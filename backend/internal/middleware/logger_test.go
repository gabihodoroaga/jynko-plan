package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/logger"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestStructuredLoggerMiddleware(t *testing.T) {
	buf := &bytes.Buffer{}
	l := logger.New(logger.Config{
		Format: "json",
		Level:  slog.LevelDebug,
		Writer: buf,
	})
	slog.SetDefault(l)

	r := gin.New()
	r.Use(StructuredLogger("test-gcp-project"))
	r.GET("/test-endpoint", func(c *gin.Context) {
		c.Set(userContextKey, model.UserContext{
			UserID: "user-123",
			Email:  "test@example.com",
			Name:   "Test User",
		})
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test-endpoint?query=1", nil)
	req.Header.Set("X-Cloud-Trace-Context", "trace-abc-123/span-456;o=1")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log JSON: %v, raw: %s", err, buf.String())
	}

	// Verify severity
	if entry["severity"] != "INFO" {
		t.Errorf("expected severity INFO, got %v", entry["severity"])
	}

	// Verify trace correlation
	expectedTrace := "projects/test-gcp-project/traces/trace-abc-123"
	if entry["logging.googleapis.com/trace"] != expectedTrace {
		t.Errorf("expected trace %s, got %v", expectedTrace, entry["logging.googleapis.com/trace"])
	}
	if entry["logging.googleapis.com/spanId"] != "span-456" {
		t.Errorf("expected spanId span-456, got %v", entry["logging.googleapis.com/spanId"])
	}

	// Verify user context
	if entry["user_id"] != "user-123" {
		t.Errorf("expected user_id user-123, got %v", entry["user_id"])
	}
	if entry["user_email"] != "test@example.com" {
		t.Errorf("expected user_email test@example.com, got %v", entry["user_email"])
	}

	// Verify httpRequest payload
	httpReq, ok := entry["httpRequest"].(map[string]any)
	if !ok {
		t.Fatalf("expected httpRequest object in log, got %v", entry["httpRequest"])
	}
	if httpReq["requestMethod"] != "GET" {
		t.Errorf("expected requestMethod GET, got %v", httpReq["requestMethod"])
	}
	if httpReq["requestUrl"] != "/test-endpoint?query=1" {
		t.Errorf("expected requestUrl with query, got %v", httpReq["requestUrl"])
	}
	if httpReq["status"] != float64(200) {
		t.Errorf("expected status 200, got %v", httpReq["status"])
	}
}

func TestStructuredRecoveryMiddleware(t *testing.T) {
	buf := &bytes.Buffer{}
	l := logger.New(logger.Config{
		Format: "json",
		Level:  slog.LevelDebug,
		Writer: buf,
	})
	slog.SetDefault(l)

	r := gin.New()
	r.Use(StructuredRecovery("test-gcp-project"))
	r.GET("/panic-endpoint", func(c *gin.Context) {
		panic("database connection exploded")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic-endpoint", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse recovery log JSON: %v, raw: %s", err, buf.String())
	}

	// Severity should be CRITICAL
	if entry["severity"] != "CRITICAL" {
		t.Errorf("expected CRITICAL severity, got %v", entry["severity"])
	}

	if entry["panic"] != "database connection exploded" {
		t.Errorf("expected panic details, got %v", entry["panic"])
	}

	if _, ok := entry["stack_trace"].(string); !ok {
		t.Errorf("expected stack_trace in log entry")
	}
}
