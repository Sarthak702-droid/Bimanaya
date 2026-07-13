package clarifications

import (
	"encoding/json"
	"net/http"
	"time"

	"bimanyaya/api/internal/auth"
	"bimanyaya/api/internal/db"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Question struct {
	ID                 string      `json:"id"`
	CaseID             string      `json:"case_id"`
	QuestionType       string      `json:"question_type"` // YES_NO, DATE, AMOUNT, DOCUMENT_UPLOAD, TEXT, MULTIPLE_CHOICE
	QuestionText       string      `json:"question_text"`
	Options            interface{} `json:"options,omitempty"`
	ContextExplanation string      `json:"context_explanation"`
	SourceDocumentRef  string      `json:"source_document_ref"`
	IsResolved         bool        `json:"is_resolved"`
	CreatedAt          time.Time   `json:"created_at"`
}

type AnswerRequest struct {
	AnswerText                string `json:"answer_text"`
	UploadedEvidenceDocumentID string `json:"uploaded_evidence_document_id"`
}

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

func (s *Service) GetQuestions(w http.ResponseWriter, r *http.Request) {
	caseID := chi.URLParam(r, "caseId")
	if caseID == "" {
		http.Error(w, "Missing case ID", http.StatusBadRequest)
		return
	}

	rows, err := s.db.Pool.Query(r.Context(),
		`SELECT id, case_id, question_type, question_text, options, 
		COALESCE(context_explanation, ''), COALESCE(source_document_ref, ''), is_resolved, created_at 
		FROM clarification_questions WHERE case_id = $1 ORDER BY created_at ASC`, caseID)
	if err != nil {
		http.Error(w, "Failed to load questions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	questions := make([]Question, 0)
	for rows.Next() {
		var q Question
		var optionsBytes []byte
		err = rows.Scan(
			&q.ID, &q.CaseID, &q.QuestionType, &q.QuestionText, &optionsBytes,
			&q.ContextExplanation, &q.SourceDocumentRef, &q.IsResolved, &q.CreatedAt,
		)
		if err != nil {
			http.Error(w, "Error parsing question row", http.StatusInternalServerError)
			return
		}
		if len(optionsBytes) > 0 {
			_ = json.Unmarshal(optionsBytes, &q.Options)
		}
		questions = append(questions, q)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(questions)
}

func (s *Service) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	caseID := chi.URLParam(r, "caseId")
	questionID := chi.URLParam(r, "questionId")
	if caseID == "" || questionID == "" {
		http.Error(w, "Missing case or question ID", http.StatusBadRequest)
		return
	}

	var req AnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input payload", http.StatusBadRequest)
		return
	}

	tx, err := s.db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Database transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	// 1. Check question existence
	var resolved bool
	err = tx.QueryRow(r.Context(),
		"SELECT is_resolved FROM clarification_questions WHERE id = $1 AND case_id = $2",
		questionID, caseID).Scan(&resolved)
	if err != nil {
		http.Error(w, "Question not found for this case", http.StatusNotFound)
		return
	}

	// 2. Insert answer
	answerID := uuid.New().String()
	var docIDVal *string
	if req.UploadedEvidenceDocumentID != "" {
		docIDVal = &req.UploadedEvidenceDocumentID
	}

	_, err = tx.Exec(r.Context(),
		`INSERT INTO clarification_answers (
			id, question_id, answer_text, uploaded_evidence_document_id, answered_by
		) VALUES ($1, $2, $3, $4, $5)`,
		answerID, questionID, req.AnswerText, docIDVal, user.ID,
	)
	if err != nil {
		http.Error(w, "Failed to submit answer", http.StatusInternalServerError)
		return
	}

	// 3. Update question state
	_, err = tx.Exec(r.Context(),
		"UPDATE clarification_questions SET is_resolved = TRUE WHERE id = $1", questionID)
	if err != nil {
		http.Error(w, "Failed to update question status", http.StatusInternalServerError)
		return
	}

	// 4. Check if all questions are now resolved
	var unresolvedCount int
	err = tx.QueryRow(r.Context(),
		"SELECT COUNT(*) FROM clarification_questions WHERE case_id = $1 AND is_resolved = FALSE",
		caseID).Scan(&unresolvedCount)

	if err == nil && unresolvedCount == 0 {
		// All resolved, transition case workflow state back from NEEDS_CLARIFICATION to ANALYSIS_READY
		var oldState string
		_ = tx.QueryRow(r.Context(), "SELECT workflow_state FROM cases WHERE id = $1", caseID).Scan(&oldState)
		if oldState == "NEEDS_CLARIFICATION" {
			_, _ = tx.Exec(r.Context(), "UPDATE cases SET workflow_state = 'ANALYSIS_READY', updated_at = NOW() WHERE id = $1", caseID)
			_, _ = tx.Exec(r.Context(),
				"INSERT INTO case_status_history (case_id, from_state, to_state, changed_by, reason) VALUES ($1, $2, $3, $4, $5)",
				caseID, "NEEDS_CLARIFICATION", "ANALYSIS_READY", user.ID, "All clarifying questions resolved by user",
			)
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Failed to commit answer transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":               "Answer submitted successfully",
		"answer_id":             answerID,
		"all_questions_resolved": unresolvedCount == 0,
	})
}
