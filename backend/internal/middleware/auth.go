package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
)

const userContextKey = "jynko_user_context"

// IAPAuthMiddleware validates Google Cloud Identity-Aware Proxy (IAP) / OIDC headers.
// In development mode (ENV=development), it falls back to X-Dev-User-* headers or configured dev defaults.
func IAPAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawIAPUserID := strings.TrimSpace(c.GetHeader("X-Goog-Authenticated-User-Id"))
		rawIAPEmail := strings.TrimSpace(c.GetHeader("X-Goog-Authenticated-User-Email"))

		userID := stripIAPPrefix(rawIAPUserID)
		email := strings.ToLower(stripIAPPrefix(rawIAPEmail))
		var name string

		if userID != "" || email != "" {
			if userID == "" {
				userID = email
			}
			if email == "" {
				email = userID
			}
			name = deriveDisplayName(email)
		} else if cfg != nil && cfg.IsDevelopment() {
			devID := strings.TrimSpace(c.GetHeader("X-Dev-User-Id"))
			devEmail := strings.ToLower(strings.TrimSpace(c.GetHeader("X-Dev-User-Email")))
			devName := strings.TrimSpace(c.GetHeader("X-Dev-User-Name"))

			if devID == "" {
				devID = cfg.DevUserID
			}
			if devEmail == "" {
				devEmail = strings.ToLower(cfg.DevUserEmail)
			}
			if devName == "" {
				devName = cfg.DevUserName
			}

			userID = devID
			email = devEmail
			name = devName
		}

		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing Google Cloud IAP authentication headers",
			})
			return
		}

		if name == "" {
			name = deriveDisplayName(email)
		}

		userCtx := model.UserContext{
			UserID: userID,
			Email:  email,
			Name:   name,
		}
		c.Set(userContextKey, userCtx)
		c.Next()
	}
}

// GetUserContext retrieves the authenticated UserContext from the Gin context.
func GetUserContext(c *gin.Context) (model.UserContext, bool) {
	val, exists := c.Get(userContextKey)
	if !exists {
		return model.UserContext{}, false
	}
	u, ok := val.(model.UserContext)
	return u, ok
}

// stripIAPPrefix removes provider prefixes like "accounts.google.com:" from IAP header values.
func stripIAPPrefix(val string) string {
	if idx := strings.Index(val, ":"); idx != -1 && idx+1 < len(val) {
		return strings.TrimSpace(val[idx+1:])
	}
	return val
}

func deriveDisplayName(email string) string {
	if email == "" {
		return "User"
	}
	local := email
	if idx := strings.Index(email, "@"); idx > 0 {
		local = email[:idx]
	}
	parts := strings.FieldsFunc(local, func(r rune) bool {
		return r == '.' || r == '_' || r == '-'
	})
	if len(parts) == 0 {
		return local
	}
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
