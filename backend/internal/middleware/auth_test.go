package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
)

func TestIAPAuthMiddleware_JWTVerificationAndSpoofingProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockValidator := func(ctx context.Context, token, audience string) (*idtoken.Payload, error) {
		if token == "valid-jwt" && audience == "/projects/123/apps/jynko" {
			return &idtoken.Payload{
				Subject: "accounts.google.com:google-uid-999",
				Claims: map[string]interface{}{
					"email": "verified.user@company.com",
					"name":  "Verified User",
				},
			}, nil
		}
		return nil, errors.New("invalid signature or audience")
	}

	cfg := &config.Config{
		Env:           "prd",
		RawEnv:        "prd",
		IAPAudience:   "/projects/123/apps/jynko",
		IAPRequireJWT: true,
	}

	r := gin.New()
	r.Use(IAPAuthMiddlewareWithValidator(cfg, mockValidator))
	r.GET("/me", func(c *gin.Context) {
		u, ok := GetUserContext(c)
		if !ok {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, u)
	})

	// 1. Spoofed plain-text IAP headers WITHOUT signed JWT must be rejected when IAPAudience is set
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Goog-Authenticated-User-Id", "accounts.google.com:admin-1")
	req.Header.Set("X-Goog-Authenticated-User-Email", "accounts.google.com:admin@company.com")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when JWT assertion missing with IAPAudience configured, got %d", w.Code)
	}

	// 2. Forged JWT assertion MUST be rejected even if plain-text headers are present
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Goog-IAP-JWT-Assertion", "forged-jwt")
	req.Header.Set("X-Goog-Authenticated-User-Email", "accounts.google.com:admin@company.com")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for forged JWT assertion, got %d", w.Code)
	}

	// 3. Valid JWT assertion succeeds and extracts identity from verified claims (ignoring spoofed plain-text header)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Goog-IAP-JWT-Assertion", "valid-jwt")
	req.Header.Set("X-Goog-Authenticated-User-Email", "accounts.google.com:attacker@company.com")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid JWT assertion, got %d", w.Code)
	}
	body := w.Body.String()
	if body == "" || !containsAll(body, "google-uid-999", "verified.user@company.com", "Verified User") {
		t.Fatalf("expected verified JWT claims in response, got %s", body)
	}

	// 4. Misconfigured ENV="production" fails closed (does NOT allow dev fallback)
	badCfg := &config.Config{
		Env:          "dev",
		RawEnv:       "production",
		DevUserID:    "dev-user-1",
		DevUserEmail: "dev@jynko.local",
	}
	rBad := gin.New()
	rBad.Use(IAPAuthMiddlewareWithValidator(badCfg, mockValidator))
	rBad.GET("/me", func(c *gin.Context) { c.Status(http.StatusOK) })

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("X-Dev-User-Id", "admin-impersonation")
	rBad.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when RawEnv='production', got %d", w.Code)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		found := false
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
