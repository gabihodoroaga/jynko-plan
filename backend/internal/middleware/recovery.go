package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/logger"
)

// StructuredRecovery returns a Gin recovery middleware that logs panics with CRITICAL
// severity, stack traces, and Cloud Trace correlation, returning an RFC-compliant JSON error.
func StructuredRecovery(projectID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				errStr := fmt.Sprintf("%v", r)

				traceAttrs := logger.TraceContextFromSpan(c.Request.Context(), projectID, c.Request.Header)

				attrs := []slog.Attr{
					slog.String("panic", errStr),
					slog.String("stack_trace", stack),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
				}
				attrs = append(attrs, traceAttrs...)

				reqLogger := logger.FromContext(c.Request.Context())
				reqLogger.LogAttrs(c.Request.Context(), logger.LevelCritical, "Panic recovered in HTTP handler: "+errStr, attrs...)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
			}
		}()

		c.Next()
	}
}
