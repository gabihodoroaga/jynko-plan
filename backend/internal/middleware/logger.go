package middleware

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/logger"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
)

// StructuredLogger returns a Gin middleware that emits GCP Cloud Logging-compatible
// structured access logs for each HTTP request, including the httpRequest payload and Cloud Trace correlation.
func StructuredLogger(projectID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		status := c.Writer.Status()

		fullURL := path
		if rawQuery != "" {
			fullURL = path + "?" + rawQuery
		}

		// Extract trace attributes from active OpenTelemetry span (or headers as fallback)
		traceAttrs := logger.TraceContextFromSpan(c.Request.Context(), projectID, c.Request.Header)

		var reqLogger *slog.Logger
		if len(traceAttrs) > 0 {
			var anyAttrs []any
			for _, a := range traceAttrs {
				anyAttrs = append(anyAttrs, a)
			}
			reqLogger = slog.Default().With(anyAttrs...)
		} else {
			reqLogger = slog.Default()
		}

		// Format httpRequest object according to Google Cloud Logging specification
		httpReq := logger.HTTPRequestPayload{
			RequestMethod: c.Request.Method,
			RequestURL:    fullURL,
			RequestSize:   c.Request.ContentLength,
			Status:        status,
			ResponseSize:  int64(c.Writer.Size()),
			UserAgent:     c.Request.UserAgent(),
			RemoteIP:      c.ClientIP(),
			Referer:       c.Request.Referer(),
			Latency:       fmt.Sprintf("%.6fs", latency.Seconds()),
			Protocol:      c.Request.Proto,
		}

		attrs := []slog.Attr{
			logger.HTTPRequestAttr(httpReq),
		}
		attrs = append(attrs, traceAttrs...)

		// Add authenticated user context if present
		if userVal, exists := c.Get(userContextKey); exists {
			if user, ok := userVal.(model.UserContext); ok {
				attrs = append(attrs,
					slog.String("user_id", user.UserID),
					slog.String("user_email", user.Email),
				)
			}
		}

		// Add Gin handler errors if any occurred
		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("gin_errors", c.Errors.String()))
		}

		// Determine severity based on HTTP status
		var level slog.Level
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		default:
			level = slog.LevelInfo
		}

		msg := fmt.Sprintf("%s %s -> %d (%s)", c.Request.Method, path, status, latency.Round(time.Millisecond))
		reqLogger.LogAttrs(c.Request.Context(), level, msg, attrs...)
	}
}
