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
	ID             string    `json:"_id"`
	CaseID         string    `json:"caseId"`
	Language       string    `json:"language"`
	Status         string    `json:"status"`
	CurrentVersion int       `json:"currentVersion"`
	SafetyStatus   string    `json:"safetyStatus"`
	CreatedBy      *string   `json:"createdBy,omitempty"`
	ApprovedBy     *string   `json:"approvedBy,omitempty"`
	LegacyID       string    `json:"legacyId,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type DraftVersion struct {
	ID            string      `json:"_id"`
	DraftID       string      `json:"draftId"`
	VersionNumber int         `json:"versionNumber"`
	Subject       string      `json:"subject"`
	Content       string      `json:"content"`
	MetaDetails   interface{} `json:"metaDetails,omitempty"`
	CreatedBy     *string     `json:"createdBy,omitempty"`
	LegacyID      string      `json:"legacyId,omitempty"`
	CreatedAt     time.Time   `json:"createdAt"`
}

type CreateDraftRequest struct {
	Language string `json:"language"`
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
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var drafts []Draft
	err := s.db.CallQuery(r.Context(), "drafts:listByCase", map[string]interface{}{"caseId": caseID}, &drafts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load drafts")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(drafts)
}

func (s *Service) CreateDraft(w http.ResponseWriter, r *http.Request) {
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

	var req CreateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input payload")
		return
	}

	if req.Language == "" {
		req.Language = "en"
	}

	// 1. Check if draft already exists
	var existing Draft
	_ = s.db.CallQuery(r.Context(), "drafts:getByLegacyId", map[string]interface{}{"legacyId": caseID + "-" + req.Language}, &existing)
	if existing.ID != "" {
		writeError(w, http.StatusConflict, "CONFLICT", "Draft for this language already exists")
		return
	}

	// 2. Fetch draft from AI worker
	aiPayload := map[string]interface{}{
		"case_id":  caseID,
		"language": req.Language,
	}
	payloadBytes, _ := json.Marshal(aiPayload)

	reqCtx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	aiReq, err := http.NewRequestWithContext(reqCtx, "POST", fmt.Sprintf("%s/generate-draft", s.aiWorkerURL), bytes.NewBuffer(payloadBytes))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal AI integration issue")
		return
	}
	aiReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(aiReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "AI Worker offline")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "AI Worker failed to generate draft")
		return
	}

	var aiResult struct {
		Subject      string `json:"subject"`
		Content      string `json:"content"`
		SafetyStatus string `json:"safety_status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&aiResult); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to parse AI output")
		return
	}

	// 3. Save draft to Convex
	draftID := uuid.New().String()
	versionID := uuid.New().String()

	draftArgs := map[string]interface{}{
		"caseId":         caseID,
		"language":       req.Language,
		"status":         "DRAFT",
		"currentVersion": 1,
		"safetyStatus":   aiResult.SafetyStatus,
		"createdBy":      user.ID,
		"legacyId":       draftID,
	}

	var convexDraftID string
	err = s.db.CallMutation(r.Context(), "drafts:create", draftArgs, &convexDraftID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create draft record: "+err.Error())
		return
	}

	versionArgs := map[string]interface{}{
		"draftId":       convexDraftID,
		"versionNumber": 1,
		"subject":       aiResult.Subject,
		"content":       aiResult.Content,
		"createdBy":     user.ID,
		"legacyId":      versionID,
	}

	var convexVersionID string
	err = s.db.CallMutation(r.Context(), "drafts:createVersion", versionArgs, &convexVersionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create draft version: "+err.Error())
		return
	}

	// Update case state
	caseState := "REVIEW_REQUIRED"
	if aiResult.SafetyStatus == "BLOCK" {
		caseState = "NEEDS_CLARIFICATION"
	}

	caseUpdateArgs := map[string]interface{}{
		"legacyId":      caseID,
		"workflowState": caseState,
	}
	var resCaseID string
	_ = s.db.CallMutation(r.Context(), "cases:update", caseUpdateArgs, &resCaseID)

	timelineArgs := map[string]interface{}{
		"caseId":    caseID,
		"fromState": "ANALYSIS_READY",
		"toState":   caseState,
		"changedBy": user.ID,
		"reason":    "Grievance draft generated by AI",
	}
	var resTimelineID string
	_ = s.db.CallMutation(r.Context(), "cases:logTimeline", timelineArgs, &resTimelineID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"draft_id":      draftID,
		"subject":       aiResult.Subject,
		"content":       aiResult.Content,
		"safety_status": aiResult.SafetyStatus,
	})
}

func (s *Service) PatchDraft(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	draftID := chi.URLParam(r, "draftId")
	if draftID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing draft ID")
		return
	}

	var req UpdateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input payload")
		return
	}

	var d Draft
	err := s.db.CallQuery(r.Context(), "drafts:getByLegacyId", map[string]interface{}{"legacyId": draftID}, &d)
	if err != nil || d.ID == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Draft not found")
		return
	}

	nextVersion := d.CurrentVersion + 1
	versionID := uuid.New().String()

	versionArgs := map[string]interface{}{
		"draftId":       d.ID,
		"versionNumber": nextVersion,
		"subject":       req.Subject,
		"content":       req.Content,
		"createdBy":     user.ID,
		"legacyId":      versionID,
	}

	var convexVerID string
	err = s.db.CallMutation(r.Context(), "drafts:createVersion", versionArgs, &convexVerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create new draft version: "+err.Error())
		return
	}

	updateArgs := map[string]interface{}{
		"legacyId":       draftID,
		"currentVersion": nextVersion,
	}
	var resDraftID string
	err = s.db.CallMutation(r.Context(), "drafts:update", updateArgs, &resDraftID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update draft header: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":         "Draft version updated successfully",
		"current_version": nextVersion,
	})
}

func (s *Service) TranslateDraft(w http.ResponseWriter, r *http.Request) {
	draftID := chi.URLParam(r, "draftId")
	if draftID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing draft ID")
		return
	}

	var req TranslateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input payload")
		return
	}

	var d Draft
	err := s.db.CallQuery(r.Context(), "drafts:getByLegacyId", map[string]interface{}{"legacyId": draftID}, &d)
	if err != nil || d.ID == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Draft not found")
		return
	}

	var ver DraftVersion
	err = s.db.CallQuery(r.Context(), "drafts:getVersion", map[string]interface{}{
		"draftId":       d.ID,
		"versionNumber": d.CurrentVersion,
	}, &ver)
	if err != nil || ver.ID == "" {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Draft version content not found")
		return
	}

	transPayload := map[string]string{
		"text":            ver.Content,
		"subject":         ver.Subject,
		"target_language": req.TargetLanguage,
	}
	payloadBytes, _ := json.Marshal(transPayload)

	reqCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	transReq, err := http.NewRequestWithContext(reqCtx, "POST", fmt.Sprintf("%s/translate", s.aiWorkerURL), bytes.NewBuffer(payloadBytes))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal AI translation setup issue")
		return
	}
	transReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(transReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Translation worker offline")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Translation worker failed request")
		return
	}

	var translationResult struct {
		TranslatedSubject string `json:"translated_subject"`
		TranslatedText    string `json:"translated_text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&translationResult); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to parse translation result")
		return
	}

	newDraftID := uuid.New().String()
	newVersionID := uuid.New().String()

	newDraftArgs := map[string]interface{}{
		"caseId":         d.CaseID,
		"language":       req.TargetLanguage,
		"status":         "DRAFT",
		"currentVersion": 1,
		"safetyStatus":   "PASS",
		"legacyId":       newDraftID,
	}

	var convNewDraftID string
	err = s.db.CallMutation(r.Context(), "drafts:create", newDraftArgs, &convNewDraftID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to insert translated draft record")
		return
	}

	newVerArgs := map[string]interface{}{
		"draftId":       convNewDraftID,
		"versionNumber": 1,
		"subject":       translationResult.TranslatedSubject,
		"content":       translationResult.TranslatedText,
		"legacyId":      newVersionID,
	}

	var convNewVerID string
	_ = s.db.CallMutation(r.Context(), "drafts:createVersion", newVerArgs, &convNewVerID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"draft_id":           newDraftID,
		"translated_subject": translationResult.TranslatedSubject,
		"translated_content": translationResult.TranslatedText,
	})
}

func (s *Service) ExportPDF(w http.ResponseWriter, r *http.Request) {
	draftID := chi.URLParam(r, "draftId")
	if draftID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing draft ID")
		return
	}

	var d Draft
	err := s.db.CallQuery(r.Context(), "drafts:getByLegacyId", map[string]interface{}{"legacyId": draftID}, &d)
	if err != nil || d.ID == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Draft not found")
		return
	}

	var ver DraftVersion
	err = s.db.CallQuery(r.Context(), "drafts:getVersion", map[string]interface{}{
		"draftId":       d.ID,
		"versionNumber": d.CurrentVersion,
	}, &ver)
	if err != nil || ver.ID == "" {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Draft version content not found")
		return
	}

	pdfPayload := map[string]string{
		"subject":      ver.Subject,
		"html_content": ver.Content,
	}
	payloadBytes, _ := json.Marshal(pdfPayload)

	reqCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	pdfReq, err := http.NewRequestWithContext(reqCtx, "POST", fmt.Sprintf("%s/generate-pdf", s.aiWorkerURL), bytes.NewBuffer(payloadBytes))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal PDF generation setup issue")
		return
	}
	pdfReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(pdfReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "PDF generation worker offline")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "PDF generation failed")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"grievance_%s.pdf\"", draftID))

	_, _ = io.Copy(w, resp.Body)
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
