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

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// GetReviewerCases lists cases based on reviewer assignments
func (s *Service) GetReviewerCases(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if user.Role != "REVIEWER" && user.Role != "SENIOR_REVIEWER" && user.Role != "ADMIN" {
		http.Error(w, "Forbidden: reviewers only", http.StatusForbidden)
		return
	}

	// Fetch all cases in REVIEW_REQUIRED or IN_REVIEW state
	rows, err := s.db.Pool.Query(r.Context(),
		`SELECT id, case_number, owner_user_id, insurance_type, COALESCE(claim_category,''), COALESCE(claim_status,''), 
		amount_claimed, amount_paid, amount_disputed, risk_level, workflow_state, assigned_reviewer_id, created_at 
		FROM cases WHERE workflow_state IN ('REVIEW_REQUIRED', 'IN_REVIEW', 'MORE_INFORMATION_REQUIRED') ORDER BY created_at ASC`)
	if err != nil {
		http.Error(w, "Failed to load queue", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type SimpleCase struct {
		ID                 string    `json:"id"`
		CaseNumber         string    `json:"case_number"`
		OwnerUserID        string    `json:"owner_user_id"`
		InsuranceType      string    `json:"insurance_type"`
		ClaimCategory      string    `json:"claim_category"`
		ClaimStatus        string    `json:"claim_status"`
		AmountClaimed      float64   `json:"amount_claimed"`
		AmountPaid         float64   `json:"amount_paid"`
		AmountDisputed     float64   `json:"amount_disputed"`
		RiskLevel          string    `json:"risk_level"`
		WorkflowState      string    `json:"workflow_state"`
		AssignedReviewerID *string   `json:"assigned_reviewer_id,omitempty"`
		CreatedAt          time.Time `json:"created_at"`
	}

	cases := make([]SimpleCase, 0)
	for rows.Next() {
		var c SimpleCase
		err = rows.Scan(
			&c.ID, &c.CaseNumber, &c.OwnerUserID, &c.InsuranceType, &c.ClaimCategory, &c.ClaimStatus,
			&c.AmountClaimed, &c.AmountPaid, &c.AmountDisputed, &c.RiskLevel, &c.WorkflowState,
			&c.AssignedReviewerID, &c.CreatedAt,
		)
		if err == nil {
			cases = append(cases, c)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cases)
}

func (s *Service) ClaimCase(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Transaction error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	// Check if already claimed
	var currentReviewer *string
	var currentState string
	err = tx.QueryRow(r.Context(), "SELECT assigned_reviewer_id, workflow_state FROM cases WHERE id = $1", caseID).Scan(&currentReviewer, &currentState)
	if err != nil {
		http.Error(w, "Case not found", http.StatusNotFound)
		return
	}

	if currentReviewer != nil && *currentReviewer != "" {
		http.Error(w, "Case already claimed by another reviewer", http.StatusConflict)
		return
	}

	// Update assignment and workflow state
	_, err = tx.Exec(r.Context(),
		"UPDATE cases SET assigned_reviewer_id = $1, workflow_state = 'IN_REVIEW', updated_at = NOW() WHERE id = $2",
		user.ID, caseID,
	)
	if err != nil {
		http.Error(w, "Failed to claim case", http.StatusInternalServerError)
		return
	}

	// Record timeline
	_, _ = tx.Exec(r.Context(),
		"INSERT INTO case_status_history (case_id, from_state, to_state, changed_by, reason) VALUES ($1, $2, $3, $4, $5)",
		caseID, currentState, "IN_REVIEW", user.ID, "Case claimed by reviewer",
	)

	// Create a review SLA entry (e.g. 48 hours SLA)
	reviewID := uuid.New().String()
	slaDue := time.Now().Add(48 * time.Hour)
	_, _ = tx.Exec(r.Context(),
		`INSERT INTO reviews (id, case_id, reviewer_id, decision, comments, started_at, sla_due_at) 
		VALUES ($1, $2, $3, $4, $5, NOW(), $6)`,
		reviewID, caseID, user.ID, "CLAIMED", "Claimed review queue item", slaDue,
	)

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Commit failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Case successfully claimed"})
}

func (s *Service) RequestInformation(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	var req ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	// Transition state to NEEDS_CLARIFICATION
	_, err = tx.Exec(r.Context(),
		"UPDATE cases SET workflow_state = 'NEEDS_CLARIFICATION', updated_at = NOW() WHERE id = $1 AND assigned_reviewer_id = $2",
		caseID, user.ID,
	)
	if err != nil {
		http.Error(w, "Failed to update case state or unauthorized reviewer", http.StatusInternalServerError)
		return
	}

	// Insert reviewer clarification question
	questionID := uuid.New().String()
	_, err = tx.Exec(r.Context(),
		`INSERT INTO clarification_questions (id, case_id, question_type, question_text, context_explanation) 
		VALUES ($1, $2, 'TEXT', $3, 'Question raised during manual human review')`,
		questionID, caseID, req.Comments,
	)
	if err != nil {
		http.Error(w, "Failed to register clarification question", http.StatusInternalServerError)
		return
	}

	// Update review status
	_, _ = tx.Exec(r.Context(),
		"UPDATE reviews SET decision = 'NEEDS_INFO', comments = $1, completed_at = NOW() WHERE case_id = $2 AND reviewer_id = $3 AND completed_at IS NULL",
		req.Comments, caseID, user.ID,
	)

	// Timeline
	_, _ = tx.Exec(r.Context(),
		"INSERT INTO case_status_history (case_id, from_state, to_state, changed_by, reason) VALUES ($1, $2, $3, $4, $5)",
		caseID, "IN_REVIEW", "NEEDS_CLARIFICATION", user.ID, "Reviewer requested additional clarification details: "+req.Comments,
	)

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Failed to commit information request", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Clarification successfully requested from policyholder"})
}

func (s *Service) ApproveCase(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	// Update case state to APPROVED
	_, err = tx.Exec(r.Context(),
		"UPDATE cases SET workflow_state = 'APPROVED', updated_at = NOW() WHERE id = $1 AND assigned_reviewer_id = $2",
		caseID, user.ID,
	)
	if err != nil {
		http.Error(w, "Failed to update case state or unauthorized reviewer", http.StatusInternalServerError)
		return
	}

	// Update draft approval details
	_, err = tx.Exec(r.Context(),
		"UPDATE drafts SET status = 'APPROVED', approved_by = $1, updated_at = NOW() WHERE case_id = $2",
		user.ID, caseID,
	)
	if err != nil {
		http.Error(w, "Failed to update draft approval details", http.StatusInternalServerError)
		return
	}

	// Finalize active review log
	_, err = tx.Exec(r.Context(),
		"UPDATE reviews SET decision = 'APPROVED', completed_at = NOW() WHERE case_id = $1 AND reviewer_id = $2 AND completed_at IS NULL",
		caseID, user.ID,
	)
	if err != nil {
		http.Error(w, "Failed to finalize active review log", http.StatusInternalServerError)
		return
	}

	// Timeline
	_, err = tx.Exec(r.Context(),
		"INSERT INTO case_status_history (case_id, from_state, to_state, changed_by, reason) VALUES ($1, $2, $3, $4, $5)",
		caseID, "IN_REVIEW", "APPROVED", user.ID, "Reviewer approved draft and claim grievance blueprint",
	)
	if err != nil {
		http.Error(w, "Failed to record timeline history", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Failed to commit approval", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Case and grievance draft approved successfully"})
}

func (s *Service) EscalateCase(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	var req ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	// Escalate case: reset assignment, raise risk to HIGH/CRITICAL
	_, err = tx.Exec(r.Context(),
		`UPDATE cases SET 
			assigned_reviewer_id = NULL, 
			risk_level = 'CRITICAL', 
			workflow_state = 'REVIEW_REQUIRED',
			updated_at = NOW() 
		WHERE id = $1 AND assigned_reviewer_id = $2`,
		caseID, user.ID,
	)
	if err != nil {
		http.Error(w, "Failed to escalate case", http.StatusInternalServerError)
		return
	}

	// Complete junior reviewer log
	_, _ = tx.Exec(r.Context(),
		"UPDATE reviews SET decision = 'ESCALATED', comments = $1, completed_at = NOW() WHERE case_id = $2 AND reviewer_id = $3 AND completed_at IS NULL",
		req.Comments, caseID, user.ID,
	)

	// Timeline
	_, _ = tx.Exec(r.Context(),
		"INSERT INTO case_status_history (case_id, from_state, to_state, changed_by, reason) VALUES ($1, $2, $3, $4, $5)",
		caseID, "IN_REVIEW", "REVIEW_REQUIRED", user.ID, "Reviewer escalated case to Senior Reviewer pool: "+req.Comments,
	)

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Failed to commit escalation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Case escalated to Senior Reviewer pool successfully"})
}

func (s *Service) RejectCase(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	var req ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	// Reject case/AI findings: shift case to CLOSED
	_, err = tx.Exec(r.Context(),
		"UPDATE cases SET workflow_state = 'CLOSED', closed_at = NOW(), updated_at = NOW() WHERE id = $1 AND assigned_reviewer_id = $2",
		caseID, user.ID,
	)
	if err != nil {
		http.Error(w, "Failed to reject case or unauthorized reviewer", http.StatusInternalServerError)
		return
	}

	// Update draft status
	_, err = tx.Exec(r.Context(),
		"UPDATE drafts SET status = 'REJECTED', updated_at = NOW() WHERE case_id = $1",
		caseID,
	)
	if err != nil {
		http.Error(w, "Failed to reject draft", http.StatusInternalServerError)
		return
	}

	// Complete review log
	_, err = tx.Exec(r.Context(),
		"UPDATE reviews SET decision = 'REJECTED', comments = $1, completed_at = NOW() WHERE case_id = $2 AND reviewer_id = $3 AND completed_at IS NULL",
		req.Comments, caseID, user.ID,
	)
	if err != nil {
		http.Error(w, "Failed to complete review log", http.StatusInternalServerError)
		return
	}

	// Timeline
	_, err = tx.Exec(r.Context(),
		"INSERT INTO case_status_history (case_id, from_state, to_state, changed_by, reason) VALUES ($1, $2, $3, $4, $5)",
		caseID, "IN_REVIEW", "CLOSED", user.ID, "Reviewer rejected grievance case details: "+req.Comments,
	)
	if err != nil {
		http.Error(w, "Failed to write status history timeline", http.StatusInternalServerError)
		return
	}


	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Failed to commit rejection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Case closed with rejection state"})
}

// AddReviewComment adds a manual feedback comment to a case being reviewed.
func (s *Service) AddReviewComment(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	var req struct {
		CommentText string `json:"comment_text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	if req.CommentText == "" {
		http.Error(w, "Comment text cannot be empty", http.StatusBadRequest)
		return
	}

	commentID := uuid.New().String()
	_, err := s.db.Pool.Exec(r.Context(),
		`INSERT INTO review_comments (id, case_id, reviewer_id, comment_text, created_at) 
		VALUES ($1, $2, $3, $4, NOW())`,
		commentID, caseID, user.ID, req.CommentText,
	)
	if err != nil {
		http.Error(w, "Failed to insert review comment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"comment_id": commentID,
		"message":    "Review comment added successfully",
	})
}

// GetReviewComments retrieves all review feedback comments for a case.
func (s *Service) GetReviewComments(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	rows, err := s.db.Pool.Query(r.Context(),
		`SELECT id, case_id, reviewer_id, comment_text, created_at 
		FROM review_comments WHERE case_id = $1 ORDER BY created_at ASC`, caseID)
	if err != nil {
		http.Error(w, "Failed to load review comments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type SimpleComment struct {
		ID          string    `json:"id"`
		CaseID      string    `json:"case_id"`
		ReviewerID  string    `json:"reviewer_id"`
		CommentText string    `json:"comment_text"`
		CreatedAt   time.Time `json:"created_at"`
	}

	comments := make([]SimpleComment, 0)
	for rows.Next() {
		var c SimpleComment
		err = rows.Scan(&c.ID, &c.CaseID, &c.ReviewerID, &c.CommentText, &c.CreatedAt)
		if err == nil {
			comments = append(comments, c)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

// CheckReviewSLAs scans for cases that have exceeded their 48hr SLA deadline and escalates them automatically.
func (s *Service) CheckReviewSLAs(w http.ResponseWriter, r *http.Request) {
	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	// Fetch active reviews that have breached their SLA
	rows, err := tx.Query(r.Context(),
		`SELECT id, case_id, reviewer_id FROM reviews 
		WHERE completed_at IS NULL AND sla_due_at < NOW()`)
	if err != nil {
		http.Error(w, "Failed to scan SLA breaches", http.StatusInternalServerError)
		return
	}

	type Breach struct {
		ReviewID   string
		CaseID     string
		ReviewerID string
	}

	var breaches []Breach
	for rows.Next() {
		var b Breach
		if err := rows.Scan(&b.ReviewID, &b.CaseID, &b.ReviewerID); err == nil {
			breaches = append(breaches, b)
		}
	}
	rows.Close()

	var escalatedCaseIDs []string

	for _, b := range breaches {
		// 1. Escalate review decision
		_, err = tx.Exec(r.Context(),
			`UPDATE reviews SET decision = 'ESCALATED', comments = 'Auto-escalated due to review SLA breach', completed_at = NOW() 
			WHERE id = $1`, b.ReviewID,
		)
		if err != nil {
			continue
		}

		// 2. Reset case assignment and flag as CRITICAL risk
		_, err = tx.Exec(r.Context(),
			`UPDATE cases SET 
				assigned_reviewer_id = NULL, 
				risk_level = 'CRITICAL', 
				workflow_state = 'REVIEW_REQUIRED',
				updated_at = NOW() 
			WHERE id = $1`, b.CaseID,
		)
		if err != nil {
			continue
		}

		// 3. Log event in status history
		_, _ = tx.Exec(r.Context(),
			`INSERT INTO case_status_history (case_id, from_state, to_state, reason) 
			VALUES ($1, 'IN_REVIEW', 'REVIEW_REQUIRED', 'Review SLA deadline breached. Case auto-escalated to Senior Reviewer pool.')`,
			b.CaseID,
		)

		escalatedCaseIDs = append(escalatedCaseIDs, b.CaseID)
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Failed to commit SLA escalation changes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":             fmt.Sprintf("SLA verification completed. Escalated %d cases.", len(escalatedCaseIDs)),
		"escalated_case_ids": escalatedCaseIDs,
	})
}

