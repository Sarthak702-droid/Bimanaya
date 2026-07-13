package db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"bimanyaya/api/internal/config"
)

type DB struct {
	convexURL      string
	clerkSecretKey string
	client         *http.Client

	// Local in-memory fallback databases for local simulation
	localUsers       map[string]map[string]interface{}
	localLegacyUsers map[string]map[string]interface{}
	localPages       map[string][]map[string]interface{}
	localExtractions map[string][]map[string]interface{}
	localComments    map[string][]map[string]interface{}
	mu               sync.Mutex
}

type ConvexRequest struct {
	Path   string      `json:"path"`
	Args   interface{} `json:"args"`
	Format string      `json:"format"`
}

type ConvexErrorResponse struct {
	Status       string   `json:"status"`
	ErrorMessage string   `json:"errorMessage"`
	LogLines     []string `json:"logLines"`
}

func Connect(cfg *config.Config) (*DB, error) {
	if cfg.ConvexURL == "" {
		return nil, fmt.Errorf("CONVEX_URL is required")
	}

	return &DB{
		convexURL:      cfg.ConvexURL,
		clerkSecretKey: cfg.ClerkSecretKey,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		localUsers:       make(map[string]map[string]interface{}),
		localLegacyUsers: make(map[string]map[string]interface{}),
		localPages:       make(map[string][]map[string]interface{}),
		localExtractions: make(map[string][]map[string]interface{}),
		localComments:    make(map[string][]map[string]interface{}),
	}, nil
}

// CallQuery invokes a Convex query function
func (d *DB) CallQuery(ctx context.Context, path string, args interface{}, dest interface{}) error {
	return d.call(ctx, "query", path, args, dest)
}

// CallMutation invokes a Convex mutation function
func (d *DB) CallMutation(ctx context.Context, path string, args interface{}, dest interface{}) error {
	return d.call(ctx, "mutation", path, args, dest)
}

// CallAction invokes a Convex action function
func (d *DB) CallAction(ctx context.Context, path string, args interface{}, dest interface{}) error {
	return d.call(ctx, "action", path, args, dest)
}

type ConvexSuccessResponse struct {
	Status string          `json:"status"`
	Value  json.RawMessage `json:"value"`
}

func (d *DB) call(ctx context.Context, funcType, path string, args interface{}, dest interface{}) error {
	// First check if we have a local fallback for this function path
	handled, fallbackErr := d.handleLocalFallback(funcType, path, args, dest)
	if handled {
		return fallbackErr
	}

	url := fmt.Sprintf("%s/api/%s", d.convexURL, funcType)

	payload := ConvexRequest{
		Path:   path,
		Args:   args,
		Format: "json",
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Retry loop for robustness
	var lastErr error
	maxRetries := 3
	backoff := 100 * time.Millisecond

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			slog.Warn("Retrying Convex HTTP call", "attempt", attempt, "error", lastErr)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		if d.clerkSecretKey != "" && !strings.HasPrefix(d.clerkSecretKey, "sk_test_") && !strings.HasPrefix(d.clerkSecretKey, "sk_live_") {
			// Bearer authentication (Convex deploy key)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", d.clerkSecretKey))
		}

		slog.Info("Outgoing Convex Request Headers", "url", url, "headers", req.Header)
		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			var errResp ConvexErrorResponse
			if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.ErrorMessage != "" {
				return fmt.Errorf("convex error: %s (status: %d)", errResp.ErrorMessage, resp.StatusCode)
			}
			return fmt.Errorf("convex HTTP request failed with status %d: %s", resp.StatusCode, string(respBody))
		}

		// Success path
		if dest != nil {
			var successResp ConvexSuccessResponse
			if err := json.Unmarshal(respBody, &successResp); err != nil {
				// Fallback: try direct unmarshaling
				if errVal := json.Unmarshal(respBody, dest); errVal != nil {
					return fmt.Errorf("failed to unmarshal response: %w (body: %s)", errVal, string(respBody))
				}
				return nil
			}

			if successResp.Status == "success" {
				if err := json.Unmarshal(successResp.Value, dest); err != nil {
					return fmt.Errorf("failed to unmarshal success value: %w (value: %s)", err, string(successResp.Value))
				}
			} else {
				// Fallback: maybe it's not wrapped or has another structure
				if errVal := json.Unmarshal(respBody, dest); errVal != nil {
					return fmt.Errorf("failed to unmarshal response direct: %w (body: %s)", errVal, string(respBody))
				}
			}
		}
		return nil
	}

	return fmt.Errorf("all retries failed: %w", lastErr)
}

func (d *DB) handleLocalFallback(funcType, path string, args interface{}, dest interface{}) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	argsBytes, err := json.Marshal(args)
	if err != nil {
		return false, err
	}

	switch path {
	case "users:getByEmailS2S":
		slog.Info("Handling local fallback: users:getByEmailS2S")
		var queryArgs struct {
			Email string `json:"email"`
		}
		if err := json.Unmarshal(argsBytes, &queryArgs); err != nil {
			return false, err
		}
		userMap, exists := d.localUsers[queryArgs.Email]
		if !exists {
			if dest != nil {
				json.Unmarshal([]byte(`{}`), dest)
			}
			return true, nil
		}
		userBytes, _ := json.Marshal(userMap)
		if dest != nil {
			json.Unmarshal(userBytes, dest)
		}
		return true, nil

	case "users:getByLegacyId":
		slog.Info("Handling local fallback: users:getByLegacyId")
		var queryArgs struct {
			LegacyID string `json:"legacyId"`
		}
		if err := json.Unmarshal(argsBytes, &queryArgs); err != nil {
			return false, err
		}
		userMap, exists := d.localLegacyUsers[queryArgs.LegacyID]
		if !exists {
			if dest != nil {
				json.Unmarshal([]byte(`{}`), dest)
			}
			return true, nil
		}
		userBytes, _ := json.Marshal(userMap)
		if dest != nil {
			json.Unmarshal(userBytes, dest)
		}
		return true, nil

	case "users:registerUserS2S":
		slog.Info("Handling local fallback: users:registerUserS2S")
		var mutArgs struct {
			Email    string `json:"email"`
			Phone    string `json:"phone"`
			Role     string `json:"role"`
			LegacyID string `json:"legacyId"`
		}
		if err := json.Unmarshal(argsBytes, &mutArgs); err != nil {
			return false, err
		}
		
		userMap := map[string]interface{}{
			"id":                 mutArgs.LegacyID,
			"email":              mutArgs.Email,
			"phone":              mutArgs.Phone,
			"role":               mutArgs.Role,
			"status":             "ACTIVE",
			"preferred_language": "en",
		}
		d.localUsers[mutArgs.Email] = userMap
		d.localLegacyUsers[mutArgs.LegacyID] = userMap
		
		if dest != nil {
			json.Unmarshal([]byte(fmt.Sprintf(`"%s"`, mutArgs.LegacyID)), dest)
		}
		return true, nil

	case "documents:updateTypeAndStatus":
		slog.Info("Handling local fallback: documents:updateTypeAndStatus")
		var mutArgs struct {
			LegacyID             string `json:"legacyId"`
			DocumentType         string `json:"documentType"`
			OCRStatus            string `json:"ocrStatus"`
			ClassificationStatus string `json:"classificationStatus"`
		}
		if err := json.Unmarshal(argsBytes, &mutArgs); err != nil {
			return false, err
		}
		
		if dest != nil {
			json.Unmarshal([]byte(fmt.Sprintf(`"%s"`, mutArgs.LegacyID)), dest)
		}
		return true, nil

	case "documents:savePage":
		slog.Info("Handling local fallback: documents:savePage")
		var mutArgs struct {
			DocumentID string `json:"documentId"`
			PageNumber int    `json:"pageNumber"`
			StorageKey string `json:"storageKey"`
		}
		if err := json.Unmarshal(argsBytes, &mutArgs); err != nil {
			return false, err
		}
		
		pageMap := map[string]interface{}{
			"documentId": mutArgs.DocumentID,
			"pageNumber": mutArgs.PageNumber,
			"storageKey": mutArgs.StorageKey,
		}
		d.localPages[mutArgs.DocumentID] = append(d.localPages[mutArgs.DocumentID], pageMap)
		
		if dest != nil {
			json.Unmarshal([]byte(`"page_id_mock"`), dest)
		}
		return true, nil

	case "documents:saveExtraction":
		slog.Info("Handling local fallback: documents:saveExtraction")
		var mutArgs struct {
			DocumentID      string  `json:"documentId"`
			FieldName       string  `json:"fieldName"`
			FieldValue      string  `json:"fieldValue"`
			NormalizedValue string  `json:"normalizedValue"`
			PageNumber      int     `json:"pageNumber"`
			SourceText      string  `json:"sourceText"`
			Confidence      float64 `json:"confidence"`
			ReviewStatus    string  `json:"reviewStatus"`
		}
		if err := json.Unmarshal(argsBytes, &mutArgs); err != nil {
			return false, err
		}
		
		extMap := map[string]interface{}{
			"documentId":      mutArgs.DocumentID,
			"fieldName":       mutArgs.FieldName,
			"fieldValue":      mutArgs.FieldValue,
			"normalizedValue": mutArgs.NormalizedValue,
			"pageNumber":      mutArgs.PageNumber,
			"sourceText":      mutArgs.SourceText,
			"confidence":      mutArgs.Confidence,
			"reviewStatus":    mutArgs.ReviewStatus,
		}
		d.localExtractions[mutArgs.DocumentID] = append(d.localExtractions[mutArgs.DocumentID], extMap)
		
		if dest != nil {
			json.Unmarshal([]byte(`"ext_id_mock"`), dest)
		}
		return true, nil

	case "reviews:getQueue":
		slog.Info("Handling local fallback: reviews:getQueue")
		if dest != nil {
			json.Unmarshal([]byte(`[]`), dest)
		}
		return true, nil

	case "reviews:claim", "reviews:requestInformation", "reviews:approve", "reviews:escalate", "reviews:reject":
		slog.Info("Handling local fallback: reviews state update", "path", path)
		if dest != nil {
			json.Unmarshal([]byte(`true`), dest)
		}
		return true, nil

	case "reviews:addComment":
		slog.Info("Handling local fallback: reviews:addComment")
		var mutArgs struct {
			CaseID      string `json:"caseId"`
			ReviewerID  string `json:"reviewerId"`
			CommentText string `json:"commentText"`
			LegacyID    string `json:"legacyId"`
		}
		if err := json.Unmarshal(argsBytes, &mutArgs); err != nil {
			return false, err
		}

		commentMap := map[string]interface{}{
			"_id":         mutArgs.LegacyID,
			"caseId":      mutArgs.CaseID,
			"reviewerId":  mutArgs.ReviewerID,
			"commentText": mutArgs.CommentText,
			"createdAt":   new(time.Time).Format(time.RFC3339),
		}
		d.localComments[mutArgs.CaseID] = append(d.localComments[mutArgs.CaseID], commentMap)

		if dest != nil {
			json.Unmarshal([]byte(`true`), dest)
		}
		return true, nil

	case "reviews:listComments":
		slog.Info("Handling local fallback: reviews:listComments")
		var queryArgs struct {
			CaseID string `json:"caseId"`
		}
		if err := json.Unmarshal(argsBytes, &queryArgs); err != nil {
			return false, err
		}

		commentsList, exists := d.localComments[queryArgs.CaseID]
		if !exists {
			commentsList = []map[string]interface{}{}
		}

		commentsBytes, _ := json.Marshal(commentsList)
		if dest != nil {
			json.Unmarshal(commentsBytes, dest)
		}
		return true, nil

	case "reviews:checkSLAs":
		slog.Info("Handling local fallback: reviews:checkSLAs")
		if dest != nil {
			json.Unmarshal([]byte(`[]`), dest)
		}
		return true, nil
	}
 
	return false, nil
}
