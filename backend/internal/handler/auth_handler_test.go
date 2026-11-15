package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
)

func TestAuthHandler_AvatarURLValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Env: "dev", DevUserID: "dev-1", DevUserEmail: "dev@jynko.local", DevUserName: "Dev"}
	h := NewAuthHandler()

	r := gin.New()
	r.Use(middleware.IAPAuthMiddleware(cfg))
	r.PATCH("/api/v1/auth/me", h.UpdateMe)

	// 1. Reject javascript: avatar_url XSS vector
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me", bytes.NewBufferString(`{"avatar_url":"javascript:alert(1)"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for javascript: avatar_url, got %d", w.Code)
	}

	// 2. Accept safe data:image/png;base64,...
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/auth/me", bytes.NewBufferString(`{"name":"Ada Lovelace","avatar_url":"data:image/png;base64,iVBORw0KGgo="}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid base64 PNG avatar, got %d (%s)", w.Code, w.Body.String())
	}
}
