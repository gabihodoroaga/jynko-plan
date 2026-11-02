package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware configures Cross-Origin Resource Sharing headers for the Angular SPA.
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowAll := false
	originMap := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
		}
		originMap[strings.TrimRight(o, "/")] = true
	}

	return func(c *gin.Context) {
		origin := strings.TrimRight(c.GetHeader("Origin"), "/")
		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" && originMap[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		} else if len(allowedOrigins) > 0 {
			c.Header("Access-Control-Allow-Origin", allowedOrigins[0])
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Goog-Authenticated-User-Id, X-Goog-Authenticated-User-Email, X-Goog-IAP-JWT-Assertion, X-Dev-User-Id, X-Dev-User-Email, X-Dev-User-Name")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
