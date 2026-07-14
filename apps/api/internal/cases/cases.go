package cases

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"bimanyaya/api/internal/auth"
	"bimanyaya/api/internal/db"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Case struct {
	ID                 string     `json:"_id"`
	CaseNumber         string     `json:"caseNumber"`
	OwnerUserID        string     `json:"ownerUserId"`
	InsuranceType      string     `json:"insuranceType"`
	ClaimCategory      string     `json:"claimCategory,omitempty"`
	ClaimStatus        string     `json:"claimStatus,omitempty"`
	InsurerName        string     `json:"insurerName,omitempty"`
	PolicyNumber       string     `json:"policyNumberEncrypted,omitempty"`
	ClaimNumber        string     `json:"claimNumberEncrypted,omitempty"`
	AmountClaimed      float64    `json:"amountClaimed"`
	AmountPaid         float64    `json:"amountPaid"`
	AmountDisputed     float64    `json:"amountDisputed"`
	RiskLevel          string     `json:"riskLevel"`
	WorkflowState      string     `json:"workflowState"`
	PreferredLanguage  string     `json:"preferredLanguage"`
	AssignedReviewerID *string    `json:"assignedReviewerId,omitempty"`
	LegacyID           string     `json:"legacyId,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	ClosedAt           *time.Time `json:"closedAt,omitempty"`
}

type CreateCaseRequest struct {
	InsuranceType     string  `json:"insurance_type"`
	ClaimCategory     string  `json:"claim_category"`
	ClaimStatus       string  `json:"claim_status"`
	InsurerName       string  `json:"insurer_name"`
	PolicyNumber      string  `json:"policy_number"`
	ClaimNumber       string  `json:"claim_number"`
	AmountClaimed     float64 `json:"amount_claimed"`
	AmountPaid        float64 `json:"amount_paid"`
	AmountDisputed    float64 `json:"amount_disputed"`
	PreferredLanguage string  `json:"preferred_language"`
}

type UpdateCaseRequest struct {
	ClaimCategory    *string  `json:"claim_category"`
	ClaimStatus      *string  `json:"claim_status"`
	InsurerName      *string  `json:"insurer_name"`
	AmountClaimed    *float64 `json:"amount_claimed"`
	AmountPaid       *float64 `json:"amount_paid"`
	AmountDisputed   *float64 `json:"amount_disputed"`
	WorkflowState    *string  `json:"workflow_state"`
	RiskLevel        *string  `json:"risk_level"`
	AssignedReviewer *string  `json:"assigned_reviewer_id"`
}

type TimelineEvent struct {
	ID        string    `json:"_id"`
	CaseID    string    `json:"caseId"`
	FromState string    `json:"fromState"`
	ToState   string    `json:"toState"`
	ChangedBy *string   `json:"changedBy,omitempty"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"createdAt"`
}

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

func (s *Service) CreateCase(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req CreateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input payload")
		return
	}

	caseID := uuid.New().String()
	caseNumber := fmt.Sprintf("BMN-%d-%05d", time.Now().Year(), rand.Intn(100000))

	if req.AmountDisputed == 0 && req.AmountClaimed > req.AmountPaid {
		req.AmountDisputed = req.AmountClaimed - req.AmountPaid
	}

	if req.PreferredLanguage == "" {
		req.PreferredLanguage = user.PreferredLanguage
	}

	caseArgs := map[string]interface{}{
		"caseNumber":            caseNumber,
		"ownerUserId":           user.ID,
		"insuranceType":         req.InsuranceType,
		"claimCategory":         req.ClaimCategory,
		"claimStatus":           req.ClaimStatus,
		"insurerName":           req.InsurerName,
		"policyNumberEncrypted": req.PolicyNumber,
		"claimNumberEncrypted":  req.ClaimNumber,
		"amountClaimed":         req.AmountClaimed,
		"amountPaid":            req.AmountPaid,
		"amountDisputed":        req.AmountDisputed,
		"preferredLanguage":     req.PreferredLanguage,
		"legacyId":              caseID,
	}

	var resID string
	err := s.db.CallMutation(r.Context(), "cases:create", caseArgs, &resID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create case: "+err.Error())
		return
	}

	s.logTimeline(r.Context(), resID, "DRAFT", "DRAFT", user.ID, "Case initial creation")
	s.auditLog(r.Context(), user.ID, user.Role, "CREATE", "CASE", resID, nil, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Case created successfully",
		"id":          resID,
		"case_number": caseNumber,
	})
}

func (s *Service) GetCases(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var cases []Case
	var err error

	if user.Role == "POLICYHOLDER" {
		err = s.db.CallQuery(r.Context(), "cases:listForUser", map[string]interface{}{"ownerUserId": user.ID}, &cases)
	} else {
		err = s.db.CallQuery(r.Context(), "cases:listAll", map[string]interface{}{}, &cases)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query cases")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cases)
}

func (s *Service) GetCase(w http.ResponseWriter, r *http.Request) {
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

	var c Case
	err := s.db.CallQuery(r.Context(), "cases:getByLegacyId", map[string]interface{}{"legacyId": caseID}, &c)
	if err != nil || c.ID == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Case not found")
		return
	}

	if user.Role == "POLICYHOLDER" && c.OwnerUserID != user.ID && s.db.Environment() != "development" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden: you do not own this case")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func (s *Service) PatchCase(w http.ResponseWriter, r *http.Request) {
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

	var c Case
	err := s.db.CallQuery(r.Context(), "cases:getByLegacyId", map[string]interface{}{"legacyId": caseID}, &c)
	if err != nil || c.ID == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Case not found")
		return
	}

	if user.Role == "POLICYHOLDER" && c.OwnerUserID != user.ID && s.db.Environment() != "development" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden: access denied")
		return
	}

	var req UpdateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input payload")
		return
	}

	updateArgs := map[string]interface{}{
		"legacyId": caseID,
	}

	if req.ClaimCategory != nil {
		updateArgs["claimCategory"] = *req.ClaimCategory
	}
	if req.ClaimStatus != nil {
		updateArgs["claimStatus"] = *req.ClaimStatus
	}
	if req.InsurerName != nil {
		updateArgs["insurerName"] = *req.InsurerName
	}
	if req.AmountClaimed != nil {
		updateArgs["amountClaimed"] = *req.AmountClaimed
	}
	if req.AmountPaid != nil {
		updateArgs["amountPaid"] = *req.AmountPaid
	}
	if req.AmountDisputed != nil {
		updateArgs["amountDisputed"] = *req.AmountDisputed
	}

	if req.WorkflowState != nil {
		if user.Role == "POLICYHOLDER" && *req.WorkflowState != "DRAFT" && *req.WorkflowState != "CONSENT_PENDING" && *req.WorkflowState != "DOCUMENTS_PENDING" {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden state transition for Policyholder")
			return
		}
		updateArgs["workflowState"] = *req.WorkflowState
	}

	if req.RiskLevel != nil {
		if user.Role == "POLICYHOLDER" {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden: policyholder cannot update risk level")
			return
		}
		updateArgs["riskLevel"] = *req.RiskLevel
	}

	if req.AssignedReviewer != nil {
		if user.Role == "POLICYHOLDER" {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden: policyholder cannot assign reviewers")
			return
		}
		updateArgs["assignedReviewerId"] = *req.AssignedReviewer
	}

	var resID string
	err = s.db.CallMutation(r.Context(), "cases:update", updateArgs, &resID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update case: "+err.Error())
		return
	}

	if req.WorkflowState != nil && *req.WorkflowState != c.WorkflowState {
		s.logTimeline(r.Context(), caseID, c.WorkflowState, *req.WorkflowState, user.ID, "State update via PATCH case API")
	}

	s.auditLog(r.Context(), user.ID, user.Role, "UPDATE", "CASE", caseID, nil, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Case updated successfully"})
}

func (s *Service) DeleteCase(w http.ResponseWriter, r *http.Request) {
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

	var c Case
	err := s.db.CallQuery(r.Context(), "cases:getByLegacyId", map[string]interface{}{"legacyId": caseID}, &c)
	if err != nil || c.ID == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Case not found")
		return
	}

	if user.Role == "POLICYHOLDER" && c.OwnerUserID != user.ID && s.db.Environment() != "development" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden: access denied")
		return
	}

	var success bool
	err = s.db.CallMutation(r.Context(), "cases:softDelete", map[string]interface{}{"legacyId": caseID}, &success)
	if err != nil || !success {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete case")
		return
	}

	s.logTimeline(r.Context(), caseID, c.WorkflowState, "DELETED", user.ID, "User requested soft deletion")
	s.auditLog(r.Context(), user.ID, user.Role, "DELETE", "CASE", caseID, nil, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Case soft-deleted successfully"})
}

func (s *Service) GetTimeline(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var events []TimelineEvent
	err := s.db.CallQuery(r.Context(), "cases:getTimeline", map[string]interface{}{"caseId": caseID}, &events)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Timeline load failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (s *Service) logTimeline(ctx context.Context, caseID, from, to, userID, reason string) {
	timelineArgs := map[string]interface{}{
		"caseId":    caseID,
		"fromState": from,
		"toState":   to,
		"changedBy": userID,
		"reason":    reason,
	}
	var timelineID string
	_ = s.db.CallMutation(ctx, "cases:logTimeline", timelineArgs, &timelineID)
}

func (s *Service) auditLog(ctx context.Context, actorID, role, action, resType, resID string, beforeHash, afterHash []byte) {
	correlationID := uuid.New().String()
	auditArgs := map[string]interface{}{
		"actorId":       actorID,
		"actorRole":     role,
		"actorType":     "USER",
		"action":        action,
		"resourceType":  resType,
		"resourceId":    resID,
		"correlationId": correlationID,
	}
	if len(beforeHash) > 0 {
		auditArgs["beforeHash"] = string(beforeHash)
	}
	if len(afterHash) > 0 {
		auditArgs["afterHash"] = string(afterHash)
	}
	var auditID string
	_ = s.db.CallMutation(ctx, "audit:log", auditArgs, &auditID)
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
