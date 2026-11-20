package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/handler"
	"github.com/gabihodoroaga/jynko-plan/internal/mcp"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

type stubProjectRepo struct {
	members map[string]*model.ProjectMember
}

func (s *stubProjectRepo) Create(ctx context.Context, project *model.Project, initialAdmin *model.ProjectMember) error {
	return nil
}
func (s *stubProjectRepo) GetByID(ctx context.Context, projectID string) (*model.Project, error) {
	return &model.Project{ID: projectID, Name: "Core", KeyPrefix: "COR"}, nil
}
func (s *stubProjectRepo) ListByUser(ctx context.Context, userID, userEmail string) ([]model.Project, error) {
	return []model.Project{{ID: "proj-1", Name: "Core", KeyPrefix: "COR"}}, nil
}
func (s *stubProjectRepo) Update(ctx context.Context, project *model.Project) error { return nil }
func (s *stubProjectRepo) Delete(ctx context.Context, projectID string) error       { return nil }
func (s *stubProjectRepo) GetMember(ctx context.Context, projectID, userID string) (*model.ProjectMember, error) {
	if mem, ok := s.members[projectID+":"+userID]; ok {
		return mem, nil
	}
	return nil, repository.ErrNotFound
}
func (s *stubProjectRepo) GetMemberByEmail(ctx context.Context, projectID, email string) (*model.ProjectMember, error) {
	if mem, ok := s.members[projectID+":"+strings.ToLower(strings.TrimSpace(email))]; ok {
		return mem, nil
	}
	return nil, repository.ErrNotFound
}
func (s *stubProjectRepo) ListMembers(ctx context.Context, projectID string) ([]model.ProjectMember, error) {
	return []model.ProjectMember{}, nil
}
func (s *stubProjectRepo) UpsertMember(ctx context.Context, projectID string, member *model.ProjectMember) error {
	return nil
}
func (s *stubProjectRepo) RemoveMember(ctx context.Context, projectID, userID string) error {
	return nil
}

func TestRouter_MCPEndpointBehindCloudRunIAP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html><body>Jynko SPA</body></html>"), 0o644); err != nil {
		t.Fatalf("failed to write dummy index.html: %v", err)
	}

	saEmail := "jynko-scion-agent@aparaturafiscala-178410.iam.gserviceaccount.com"
	saSub := "109876543210987654321"
	projRepo := &stubProjectRepo{
		members: map[string]*model.ProjectMember{
			"proj-1:" + saEmail: {
				UserID: saEmail,
				Email:  saEmail,
				Role:   model.RoleMember,
			},
		},
	}

	cfg := &config.Config{
		Env:                config.EnvPrd,
		RawEnv:             config.EnvPrd,
		DisableDevAuth:     true,
		GoogleCloudProject: "aparaturafiscala-178410",
		IAPAudience:        "/projects/12345/locations/europe-west3/services/jynko-plan-prd",
		IAPRequireJWT:      true,
		StaticDir:          staticDir,
	}

	mockValidator := func(ctx context.Context, token, audience string) (*idtoken.Payload, error) {
		if token == "valid-sa-token" && audience == cfg.IAPAudience {
			return &idtoken.Payload{
				Subject: saSub,
				Claims: map[string]interface{}{
					"email": saEmail,
				},
			}, nil
		}
		return nil, errors.New("invalid token")
	}

	projectSvc := service.NewProjectService(projRepo)
	mcpSrv := mcp.NewServer(mcp.ServerDeps{
		Config:       mcp.Config{DefaultProjectID: "proj-1"},
		AppConfig:    cfg,
		JWTValidator: mockValidator,
		ProjectRepo:  projRepo,
		ProjectSvc:   projectSvc,
	})

	router := SetupRouterWithMCP(
		cfg,
		projRepo,
		handler.NewAuthHandler(),
		handler.NewProjectHandler(projectSvc),
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		mcpSrv.HTTPHandler(),
	)

	// 1. Unauthenticated POST /mcp returns 401 Unauthorized (never 200 index.html)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated POST /mcp, got %d (%s)", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "Jynko SPA") {
		t.Fatalf("unauthenticated /mcp must not return index.html")
	}

	// 2. Unmatched /mcp/subpath returns 404 JSON (never 200 index.html)
	reqSub := httptest.NewRequest(http.MethodGet, "/mcp/unknown", nil)
	wSub := httptest.NewRecorder()
	router.ServeHTTP(wSub, reqSub)
	if wSub.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for /mcp/unknown, got %d (%s)", wSub.Code, wSub.Body.String())
	}
	if !strings.Contains(wSub.Body.String(), "route not found") || strings.Contains(wSub.Body.String(), "Jynko SPA") {
		t.Fatalf("expected JSON 404 for /mcp/unknown, got: %s", wSub.Body.String())
	}

	doAuthMCP := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
		r.RemoteAddr = "127.0.0.1:12345"
		r.Host = "jynko-plan-prd-d2lb6vh75a-ey.a.run.app"
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Goog-IAP-JWT-Assertion", "valid-sa-token")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, r)
		return rec
	}

	// 3. Authenticated initialize behind Cloud Run local reverse proxy
	wInit := doAuthMCP(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"scion","version":"1.0"}}}`)
	if wInit.Code != http.StatusOK {
		t.Fatalf("expected 200 on authenticated initialize, got %d: %s", wInit.Code, wInit.Body.String())
	}

	// 4. Authenticated tools/list without session state
	wList := doAuthMCP(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if wList.Code != http.StatusOK || !strings.Contains(wList.Body.String(), "list_projects") {
		t.Fatalf("expected 200 with tools list on stateless tools/list, got %d: %s", wList.Code, wList.Body.String())
	}

	// 5. Stateless tools/call across independent requests without Mcp-Session-Id
	wCall := doAuthMCP(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_projects","arguments":{}}}`)
	if wCall.Code != http.StatusOK || !strings.Contains(wCall.Body.String(), "proj-1") {
		t.Fatalf("expected 200 with proj-1 on stateless tools/call, got %d: %s", wCall.Code, wCall.Body.String())
	}
}
