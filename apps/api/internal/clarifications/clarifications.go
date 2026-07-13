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
	ID                 string      `json:"_id"`
	CaseID             string      `json:"caseId"`
	QuestionType       string      `json:"questionType"`
	QuestionText       string      `json:"questionText"`
	Options            interface{} `json:"options,omitempty"`
	ContextExplanation string      `json:"contextExplanation"`
	SourceDocumentRef  string      `json:"sourceDocumentRef"`
	IsResolved         bool        `json:"isResolved"`
	CreatedAt          time.Time   `json:"createdAt"`
	LegacyID           *string     `json:"legacyId,omitempty"`
}

type AnswerRequest struct {
	AnswerText                 string `json:"answer_text"`
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
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case ID")
		return
	}

	var questions []Question
	err := s.db.CallQuery(r.Context(), "clarifications:getQuestions", map[string]interface{}{"caseId": caseID}, &questions)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load questions")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(questions)
}

func (s *Service) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	caseID := chi.URLParam(r, "caseId")
	questionID := chi.URLParam(r, "questionId")
	if caseID == "" || questionID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing case or question ID")
		return
	}

	var req AnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid input payload")
		return
	}

	legacyAnswerID := uuid.New().String()

	args := map[string]interface{}{
		"caseId":         caseID,
		"questionId":     questionID,
		"answerText":     req.AnswerText,
		"answeredBy":     user.ID,
		"legacyAnswerId": legacyAnswerID,
	}

	if req.UploadedEvidenceDocumentID != "" {
		args["uploadedEvidenceDocumentId"] = req.UploadedEvidenceDocumentID
	}

	var result struct {
		AnswerID             string `json:"answerId"`
		AllQuestionsResolved bool   `json:"allQuestionsResolved"`
	}

	err := s.db.CallMutation(r.Context(), "clarifications:submitAnswer", args, &result)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":                "Answer submitted successfully",
		"answer_id":              result.AnswerID,
		"all_questions_resolved": result.AllQuestionsResolved,
	})
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
