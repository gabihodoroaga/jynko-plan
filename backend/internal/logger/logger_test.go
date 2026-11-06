package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestGCPStructuredLoggingFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(Config{
		Format: "json",
		Level:  slog.LevelDebug,
		Writer: buf,
	})

	l.Info("Server booted successfully", slog.String("port", "8080"))

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal JSON log: %v, raw: %s", err, buf.String())
	}

	// 1. Verify severity is used instead of level
	if _, ok := entry["level"]; ok {
		t.Errorf("expected 'level' field to be replaced by 'severity'")
	}
	if sev, ok := entry["severity"].(string); !ok || sev != "INFO" {
		t.Errorf("expected severity 'INFO', got %v", entry["severity"])
	}

	// 2. Verify message is used instead of msg
	if _, ok := entry["msg"]; ok {
		t.Errorf("expected 'msg' field to be replaced by 'message'")
	}
	if msg, ok := entry["message"].(string); !ok || msg != "Server booted successfully" {
		t.Errorf("expected message 'Server booted successfully', got %v", entry["message"])
	}

	// 3. Verify timestamp
	if ts, ok := entry["timestamp"].(string); !ok || ts == "" {
		t.Errorf("expected non-empty 'timestamp' string, got %v", entry["timestamp"])
	}

	// 4. Verify custom attributes
	if port, ok := entry["port"].(string); !ok || port != "8080" {
		t.Errorf("expected port '8080', got %v", entry["port"])
	}
}

func TestGCPLevelSeverities(t *testing.T) {
	tests := []struct {
		level    slog.Level
		expected string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{LevelNotice, "NOTICE"},
		{slog.LevelWarn, "WARNING"},
		{slog.LevelError, "ERROR"},
		{LevelCritical, "CRITICAL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			buf := &bytes.Buffer{}
			l := New(Config{
				Format: "json",
				Level:  slog.LevelDebug,
				Writer: buf,
			})

			l.Log(context.Background(), tt.level, "test event")

			var entry map[string]any
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if entry["severity"] != tt.expected {
				t.Errorf("level %v: expected severity %s, got %v", tt.level, tt.expected, entry["severity"])
			}
		})
	}
}

func TestTraceContextParsing(t *testing.T) {
	projectID := "my-gcp-project"

	t.Run("X-Cloud-Trace-Context sampled", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-Cloud-Trace-Context", "105445aa7843bc8bf206b120001000/12345;o=1")

		attrs := TraceContext(projectID, h)
		m := make(map[string]any)
		for _, a := range attrs {
			m[a.Key] = a.Value.Any()
		}

		expectedTrace := "projects/my-gcp-project/traces/105445aa7843bc8bf206b120001000"
		if m["logging.googleapis.com/trace"] != expectedTrace {
			t.Errorf("trace mismatch: expected %s, got %v", expectedTrace, m["logging.googleapis.com/trace"])
		}
		if m["logging.googleapis.com/spanId"] != "12345" {
			t.Errorf("spanId mismatch: expected 12345, got %v", m["logging.googleapis.com/spanId"])
		}
		if m["logging.googleapis.com/trace_sampled"] != true {
			t.Errorf("trace_sampled mismatch: expected true, got %v", m["logging.googleapis.com/trace_sampled"])
		}
	})

	t.Run("W3C traceparent", func(t *testing.T) {
		h := http.Header{}
		h.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

		attrs := TraceContext(projectID, h)
		m := make(map[string]any)
		for _, a := range attrs {
			m[a.Key] = a.Value.Any()
		}

		expectedTrace := "projects/my-gcp-project/traces/4bf92f3577b34da6a3ce929d0e0e4736"
		if m["logging.googleapis.com/trace"] != expectedTrace {
			t.Errorf("trace mismatch: expected %s, got %v", expectedTrace, m["logging.googleapis.com/trace"])
		}
		if m["logging.googleapis.com/spanId"] != "00f067aa0ba902b7" {
			t.Errorf("spanId mismatch: expected 00f067aa0ba902b7, got %v", m["logging.googleapis.com/spanId"])
		}
		if m["logging.googleapis.com/trace_sampled"] != true {
			t.Errorf("trace_sampled mismatch: expected true, got %v", m["logging.googleapis.com/trace_sampled"])
		}
	})
}

func TestHTTPRequestAttr(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(Config{
		Format: "json",
		Level:  slog.LevelInfo,
		Writer: buf,
	})

	reqAttr := HTTPRequestAttr(HTTPRequestPayload{
		RequestMethod: "GET",
		RequestURL:    "/api/v1/projects",
		Status:        200,
		ResponseSize:  1024,
		Latency:       "0.004123s",
		RemoteIP:      "192.168.1.10",
		UserAgent:     "Mozilla/5.0",
		Protocol:      "HTTP/1.1",
	})

	l.LogAttrs(context.Background(), slog.LevelInfo, "HTTP request handled", reqAttr)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	httpReq, ok := entry["httpRequest"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'httpRequest' object in log entry, got %v", entry["httpRequest"])
	}

	if httpReq["requestMethod"] != "GET" {
		t.Errorf("expected requestMethod GET, got %v", httpReq["requestMethod"])
	}
	if httpReq["status"] != float64(200) {
		t.Errorf("expected status 200, got %v", httpReq["status"])
	}
	if httpReq["latency"] != "0.004123s" {
		t.Errorf("expected latency 0.004123s, got %v", httpReq["latency"])
	}
}

func TestContextHelpers(t *testing.T) {
	buf := &bytes.Buffer{}
	customLogger := New(Config{
		Format: "text",
		Writer: buf,
	})

	ctx := WithLogger(context.Background(), customLogger)
	retrieved := FromContext(ctx)
	if retrieved != customLogger {
		t.Fatalf("expected custom logger to be retrieved from context")
	}

	emptyCtx := context.Background()
	defaultRetrieved := FromContext(emptyCtx)
	if defaultRetrieved == nil {
		t.Fatalf("expected non-nil default logger from empty context")
	}
}

func TestTextFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(Config{
		Format: "text",
		Level:  slog.LevelInfo,
		Writer: buf,
	})

	l.Info("Plain text test", slog.String("key", "val"))
	out := buf.String()
	if !strings.Contains(out, "Plain text test") || !strings.Contains(out, "key=val") {
		t.Errorf("unexpected text log format: %s", out)
	}
}
