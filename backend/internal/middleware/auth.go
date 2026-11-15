package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
)

const userContextKey = "jynko_user_context"

// JWTValidatorFunc validates a signed Google IAP / OIDC JWT token and returns its verified payload.
type JWTValidatorFunc func(ctx context.Context, token, audience string) (*idtoken.Payload, error)

// IAPAuthMiddleware validates Google Cloud Identity-Aware Proxy (IAP) / OIDC JWTs and headers.
// In dev mode (ENV=dev), it falls back to X-Dev-User-* headers or configured dev defaults.
func IAPAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return IAPAuthMiddlewareWithValidator(cfg, idtoken.Validate)
}

// IAPAuthMiddlewareWithValidator allows injecting a custom JWT validator in unit tests.
func IAPAuthMiddlewareWithValidator(cfg *config.Config, validator JWTValidatorFunc) gin.HandlerFunc {
	if validator == nil {
		validator = idtoken.Validate
	}
	return func(c *gin.Context) {
		userCtx, status, errMsg := AuthenticateHTTPRequest(c.Request.Context(), c.Request.Header, cfg, validator)
		if status != http.StatusOK {
			c.AbortWithStatusJSON(status, gin.H{"error": errMsg})
			return
		}
		c.Set(userContextKey, userCtx)
		c.Next()
	}
}

// AuthenticateHTTPRequest authenticates an HTTP request using IAP JWT assertion, OIDC Bearer token,
// GFE-stripped IAP headers (when JWT enforcement is not configured), or local dev headers.
func AuthenticateHTTPRequest(
	ctx context.Context,
	headers http.Header,
	cfg *config.Config,
	validator JWTValidatorFunc,
) (model.UserContext, int, string) {
	if validator == nil {
		validator = idtoken.Validate
	}

	audience := ""
	requireJWT := false
	allowDev := false
	if cfg != nil {
		audience = strings.TrimSpace(cfg.IAPAudience)
		requireJWT = cfg.IAPRequireJWT || audience != ""
		allowDev = cfg.AllowDevAuth()
	}

	// 1. Cryptographically verify X-Goog-IAP-JWT-Assertion (or Authorization: Bearer OIDC token) whenever present.
	jwtAssertion := strings.TrimSpace(headers.Get("X-Goog-IAP-JWT-Assertion"))
	if jwtAssertion == "" {
		if authHeader := strings.TrimSpace(headers.Get("Authorization")); strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			jwtAssertion = strings.TrimSpace(authHeader[7:])
		}
	}

	if jwtAssertion != "" {
		payload, err := validator(ctx, jwtAssertion, audience)
		if err != nil {
			return model.UserContext{}, http.StatusUnauthorized, "invalid Google Cloud IAP / OIDC JWT assertion"
		}

		sub := stripIAPPrefix(strings.TrimSpace(payload.Subject))
		var email string
		if rawEmail, ok := payload.Claims["email"].(string); ok {
			email = strings.ToLower(stripIAPPrefix(strings.TrimSpace(rawEmail)))
		}
		var name string
		if rawName, ok := payload.Claims["name"].(string); ok {
			name = strings.TrimSpace(rawName)
		}

		if sub == "" && email == "" {
			return model.UserContext{}, http.StatusUnauthorized, "verified JWT assertion missing subject and email claims"
		}
		if sub == "" {
			sub = email
		}
		if email == "" {
			email = sub
		}
		if name == "" {
			name = deriveDisplayName(email)
		}

		return model.UserContext{
			UserID: sub,
			Email:  email,
			Name:   name,
		}, http.StatusOK, ""
	}

	// 2. If running outside dev and JWT verification is required, reject requests missing a signed JWT.
	if !allowDev && requireJWT {
		return model.UserContext{}, http.StatusUnauthorized, "missing X-Goog-IAP-JWT-Assertion header"
	}

	// 3. In local dev mode, prioritize explicit X-Dev-User-* headers (preventing spoofed X-Goog-* overrides).
	if allowDev {
		devID := strings.TrimSpace(headers.Get("X-Dev-User-Id"))
		devEmail := strings.ToLower(strings.TrimSpace(headers.Get("X-Dev-User-Email")))
		devName := strings.TrimSpace(headers.Get("X-Dev-User-Name"))

		if devID != "" || devEmail != "" {
			if devID == "" {
				devID = devEmail
			}
			if devEmail == "" {
				devEmail = strings.ToLower(cfg.DevUserEmail)
			}
			if devName == "" {
				devName = deriveDisplayName(devEmail)
			}
			return model.UserContext{
				UserID: devID,
				Email:  devEmail,
				Name:   devName,
			}, http.StatusOK, ""
		}
	}

	// 4. Fallback to GFE-stripped IAP identity headers when JWT audience/requirement is not configured.
	rawIAPUserID := strings.TrimSpace(headers.Get("X-Goog-Authenticated-User-Id"))
	rawIAPEmail := strings.TrimSpace(headers.Get("X-Goog-Authenticated-User-Email"))
	userID := stripIAPPrefix(rawIAPUserID)
	email := strings.ToLower(stripIAPPrefix(rawIAPEmail))

	if userID != "" || email != "" {
		if userID == "" {
			userID = email
		}
		if email == "" {
			email = userID
		}
		return model.UserContext{
			UserID: userID,
			Email:  email,
			Name:   deriveDisplayName(email),
		}, http.StatusOK, ""
	}

	// 5. Default dev user when no headers are supplied in local development.
	if allowDev && cfg != nil {
		userID = cfg.DevUserID
		email = strings.ToLower(cfg.DevUserEmail)
		name := cfg.DevUserName
		if name == "" {
			name = deriveDisplayName(email)
		}
		if userID != "" {
			return model.UserContext{
				UserID: userID,
				Email:  email,
				Name:   name,
			}, http.StatusOK, ""
		}
	}

	return model.UserContext{}, http.StatusUnauthorized, "missing Google Cloud IAP authentication headers"
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
