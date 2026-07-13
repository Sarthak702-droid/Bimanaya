package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"bimanyaya/api/internal/auth"
	"bimanyaya/api/internal/db"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ProcessResponse struct {
	Status        string `json:"status"` // PROCESSING, COMPLETED, FAILED
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id"`
}

type CaseIssue struct {
	ID            string      `json:"id"`
	CaseID        string      `json:"case_id"`
	IssueCategory string      `json:"issue_category"`
	Summary       string      `json:"summary"`
	Details       interface{} `json:"details"`
	Confidence    float64     `json:"confidence"`
	CreatedAt     time.Time   `json:"created_at"`
}

type Citation struct {
	ID                string      `json:"id"`
	CaseID            string      `json:"case_id"`
	SourceType        string      `json:"source_type"` // POLICY, REGULATION
	DocumentID        *string     `json:"document_id,omitempty"`
	KnowledgeSourceID *string     `json:"knowledge_source_id,omitempty"`
	PageNumber        int         `json:"page_number"`
	SectionName       string      `json:"section_name"`
	ClauseNumber      string      `json:"clause_number"`
	QuotedText        string      `json:"quoted_text"`
	BoundingBox       interface{} `json:"bounding_box,omitempty"`
	Confidence        float64     `json:"confidence"`
	ValidationStatus  string      `json:"validation_status"`
	CreatedAt         time.Time   `json:"created_at"`
}

type EvidenceItem struct {
	ID                 string    `json:"id"`
	CaseID             string    `json:"case_id"`
	DocumentName       string    `json:"document_name"`
	WhyRequired        string    `json:"why_required"`
	Priority           string    `json:"priority"` // HIGH, MEDIUM, LOW
	IsMandatory        bool      `json:"is_mandatory"`
	Status             string    `json:"status"` // AVAILABLE, MISSING, CONTRADICTORY
	UploadedDocumentID *string   `json:"uploaded_document_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
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

// TriggerCaseProcessing starts the Python AI pipeline (FastAPI call)
func (s *Service) TriggerCaseProcessing(w http.ResponseWriter, r *http.Request) {
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

	// 1. Fetch case details from DB
	var claimNumber, insurer, claimStatus string
	var amountClaimed, amountPaid, amountDisputed float64
	err := s.db.Pool.QueryRow(r.Context(),
		"SELECT COALESCE(claim_number_encrypted,''), COALESCE(insurer_name,''), COALESCE(claim_status,''), amount_claimed, amount_paid, amount_disputed FROM cases WHERE id = $1",
		caseID).Scan(&claimNumber, &insurer, &claimStatus, &amountClaimed, &amountPaid, &amountDisputed)
	if err != nil {
		http.Error(w, "Case not found", http.StatusNotFound)
		return
	}

	// 2. Fetch associated documents
	rows, err := s.db.Pool.Query(r.Context(), "SELECT id, storage_key, document_type FROM documents WHERE case_id = $1 AND deleted_at IS NULL", caseID)
	if err != nil {
		http.Error(w, "Failed to load documents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type docMeta struct {
		ID           string `json:"id"`
		StorageKey   string `json:"storage_key"`
		DocumentType string `json:"document_type"`
	}
	docs := make([]docMeta, 0)
	for rows.Next() {
		var dm docMeta
		if err := rows.Scan(&dm.ID, &dm.StorageKey, &dm.DocumentType); err == nil {
			docs = append(docs, dm)
		}
	}

	// 3. Make HTTP request to Python AI worker FastAPI server
	aiPayload := map[string]interface{}{
		"case_id":         caseID,
		"claim_number":    claimNumber,
		"insurer":         insurer,
		"claim_status":    claimStatus,
		"amount_claimed":  amountClaimed,
		"amount_paid":     amountPaid,
		"amount_disputed": amountDisputed,
		"documents":       docs,
	}

	payloadBytes, err := json.Marshal(aiPayload)
	if err != nil {
		http.Error(w, "Serialization error", http.StatusInternalServerError)
		return
	}

	// Fire and forget or synchronous depending on load. Let's make it fire-and-forget in background, but returning 202 Accepted.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/process-case", s.aiWorkerURL), bytes.NewBuffer(payloadBytes))
		if err != nil {
			fmt.Printf("[AI WORKER ERROR] Failed to construct request: %v\n", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("[AI WORKER ERROR] Connection failed: %v\n", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("[AI WORKER ERROR] Returned status: %d\n", resp.StatusCode)
			return
		}

		// Save the returned findings to database (simulating Python response parsing)
		var aiResult struct {
			Issues            []CaseIssue    `json:"issues"`
			Citations         []Citation     `json:"citations"`
			EvidenceChecklist []EvidenceItem `json:"evidence_checklist"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&aiResult); err != nil {
			fmt.Printf("[AI WORKER ERROR] Failed to parse result: %v\n", err)
			return
		}

		// Store in database
		dbCtx := context.Background()
		tx, err := s.db.Pool.Begin(dbCtx)
		if err != nil {
			return
		}
		defer tx.Rollback(dbCtx)

		// Clear old findings
		_, _ = tx.Exec(dbCtx, "DELETE FROM case_issues WHERE case_id = $1", caseID)
		_, _ = tx.Exec(dbCtx, "DELETE FROM citations WHERE case_id = $1", caseID)
		_, _ = tx.Exec(dbCtx, "DELETE FROM evidence_items WHERE case_id = $1", caseID)

		// Insert issues
		for _, issue := range aiResult.Issues {
			issueDetailsJSON, _ := json.Marshal(issue.Details)
			_, _ = tx.Exec(dbCtx,
				"INSERT INTO case_issues (id, case_id, issue_category, summary, details, confidence) VALUES ($1, $2, $3, $4, $5, $6)",
				uuid.New().String(), caseID, issue.IssueCategory, issue.Summary, issueDetailsJSON, issue.Confidence,
			)
		}

		// Insert citations
		for _, cit := range aiResult.Citations {
			bboxJSON, _ := json.Marshal(cit.BoundingBox)
			_, _ = tx.Exec(dbCtx,
				`INSERT INTO citations (
					id, case_id, source_type, document_id, knowledge_source_id, page_number, 
					section_name, clause_number, quoted_text, bounding_box, confidence, validation_status
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
				uuid.New().String(), caseID, cit.SourceType, cit.DocumentID, cit.KnowledgeSourceID,
				cit.PageNumber, cit.SectionName, cit.ClauseNumber, cit.QuotedText, bboxJSON, cit.Confidence, "VALIDATED",
			)
		}

		// Insert checklist
		for _, ev := range aiResult.EvidenceChecklist {
			_, _ = tx.Exec(dbCtx,
				`INSERT INTO evidence_items (
					id, case_id, document_name, why_required, priority, is_mandatory, status, uploaded_document_id
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				uuid.New().String(), caseID, ev.DocumentName, ev.WhyRequired, ev.Priority, ev.IsMandatory, ev.Status, ev.UploadedDocumentID,
			)
		}

		// Update case status to ANALYSIS_READY or REVIEW_REQUIRED
		_, _ = tx.Exec(dbCtx, "UPDATE cases SET workflow_state = 'ANALYSIS_READY', updated_at = NOW() WHERE id = $1", caseID)
		_, _ = tx.Exec(dbCtx,
			"INSERT INTO case_status_history (case_id, from_state, to_state, changed_by, reason) VALUES ($1, $2, $3, $4, $5)",
			caseID, "PROCESSING", "ANALYSIS_READY", user.ID, "AI analysis completed",
		)

		_ = tx.Commit(dbCtx)
		fmt.Printf("[AI WORKER SUCCESS] Fully updated case %s with findings\n", caseID)
	}()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(ProcessResponse{
		Status:        "PROCESSING",
		Message:       "Case analysis triggered on Python worker pool successfully.",
		CorrelationID: uuid.New().String(),
	})
}

// GetProcessingStatus polls case state
func (s *Service) GetProcessingStatus(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	var state string
	err := s.db.Pool.QueryRow(r.Context(), "SELECT workflow_state FROM cases WHERE id = $1", caseID).Scan(&state)
	if err != nil {
		http.Error(w, "Case not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"case_id":        caseID,
		"workflow_state": state,
	})
}

// GetAnalysis gets issues
func (s *Service) GetAnalysis(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	rows, err := s.db.Pool.Query(r.Context(),
		"SELECT id, case_id, issue_category, summary, details, confidence, created_at FROM case_issues WHERE case_id = $1",
		caseID)
	if err != nil {
		http.Error(w, "Failed to load issues", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	issues := make([]CaseIssue, 0)
	for rows.Next() {
		var is CaseIssue
		var detailsBytes []byte
		err = rows.Scan(&is.ID, &is.CaseID, &is.IssueCategory, &is.Summary, &detailsBytes, &is.Confidence, &is.CreatedAt)
		if err != nil {
			http.Error(w, "Error parsing issue row", http.StatusInternalServerError)
			return
		}
		_ = json.Unmarshal(detailsBytes, &is.Details)
		issues = append(issues, is)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(issues)
}

// GetCitations gets supporting/adverse citations
func (s *Service) GetCitations(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	rows, err := s.db.Pool.Query(r.Context(),
		`SELECT id, case_id, source_type, document_id, knowledge_source_id, page_number, 
		section_name, clause_number, quoted_text, bounding_box, confidence, validation_status, created_at 
		FROM citations WHERE case_id = $1`, caseID)
	if err != nil {
		http.Error(w, "Failed to load citations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	citations := make([]Citation, 0)
	for rows.Next() {
		var cit Citation
		var bboxBytes []byte
		err = rows.Scan(
			&cit.ID, &cit.CaseID, &cit.SourceType, &cit.DocumentID, &cit.KnowledgeSourceID, &cit.PageNumber,
			&cit.SectionName, &cit.ClauseNumber, &cit.QuotedText, &bboxBytes, &cit.Confidence, &cit.ValidationStatus, &cit.CreatedAt,
		)
		if err != nil {
			http.Error(w, "Error parsing citation row", http.StatusInternalServerError)
			return
		}
		_ = json.Unmarshal(bboxBytes, &cit.BoundingBox)
		citations = append(citations, cit)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(citations)
}

// GetEvidence fetches the checklist
func (s *Service) GetEvidence(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	rows, err := s.db.Pool.Query(r.Context(),
		`SELECT id, case_id, document_name, why_required, priority, is_mandatory, status, uploaded_document_id, created_at 
		FROM evidence_items WHERE case_id = $1`, caseID)
	if err != nil {
		http.Error(w, "Failed to load evidence", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := make([]EvidenceItem, 0)
	for rows.Next() {
		var ev EvidenceItem
		err = rows.Scan(
			&ev.ID, &ev.CaseID, &ev.DocumentName, &ev.WhyRequired, &ev.Priority, &ev.IsMandatory,
			&ev.Status, &ev.UploadedDocumentID, &ev.CreatedAt,
		)
		if err != nil {
			http.Error(w, "Error parsing evidence row", http.StatusInternalServerError)
			return
		}
		items = append(items, ev)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}
