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
	"github.com/jackc/pgx/v5"
)

type Case struct {
	ID                 string    `json:"id"`
	CaseNumber         string    `json:"case_number"`
	OwnerUserID        string    `json:"owner_user_id"`
	InsuranceType      string    `json:"insurance_type"`
	ClaimCategory      string    `json:"claim_category"`
	ClaimStatus        string    `json:"claim_status"`
	InsurerName        string    `json:"insurer_name"`
	PolicyNumber       string    `json:"policy_number"` // Encr/Decr simulated for demo
	ClaimNumber        string    `json:"claim_number"`
	AmountClaimed      float64   `json:"amount_claimed"`
	AmountPaid         float64   `json:"amount_paid"`
	AmountDisputed     float64   `json:"amount_disputed"`
	RiskLevel          string    `json:"risk_level"`
	WorkflowState      string    `json:"workflow_state"`
	PreferredLanguage  string    `json:"preferred_language"`
	AssignedReviewerID *string   `json:"assigned_reviewer_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	ClosedAt           *time.Time `json:"closed_at,omitempty"`
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
	ClaimCategory     *string  `json:"claim_category"`
	ClaimStatus       *string  `json:"claim_status"`
	InsurerName       *string  `json:"insurer_name"`
	AmountClaimed     *float64 `json:"amount_claimed"`
	AmountPaid        *float64 `json:"amount_paid"`
	AmountDisputed    *float64 `json:"amount_disputed"`
	WorkflowState     *string  `json:"workflow_state"`
	RiskLevel         *string  `json:"risk_level"`
	AssignedReviewer  *string  `json:"assigned_reviewer_id"`
}

type TimelineEvent struct {
	ID        string    `json:"id"`
	CaseID    string    `json:"case_id"`
	FromState string    `json:"from_state"`
	ToState   string    `json:"to_state"`
	ChangedBy *string   `json:"changed_by"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

func (s *Service) CreateCase(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	caseID := uuid.New().String()
	caseNumber := fmt.Sprintf("BMN-%d-%05d", time.Now().Year(), rand.Intn(100000))

	// Pre-calculation
	if req.AmountDisputed == 0 && req.AmountClaimed > req.AmountPaid {
		req.AmountDisputed = req.AmountClaimed - req.AmountPaid
	}

	if req.PreferredLanguage == "" {
		req.PreferredLanguage = user.PreferredLanguage
	}

	// Insert into DB
	_, err := s.db.Pool.Exec(r.Context(),
		`INSERT INTO cases (
			id, case_number, owner_user_id, insurance_type, claim_category, claim_status, 
			insurer_name, policy_number_encrypted, claim_number_encrypted, 
			amount_claimed, amount_paid, amount_disputed, risk_level, workflow_state, preferred_language
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		caseID, caseNumber, user.ID, req.InsuranceType, req.ClaimCategory, req.ClaimStatus,
		req.InsurerName, req.PolicyNumber, req.ClaimNumber, // In production, encrypt these
		req.AmountClaimed, req.AmountPaid, req.AmountDisputed, "LOW", "DRAFT", req.PreferredLanguage,
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create case: %v", err), http.StatusInternalServerError)
		return
	}

	// Log timeline event
	s.logTimeline(r.Context(), caseID, "DRAFT", "DRAFT", user.ID, "Case initial creation")

	// Audit log
	s.auditLog(r.Context(), user.ID, user.Role, "CREATE", "CASE", caseID, nil, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":     "Case created successfully",
		"id":          caseID,
		"case_number": caseNumber,
	})
}

func (s *Service) GetCases(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var rows pgx.Rows
	var err error

	// Role limits: policyholders only see their own cases, reviewers/admins see all or filter.
	if user.Role == "POLICYHOLDER" {
		rows, err = s.db.Pool.Query(r.Context(),
			`SELECT id, case_number, owner_user_id, insurance_type, COALESCE(claim_category, ''), COALESCE(claim_status, ''), 
			COALESCE(insurer_name, ''), COALESCE(policy_number_encrypted, ''), COALESCE(claim_number_encrypted, ''), 
			amount_claimed, amount_paid, amount_disputed, risk_level, workflow_state, preferred_language, assigned_reviewer_id, 
			created_at, updated_at, closed_at FROM cases WHERE owner_user_id = $1 ORDER BY created_at DESC`,
			user.ID)
	} else {
		// Reviewer / Admin sees all
		rows, err = s.db.Pool.Query(r.Context(),
			`SELECT id, case_number, owner_user_id, insurance_type, COALESCE(claim_category, ''), COALESCE(claim_status, ''), 
			COALESCE(insurer_name, ''), COALESCE(policy_number_encrypted, ''), COALESCE(claim_number_encrypted, ''), 
			amount_claimed, amount_paid, amount_disputed, risk_level, workflow_state, preferred_language, assigned_reviewer_id, 
			created_at, updated_at, closed_at FROM cases ORDER BY created_at DESC`)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query cases: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	cases := make([]Case, 0)
	for rows.Next() {
		var c Case
		err = rows.Scan(
			&c.ID, &c.CaseNumber, &c.OwnerUserID, &c.InsuranceType, &c.ClaimCategory, &c.ClaimStatus,
			&c.InsurerName, &c.PolicyNumber, &c.ClaimNumber, &c.AmountClaimed, &c.AmountPaid, &c.AmountDisputed,
			&c.RiskLevel, &c.WorkflowState, &c.PreferredLanguage, &c.AssignedReviewerID, &c.CreatedAt, &c.UpdatedAt, &c.ClosedAt,
		)
		if err != nil {
			http.Error(w, "Error parsing case data", http.StatusInternalServerError)
			return
		}
		cases = append(cases, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cases)
}

func (s *Service) GetCase(w http.ResponseWriter, r *http.Request) {
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

	var c Case
	err := s.db.Pool.QueryRow(r.Context(),
		`SELECT id, case_number, owner_user_id, insurance_type, COALESCE(claim_category, ''), COALESCE(claim_status, ''), 
		COALESCE(insurer_name, ''), COALESCE(policy_number_encrypted, ''), COALESCE(claim_number_encrypted, ''), 
		amount_claimed, amount_paid, amount_disputed, risk_level, workflow_state, preferred_language, assigned_reviewer_id, 
		created_at, updated_at, closed_at FROM cases WHERE id = $1`, caseID).Scan(
		&c.ID, &c.CaseNumber, &c.OwnerUserID, &c.InsuranceType, &c.ClaimCategory, &c.ClaimStatus,
		&c.InsurerName, &c.PolicyNumber, &c.ClaimNumber, &c.AmountClaimed, &c.AmountPaid, &c.AmountDisputed,
		&c.RiskLevel, &c.WorkflowState, &c.PreferredLanguage, &c.AssignedReviewerID, &c.CreatedAt, &c.UpdatedAt, &c.ClosedAt,
	)

	if err != nil {
		http.Error(w, "Case not found", http.StatusNotFound)
		return
	}

	// Policyholder can only see their own cases
	if user.Role == "POLICYHOLDER" && c.OwnerUserID != user.ID {
		http.Error(w, "Forbidden: you do not own this case", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func (s *Service) PatchCase(w http.ResponseWriter, r *http.Request) {
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

	// Verify existence and authorization
	var ownerID string
	var currentState string
	err := s.db.Pool.QueryRow(r.Context(), "SELECT owner_user_id, workflow_state FROM cases WHERE id = $1", caseID).Scan(&ownerID, &currentState)
	if err != nil {
		http.Error(w, "Case not found", http.StatusNotFound)
		return
	}

	if user.Role == "POLICYHOLDER" && ownerID != user.ID {
		http.Error(w, "Forbidden: access denied", http.StatusForbidden)
		return
	}

	var req UpdateCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	// Build update query dynamically
	query := "UPDATE cases SET updated_at = NOW()"
	args := []interface{}{caseID}
	argIdx := 2

	if req.ClaimCategory != nil {
		query += fmt.Sprintf(", claim_category = $%d", argIdx)
		args = append(args, *req.ClaimCategory)
		argIdx++
	}
	if req.ClaimStatus != nil {
		query += fmt.Sprintf(", claim_status = $%d", argIdx)
		args = append(args, *req.ClaimStatus)
		argIdx++
	}
	if req.InsurerName != nil {
		query += fmt.Sprintf(", insurer_name = $%d", argIdx)
		args = append(args, *req.InsurerName)
		argIdx++
	}
	if req.AmountClaimed != nil {
		query += fmt.Sprintf(", amount_claimed = $%d", argIdx)
		args = append(args, *req.AmountClaimed)
		argIdx++
	}
	if req.AmountPaid != nil {
		query += fmt.Sprintf(", amount_paid = $%d", argIdx)
		args = append(args, *req.AmountPaid)
		argIdx++
	}
	if req.AmountDisputed != nil {
		query += fmt.Sprintf(", amount_disputed = $%d", argIdx)
		args = append(args, *req.AmountDisputed)
		argIdx++
	}
	if req.WorkflowState != nil {
		// Verify workflow state rules if user is policyholder (policyholders can't transition to reviews)
		if user.Role == "POLICYHOLDER" && *req.WorkflowState != "DRAFT" && *req.WorkflowState != "CONSENT_PENDING" && *req.WorkflowState != "DOCUMENTS_PENDING" {
			http.Error(w, "Forbidden state transition for Policyholder", http.StatusForbidden)
			return
		}
		query += fmt.Sprintf(", workflow_state = $%d", argIdx)
		args = append(args, *req.WorkflowState)
		argIdx++
	}
	if req.RiskLevel != nil {
		// Only reviewer/admin can alter risk level
		if user.Role == "POLICYHOLDER" {
			http.Error(w, "Forbidden: policyholder cannot update risk level", http.StatusForbidden)
			return
		}
		query += fmt.Sprintf(", risk_level = $%d", argIdx)
		args = append(args, *req.RiskLevel)
		argIdx++
	}
	if req.AssignedReviewer != nil {
		if user.Role == "POLICYHOLDER" {
			http.Error(w, "Forbidden: policyholder cannot assign reviewers", http.StatusForbidden)
			return
		}
		query += fmt.Sprintf(", assigned_reviewer_id = $%d", argIdx)
		if *req.AssignedReviewer == "" {
			args = append(args, nil)
		} else {
			args = append(args, *req.AssignedReviewer)
		}
		argIdx++
	}

	query += " WHERE id = $1"

	_, err = s.db.Pool.Exec(r.Context(), query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update case: %v", err), http.StatusInternalServerError)
		return
	}

	// If workflow state changed, record history
	if req.WorkflowState != nil && *req.WorkflowState != currentState {
		s.logTimeline(r.Context(), caseID, currentState, *req.WorkflowState, user.ID, "State update via PATCH case API")
	}

	// Audit log
	s.auditLog(r.Context(), user.ID, user.Role, "UPDATE", "CASE", caseID, nil, nil)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Case updated successfully"})
}

func (s *Service) DeleteCase(w http.ResponseWriter, r *http.Request) {
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

	// Verify owner
	var ownerID string
	err := s.db.Pool.QueryRow(r.Context(), "SELECT owner_user_id FROM cases WHERE id = $1", caseID).Scan(&ownerID)
	if err != nil {
		http.Error(w, "Case not found", http.StatusNotFound)
		return
	}

	if user.Role == "POLICYHOLDER" && ownerID != user.ID {
		http.Error(w, "Forbidden: access denied", http.StatusForbidden)
		return
	}

	// Hard or soft delete depending on regulation. Let's do soft delete workflow_state update.
	_, err = s.db.Pool.Exec(r.Context(), "UPDATE cases SET workflow_state = 'DELETED', closed_at = NOW(), updated_at = NOW() WHERE id = $1", caseID)
	if err != nil {
		http.Error(w, "Failed to delete case", http.StatusInternalServerError)
		return
	}

	s.logTimeline(r.Context(), caseID, "ACTIVE", "DELETED", user.ID, "User requested soft deletion")
	s.auditLog(r.Context(), user.ID, user.Role, "DELETE", "CASE", caseID, nil, nil)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Case soft-deleted successfully"})
}

func (s *Service) GetTimeline(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	rows, err := s.db.Pool.Query(r.Context(),
		`SELECT id, case_id, from_state, to_state, changed_by, COALESCE(reason, ''), created_at 
		FROM case_status_history WHERE case_id = $1 ORDER BY created_at ASC`, caseID)
	if err != nil {
		http.Error(w, "Timeline load failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	events := make([]TimelineEvent, 0)
	for rows.Next() {
		var ev TimelineEvent
		err = rows.Scan(&ev.ID, &ev.CaseID, &ev.FromState, &ev.ToState, &ev.ChangedBy, &ev.Reason, &ev.CreatedAt)
		if err != nil {
			http.Error(w, "Error reading history", http.StatusInternalServerError)
			return
		}
		events = append(events, ev)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (s *Service) logTimeline(ctx context.Context, caseID, from, to, userID, reason string) {
	_, _ = s.db.Pool.Exec(ctx,
		"INSERT INTO case_status_history (case_id, from_state, to_state, changed_by, reason) VALUES ($1, $2, $3, $4, $5)",
		caseID, from, to, userID, reason)
}

func (s *Service) auditLog(ctx context.Context, actorID, role, action, resType, resID string, beforeHash, afterHash []byte) {
	correlationID := uuid.New().String()
	_, _ = s.db.Pool.Exec(ctx,
		`INSERT INTO audit_events (
			actor_id, actor_role, action, resource_type, resource_id, before_hash, after_hash, correlation_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		actorID, role, action, resType, resID, beforeHash, afterHash, correlationID,
	)
}
