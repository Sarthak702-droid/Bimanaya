package db

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"bimanyaya/api/internal/config"
)

type DB struct {
	convexURL      string
	clerkSecretKey string
	client         *http.Client
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

func (d *DB) call(ctx context.Context, funcType, path string, args interface{}, dest interface{}) error {
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
		if d.clerkSecretKey != "" {
			// Bearer authentication (Clerk Secret Key or Convex deploy key)
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", d.clerkSecretKey))
		}

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
			if err := json.Unmarshal(respBody, dest); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w (body: %s)", err, string(respBody))
			}
		}
		return nil
	}

	return fmt.Errorf("all retries failed: %w", lastErr)
}
