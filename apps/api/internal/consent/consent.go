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
	ID                         string     `json:"_id"`
	CaseID                     string     `json:"caseId"`
	UserID                     string     `json:"userId"`
	ConsentVersion             string     `json:"consentVersion"`
	DocumentProcessingConsent  bool       `json:"documentProcessingConsent"`
	ReviewerAccessConsent      bool       `json:"reviewerAccessConsent"`
	DataRetentionConsent       bool       `json:"dataRetentionConsent"`
	AuthorityConfirmation      bool       `json:"authorityConfirmation"`
	ResearchConsent            bool       `json:"researchConsent"`
	IPAddress                  string     `json:"ipAddress"`
	UserAgent                  string     `json:"userAgent"`
	WithdrawnAt                *time.Time `json:"withdrawnAt,omitempty"`
	CreatedAt                  time.Time  `json:"createdAt"`
}

type RecordConsentRequest struct {
	ConsentVersion            string `json:"consent_version"`
	DocumentProcessingConsent bool   `json:"document_processing_consent"`
	ReviewerAccessConsent     bool   `json:"reviewer_access_consent"`
	DataRetentionConsent      bool   `json:"data_retention_consent"`
	AuthorityConfirmation     bool   `json:"authority_confirmation"`
	ResearchConsent           bool   `json:"research_consent"`
}

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

type ConvexCase struct {
	ID          string `json:"_id"`
	OwnerUserID string `json:"ownerUserId"`
	LegacyID    string `json:"legacyId"`
}

func (s *Service) RecordConsent(w http.ResponseWriter, r *http.Request) {
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

	// Verify case ownership
	var caseItem ConvexCase
	err := s.db.CallQuery(r.Context(), "cases:getByLegacyId", map[string]interface{}{"legacyId": caseID}, &caseItem)
	if err != nil || caseItem.ID == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Case not found")
		return
	}
	if caseItem.OwnerUserID != user.ID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden: you do not own this case")
		return
	}

	var req RecordConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input payload")
		return
	}

	consentID := uuid.New().String()
	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()

	consentArgs := map[string]interface{}{
		"caseId":                     caseID,
		"userId":                     user.ID,
		"consentVersion":             req.ConsentVersion,
		"documentProcessingConsent":  req.DocumentProcessingConsent,
		"reviewerAccessConsent":      req.ReviewerAccessConsent,
		"dataRetentionConsent":       req.DataRetentionConsent,
		"authorityConfirmation":      req.AuthorityConfirmation,
		"researchConsent":            req.ResearchConsent,
		"ipAddress":                  ipAddress,
		"userAgent":                  userAgent,
		"legacyId":                   consentID,
	}

	var resID string
	err = s.db.CallMutation(r.Context(), "consents:record", consentArgs, &resID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to record consent: "+err.Error())
		return
	}

	// Transition case state if consent is positive
	if req.DocumentProcessingConsent && req.AuthorityConfirmation {
		updateArgs := map[string]interface{}{
			"legacyId":      caseID,
			"workflowState": "DOCUMENTS_PENDING",
		}
		var updatedCaseID string
		_ = s.db.CallMutation(r.Context(), "cases:update", updateArgs, &updatedCaseID)

		// Record timeline event
		timelineArgs := map[string]interface{}{
			"caseId":    caseID,
			"fromState": "DRAFT",
			"toState":   "DOCUMENTS_PENDING",
			"changedBy": user.ID,
			"reason":    "Consent granted by user",
		}
		var timelineID string
		_ = s.db.CallMutation(r.Context(), "cases:logTimeline", timelineArgs, &timelineID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Consent recorded successfully",
		"consent_id": consentID,
	})
}

func (s *Service) GetConsents(w http.ResponseWriter, r *http.Request) {
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

	var caseItem ConvexCase
	err := s.db.CallQuery(r.Context(), "cases:getByLegacyId", map[string]interface{}{"legacyId": caseID}, &caseItem)
	if err != nil || caseItem.ID == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Case not found")
		return
	}

	if user.Role == "POLICYHOLDER" && caseItem.OwnerUserID != user.ID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden: you do not own this case")
		return
	}

	var consents []Consent
	err = s.db.CallQuery(r.Context(), "consents:listByCase", map[string]interface{}{"caseId": caseID}, &consents)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load consents")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(consents)
}

func (s *Service) WithdrawConsent(w http.ResponseWriter, r *http.Request) {
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

	var caseItem ConvexCase
	err := s.db.CallQuery(r.Context(), "cases:getByLegacyId", map[string]interface{}{"legacyId": caseID}, &caseItem)
	if err != nil || caseItem.ID == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Case not found")
		return
	}
	if caseItem.OwnerUserID != user.ID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden: you do not own this case")
		return
	}

	var success bool
	err = s.db.CallMutation(r.Context(), "consents:withdraw", map[string]interface{}{"caseId": caseID}, &success)
	if err != nil || !success {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to withdraw consent")
		return
	}

	updateArgs := map[string]interface{}{
		"legacyId":      caseID,
		"workflowState": "CLOSED",
		"closedAt":      time.Now().Format(time.RFC3339),
	}
	var updatedCaseID string
	_ = s.db.CallMutation(r.Context(), "cases:update", updateArgs, &updatedCaseID)

	timelineArgs := map[string]interface{}{
		"caseId":    caseID,
		"fromState": "ACTIVE", // Or whichever state it was in, default to active timeline trace representation
		"toState":   "CLOSED",
		"changedBy": user.ID,
		"reason":    "Consent withdrawn by user",
	}
	var timelineID string
	_ = s.db.CallMutation(r.Context(), "cases:logTimeline", timelineArgs, &timelineID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Consent successfully withdrawn and case closed"})
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
