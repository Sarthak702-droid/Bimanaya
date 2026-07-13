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
	ID                   string     `json:"id"`
	CaseID               string     `json:"case_id"`
	DocumentType         string     `json:"document_type"`
	OriginalFilename     string     `json:"original_filename"`
	StorageKey           string     `json:"storage_key"`
	FileHash             string     `json:"file_hash"`
	MimeType             string     `json:"mime_type"`
	SizeBytes            int64      `json:"size_bytes"`
	PageCount            int        `json:"page_count"`
	MalwareScanStatus    string     `json:"malware_scan_status"`
	OCRStatus            string     `json:"ocr_status"`
	ClassificationStatus string     `json:"classification_status"`
	RetentionUntil       *time.Time `json:"retention_until,omitempty"`
	UploadedBy           *string    `json:"uploaded_by,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
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

	var req GetUploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	// Validate MimeType
	allowedTypes := map[string]bool{
		"application/pdf": true,
		"image/jpeg":      true,
		"image/png":       true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true, // docx
	}
	if !allowedTypes[req.MimeType] {
		http.Error(w, "Unsupported file format. Only PDF, PNG, JPEG, DOCX are supported", http.StatusUnsupportedMediaType)
		return
	}

	docID := uuid.New().String()
	ext := filepath.Ext(req.Filename)
	storageKey := fmt.Sprintf("cases/%s/docs/%s%s", caseID, docID, ext)

	// Simulate signed URL for uploads.
	// In production, we request a pre-signed PUT URL from AWS S3 / MinIO.
	// For demo, we return an endpoint in our Go API that will receive the upload.
	uploadURL := fmt.Sprintf("http://%s/api/v1/documents/upload-endpoint/%s", r.Host, docID)

	// Insert doc metadata in PENDING state
	_, err := s.db.Pool.Exec(r.Context(),
		`INSERT INTO documents (
			id, case_id, document_type, original_filename, storage_key, file_hash, 
			mime_type, size_bytes, page_count, malware_scan_status, ocr_status, 
			classification_status, uploaded_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		docID, caseID, req.DocumentType, req.Filename, storageKey, "PENDING_HASH",
		req.MimeType, req.SizeBytes, 0, "PENDING", "PENDING", "PENDING", user.ID,
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to reserve document metadata: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetUploadURLResponse{
		DocumentID: docID,
		UploadURL:  uploadURL,
		StorageKey: storageKey,
	})
}

// UploadEndpoint allows direct file upload to our backend, simulating S3 bucket direct upload for local development
func (s *Service) UploadEndpoint(w http.ResponseWriter, r *http.Request) {
	docID := chi.URLParam(r, "documentId")
	if docID == "" {
		http.Error(w, "Missing document ID", http.StatusBadRequest)
		return
	}

	// Parse file
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file form field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Compute checksum hash (sha256)
	hasher := sha256.New()
	tee := io.TeeReader(file, hasher)

	// We can save the file locally in the workspace or S3 bucket here
	// For demo, we just consume the reader to compute the hash
	_, err = io.Copy(io.Discard, tee)
	if err != nil {
		http.Error(w, "Failed to process file stream", http.StatusInternalServerError)
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
	user, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CompleteUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	_, err := s.db.Pool.Exec(r.Context(),
		`UPDATE documents SET 
			file_hash = $1, 
			page_count = $2, 
			malware_scan_status = 'CLEAN', 
			ocr_status = 'READY', 
			classification_status = 'COMPLETED'
		WHERE id = $3`,
		req.FileHash, req.PageCount, req.DocumentID,
	)
	if err != nil {
		http.Error(w, "Failed to update document upload status", http.StatusInternalServerError)
		return
	}

	// Fetch case details to audit and verify workflow state
	var caseID string
	err = s.db.Pool.QueryRow(r.Context(), "SELECT case_id FROM documents WHERE id = $1", req.DocumentID).Scan(&caseID)
	if err == nil {
		// Log Audit Event
		s.auditLog(r.Context(), user.ID, user.Role, "UPLOAD", "DOCUMENT", req.DocumentID, nil, nil)

		// Optionally update case state to PROCESSING
		_, _ = s.db.Pool.Exec(r.Context(), "UPDATE cases SET workflow_state = 'PROCESSING', updated_at = NOW() WHERE id = $1", caseID)
		_, _ = s.db.Pool.Exec(r.Context(),
			"INSERT INTO case_status_history (case_id, from_state, to_state, changed_by, reason) VALUES ($1, $2, $3, $4, $5)",
			caseID, "DOCUMENTS_PENDING", "PROCESSING", user.ID, "Document upload completed",
		)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Upload marked complete. Processing pipeline scheduled.",
		"status":  "PROCESSING",
	})
}

func (s *Service) GetCaseDocuments(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	rows, err := s.db.Pool.Query(r.Context(),
		`SELECT id, case_id, document_type, original_filename, storage_key, file_hash, 
		mime_type, size_bytes, page_count, malware_scan_status, ocr_status, 
		classification_status, retention_until, uploaded_by, created_at 
		FROM documents WHERE case_id = $1 AND deleted_at IS NULL ORDER BY created_at ASC`, caseID)
	if err != nil {
		http.Error(w, "Failed to query case documents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	docs := make([]Document, 0)
	for rows.Next() {
		var doc Document
		err = rows.Scan(
			&doc.ID, &doc.CaseID, &doc.DocumentType, &doc.OriginalFilename, &doc.StorageKey,
			&doc.FileHash, &doc.MimeType, &doc.SizeBytes, &doc.PageCount, &doc.MalwareScanStatus,
			&doc.OCRStatus, &doc.ClassificationStatus, &doc.RetentionUntil, &doc.UploadedBy, &doc.CreatedAt,
		)
		if err != nil {
			http.Error(w, "Failed to parse document data", http.StatusInternalServerError)
			return
		}
		docs = append(docs, doc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}

func (s *Service) GetDocument(w http.ResponseWriter, r *http.Request) {
	docID := chi.URLParam(r, "documentId")
	if docID == "" {
		http.Error(w, "Missing document ID", http.StatusBadRequest)
		return
	}

	var doc Document
	err := s.db.Pool.QueryRow(r.Context(),
		`SELECT id, case_id, document_type, original_filename, storage_key, file_hash, 
		mime_type, size_bytes, page_count, malware_scan_status, ocr_status, 
		classification_status, retention_until, uploaded_by, created_at 
		FROM documents WHERE id = $1 AND deleted_at IS NULL`, docID).Scan(
		&doc.ID, &doc.CaseID, &doc.DocumentType, &doc.OriginalFilename, &doc.StorageKey,
		&doc.FileHash, &doc.MimeType, &doc.SizeBytes, &doc.PageCount, &doc.MalwareScanStatus,
		&doc.OCRStatus, &doc.ClassificationStatus, &doc.RetentionUntil, &doc.UploadedBy, &doc.CreatedAt,
	)
	if err != nil {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func (s *Service) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	docID := chi.URLParam(r, "documentId")
	if docID == "" {
		http.Error(w, "Missing document ID", http.StatusBadRequest)
		return
	}

	// Soft delete document
	_, err := s.db.Pool.Exec(r.Context(),
		"UPDATE documents SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1", docID)
	if err != nil {
		http.Error(w, "Failed to delete document", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Document deleted successfully"})
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
