package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestFilterUntracedRoutes(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/healthz", false},
		{"/metrics", false},
		{"/api/v1/projects", true},
		{"/api/v1/projects/proj-jynko-core/items", true},
		{"/home", true},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		actual := FilterUntracedRoutes(req)
		if actual != tt.expected {
			t.Errorf("path %s: expected %v, got %v", tt.path, tt.expected, actual)
		}
	}
}

func TestTelemetryLifecycleAndMiddleware(t *testing.T) {
	cfg := &config.Config{
		GoogleCloudProject:      "test-project",
		EnableCloudTrace:        false,
		EnableCloudMetrics:      false,
		EnablePrometheusMetrics: true,
		OTelSampleRatio:         1.0,
	}

	shutdown, err := Init(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to init telemetry: %v", err)
	}
	defer func() {
		_ = shutdown(context.Background())
	}()

	r := gin.New()
	r.Use(GinTracingMiddleware("jynko-test"))
	r.Use(HTTPMetricsMiddleware())

	var capturedSpan trace.Span
	r.GET("/api/v1/test", func(c *gin.Context) {
		capturedSpan = trace.SpanFromContext(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/healthz", func(c *gin.Context) {
		capturedSpan = trace.SpanFromContext(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/metrics", gin.WrapH(PrometheusHandler()))

	// 1. Traced route
	reqTraced := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	wTraced := httptest.NewRecorder()
	r.ServeHTTP(wTraced, reqTraced)

	if wTraced.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", wTraced.Code)
	}
	if capturedSpan == nil || !capturedSpan.SpanContext().IsValid() {
		t.Errorf("expected valid OpenTelemetry span context on /api/v1/test")
	}

	// 2. Untraced /healthz route
	reqHealth := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	wHealth := httptest.NewRecorder()
	r.ServeHTTP(wHealth, reqHealth)

	if wHealth.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", wHealth.Code)
	}
	if capturedSpan != nil && capturedSpan.SpanContext().IsValid() {
		t.Errorf("expected no valid span on /healthz")
	}

	// 3. Prometheus /metrics endpoint
	time.Sleep(10 * time.Millisecond)
	reqProm := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	wProm := httptest.NewRecorder()
	r.ServeHTTP(wProm, reqProm)

	if wProm.Code != http.StatusOK {
		t.Errorf("expected 200 on /metrics, got %d", wProm.Code)
	}
	promBody := wProm.Body.String()
	if len(promBody) == 0 {
		t.Errorf("expected non-empty Prometheus metrics body")
	}
}

func TestTracerProviderRegistered(t *testing.T) {
	tp := otel.GetTracerProvider()
	if tp == nil {
		t.Fatalf("expected non-nil global tracer provider")
	}
}
