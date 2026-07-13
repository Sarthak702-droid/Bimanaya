package consent

import (
	"encoding/json"
	"net/http"
	"time"

	"bimanyaya/api/internal/auth"
	"bimanyaya/api/internal/db"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Consent struct {
	ID                         string     `json:"id"`
	CaseID                     string     `json:"case_id"`
	UserID                     string     `json:"user_id"`
	ConsentVersion             string     `json:"consent_version"`
	DocumentProcessingConsent  bool       `json:"document_processing_consent"`
	ReviewerAccessConsent      bool       `json:"reviewer_access_consent"`
	DataRetentionConsent       bool       `json:"data_retention_consent"`
	AuthorityConfirmation      bool       `json:"authority_confirmation"`
	ResearchConsent            bool       `json:"research_consent"`
	IPAddress                  string     `json:"ip_address"`
	UserAgent                  string     `json:"user_agent"`
	WithdrawnAt                *time.Time `json:"withdrawn_at,omitempty"`
	CreatedAt                  time.Time  `json:"created_at"`
}

type RecordConsentRequest struct {
	ConsentVersion             string `json:"consent_version"`
	DocumentProcessingConsent  bool   `json:"document_processing_consent"`
	ReviewerAccessConsent      bool   `json:"reviewer_access_consent"`
	DataRetentionConsent       bool   `json:"data_retention_consent"`
	AuthorityConfirmation      bool   `json:"authority_confirmation"`
	ResearchConsent            bool   `json:"research_consent"`
}

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

func (s *Service) RecordConsent(w http.ResponseWriter, r *http.Request) {
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

	// Verify case ownership
	var ownerID string
	err := s.db.Pool.QueryRow(r.Context(), "SELECT owner_user_id FROM cases WHERE id = $1", caseID).Scan(&ownerID)
	if err != nil {
		http.Error(w, "Case not found", http.StatusNotFound)
		return
	}
	if ownerID != user.ID {
		http.Error(w, "Forbidden: you do not own this case", http.StatusForbidden)
		return
	}

	var req RecordConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	consentID := uuid.New().String()
	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()

	_, err = s.db.Pool.Exec(r.Context(),
		`INSERT INTO consents (
			id, case_id, user_id, consent_version, document_processing_consent, 
			reviewer_access_consent, data_retention_consent, authority_confirmation, 
			research_consent, ip_address, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		consentID, caseID, user.ID, req.ConsentVersion, req.DocumentProcessingConsent,
		req.ReviewerAccessConsent, req.DataRetentionConsent, req.AuthorityConfirmation,
		req.ResearchConsent, ipAddress, userAgent,
	)

	if err != nil {
		http.Error(w, "Failed to record consent", http.StatusInternalServerError)
		return
	}

	// Transition case state if consent is positive
	if req.DocumentProcessingConsent && req.AuthorityConfirmation {
		_, _ = s.db.Pool.Exec(r.Context(),
			"UPDATE cases SET workflow_state = 'DOCUMENTS_PENDING', updated_at = NOW() WHERE id = $1", caseID)
		
		// Record timeline event
		_, _ = s.db.Pool.Exec(r.Context(),
			"INSERT INTO case_status_history (case_id, from_state, to_state, changed_by, reason) VALUES ($1, $2, $3, $4, $5)",
			caseID, "DRAFT", "DOCUMENTS_PENDING", user.ID, "Consent granted by user",
		)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Consent recorded successfully",
		"consent_id": consentID,
	})
}

func (s *Service) GetConsents(w http.ResponseWriter, r *http.Request) {
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

	// Verify case ownership or reviewer access
	var ownerID string
	err := s.db.Pool.QueryRow(r.Context(), "SELECT owner_user_id FROM cases WHERE id = $1", caseID).Scan(&ownerID)
	if err != nil {
		http.Error(w, "Case not found", http.StatusNotFound)
		return
	}
	if user.Role == "POLICYHOLDER" && ownerID != user.ID {
		http.Error(w, "Forbidden: you do not own this case", http.StatusForbidden)
		return
	}

	rows, err := s.db.Pool.Query(r.Context(),
		`SELECT id, case_id, user_id, consent_version, document_processing_consent, 
		reviewer_access_consent, data_retention_consent, authority_confirmation, 
		research_consent, COALESCE(ip_address, ''), COALESCE(user_agent, ''), withdrawn_at, created_at 
		FROM consents WHERE case_id = $1 ORDER BY created_at DESC`, caseID)
	if err != nil {
		http.Error(w, "Failed to load consents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	consents := make([]Consent, 0)
	for rows.Next() {
		var c Consent
		err = rows.Scan(
			&c.ID, &c.CaseID, &c.UserID, &c.ConsentVersion, &c.DocumentProcessingConsent,
			&c.ReviewerAccessConsent, &c.DataRetentionConsent, &c.AuthorityConfirmation,
			&c.ResearchConsent, &c.IPAddress, &c.UserAgent, &c.WithdrawnAt, &c.CreatedAt,
		)
		if err != nil {
			http.Error(w, "Failed parsing consent entry", http.StatusInternalServerError)
			return
		}
		consents = append(consents, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(consents)
}

func (s *Service) WithdrawConsent(w http.ResponseWriter, r *http.Request) {
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

	var ownerID string
	err := s.db.Pool.QueryRow(r.Context(), "SELECT owner_user_id FROM cases WHERE id = $1", caseID).Scan(&ownerID)
	if err != nil {
		http.Error(w, "Case not found", http.StatusNotFound)
		return
	}
	if ownerID != user.ID {
		http.Error(w, "Forbidden: you do not own this case", http.StatusForbidden)
		return
	}

	// Set withdrawn_at timestamp on all active consents for this case
	_, err = s.db.Pool.Exec(r.Context(),
		"UPDATE consents SET withdrawn_at = NOW() WHERE case_id = $1 AND withdrawn_at IS NULL", caseID)
	if err != nil {
		http.Error(w, "Failed to withdraw consent", http.StatusInternalServerError)
		return
	}

	// Update workflow state to close case on consent withdrawal
	_, _ = s.db.Pool.Exec(r.Context(),
		"UPDATE cases SET workflow_state = 'CLOSED', closed_at = NOW(), updated_at = NOW() WHERE id = $1", caseID)

	_, _ = s.db.Pool.Exec(r.Context(),
		"INSERT INTO case_status_history (case_id, from_state, to_state, changed_by, reason) VALUES ($1, $2, $3, $4, $5)",
		caseID, "ACTIVE", "CLOSED", user.ID, "Consent withdrawn by user",
	)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Consent successfully withdrawn and case closed"})
}
