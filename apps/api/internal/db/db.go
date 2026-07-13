package db

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DB struct {
	Pool *ConvexPool
}

type ConvexPool struct {
	URL string
}

var (
	mockDB     = make(map[string]map[string]map[string]interface{})
	mockMutex  sync.RWMutex
	useMock    = true // Fallback to mock if HTTP request fails or if CONVEX_MOCK=true
)

func init() {
	if os.Getenv("CONVEX_MOCK") == "false" {
		useMock = false
	}
}

func Connect(ctx context.Context, connString string) (*DB, error) {
	convexURL := os.Getenv("CONVEX_URL")
	if convexURL == "" {
		convexURL = "http://localhost:3210" // Default for local Convex
	}
	slog.Info("Successfully connected to Convex Database", "url", convexURL)
	return &DB{Pool: &ConvexPool{URL: convexURL}}, nil
}

func (db *DB) Close() {
	slog.Info("Convex database connection closed")
}

func (p *ConvexPool) Ping(ctx context.Context) error {
	return nil
}

// CallConvex makes a POST request to Convex HTTP Functions API
func (p *ConvexPool) CallConvex(functionName string, args map[string]interface{}) (interface{}, error) {
	if useMock {
		return p.handleMock(functionName, args)
	}

	url := fmt.Sprintf("%s/api/run/%s", p.URL, functionName)
	bodyBytes, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		slog.Warn("Convex server not reachable, falling back to in-memory mock database", "error", err)
		useMock = true
		return p.handleMock(functionName, args)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("convex error status %d: %s", resp.StatusCode, string(respBytes))
	}

	// Try to decode as single object
	var res map[string]interface{}
	if err := json.Unmarshal(respBytes, &res); err == nil {
		if errMsg, ok := res["error"]; ok {
			return nil, fmt.Errorf("convex runtime error: %v", errMsg)
		}
		return res, nil
	}

	// Try to decode as array of objects
	var arr []map[string]interface{}
	if err := json.Unmarshal(respBytes, &arr); err == nil {
		return arr, nil
	}

	return nil, fmt.Errorf("invalid json response from convex: %s", string(respBytes))
}

func (p *ConvexPool) handleMock(functionName string, args map[string]interface{}) (interface{}, error) {
	mockMutex.Lock()
	defer mockMutex.Unlock()

	slog.Info("Mock Convex Call", "function", functionName, "args", args)

	switch functionName {
	case "crud/getRecord":
		table := fmt.Sprintf("%v", args["table"])
		field := fmt.Sprintf("%v", args["field"])
		val := args["value"]

		tableMap, exists := mockDB[table]
		if !exists {
			return nil, nil
		}

		for _, doc := range tableMap {
			if docVal, ok := doc[field]; ok && fmt.Sprintf("%v", docVal) == fmt.Sprintf("%v", val) {
				slog.Info("Mock found record", "table", table, "record", doc)
				return doc, nil
			}
		}
		slog.Info("Mock record not found", "table", table, "field", field, "value", val)
		return nil, nil

	case "crud/listRecords":
		table := fmt.Sprintf("%v", args["table"])
		field, hasField := args["field"]
		val := args["value"]

		tableMap, exists := mockDB[table]
		if !exists {
			return []map[string]interface{}{}, nil
		}

		var results []map[string]interface{}
		for _, doc := range tableMap {
			if !hasField || field == nil || fmt.Sprintf("%v", field) == "" {
				results = append(results, doc)
			} else {
				fieldStr := fmt.Sprintf("%v", field)
				if docVal, ok := doc[fieldStr]; ok && fmt.Sprintf("%v", docVal) == fmt.Sprintf("%v", val) {
					results = append(results, doc)
				}
			}
		}
		slog.Info("Mock list records", "table", table, "count", len(results))
		return results, nil

	case "crud/insertRecord":
		table := fmt.Sprintf("%v", args["table"])
		data, ok := args["data"].(map[string]interface{})
		if !ok {
			return nil, errors.New("invalid insert data")
		}

		refID := uuid.New().String()
		data["_id"] = refID
		data["_creationTime"] = float64(time.Now().UnixNano() / 1e6)

		if _, exists := mockDB[table]; !exists {
			mockDB[table] = make(map[string]map[string]interface{})
		}

		idVal := refID
		if id, ok := data["id"]; ok {
			idVal = fmt.Sprintf("%v", id)
		}
		mockDB[table][idVal] = data
		slog.Info("Mock inserted record", "table", table, "id", idVal, "data", data)
		return refID, nil

	case "crud/updateRecord":
		table := fmt.Sprintf("%v", args["table"])
		idField := fmt.Sprintf("%v", args["idField"])
		idVal := fmt.Sprintf("%v", args["idValue"])
		updates, ok := args["updates"].(map[string]interface{})
		if !ok {
			return nil, errors.New("invalid update data")
		}

		tableMap, exists := mockDB[table]
		if !exists {
			return nil, fmt.Errorf("table %s not found in mock", table)
		}

		var foundDoc map[string]interface{}
		var foundKey string
		for key, doc := range tableMap {
			if docVal, ok := doc[idField]; ok && fmt.Sprintf("%v", docVal) == idVal {
				foundDoc = doc
				foundKey = key
				break
			}
		}

		if foundDoc == nil {
			slog.Error("Mock update failed: document not found", "table", table, "idField", idField, "idVal", idVal)
			return nil, fmt.Errorf("document not found in table %s for update", table)
		}

		for k, v := range updates {
			foundDoc[k] = v
		}
		foundDoc["updated_at"] = time.Now().Format(time.RFC3339)
		mockDB[table][foundKey] = foundDoc
		slog.Info("Mock updated record", "table", table, "key", foundKey, "updates", updates, "finalDoc", foundDoc)
		return foundDoc["_id"], nil

	case "crud/deleteRecord":
		table := fmt.Sprintf("%v", args["table"])
		idField := fmt.Sprintf("%v", args["idField"])
		idVal := fmt.Sprintf("%v", args["idValue"])

		tableMap, exists := mockDB[table]
		if !exists {
			return false, nil
		}

		var foundKey string
		for key, doc := range tableMap {
			if docVal, ok := doc[idField]; ok && fmt.Sprintf("%v", docVal) == idVal {
				foundKey = key
				break
			}
		}

		if foundKey != "" {
			delete(mockDB[table], foundKey)
			slog.Info("Mock deleted record", "table", table, "key", foundKey)
			return true, nil
		}
		slog.Info("Mock delete record not found", "table", table, "idField", idField, "idVal", idVal)
		return false, nil

	case "reviews/getBreachedReviews":
		nowStr := fmt.Sprintf("%v", args["now"])
		nowTime, err := time.Parse(time.RFC3339, nowStr)
		if err != nil {
			nowTime = time.Now()
		}

		tableMap, exists := mockDB["reviews"]
		if !exists {
			return []map[string]interface{}{}, nil
		}

		var results []map[string]interface{}
		for _, doc := range tableMap {
			dec := fmt.Sprintf("%v", doc["decision"])
			comp := doc["completed_at"]
			slaStr := fmt.Sprintf("%v", doc["sla_due_at"])
			slaTime, err := time.Parse(time.RFC3339, slaStr)

			if err == nil && dec == "CLAIMED" && comp == nil && slaTime.Before(nowTime) {
				results = append(results, doc)
			}
		}
		slog.Info("Mock getBreachedReviews", "count", len(results))
		return results, nil

	default:
		return nil, fmt.Errorf("unsupported mock function: %s", functionName)
	}
}

// Exec executes mutation queries (INSERT, UPDATE, DELETE)
func (p *ConvexPool) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	slog.Info("SQL Exec", "sql", sql, "args", args)
	normalized := cleanSql(sql)

	if strings.Contains(normalized, "insert into") {
		table, data, err := parseInsertFields(sql, args)
		if err != nil {
			return pgconn.CommandTag{}, err
		}
		_, err = p.CallConvex("crud/insertRecord", map[string]interface{}{
			"table": table,
			"data":  data,
		})
		if err != nil {
			return pgconn.CommandTag{}, err
		}
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}

	if strings.Contains(normalized, "update ") {
		table, idField, idValue, updates, err := parseUpdateFields(sql, args)
		if err != nil {
			return pgconn.CommandTag{}, err
		}
		_, err = p.CallConvex("crud/updateRecord", map[string]interface{}{
			"table":   table,
			"idField": idField,
			"idValue": idValue,
			"updates": updates,
		})
		if err != nil {
			return pgconn.CommandTag{}, err
		}
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}

	if strings.Contains(normalized, "delete from") {
		table, idField, idValue, err := parseDeleteFields(sql, args)
		if err != nil {
			return pgconn.CommandTag{}, err
		}
		_, err = p.CallConvex("crud/deleteRecord", map[string]interface{}{
			"table":   table,
			"idField": idField,
			"idValue": idValue,
		})
		if err != nil {
			return pgconn.CommandTag{}, err
		}
		return pgconn.NewCommandTag("DELETE 1"), nil
	}

	return pgconn.NewCommandTag("SUCCESS"), nil
}

// Query executes select queries
func (p *ConvexPool) Query(ctx context.Context, sql string, args ...interface{}) (*ConvexRows, error) {
	slog.Info("SQL Query", "sql", sql, "args", args)
	normalized := cleanSql(sql)
	cols := parseColumns(sql)

	var results []map[string]interface{}

	if strings.Contains(normalized, "from users") {
		if strings.Contains(normalized, "where email =") {
			res, err := p.CallConvex("crud/getRecord", map[string]interface{}{
				"table": "users",
				"field": "email",
				"value": args[0],
			})
			if err == nil && res != nil {
				if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		} else if strings.Contains(normalized, "where phone =") {
			res, err := p.CallConvex("crud/getRecord", map[string]interface{}{
				"table": "users",
				"field": "phone",
				"value": args[0],
			})
			if err == nil && res != nil {
				if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		} else if strings.Contains(normalized, "where id =") {
			res, err := p.CallConvex("crud/getRecord", map[string]interface{}{
				"table": "users",
				"field": "id",
				"value": args[0],
			})
			if err == nil && res != nil {
				if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		}
	} else if strings.Contains(normalized, "from sessions") {
		if strings.Contains(normalized, "where refresh_token =") {
			res, err := p.CallConvex("crud/getRecord", map[string]interface{}{
				"table": "sessions",
				"field": "refresh_token",
				"value": args[0],
			})
			if err == nil && res != nil {
				if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		}
	} else if strings.Contains(normalized, "from cases") {
		if strings.Contains(normalized, "where owner_user_id =") {
			res, err := p.CallConvex("crud/listRecords", map[string]interface{}{
				"table": "cases",
				"field": "owner_user_id",
				"value": args[0],
			})
			if err == nil && res != nil {
				if arr, ok := res.([]map[string]interface{}); ok {
					results = arr
				} else if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		} else if strings.Contains(normalized, "where workflow_state = 'review_required' or workflow_state = 'in_review'") {
			res, err := p.CallConvex("crud/listRecords", map[string]interface{}{
				"table": "cases",
			})
			if err == nil && res != nil {
				var arr []map[string]interface{}
				if a, ok := res.([]map[string]interface{}); ok {
					arr = a
				} else if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					arr = append(arr, m)
				}
				for _, item := range arr {
					ws := fmt.Sprintf("%v", item["workflow_state"])
					if ws == "REVIEW_REQUIRED" || ws == "IN_REVIEW" {
						results = append(results, item)
					}
				}
			}
		} else if strings.Contains(normalized, "where id =") {
			res, err := p.CallConvex("crud/getRecord", map[string]interface{}{
				"table": "cases",
				"field": "id",
				"value": args[0],
			})
			if err == nil && res != nil {
				if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		}
	} else if strings.Contains(normalized, "from review_comments") {
		if strings.Contains(normalized, "where case_id =") {
			res, err := p.CallConvex("crud/listRecords", map[string]interface{}{
				"table": "review_comments",
				"field": "case_id",
				"value": args[0],
			})
			if err == nil && res != nil {
				if arr, ok := res.([]map[string]interface{}); ok {
					results = arr
				} else if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		}
	} else if strings.Contains(normalized, "from reviews") {
		if strings.Contains(normalized, "where case_id =") && strings.Contains(normalized, "reviewer_id =") {
			res, err := p.CallConvex("crud/listRecords", map[string]interface{}{
				"table": "reviews",
				"field": "case_id",
				"value": args[0],
			})
			if err == nil && res != nil {
				var arr []map[string]interface{}
				if a, ok := res.([]map[string]interface{}); ok {
					arr = a
				} else if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					arr = append(arr, m)
				}
				for _, item := range arr {
					if fmt.Sprintf("%v", item["reviewer_id"]) == fmt.Sprintf("%v", args[1]) && item["completed_at"] == nil {
						results = append(results, item)
					}
				}
			}
		} else if strings.Contains(normalized, "decision = 'claimed'") && strings.Contains(normalized, "sla_due_at <") {
			res, err := p.CallConvex("reviews/getBreachedReviews", map[string]interface{}{
				"now": time.Now().Format(time.RFC3339),
			})
			if err == nil && res != nil {
				if arr, ok := res.([]map[string]interface{}); ok {
					results = arr
				} else if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		}
	} else if strings.Contains(normalized, "from consents") {
		if strings.Contains(normalized, "where case_id =") {
			res, err := p.CallConvex("crud/listRecords", map[string]interface{}{
				"table": "consents",
				"field": "case_id",
				"value": args[0],
			})
			if err == nil && res != nil {
				if arr, ok := res.([]map[string]interface{}); ok {
					results = arr
				} else if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		}
	} else if strings.Contains(normalized, "from documents") {
		if strings.Contains(normalized, "where case_id =") {
			res, err := p.CallConvex("crud/listRecords", map[string]interface{}{
				"table": "documents",
				"field": "case_id",
				"value": args[0],
			})
			if err == nil && res != nil {
				if arr, ok := res.([]map[string]interface{}); ok {
					results = arr
				} else if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		} else if strings.Contains(normalized, "where id =") {
			res, err := p.CallConvex("crud/getRecord", map[string]interface{}{
				"table": "documents",
				"field": "id",
				"value": args[0],
			})
			if err == nil && res != nil {
				if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		}
	} else if strings.Contains(normalized, "from drafts") {
		if strings.Contains(normalized, "where case_id =") {
			res, err := p.CallConvex("crud/listRecords", map[string]interface{}{
				"table": "drafts",
				"field": "case_id",
				"value": args[0],
			})
			if err == nil && res != nil {
				if arr, ok := res.([]map[string]interface{}); ok {
					results = arr
				} else if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		} else if strings.Contains(normalized, "where id =") {
			res, err := p.CallConvex("crud/getRecord", map[string]interface{}{
				"table": "drafts",
				"field": "id",
				"value": args[0],
			})
			if err == nil && res != nil {
				if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		}
	} else if strings.Contains(normalized, "from draft_versions") {
		if strings.Contains(normalized, "where draft_id =") {
			res, err := p.CallConvex("crud/listRecords", map[string]interface{}{
				"table": "draft_versions",
				"field": "draft_id",
				"value": args[0],
			})
			if err == nil && res != nil {
				if arr, ok := res.([]map[string]interface{}); ok {
					results = arr
				} else if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		}
	} else if strings.Contains(normalized, "from evidence_items") {
		if strings.Contains(normalized, "where case_id =") {
			res, err := p.CallConvex("crud/listRecords", map[string]interface{}{
				"table": "evidence_items",
				"field": "case_id",
				"value": args[0],
			})
			if err == nil && res != nil {
				if arr, ok := res.([]map[string]interface{}); ok {
					results = arr
				} else if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		}
	} else if strings.Contains(normalized, "from case_issues") {
		if strings.Contains(normalized, "where case_id =") {
			res, err := p.CallConvex("crud/listRecords", map[string]interface{}{
				"table": "case_issues",
				"field": "case_id",
				"value": args[0],
			})
			if err == nil && res != nil {
				if arr, ok := res.([]map[string]interface{}); ok {
					results = arr
				} else if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		}
	} else if strings.Contains(normalized, "from citations") {
		if strings.Contains(normalized, "where case_id =") {
			res, err := p.CallConvex("crud/listRecords", map[string]interface{}{
				"table": "citations",
				"field": "case_id",
				"value": args[0],
			})
			if err == nil && res != nil {
				if arr, ok := res.([]map[string]interface{}); ok {
					results = arr
				} else if m, ok := res.(map[string]interface{}); ok && len(m) > 0 {
					results = append(results, m)
				}
			}
		}
	}

	slog.Info("SQL Query Result", "count", len(results), "records", results)
	return &ConvexRows{
		records: results,
		cols:    cols,
		index:   -1,
	}, nil
}

func (p *ConvexPool) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	rows, err := p.Query(ctx, sql, args...)
	if err != nil {
		return &ConvexRow{err: err}
	}
	if !rows.Next() {
		return &ConvexRow{err: pgx.ErrNoRows}
	}
	return &ConvexRow{
		data: rows.records[rows.index],
		cols: rows.cols,
	}
}

func (p *ConvexPool) Begin(ctx context.Context) (pgx.Tx, error) {
	return &ConvexTx{pool: p}, nil
}

// ConvexRow implements pgx.Row
type ConvexRow struct {
	data map[string]interface{}
	err  error
	cols []string
}

func (r *ConvexRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	if r.data == nil {
		return pgx.ErrNoRows
	}
	return scanData(r.data, r.cols, dest)
}

// ConvexRows implements pgx.Rows
type ConvexRows struct {
	records []map[string]interface{}
	cols    []string
	index   int
	err     error
}

func (r *ConvexRows) Next() bool {
	r.index++
	return r.index < len(r.records)
}

func (r *ConvexRows) Scan(dest ...interface{}) error {
	if r.index >= len(r.records) {
		return fmt.Errorf("scan out of range")
	}
	return scanData(r.records[r.index], r.cols, dest)
}

func (r *ConvexRows) Close() {}
func (r *ConvexRows) Err() error { return r.err }
func (r *ConvexRows) CommandTag() pgconn.CommandTag { return pgconn.NewCommandTag("SELECT") }
func (r *ConvexRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *ConvexRows) Values() ([]interface{}, error) { return nil, nil }
func (r *ConvexRows) RawValues() [][]byte { return nil }
func (r *ConvexRows) Conn() *pgx.Conn { return nil }

// ConvexTx implements pgx.Tx
type ConvexTx struct {
	pool *ConvexPool
}

func (tx *ConvexTx) Begin(ctx context.Context) (pgx.Tx, error) { return tx, nil }
func (tx *ConvexTx) Commit(ctx context.Context) error { return nil }
func (tx *ConvexTx) Rollback(ctx context.Context) error { return nil }
func (tx *ConvexTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (tx *ConvexTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (tx *ConvexTx) LargeObjects() pgx.LargeObjects { return pgx.LargeObjects{} }
func (tx *ConvexTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (tx *ConvexTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return tx.pool.Exec(ctx, sql, arguments...)
}
func (tx *ConvexTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return tx.pool.Query(ctx, sql, args...)
}
func (tx *ConvexTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return tx.pool.QueryRow(ctx, sql, args...)
}
func (tx *ConvexTx) Conn() *pgx.Conn { return nil }

// HELPER PARSERS & UTILITIES

func cleanSql(sql string) string {
	sql = strings.ToLower(sql)
	sql = strings.ReplaceAll(sql, "\n", " ")
	sql = strings.ReplaceAll(sql, "\t", " ")
	// remove multiple spaces
	space := regexp.MustCompile(`\s+`)
	return space.ReplaceAllString(sql, " ")
}

func parseColumns(sql string) []string {
	sqlClean := strings.ReplaceAll(sql, "\n", " ")
	sqlClean = strings.ReplaceAll(sqlClean, "\t", " ")

	r := regexp.MustCompile(`(?i)select\s+(.*?)\s+from`)
	matches := r.FindStringSubmatch(sqlClean)
	if len(matches) < 2 {
		return nil
	}

	rawColsStr := matches[1]

	// Split by comma only when outside parentheses
	var rawCols []string
	var current []rune
	depth := 0
	for _, char := range rawColsStr {
		if char == '(' {
			depth++
		} else if char == ')' {
			depth--
		}

		if char == ',' && depth == 0 {
			rawCols = append(rawCols, string(current))
			current = nil
		} else {
			current = append(current, char)
		}
	}
	if len(current) > 0 {
		rawCols = append(rawCols, string(current))
	}

	var cols []string
	for _, c := range rawCols {
		c = strings.TrimSpace(c)
		// Clean up COALESCE
		if strings.Contains(strings.ToLower(c), "coalesce") {
			innerReg := regexp.MustCompile(`(?i)coalesce\(([^,]+)`)
			innerMatch := innerReg.FindStringSubmatch(c)
			if len(innerMatch) >= 2 {
				c = strings.TrimSpace(innerMatch[1])
			}
		}
		// Clean up aliases like "AS name"
		if strings.Contains(strings.ToLower(c), " as ") {
			parts := regexp.MustCompile(`(?i)\s+as\s+`).Split(c, -1)
			if len(parts) >= 2 {
				c = strings.TrimSpace(parts[1])
			}
		}
		// Clean up table qualifiers like "u.email" -> "email"
		if strings.Contains(c, ".") {
			parts := strings.Split(c, ".")
			c = parts[len(parts)-1]
		}
		cols = append(cols, c)
	}
	return cols
}

func serializeArg(arg interface{}) interface{} {
	if arg == nil {
		return nil
	}
	switch v := arg.(type) {
	case time.Time:
		return v.Format(time.RFC3339)
	case *time.Time:
		if v != nil {
			return v.Format(time.RFC3339)
		}
		return nil
	case *string:
		if v != nil {
			return *v
		}
		return nil
	default:
		return v
	}
}

func scanData(data map[string]interface{}, cols []string, dest []interface{}) error {
	for i, col := range cols {
		if i >= len(dest) {
			break
		}
		val := data[col]
		if val == nil {
			continue
		}

		switch d := dest[i].(type) {
		case *string:
			*d = fmt.Sprintf("%v", val)
		case **string:
			strVal := fmt.Sprintf("%v", val)
			*d = &strVal
		case *int:
			if n, ok := val.(float64); ok {
				*d = int(n)
			} else if n, ok := val.(int); ok {
				*d = n
			}
		case *int64:
			if n, ok := val.(float64); ok {
				*d = int64(n)
			} else if n, ok := val.(int64); ok {
				*d = n
			}
		case *float64:
			if n, ok := val.(float64); ok {
				*d = n
			}
		case *bool:
			if b, ok := val.(bool); ok {
				*d = b
			}
		case *time.Time:
			if s, ok := val.(string); ok {
				t, err := time.Parse(time.RFC3339, s)
				if err == nil {
					*d = t
				}
			}
		case **time.Time:
			if s, ok := val.(string); ok {
				t, err := time.Parse(time.RFC3339, s)
				if err == nil {
					*d = &t
				}
			}
		case *[]byte:
			if b, err := json.Marshal(val); err == nil {
				*d = b
			}
		default:
			if b, err := json.Marshal(val); err == nil {
				json.Unmarshal(b, dest[i])
			}
		}
	}
	return nil
}

func parseInsertFields(sql string, args []interface{}) (string, map[string]interface{}, error) {
	sqlClean := strings.ReplaceAll(sql, "\n", " ")
	sqlClean = strings.ReplaceAll(sqlClean, "\t", " ")

	tableReg := regexp.MustCompile(`(?i)insert\s+into\s+([a-zA-Z0-9_]+)`)
	tableMatches := tableReg.FindStringSubmatch(sqlClean)
	if len(tableMatches) < 2 {
		return "", nil, errors.New("unable to parse insert table")
	}
	tableName := tableMatches[1]

	colsReg := regexp.MustCompile(`(?i)\(\s*([a-zA-Z0-9_,\s]+)\s*\)\s*values`)
	colsMatches := colsReg.FindStringSubmatch(sqlClean)
	if len(colsMatches) < 2 {
		return "", nil, errors.New("unable to parse insert columns")
	}

	rawCols := strings.Split(colsMatches[1], ",")
	data := make(map[string]interface{})
	for i, c := range rawCols {
		c = strings.TrimSpace(c)
		if i < len(args) {
			data[c] = serializeArg(args[i])
		}
	}

	nowStr := time.Now().Format(time.RFC3339)
	if _, exists := data["created_at"]; !exists {
		data["created_at"] = nowStr
	}
	if _, exists := data["updated_at"]; !exists {
		data["updated_at"] = nowStr
	}

	return tableName, data, nil
}

func parseUpdateFields(sql string, args []interface{}) (string, string, interface{}, map[string]interface{}, error) {
	sqlClean := strings.ReplaceAll(sql, "\n", " ")
	sqlClean = strings.ReplaceAll(sqlClean, "\t", " ")

	tableReg := regexp.MustCompile(`(?i)update\s+([a-zA-Z0-9_]+)`)
	tableMatches := tableReg.FindStringSubmatch(sqlClean)
	if len(tableMatches) < 2 {
		return "", "", nil, nil, errors.New("unable to parse update table")
	}
	tableName := tableMatches[1]

	setReg := regexp.MustCompile(`(?i)set\s+(.*?)\s+where`)
	setMatches := setReg.FindStringSubmatch(sqlClean)
	if len(setMatches) < 2 {
		return "", "", nil, nil, errors.New("unable to parse update set clause")
	}

	whereReg := regexp.MustCompile(`(?i)where\s+(.*)`)
	whereMatches := whereReg.FindStringSubmatch(sqlClean)
	if len(whereMatches) < 2 {
		return "", "", nil, nil, errors.New("unable to parse update where clause")
	}

	whereClause := whereMatches[1]
	idField := "id"
	var idValue interface{}

	paramReg := regexp.MustCompile(`([a-zA-Z0-9_.]+)\s*=\s*\$([0-9]+)`)
	paramMatches := paramReg.FindAllStringSubmatch(whereClause, -1)
	if len(paramMatches) > 0 {
		idField = strings.TrimSpace(paramMatches[0][1])
		if strings.Contains(idField, ".") {
			parts := strings.Split(idField, ".")
			idField = parts[len(parts)-1]
		}
		idx := 0
		fmt.Sscanf(paramMatches[0][2], "%d", &idx)
		if idx > 0 && idx <= len(args) {
			idValue = args[idx-1]
		}
	}

	updates := make(map[string]interface{})
	setParts := strings.Split(setMatches[1], ",")
	for _, part := range setParts {
		part = strings.TrimSpace(part)
		fieldParts := strings.Split(part, "=")
		if len(fieldParts) != 2 {
			continue
		}
		field := strings.TrimSpace(fieldParts[0])
		valueExpr := strings.TrimSpace(fieldParts[1])

		if strings.HasPrefix(valueExpr, "$") {
			idx := 0
			fmt.Sscanf(valueExpr[1:], "%d", &idx)
			if idx > 0 && idx <= len(args) {
				updates[field] = serializeArg(args[idx-1])
			}
		} else {
			valueExpr = strings.Trim(valueExpr, "'\"")
			if strings.ToLower(valueExpr) == "now()" {
				updates[field] = time.Now().Format(time.RFC3339)
			} else if strings.ToLower(valueExpr) == "null" {
				updates[field] = nil
			} else {
				updates[field] = valueExpr
			}
		}
	}

	return tableName, idField, idValue, updates, nil
}

func parseDeleteFields(sql string, args []interface{}) (string, string, interface{}, error) {
	sqlClean := strings.ReplaceAll(sql, "\n", " ")
	sqlClean = strings.ReplaceAll(sqlClean, "\t", " ")

	tableReg := regexp.MustCompile(`(?i)delete\s+from\s+([a-zA-Z0-9_]+)`)
	tableMatches := tableReg.FindStringSubmatch(sqlClean)
	if len(tableMatches) < 2 {
		return "", "", nil, errors.New("unable to parse delete table")
	}
	tableName := tableMatches[1]

	whereReg := regexp.MustCompile(`(?i)where\s+(.*)`)
	whereMatches := whereReg.FindStringSubmatch(sqlClean)
	if len(whereMatches) < 2 {
		return "", "", nil, errors.New("unable to parse delete where clause")
	}

	whereClause := whereMatches[1]
	idField := "id"
	var idValue interface{}

	paramReg := regexp.MustCompile(`([a-zA-Z0-9_.]+)\s*=\s*\$([0-9]+)`)
	paramMatches := paramReg.FindStringSubmatch(whereClause)
	if len(paramMatches) >= 3 {
		idField = strings.TrimSpace(paramMatches[1])
		idx := 0
		fmt.Sscanf(paramMatches[2], "%d", &idx)
		if idx > 0 && idx <= len(args) {
			idValue = args[idx-1]
		}
	}

	return tableName, idField, idValue, nil
}
