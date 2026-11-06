package logger

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// Google Cloud Logging Severity levels
// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#logseverity
const (
	SeverityDefault   = "DEFAULT"
	SeverityDebug     = "DEBUG"
	SeverityInfo      = "INFO"
	SeverityNotice    = "NOTICE"
	SeverityWarning   = "WARNING"
	SeverityError     = "ERROR"
	SeverityCritical  = "CRITICAL"
	SeverityAlert     = "ALERT"
	SeverityEmergency = "EMERGENCY"
)

// Custom slog.Level definitions mapping into GCP Cloud Logging
const (
	LevelNotice   = slog.Level(2)
	LevelCritical = slog.Level(12)
)

// LevelToSeverity maps a slog.Level to its corresponding GCP severity string.
func LevelToSeverity(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return SeverityDebug
	case level < LevelNotice:
		return SeverityInfo
	case level < slog.LevelWarn:
		return SeverityNotice
	case level < slog.LevelError:
		return SeverityWarning
	case level < LevelCritical:
		return SeverityError
	default:
		return SeverityCritical
	}
}

// ParseLevel converts a level string into a slog.Level.
func ParseLevel(lvl string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(lvl)) {
	case "debug":
		return slog.LevelDebug
	case "notice":
		return LevelNotice
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "critical":
		return LevelCritical
	default:
		return slog.LevelInfo
	}
}

// Config configures the structured logger.
type Config struct {
	Format    string     // "json" or "text" (defaults to "json")
	Level     slog.Level // minimum logging level
	ProjectID string     // GCP Project ID for Cloud Trace correlation
	AddSource bool       // include source code location
	Writer    io.Writer  // output destination (defaults to os.Stdout)
}

// New creates a new *slog.Logger configured for Google Cloud Logging.
func New(cfg Config) *slog.Logger {
	out := cfg.Writer
	if out == nil {
		out = os.Stdout
	}

	opts := &slog.HandlerOptions{
		AddSource: cfg.AddSource,
		Level:     cfg.Level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Only transform root attributes
			if len(groups) == 0 {
				switch a.Key {
				case slog.LevelKey:
					// GCP Cloud Logging expects "severity" instead of "level"
					a.Key = "severity"
					if lvl, ok := a.Value.Any().(slog.Level); ok {
						a.Value = slog.StringValue(LevelToSeverity(lvl))
					}
				case slog.MessageKey:
					// GCP Cloud Logging expects "message" as the primary summary field
					a.Key = "message"
				case slog.TimeKey:
					// GCP Cloud Logging accepts "timestamp" formatted in RFC3339Nano
					a.Key = "timestamp"
					if t, ok := a.Value.Any().(time.Time); ok {
						a.Value = slog.StringValue(t.UTC().Format(time.RFC3339Nano))
					}
				case slog.SourceKey:
					// GCP recognized sourceLocation attribute
					a.Key = "logging.googleapis.com/sourceLocation"
				}
			}
			return a
		},
	}

	var handler slog.Handler
	if strings.EqualFold(cfg.Format, "text") {
		handler = slog.NewTextHandler(out, &slog.HandlerOptions{
			AddSource: cfg.AddSource,
			Level:     cfg.Level,
		})
	} else {
		handler = slog.NewJSONHandler(out, opts)
	}

	return slog.New(handler)
}

// HTTPRequestPayload formats an HTTP request for GCP Cloud Logging's httpRequest field:
// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#httprequest
type HTTPRequestPayload struct {
	RequestMethod string `json:"requestMethod"`
	RequestURL    string `json:"requestUrl"`
	RequestSize   int64  `json:"requestSize,omitempty"`
	Status        int    `json:"status"`
	ResponseSize  int64  `json:"responseSize,omitempty"`
	UserAgent     string `json:"userAgent,omitempty"`
	RemoteIP      string `json:"remoteIp,omitempty"`
	Referer       string `json:"referer,omitempty"`
	Latency       string `json:"latency,omitempty"` // Formatted as "<seconds>s", e.g. "0.003421s"
	Protocol      string `json:"protocol,omitempty"`
}

// HTTPRequestAttr returns a slog.Attr containing the GCP-formatted httpRequest.
func HTTPRequestAttr(req HTTPRequestPayload) slog.Attr {
	return slog.Any("httpRequest", req)
}

// TraceContextFromSpan extracts Google Cloud Trace correlation attributes directly from the active
// OpenTelemetry span in ctx if valid; otherwise, falls back to parsing the provided HTTP header.
func TraceContextFromSpan(ctx context.Context, projectID string, header http.Header) []slog.Attr {
	if projectID == "" {
		return nil
	}
	if ctx != nil {
		span := trace.SpanFromContext(ctx)
		if span != nil && span.SpanContext().IsValid() {
			sc := span.SpanContext()
			traceURI := "projects/" + projectID + "/traces/" + sc.TraceID().String()
			attrs := []slog.Attr{
				slog.String("logging.googleapis.com/trace", traceURI),
				slog.String("logging.googleapis.com/spanId", sc.SpanID().String()),
			}
			if sc.IsSampled() {
				attrs = append(attrs, slog.Bool("logging.googleapis.com/trace_sampled", true))
			}
			return attrs
		}
	}
	return TraceContext(projectID, header)
}

// TraceContext extracts Cloud Trace correlation attributes from HTTP headers.
// Supports both GCP's X-Cloud-Trace-Context header and W3C traceparent header.
func TraceContext(projectID string, header http.Header) []slog.Attr {
	if projectID == "" || header == nil {
		return nil
	}

	// 1. Google Cloud Run & Load Balancer: X-Cloud-Trace-Context: TRACE_ID/SPAN_ID;o=TRACE_TRUE
	if xctc := header.Get("X-Cloud-Trace-Context"); xctc != "" {
		return parseCloudTraceContext(projectID, xctc)
	}

	// 2. W3C traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
	if tp := header.Get("traceparent"); tp != "" {
		return parseTraceparent(projectID, tp)
	}

	return nil
}

func parseCloudTraceContext(projectID, header string) []slog.Attr {
	parts := strings.SplitN(header, ";", 2)
	traceSpan := strings.SplitN(parts[0], "/", 2)
	traceID := traceSpan[0]

	var attrs []slog.Attr
	if traceID != "" {
		traceURI := "projects/" + projectID + "/traces/" + traceID
		attrs = append(attrs, slog.String("logging.googleapis.com/trace", traceURI))
	}

	if len(traceSpan) > 1 && traceSpan[1] != "" {
		attrs = append(attrs, slog.String("logging.googleapis.com/spanId", traceSpan[1]))
	}

	if len(parts) > 1 && strings.Contains(parts[1], "o=1") {
		attrs = append(attrs, slog.Bool("logging.googleapis.com/trace_sampled", true))
	}

	return attrs
}

func parseTraceparent(projectID, header string) []slog.Attr {
	parts := strings.Split(header, "-")
	if len(parts) < 4 {
		return nil
	}

	traceID := parts[1]
	spanID := parts[2]
	flags := parts[3]

	var attrs []slog.Attr
	if traceID != "" {
		traceURI := "projects/" + projectID + "/traces/" + traceID
		attrs = append(attrs, slog.String("logging.googleapis.com/trace", traceURI))
	}
	if spanID != "" {
		attrs = append(attrs, slog.String("logging.googleapis.com/spanId", spanID))
	}
	if flags == "01" {
		attrs = append(attrs, slog.Bool("logging.googleapis.com/trace_sampled", true))
	}

	return attrs
}

type ctxKey struct{}

// WithLogger returns a new Context with the provided logger stored.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext extracts the logger from Context, or returns slog.Default().
func FromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return slog.Default()
}
