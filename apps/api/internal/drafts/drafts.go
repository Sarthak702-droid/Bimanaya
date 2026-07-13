package drafts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"bimanyaya/api/internal/auth"
	"bimanyaya/api/internal/db"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Draft struct {
	ID             string    `json:"id"`
	CaseID         string    `json:"case_id"`
	Language       string    `json:"language"`
	Status         string    `json:"status"` // DRAFT, APPROVED, REJECTED
	CurrentVersion int       `json:"current_version"`
	SafetyStatus   string    `json:"safety_status"` // PENDING, PASS, WARNING, BLOCK
	CreatedBy      *string   `json:"created_by,omitempty"`
	ApprovedBy     *string   `json:"approved_by,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type DraftVersion struct {
	ID            string      `json:"id"`
	DraftID       string      `json:"draft_id"`
	VersionNumber int         `json:"version_number"`
	Subject       string      `json:"subject"`
	Content       string      `json:"content"` // HTML/Markdown formatted string
	MetaDetails   interface{} `json:"meta_details,omitempty"`
	CreatedBy     *string     `json:"created_by,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
}

type CreateDraftRequest struct {
	Language string `json:"language"` // en, hi, or
}

type UpdateDraftRequest struct {
	Subject string `json:"subject"`
	Content string `json:"content"`
}

type TranslateDraftRequest struct {
	TargetLanguage string `json:"target_language"`
}

type Service struct {
	db          *db.DB
	aiWorkerURL string
}

func NewService(database *db.DB, aiWorkerURL string) *Service {
	return &Service{
		db:          database,
		aiWorkerURL: aiWorkerURL,
	}
}

func (s *Service) GetCaseDrafts(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	rows, err := s.db.Pool.Query(r.Context(),
		`SELECT id, case_id, language, status, current_version, safety_status, created_by, approved_by, created_at, updated_at 
		FROM drafts WHERE case_id = $1 ORDER BY updated_at DESC`, caseID)
	if err != nil {
		http.Error(w, "Failed to load drafts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	drafts := make([]Draft, 0)
	for rows.Next() {
		var d Draft
		err = rows.Scan(
			&d.ID, &d.CaseID, &d.Language, &d.Status, &d.CurrentVersion, &d.SafetyStatus,
			&d.CreatedBy, &d.ApprovedBy, &d.CreatedAt, &d.UpdatedAt,
		)
		if err != nil {
			http.Error(w, "Error parsing draft", http.StatusInternalServerError)
			return
		}
		drafts = append(drafts, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(drafts)
}

func (s *Service) CreateDraft(w http.ResponseWriter, r *http.Request) {
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

	var req CreateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	if req.Language == "" {
		req.Language = "en"
	}

	// 1. Check if draft already exists
	var existingID string
	err := s.db.Pool.QueryRow(r.Context(),
		"SELECT id FROM drafts WHERE case_id = $1 AND language = $2", caseID, req.Language).Scan(&existingID)

	if err == nil {
		http.Error(w, "Draft for this language already exists", http.StatusConflict)
		return
	}

	// 2. Fetch issues and citations to build prompt/payload for Python worker
	// For demo: we make a direct call to the AI worker to retrieve a generated draft
	aiPayload := map[string]interface{}{
		"case_id":  caseID,
		"language": req.Language,
	}
	payloadBytes, _ := json.Marshal(aiPayload)

	reqCtx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	aiReq, err := http.NewRequestWithContext(reqCtx, "POST", fmt.Sprintf("%s/generate-draft", s.aiWorkerURL), bytes.NewBuffer(payloadBytes))
	if err != nil {
		http.Error(w, "Internal AI integration issue", http.StatusInternalServerError)
		return
	}
	aiReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(aiReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("AI Worker offline: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "AI Worker failed to generate draft", http.StatusInternalServerError)
		return
	}

	var aiResult struct {
		Subject      string `json:"subject"`
		Content      string `json:"content"`
		SafetyStatus string `json:"safety_status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&aiResult); err != nil {
		http.Error(w, "Failed to parse AI output", http.StatusInternalServerError)
		return
	}

	// 3. Save draft to DB
	draftID := uuid.New().String()
	versionID := uuid.New().String()

	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(),
		`INSERT INTO drafts (id, case_id, language, status, current_version, safety_status, created_by) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		draftID, caseID, req.Language, "DRAFT", 1, aiResult.SafetyStatus, user.ID,
	)
	if err != nil {
		http.Error(w, "Failed to create draft record", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(r.Context(),
		`INSERT INTO draft_versions (id, draft_id, version_number, subject, content, created_by) 
		VALUES ($1, $2, $3, $4, $5, $6)`,
		versionID, draftID, 1, aiResult.Subject, aiResult.Content, user.ID,
	)
	if err != nil {
		http.Error(w, "Failed to create draft version", http.StatusInternalServerError)
		return
	}

	// Transition case status to REVIEW_REQUIRED if safety check passed, else NEEDS_CLARIFICATION
	caseState := "REVIEW_REQUIRED"
	if aiResult.SafetyStatus == "BLOCK" {
		caseState = "NEEDS_CLARIFICATION"
	}
	_, _ = tx.Exec(r.Context(), "UPDATE cases SET workflow_state = $1, updated_at = NOW() WHERE id = $2", caseState, caseID)
	_, _ = tx.Exec(r.Context(),
		"INSERT INTO case_status_history (case_id, from_state, to_state, changed_by, reason) VALUES ($1, $2, $3, $4, $5)",
		caseID, "ANALYSIS_READY", caseState, user.ID, "Grievance draft generated by AI",
	)

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Commit failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"draft_id":      draftID,
		"subject":       aiResult.Subject,
		"content":       aiResult.Content,
		"safety_status": aiResult.SafetyStatus,
	})
}

func (s *Service) PatchDraft(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	draftID := chi.URLParam(r, "draftId")
	if draftID == "" {
		http.Error(w, "Missing draft ID", http.StatusBadRequest)
		return
	}

	var req UpdateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	// Fetch current version number
	var curVersion int
	err := s.db.Pool.QueryRow(r.Context(), "SELECT current_version FROM drafts WHERE id = $1", draftID).Scan(&curVersion)
	if err != nil {
		http.Error(w, "Draft not found", http.StatusNotFound)
		return
	}

	nextVersion := curVersion + 1

	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	// 1. Insert new version
	versionID := uuid.New().String()
	_, err = tx.Exec(r.Context(),
		`INSERT INTO draft_versions (id, draft_id, version_number, subject, content, created_by) 
		VALUES ($1, $2, $3, $4, $5, $6)`,
		versionID, draftID, nextVersion, req.Subject, req.Content, user.ID,
	)
	if err != nil {
		http.Error(w, "Failed to create new version", http.StatusInternalServerError)
		return
	}

	// 2. Update parent draft meta
	_, err = tx.Exec(r.Context(),
		"UPDATE drafts SET current_version = $1, updated_at = NOW() WHERE id = $2",
		nextVersion, draftID,
	)
	if err != nil {
		http.Error(w, "Failed to update draft current version", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Failed to commit version update", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":         "Draft version updated successfully",
		"current_version": nextVersion,
	})
}

func (s *Service) TranslateDraft(w http.ResponseWriter, r *http.Request) {
	draftID := chi.URLParam(r, "draftId")
	if draftID == "" {
		http.Error(w, "Missing draft ID", http.StatusBadRequest)
		return
	}

	var req TranslateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	// Fetch current draft version content
	var curVersion int
	var caseID string
	err := s.db.Pool.QueryRow(r.Context(), "SELECT case_id, current_version FROM drafts WHERE id = $1", draftID).Scan(&caseID, &curVersion)
	if err != nil {
		http.Error(w, "Draft not found", http.StatusNotFound)
		return
	}

	var subject, content string
	err = s.db.Pool.QueryRow(r.Context(),
		"SELECT subject, content FROM draft_versions WHERE draft_id = $1 AND version_number = $2",
		draftID, curVersion).Scan(&subject, &content)
	if err != nil {
		http.Error(w, "Draft content not found", http.StatusInternalServerError)
		return
	}

	// Make HTTP call to Python worker for translation
	transPayload := map[string]string{
		"text":            content,
		"subject":         subject,
		"target_language": req.TargetLanguage,
	}
	payloadBytes, _ := json.Marshal(transPayload)

	reqCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	transReq, err := http.NewRequestWithContext(reqCtx, "POST", fmt.Sprintf("%s/translate", s.aiWorkerURL), bytes.NewBuffer(payloadBytes))
	if err != nil {
		http.Error(w, "Internal AI translation setup issue", http.StatusInternalServerError)
		return
	}
	transReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(transReq)
	if err != nil {
		http.Error(w, "Translation worker unavailable", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Translation worker failed request", http.StatusInternalServerError)
		return
	}

	var translationResult struct {
		TranslatedSubject string `json:"translated_subject"`
		TranslatedText    string `json:"translated_text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&translationResult); err != nil {
		http.Error(w, "Failed to parse translation result", http.StatusInternalServerError)
		return
	}

	// Save new translated draft
	newDraftID := uuid.New().String()
	newVersionID := uuid.New().String()

	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Database transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	// Insert translated draft
	_, err = tx.Exec(r.Context(),
		`INSERT INTO drafts (id, case_id, language, status, current_version, safety_status) 
		VALUES ($1, $2, $3, $4, $5, $6)`,
		newDraftID, caseID, req.TargetLanguage, "DRAFT", 1, "PASS",
	)
	if err != nil {
		http.Error(w, "Failed to insert translated draft record", http.StatusInternalServerError)
		return
	}

	// Insert version
	_, err = tx.Exec(r.Context(),
		`INSERT INTO draft_versions (id, draft_id, version_number, subject, content) 
		VALUES ($1, $2, $3, $4, $5)`,
		newVersionID, newDraftID, 1, translationResult.TranslatedSubject, translationResult.TranslatedText,
	)
	if err != nil {
		http.Error(w, "Failed to insert translated draft version", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Failed to save translation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"draft_id":           newDraftID,
		"translated_subject": translationResult.TranslatedSubject,
		"translated_content": translationResult.TranslatedText,
	})
}

// ExportPDF fetches the draft and generates a downloadable PDF document using the AI worker.
func (s *Service) ExportPDF(w http.ResponseWriter, r *http.Request) {
	draftID := chi.URLParam(r, "draftId")
	if draftID == "" {
		http.Error(w, "Missing draft ID", http.StatusBadRequest)
		return
	}

	// 1. Fetch current draft version content
	var curVersion int
	err := s.db.Pool.QueryRow(r.Context(), "SELECT current_version FROM drafts WHERE id = $1", draftID).Scan(&curVersion)
	if err != nil {
		http.Error(w, "Draft not found", http.StatusNotFound)
		return
	}

	var subject, content string
	err = s.db.Pool.QueryRow(r.Context(),
		"SELECT subject, content FROM draft_versions WHERE draft_id = $1 AND version_number = $2",
		draftID, curVersion).Scan(&subject, &content)
	if err != nil {
		http.Error(w, "Draft content not found", http.StatusInternalServerError)
		return
	}

	// 2. Make HTTP request to AI worker POST /generate-pdf
	pdfPayload := map[string]string{
		"subject":      subject,
		"html_content": content,
	}
	payloadBytes, _ := json.Marshal(pdfPayload)

	reqCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	pdfReq, err := http.NewRequestWithContext(reqCtx, "POST", fmt.Sprintf("%s/generate-pdf", s.aiWorkerURL), bytes.NewBuffer(payloadBytes))
	if err != nil {
		http.Error(w, "Internal PDF generation setup issue", http.StatusInternalServerError)
		return
	}
	pdfReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(pdfReq)
	if err != nil {
		http.Error(w, "PDF generation worker offline", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "PDF generation failed", http.StatusInternalServerError)
		return
	}

	// 3. Write PDF response headers
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"grievance_%s.pdf\"", draftID))

	// Copy body bytes directly to response writer
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return
	}
}

