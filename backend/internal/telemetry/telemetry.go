package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	mexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric"
	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
)

var (
	requestDurationHistogram metric.Float64Histogram
	requestCounter           metric.Int64Counter
)

// Init initializes the global OpenTelemetry TracerProvider, MeterProvider, and Propagators.
// It returns a shutdown function to flush all pending telemetry upon server exit.
func Init(ctx context.Context, cfg *config.Config) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("jynko"),
			semconv.ServiceVersionKey.String("0.1.0"),
			attribute.String("cloud.provider", "gcp"),
			attribute.String("cloud.platform", "gcp_cloud_run"),
			attribute.String("gcp.project_id", cfg.GoogleCloudProject),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenTelemetry resource: %w", err)
	}

	// -------------------------------------------------------------------------
	// 1. TracerProvider Setup
	// -------------------------------------------------------------------------
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.OTelSampleRatio))
	var tpOpts []sdktrace.TracerProviderOption
	tpOpts = append(tpOpts, sdktrace.WithResource(res), sdktrace.WithSampler(sampler))

	if cfg.EnableCloudTrace {
		exp, err := texporter.New(texporter.WithProjectID(cfg.GoogleCloudProject))
		if err != nil {
			slog.Warn("Failed to initialize Google Cloud Trace exporter; falling back to local tracing",
				slog.String("error", err.Error()),
			)
		} else {
			tpOpts = append(tpOpts, sdktrace.WithBatcher(exp))
			slog.Info("Google Cloud Trace exporter enabled", slog.String("gcp_project", cfg.GoogleCloudProject))
		}
	} else {
		slog.Info("Google Cloud Trace export disabled (running in local mode)")
	}

	tp := sdktrace.NewTracerProvider(tpOpts...)
	otel.SetTracerProvider(tp)

	// Set standard W3C TraceContext + Baggage propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// -------------------------------------------------------------------------
	// 2. MeterProvider Setup (Cloud Monitoring + Prometheus)
	// -------------------------------------------------------------------------
	var mpOpts []sdkmetric.Option
	mpOpts = append(mpOpts, sdkmetric.WithResource(res))

	if cfg.EnablePrometheusMetrics {
		promExp, err := prometheus.New()
		if err != nil {
			slog.Warn("Failed to initialize Prometheus exporter", slog.String("error", err.Error()))
		} else {
			mpOpts = append(mpOpts, sdkmetric.WithReader(promExp))
			slog.Info("Prometheus metrics reader enabled on /metrics")
		}
	}

	if cfg.EnableCloudMetrics {
		cExp, err := mexporter.New(mexporter.WithProjectID(cfg.GoogleCloudProject))
		if err != nil {
			slog.Warn("Failed to initialize Google Cloud Monitoring exporter", slog.String("error", err.Error()))
		} else {
			reader := sdkmetric.NewPeriodicReader(cExp, sdkmetric.WithInterval(60*time.Second))
			mpOpts = append(mpOpts, sdkmetric.WithReader(reader))
			slog.Info("Google Cloud Monitoring exporter enabled", slog.String("gcp_project", cfg.GoogleCloudProject))
		}
	}

	mp := sdkmetric.NewMeterProvider(mpOpts...)
	otel.SetMeterProvider(mp)

	// Initialize HTTP Latency Metrics Instruments
	meter := mp.Meter("jynko/http")
	requestDurationHistogram, err = meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Latency duration of HTTP requests in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	if err != nil {
		slog.Warn("Failed to create HTTP latency histogram instrument", slog.String("error", err.Error()))
	}

	requestCounter, err = meter.Int64Counter(
		"http.server.request.count",
		metric.WithDescription("Total number of HTTP requests processed"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		slog.Warn("Failed to create HTTP request counter instrument", slog.String("error", err.Error()))
	}

	// -------------------------------------------------------------------------
	// 3. Cleanup / Shutdown Function
	// -------------------------------------------------------------------------
	shutdown := func(shutdownCtx context.Context) error {
		var firstErr error
		if err := tp.Shutdown(shutdownCtx); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := mp.Shutdown(shutdownCtx); err != nil && firstErr == nil {
			firstErr = err
		}
		return firstErr
	}

	return shutdown, nil
}

// FilterUntracedRoutes returns true if the request path should be traced.
// It filters out /healthz and /metrics to prevent log and trace pollution.
func FilterUntracedRoutes(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return true
	}
	p := req.URL.Path
	return p != "/healthz" && p != "/metrics"
}

// GinTracingMiddleware returns the otelgin tracing middleware configured with route filtering.
func GinTracingMiddleware(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName, otelgin.WithFilter(FilterUntracedRoutes))
}

// PrometheusHandler returns an http.Handler serving the Prometheus metrics endpoint.
func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}

// HTTPMetricsMiddleware records request duration histograms and request count metrics.
func HTTPMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !FilterUntracedRoutes(c.Request) {
			c.Next()
			return
		}

		start := time.Now()

		c.Next()

		durationSec := time.Since(start).Seconds()
		statusCode := c.Writer.Status()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}

		attrs := []attribute.KeyValue{
			attribute.String("http.request.method", c.Request.Method),
			attribute.String("http.route", route),
			attribute.Int("http.response.status_code", statusCode),
		}

		if requestDurationHistogram != nil {
			requestDurationHistogram.Record(c.Request.Context(), durationSec, metric.WithAttributes(attrs...))
		}
		if requestCounter != nil {
			requestCounter.Add(c.Request.Context(), 1, metric.WithAttributes(attrs...))
		}
	}
}
