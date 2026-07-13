package audit

import (
	"encoding/json"
	"net/http"
	"time"

	"bimanyaya/api/internal/auth"
	"bimanyaya/api/internal/db"
)

type AuditEvent struct {
	ID            string    `json:"id"`
	ActorID       *string   `json:"actor_id,omitempty"`
	ActorRole     *string   `json:"actor_role,omitempty"`
	Action        string    `json:"action"`
	ResourceType  string    `json:"resource_type"`
	ResourceID    *string   `json:"resource_id,omitempty"`
	BeforeHash    *string   `json:"before_hash,omitempty"`
	AfterHash     *string   `json:"after_hash,omitempty"`
	IPAddress     *string   `json:"ip_address,omitempty"`
	UserAgent     *string   `json:"user_agent,omitempty"`
	CorrelationID string    `json:"correlation_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

func (s *Service) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value("user").(auth.User)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Only admin role can review the system audit trails
	if user.Role != "ADMIN" {
		http.Error(w, "Forbidden: administrators only", http.StatusForbidden)
		return
	}

	rows, err := s.db.Pool.Query(r.Context(),
		`SELECT id, actor_id, actor_role, action, resource_type, resource_id, 
		COALESCE(before_hash,''), COALESCE(after_hash,''), COALESCE(ip_address,''), COALESCE(user_agent,''), 
		correlation_id, created_at 
		FROM audit_events ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		http.Error(w, "Failed to load audit events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	events := make([]AuditEvent, 0)
	for rows.Next() {
		var ev AuditEvent
		err = rows.Scan(
			&ev.ID, &ev.ActorID, &ev.ActorRole, &ev.Action, &ev.ResourceType, &ev.ResourceID,
			&ev.BeforeHash, &ev.AfterHash, &ev.IPAddress, &ev.UserAgent, &ev.CorrelationID, &ev.CreatedAt,
		)
		if err == nil {
			events = append(events, ev)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
