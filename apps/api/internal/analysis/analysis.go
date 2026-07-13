package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"bimanyaya/api/internal/auth"
	"bimanyaya/api/internal/db"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ProcessResponse struct {
	Status        string `json:"status"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id"`
}

type CaseIssue struct {
	ID            string      `json:"_id"`
	CaseID        string      `json:"caseId"`
	IssueCategory string      `json:"issueCategory"`
	Summary       string      `json:"summary"`
	Details       interface{} `json:"details,omitempty"`
	Confidence    float64     `json:"confidence"`
	CreatedAt     time.Time   `json:"createdAt"`
}

type Citation struct {
	ID                string      `json:"_id"`
	CaseID            string      `json:"caseId"`
	SourceType        string      `json:"sourceType"`
	DocumentID        *string     `json:"documentId,omitempty"`
	KnowledgeSourceID *string     `json:"knowledgeSourceId,omitempty"`
	PageNumber        int         `json:"pageNumber"`
	SectionName       string      `json:"sectionName"`
	ClauseNumber      string      `json:"clauseNumber"`
	QuotedText        string      `json:"quotedText"`
	BoundingBox       interface{} `json:"boundingBox,omitempty"`
	Confidence        float64     `json:"confidence"`
	ValidationStatus  string      `json:"validationStatus"`
	CreatedAt         time.Time   `json:"createdAt"`
}

type EvidenceItem struct {
	ID                 string    `json:"_id"`
	CaseID             string    `json:"caseId"`
	DocumentName       string    `json:"documentName"`
	WhyRequired        string    `json:"whyRequired"`
	Priority           string    `json:"priority"`
	IsMandatory        bool      `json:"isMandatory"`
	Status             string    `json:"status"`
	UploadedDocumentID *string   `json:"uploadedDocumentId,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
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

type ConvexCase struct {
	ID                    string  `json:"_id"`
	CaseNumber            string  `json:"caseNumber"`
	OwnerUserID           string  `json:"ownerUserId"`
	InsuranceType         string  `json:"insuranceType"`
	ClaimCategory         string  `json:"claimCategory,omitempty"`
	ClaimStatus           string  `json:"claimStatus,omitempty"`
	InsurerName           string  `json:"insurerName,omitempty"`
	PolicyNumberEncrypted string  `json:"policyNumberEncrypted,omitempty"`
	ClaimNumberEncrypted  string  `json:"claimNumberEncrypted,omitempty"`
	AmountClaimed         float64 `json:"amountClaimed"`
	AmountPaid            float64 `json:"amountPaid"`
	AmountDisputed        float64 `json:"amountDisputed"`
	WorkflowState         string  `json:"workflowState"`
}

type ConvexDoc struct {
	ID           string `json:"_id"`
	StorageKey   string `json:"storageKey"`
	DocumentType string `json:"documentType"`
}

// TriggerCaseProcessing starts the Python AI pipeline (FastAPI call)
func (s *Service) TriggerCaseProcessing(w http.ResponseWriter, r *http.Request) {
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

	// 1. Fetch case details from Convex
	var caseItem ConvexCase
	err := s.db.CallQuery(r.Context(), "cases:getByLegacyId", map[string]interface{}{"legacyId": caseID}, &caseItem)
	if err != nil || caseItem.ID == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Case not found")
		return
	}

	// 2. Fetch associated documents from Convex
	var docs []ConvexDoc
	err = s.db.CallQuery(r.Context(), "documents:listByCase", map[string]interface{}{"caseId": caseID}, &docs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load documents")
		return
	}

	type docMeta struct {
		ID           string `json:"id"`
		StorageKey   string `json:"storage_key"`
		DocumentType string `json:"document_type"`
	}
	aiDocs := make([]docMeta, len(docs))
	for i, d := range docs {
		aiDocs[i] = docMeta{
			ID:           d.ID,
			StorageKey:   d.StorageKey,
			DocumentType: d.DocumentType,
		}
	}

	// 3. Make HTTP request to Python AI worker FastAPI server
	aiPayload := map[string]interface{}{
		"case_id":         caseID,
		"claim_number":    caseItem.ClaimNumberEncrypted,
		"insurer":         caseItem.InsurerName,
		"claim_status":    caseItem.ClaimStatus,
		"amount_claimed":  caseItem.AmountClaimed,
		"amount_paid":     caseItem.AmountPaid,
		"amount_disputed": caseItem.AmountDisputed,
		"documents":       aiDocs,
	}

	payloadBytes, err := json.Marshal(aiPayload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Serialization error")
		return
	}

	// Process in background goroutine to prevent blocking HTTP handler
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/process-case", s.aiWorkerURL), bytes.NewBuffer(payloadBytes))
		if err != nil {
			slog.Error("Failed to construct AI request", "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			slog.Error("AI Worker HTTP request failed", "error", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			slog.Error("AI Worker returned non-200 status", "status", resp.StatusCode)
			return
		}

		var aiResult struct {
			Issues []struct {
				IssueCategory string      `json:"issue_category"`
				Summary       string      `json:"summary"`
				Details       interface{} `json:"details"`
				Confidence    float64     `json:"confidence"`
			} `json:"issues"`
			Citations []struct {
				SourceType        string      `json:"source_type"`
				DocumentID        *string     `json:"document_id,omitempty"`
				KnowledgeSourceID *string     `json:"knowledge_source_id,omitempty"`
				PageNumber        int         `json:"page_number"`
				SectionName       string      `json:"section_name"`
				ClauseNumber      string      `json:"clause_number"`
				QuotedText        string      `json:"quoted_text"`
				BoundingBox       interface{} `json:"bounding_box,omitempty"`
				Confidence        float64     `json:"confidence"`
			} `json:"citations"`
			EvidenceChecklist []struct {
				DocumentName       string  `json:"document_name"`
				WhyRequired        string  `json:"why_required"`
				Priority           string  `json:"priority"`
				IsMandatory        bool    `json:"is_mandatory"`
				Status             string  `json:"status"`
				UploadedDocumentID *string `json:"uploaded_document_id,omitempty"`
			} `json:"evidence_checklist"`
			DocumentUpdates []struct {
				DocumentID           string `json:"document_id"`
				DocumentType         string `json:"document_type"`
				OCRStatus            string `json:"ocr_status"`
				ClassificationStatus string `json:"classification_status"`
				Pages                []struct {
					PageNumber int    `json:"page_number"`
					StorageKey string `json:"storage_key"`
				} `json:"pages"`
				Extractions []struct {
					FieldName       string  `json:"field_name"`
					RawValue        string  `json:"raw_value"`
					NormalizedValue string  `json:"normalized_value"`
					PageNumber      int     `json:"page_number"`
					SourceText      string  `json:"source_text"`
					Confidence      float64 `json:"confidence"`
				} `json:"extractions"`
			} `json:"document_updates"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&aiResult); err != nil {
			slog.Error("Failed to decode AI worker payload", "error", err)
			return
		}

		// Save the returned findings to Convex database atomically
		saveArgs := map[string]interface{}{
			"caseId":    caseID,
			"changedBy": user.ID,
		}

		issues := make([]map[string]interface{}, len(aiResult.Issues))
		for i, issue := range aiResult.Issues {
			detailsBytes, _ := json.Marshal(issue.Details)
			issues[i] = map[string]interface{}{
				"issueCategory": issue.IssueCategory,
				"summary":       issue.Summary,
				"details":       string(detailsBytes),
				"confidence":    issue.Confidence,
				"legacyId":      uuid.New().String(),
			}
		}
		saveArgs["issues"] = issues

		citations := make([]map[string]interface{}, len(aiResult.Citations))
		for i, cit := range aiResult.Citations {
			var docIDVal, ksIDVal string
			if cit.DocumentID != nil {
				docIDVal = *cit.DocumentID
			}
			if cit.KnowledgeSourceID != nil {
				ksIDVal = *cit.KnowledgeSourceID
			}

			bboxBytes, _ := json.Marshal(cit.BoundingBox)
			citations[i] = map[string]interface{}{
				"sourceType":        cit.SourceType,
				"pageNumber":        cit.PageNumber,
				"sectionName":       cit.SectionName,
				"clauseNumber":      cit.ClauseNumber,
				"quotedText":        cit.QuotedText,
				"boundingBox":       string(bboxBytes),
				"confidence":        cit.Confidence,
				"validationStatus":  "VALIDATED",
				"legacyId":          uuid.New().String(),
			}
			if docIDVal != "" {
				citations[i]["documentId"] = docIDVal
			}
			if ksIDVal != "" {
				citations[i]["knowledgeSourceId"] = ksIDVal
			}
		}
		saveArgs["citations"] = citations

		evidenceItems := make([]map[string]interface{}, len(aiResult.EvidenceChecklist))
		for i, ev := range aiResult.EvidenceChecklist {
			var upDocID string
			if ev.UploadedDocumentID != nil {
				upDocID = *ev.UploadedDocumentID
			}
			evidenceItems[i] = map[string]interface{}{
				"documentName": ev.DocumentName,
				"whyRequired":  ev.WhyRequired,
				"priority":     ev.Priority,
				"isMandatory":  ev.IsMandatory,
				"status":       ev.Status,
				"legacyId":     uuid.New().String(),
			}
			if upDocID != "" {
				evidenceItems[i]["uploadedDocumentId"] = upDocID
			}
		}
		saveArgs["evidenceItems"] = evidenceItems

		var success bool
		err = s.db.CallMutation(ctx, "analysis:saveAnalysisFindings", saveArgs, &success)
		if err != nil || !success {
			slog.Error("Failed to save AI findings to Convex", "error", err)
			return
		}

		// Save document classification, page text mappings, and field extractions to Convex
		for _, docUpdate := range aiResult.DocumentUpdates {
			var updatedDocID string
			err := s.db.CallMutation(ctx, "documents:updateTypeAndStatus", map[string]interface{}{
				"legacyId":             docUpdate.DocumentID,
				"documentType":         docUpdate.DocumentType,
				"ocrStatus":            docUpdate.OCRStatus,
				"classificationStatus": docUpdate.ClassificationStatus,
			}, &updatedDocID)
			if err != nil {
				slog.Error("Failed to update document type and status in Convex", "document_id", docUpdate.DocumentID, "error", err)
			}

			for _, page := range docUpdate.Pages {
				var pageID string
				err := s.db.CallMutation(ctx, "documents:savePage", map[string]interface{}{
					"documentId": docUpdate.DocumentID,
					"pageNumber": page.PageNumber,
					"storageKey": page.StorageKey,
				}, &pageID)
				if err != nil {
					slog.Error("Failed to save document page to Convex", "document_id", docUpdate.DocumentID, "page", page.PageNumber, "error", err)
				}
			}

			for _, ext := range docUpdate.Extractions {
				var extID string
				err := s.db.CallMutation(ctx, "documents:saveExtraction", map[string]interface{}{
					"documentId":      docUpdate.DocumentID,
					"fieldName":       ext.FieldName,
					"fieldValue":      ext.RawValue,
					"normalizedValue": ext.NormalizedValue,
					"pageNumber":      ext.PageNumber,
					"sourceText":      ext.SourceText,
					"confidence":      ext.Confidence,
					"reviewStatus":    "PENDING",
				}, &extID)
				if err != nil {
					slog.Error("Failed to save document extraction to Convex", "document_id", docUpdate.DocumentID, "field", ext.FieldName, "error", err)
				}
			}
		}

		slog.Info("Successfully processed and saved AI findings", "case_id", caseID)
	}()

	w.Header().Set("Content-Type", "application/json")
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
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var caseItem ConvexCase
	err := s.db.CallQuery(r.Context(), "cases:getByLegacyId", map[string]interface{}{"legacyId": caseID}, &caseItem)
	if err != nil || caseItem.ID == "" {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Case not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"case_id":        caseID,
		"workflow_state": caseItem.WorkflowState,
	})
}

// GetAnalysis gets issues
func (s *Service) GetAnalysis(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var issues []CaseIssue
	err := s.db.CallQuery(r.Context(), "analysis:getAnalysis", map[string]interface{}{"caseId": caseID}, &issues)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load issues")
		return
	}

	// Decode any string encoded details for API compatibility
	for i, issue := range issues {
		if detailsStr, ok := issue.Details.(string); ok && detailsStr != "" {
			var detailsMap interface{}
			if err := json.Unmarshal([]byte(detailsStr), &detailsMap); err == nil {
				issues[i].Details = detailsMap
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(issues)
}

// GetCitations gets supporting/adverse citations
func (s *Service) GetCitations(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var citations []Citation
	err := s.db.CallQuery(r.Context(), "analysis:getCitations", map[string]interface{}{"caseId": caseID}, &citations)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load citations")
		return
	}

	for i, cit := range citations {
		if bboxStr, ok := cit.BoundingBox.(string); ok && bboxStr != "" {
			var bboxMap interface{}
			if err := json.Unmarshal([]byte(bboxStr), &bboxMap); err == nil {
				citations[i].BoundingBox = bboxMap
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(citations)
}

// GetEvidence fetches the checklist
func (s *Service) GetEvidence(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var items []EvidenceItem
	err := s.db.CallQuery(r.Context(), "analysis:getEvidence", map[string]interface{}{"caseId": caseID}, &items)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load evidence")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
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
