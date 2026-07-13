package audit

import (
	"encoding/json"
	"net/http"
	"time"

	"bimanyaya/api/internal/auth"
	"bimanyaya/api/internal/db"
)

type AuditEvent struct {
	ID            string    `json:"_id"`
	ActorID       *string   `json:"actorId,omitempty"`
	ActorRole     *string   `json:"actorRole,omitempty"`
	Action        string    `json:"action"`
	ResourceType  string    `json:"resourceType"`
	ResourceID    *string   `json:"resourceId,omitempty"`
	BeforeHash    *string   `json:"beforeHash,omitempty"`
	AfterHash     *string   `json:"afterHash,omitempty"`
	IPAddress     *string   `json:"ipAddress,omitempty"`
	UserAgent     *string   `json:"userAgent,omitempty"`
	CorrelationID string    `json:"correlationId"`
	CreatedAt     time.Time `json:"createdAt"`
}

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

func (s *Service) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(auth.UserKey).(auth.User)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	if user.Role != "ADMIN" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden: administrators only")
		return
	}

	var events []AuditEvent
	err := s.db.CallQuery(r.Context(), "audit:list", map[string]interface{}{"limit": 100}, &events)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to load audit events")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
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
