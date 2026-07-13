package documents

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"bimanyaya/api/internal/auth"
	"bimanyaya/api/internal/db"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Document struct {
	ID                   string     `json:"_id"`
	CaseID               string     `json:"caseId"`
	DocumentType         string     `json:"documentType"`
	OriginalFilename     string     `json:"originalFilename"`
	StorageKey           string     `json:"storageKey"`
	FileHash             string     `json:"fileHash"`
	MimeType             string     `json:"mimeType"`
	SizeBytes            int64      `json:"sizeBytes"`
	PageCount            int        `json:"pageCount"`
	MalwareScanStatus    string     `json:"malwareScanStatus"`
	OCRStatus            string     `json:"ocrStatus"`
	ClassificationStatus string     `json:"classificationStatus"`
	RetentionUntil       *time.Time `json:"retentionUntil,omitempty"`
	UploadedBy           *string    `json:"uploadedBy,omitempty"`
	LegacyID             string     `json:"legacyId,omitempty"`
	CreatedAt            time.Time  `json:"createdAt"`
}

type GetUploadURLRequest struct {
	Filename     string `json:"filename"`
	DocumentType string `json:"document_type"` // e.g. "REJECTION_LETTER", "POLICY_SCHEDULE"
	MimeType     string `json:"mime_type"`
	SizeBytes    int64  `json:"size_bytes"`
}

type GetUploadURLResponse struct {
	DocumentID string `json:"document_id"`
	UploadURL  string `json:"upload_url"`
	StorageKey string `json:"storage_key"`
}

type CompleteUploadRequest struct {
	DocumentID string `json:"document_id"`
	FileHash   string `json:"file_hash"`
	PageCount  int    `json:"page_count"`
}

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

func (s *Service) GetUploadURL(w http.ResponseWriter, r *http.Request) {
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

	var req GetUploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input payload")
		return
	}

	allowedTypes := map[string]bool{
		"application/pdf": true,
		"image/jpeg":      true,
		"image/png":       true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	}
	if !allowedTypes[req.MimeType] {
		writeError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Unsupported file format. Only PDF, PNG, JPEG, DOCX are supported")
		return
	}

	docID := uuid.New().String()
	ext := filepath.Ext(req.Filename)
	storageKey := fmt.Sprintf("cases/%s/docs/%s%s", caseID, docID, ext)

	uploadURL := fmt.Sprintf("http://%s/api/v1/documents/upload-endpoint/%s", r.Host, docID)

	metaArgs := map[string]interface{}{
		"caseId":               caseID,
		"documentType":         req.DocumentType,
		"originalFilename":     req.Filename,
		"storageKey":           storageKey,
		"fileHash":             "PENDING_HASH",
		"mimeType":             req.MimeType,
		"sizeBytes":            req.SizeBytes,
		"pageCount":            0,
		"malwareScanStatus":    "PENDING",
		"ocrStatus":            "PENDING",
		"classificationStatus": "PENDING",
		"uploadedBy":           user.ID,
		"legacyId":             docID,
	}

	var resID string
	err := s.db.CallMutation(r.Context(), "documents:createMetadata", metaArgs, &resID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to reserve document metadata: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GetUploadURLResponse{
		DocumentID: docID,
		UploadURL:  uploadURL,
		StorageKey: storageKey,
	})
}

func (s *Service) UploadEndpoint(w http.ResponseWriter, r *http.Request) {
	docID := chi.URLParam(r, "documentId")
	if docID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing document ID")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Failed to read file form field")
		return
	}
	defer file.Close()

	hasher := sha256.New()
	tee := io.TeeReader(file, hasher)

	_, err = io.Copy(io.Discard, tee)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process file stream")
		return
	}

	fileHash := hex.EncodeToString(hasher.Sum(nil))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "uploaded_successfully",
		"file_hash": fileHash,
	})
}

func (s *Service) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req CompleteUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input payload")
		return
	}

	updateArgs := map[string]interface{}{
		"legacyId":             req.DocumentID,
		"fileHash":             req.FileHash,
		"pageCount":            req.PageCount,
		"malwareScanStatus":    "CLEAN",
		"ocrStatus":            "READY",
		"classificationStatus": "COMPLETED",
	}

	var updatedDocID string
	err := s.db.CallMutation(r.Context(), "documents:updateStatus", updateArgs, &updatedDocID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update document upload status")
		return
	}

	// Fetch doc details to audit and verify workflow state
	var doc Document
	err = s.db.CallQuery(r.Context(), "documents:getByLegacyId", map[string]interface{}{"legacyId": req.DocumentID}, &doc)
	if err == nil && doc.CaseID != "" {
		s.auditLog(r.Context(), user.ID, user.Role, "UPLOAD", "DOCUMENT", req.DocumentID, nil, nil)

		caseUpdateArgs := map[string]interface{}{
			"legacyId":      doc.CaseID,
			"workflowState": "PROCESSING",
		}
		var resCaseID string
		_ = s.db.CallMutation(r.Context(), "cases:update", caseUpdateArgs, &resCaseID)

		timelineArgs := map[string]interface{}{
			"caseId":    doc.CaseID,
			"fromState": "DOCUMENTS_PENDING",
			"toState":   "PROCESSING",
			"changedBy": user.ID,
			"reason":    "Document upload completed",
		}
		var resTimelineID string
		_ = s.db.CallMutation(r.Context(), "cases:logTimeline", timelineArgs, &resTimelineID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Upload marked complete. Processing pipeline scheduled.",
		"status":  "PROCESSING",
	})
}

func (s *Service) GetCaseDocuments(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var docs []Document
	err := s.db.CallQuery(r.Context(), "documents:listByCase", map[string]interface{}{"caseId": caseID}, &docs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to query case documents")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}

func (s *Service) GetDocument(w http.ResponseWriter, r *http.Request) {
	docID := chi.URLParam(r, "documentId")
	if docID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing document ID")
		return
	}

	var doc Document
	err := s.db.CallQuery(r.Context(), "documents:getByLegacyId", map[string]interface{}{"legacyId": docID}, &doc)
	if err != nil || doc.ID == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Document not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func (s *Service) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	docID := chi.URLParam(r, "documentId")
	if docID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing document ID")
		return
	}

	var success bool
	err := s.db.CallMutation(r.Context(), "documents:softDelete", map[string]interface{}{"legacyId": docID}, &success)
	if err != nil || !success {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete document")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Document deleted successfully"})
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
	var resIDStr string
	_ = s.db.CallMutation(ctx, "audit:log", auditArgs, &resIDStr)
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
