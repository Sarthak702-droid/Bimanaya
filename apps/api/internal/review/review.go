package review

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"bimanyaya/api/internal/auth"
	"bimanyaya/api/internal/db"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ReviewRequest struct {
	Comments     string `json:"comments"`
	RiskOverride string `json:"risk_override,omitempty"`
}

type SimpleCase struct {
	ID                 string    `json:"_id"`
	CaseNumber         string    `json:"caseNumber"`
	OwnerUserID        string    `json:"ownerUserId"`
	InsuranceType      string    `json:"insuranceType"`
	ClaimCategory      string    `json:"claimCategory,omitempty"`
	ClaimStatus        string    `json:"claimStatus,omitempty"`
	AmountClaimed      float64   `json:"amountClaimed"`
	AmountPaid         float64   `json:"amountPaid"`
	AmountDisputed     float64   `json:"amountDisputed"`
	RiskLevel          string    `json:"riskLevel"`
	WorkflowState      string    `json:"workflowState"`
	AssignedReviewerID *string   `json:"assignedReviewerId,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
}

type SimpleComment struct {
	ID          string    `json:"_id"`
	CaseID      string    `json:"caseId"`
	ReviewerID  string    `json:"reviewerId"`
	CommentText string    `json:"commentText"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// GetReviewerCases lists cases based on reviewer assignments
func (s *Service) GetReviewerCases(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	if user.Role != "REVIEWER" && user.Role != "SENIOR_REVIEWER" && user.Role != "ADMIN" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden: reviewers only")
		return
	}

	var cases []SimpleCase
	err := s.db.CallQuery(r.Context(), "reviews:getQueue", map[string]interface{}{}, &cases)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load queue")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cases)
}

func (s *Service) ClaimCase(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	legacyReviewID := uuid.New().String()

	var success bool
	err := s.db.CallMutation(r.Context(), "reviews:claim", map[string]interface{}{
		"caseId":         caseID,
		"reviewerId":     user.ID,
		"legacyReviewId": legacyReviewID,
	}, &success)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Case successfully claimed"})
}

func (s *Service) RequestInformation(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var req ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input payload")
		return
	}

	legacyQuestionID := uuid.New().String()

	var success bool
	err := s.db.CallMutation(r.Context(), "reviews:requestInformation", map[string]interface{}{
		"caseId":           caseID,
		"reviewerId":       user.ID,
		"comments":         req.Comments,
		"legacyQuestionId": legacyQuestionID,
	}, &success)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Clarification successfully requested from policyholder"})
}

func (s *Service) ApproveCase(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var success bool
	err := s.db.CallMutation(r.Context(), "reviews:approve", map[string]interface{}{
		"caseId":     caseID,
		"reviewerId": user.ID,
	}, &success)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Case and grievance draft approved successfully"})
}

func (s *Service) EscalateCase(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var req ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input payload")
		return
	}

	var success bool
	err := s.db.CallMutation(r.Context(), "reviews:escalate", map[string]interface{}{
		"caseId":     caseID,
		"reviewerId": user.ID,
		"comments":   req.Comments,
	}, &success)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Case escalated to Senior Reviewer pool successfully"})
}

func (s *Service) RejectCase(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var req ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input payload")
		return
	}

	var success bool
	err := s.db.CallMutation(r.Context(), "reviews:reject", map[string]interface{}{
		"caseId":     caseID,
		"reviewerId": user.ID,
		"comments":   req.Comments,
	}, &success)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Case closed with rejection state"})
}

func (s *Service) AddReviewComment(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var req struct {
		CommentText string `json:"comment_text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input payload")
		return
	}

	if req.CommentText == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Comment text cannot be empty")
		return
	}

	commentID := uuid.New().String()

	var success bool
	err := s.db.CallMutation(r.Context(), "reviews:addComment", map[string]interface{}{
		"caseId":           caseID,
		"reviewerId":       user.ID,
		"commentText":      req.CommentText,
		"legacyId":         commentID,
	}, &success)

	if err != nil || !success {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to insert review comment")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"comment_id": commentID,
		"message":    "Review comment added successfully",
	})
}

func (s *Service) GetReviewComments(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var comments []SimpleComment
	err := s.db.CallQuery(r.Context(), "reviews:listComments", map[string]interface{}{"caseId": caseID}, &comments)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load review comments")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func (s *Service) CheckReviewSLAs(w http.ResponseWriter, r *http.Request) {
	var escalatedCaseIDs []string
	err := s.db.CallMutation(r.Context(), "reviews:checkSLAs", map[string]interface{}{}, &escalatedCaseIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to run SLA checks: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":            fmt.Sprintf("SLA verification completed. Escalated %d cases.", len(escalatedCaseIDs)),
		"escalated_case_ids": escalatedCaseIDs,
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
