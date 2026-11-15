package mcp

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/api/idtoken"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

type testProjectRepo struct {
	members map[string]*model.ProjectMember
}

func (m *testProjectRepo) Create(ctx context.Context, project *model.Project, initialAdmin *model.ProjectMember) error {
	return nil
}
func (m *testProjectRepo) GetByID(ctx context.Context, projectID string) (*model.Project, error) {
	return &model.Project{ID: projectID, Name: "Core", KeyPrefix: "COR"}, nil
}
func (m *testProjectRepo) ListByUser(ctx context.Context, userID, userEmail string) ([]model.Project, error) {
	return []model.Project{{ID: "proj-1", Name: "P1"}}, nil
}
func (m *testProjectRepo) Update(ctx context.Context, project *model.Project) error { return nil }
func (m *testProjectRepo) Delete(ctx context.Context, projectID string) error       { return nil }
func (m *testProjectRepo) GetMember(ctx context.Context, projectID, userID string) (*model.ProjectMember, error) {
	if mem, ok := m.members[projectID+":"+userID]; ok {
		return mem, nil
	}
	return nil, repository.ErrNotFound
}
func (m *testProjectRepo) GetMemberByEmail(ctx context.Context, projectID, email string) (*model.ProjectMember, error) {
	if mem, ok := m.members[projectID+":"+email]; ok {
		return mem, nil
	}
	return nil, repository.ErrNotFound
}
func (m *testProjectRepo) ListMembers(ctx context.Context, projectID string) ([]model.ProjectMember, error) {
	return []model.ProjectMember{}, nil
}
func (m *testProjectRepo) UpsertMember(ctx context.Context, projectID string, member *model.ProjectMember) error {
	return nil
}
func (m *testProjectRepo) RemoveMember(ctx context.Context, projectID, userID string) error {
	return nil
}

func TestResolver_RoleChecking(t *testing.T) {
	projRepo := &testProjectRepo{
		members: map[string]*model.ProjectMember{
			"proj-1:user-admin": {
				UserID: "user-admin",
				Email:  "admin@example.com",
				Role:   model.RoleAdmin,
			},
			"proj-1:user-member": {
				UserID: "user-member",
				Email:  "member@example.com",
				Role:   model.RoleMember,
			},
			"proj-1:user-viewer": {
				UserID: "user-viewer",
				Email:  "viewer@example.com",
				Role:   model.RoleViewer,
			},
		},
	}

	ctx := context.Background()

	// 1. Admin user has Viewer, Member, Admin permissions
	adminResolver := NewResolver(Config{
		DefaultProjectID: "proj-1",
		User: model.UserContext{
			UserID: "user-admin",
			Email:  "admin@example.com",
		},
	}, projRepo)

	if err := adminResolver.CheckRole(ctx, "proj-1", model.RoleViewer); err != nil {
		t.Fatalf("expected admin to have Viewer role: %v", err)
	}
	if err := adminResolver.CheckRole(ctx, "proj-1", model.RoleMember); err != nil {
		t.Fatalf("expected admin to have Member role: %v", err)
	}
	if err := adminResolver.CheckRole(ctx, "proj-1", model.RoleAdmin); err != nil {
		t.Fatalf("expected admin to have Admin role: %v", err)
	}

	// 2. Member user has Viewer, Member permissions, but not Admin
	memberResolver := NewResolver(Config{
		DefaultProjectID: "proj-1",
		User: model.UserContext{
			UserID: "user-member",
			Email:  "member@example.com",
		},
	}, projRepo)

	if err := memberResolver.CheckRole(ctx, "proj-1", model.RoleViewer); err != nil {
		t.Fatalf("expected member to have Viewer role: %v", err)
	}
	if err := memberResolver.CheckRole(ctx, "proj-1", model.RoleMember); err != nil {
		t.Fatalf("expected member to have Member role: %v", err)
	}
	if err := memberResolver.CheckRole(ctx, "proj-1", model.RoleAdmin); err == nil {
		t.Fatalf("expected member to NOT have Admin role")
	}

	// 3. Viewer user has Viewer permissions, but not Member
	viewerResolver := NewResolver(Config{
		DefaultProjectID: "proj-1",
		User: model.UserContext{
			UserID: "user-viewer",
			Email:  "viewer@example.com",
		},
	}, projRepo)

	if err := viewerResolver.CheckRole(ctx, "proj-1", model.RoleViewer); err != nil {
		t.Fatalf("expected viewer to have Viewer role: %v", err)
	}
	if err := viewerResolver.CheckRole(ctx, "proj-1", model.RoleMember); err == nil {
		t.Fatalf("expected viewer to NOT have Member role")
	}

	// 4. Non-member user is rejected
	strangerResolver := NewResolver(Config{
		DefaultProjectID: "proj-1",
		User: model.UserContext{
			UserID: "user-stranger",
			Email:  "stranger@example.com",
		},
	}, projRepo)

	if err := strangerResolver.CheckRole(ctx, "proj-1", model.RoleViewer); err == nil {
		t.Fatalf("expected stranger to be denied access to project")
	}
}

func TestResolver_ResolveProjectID(t *testing.T) {
	r := NewResolver(Config{
		DefaultProjectID: "proj-fallback",
		User: model.UserContext{
			UserID: "user-1",
		},
	}, &testProjectRepo{members: make(map[string]*model.ProjectMember)})

	// Explicit argument provided
	reqWithProj := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"project_id": "proj-explicit",
			},
		},
	}
	proj, err := r.ResolveProjectID(reqWithProj)
	if err != nil || proj != "proj-explicit" {
		t.Fatalf("expected proj-explicit, got %s (err: %v)", proj, err)
	}

	// Fallback used when omitted
	reqEmpty := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{},
		},
	}
	fallbackProj, err := r.ResolveProjectID(reqEmpty)
	if err != nil || fallbackProj != "proj-fallback" {
		t.Fatalf("expected proj-fallback, got %s (err: %v)", fallbackProj, err)
	}
}

func TestServer_Initialization(t *testing.T) {
	projRepo := &testProjectRepo{members: make(map[string]*model.ProjectMember)}

	serverDeps := ServerDeps{
		Config: Config{
			DefaultProjectID: "proj-core",
			User: model.UserContext{
				UserID: "test-agent",
				Email:  "agent@jynko.local",
				Name:   "Test Agent",
			},
		},
		ProjectRepo: projRepo,
		ProjectSvc:  service.NewProjectService(projRepo),
	}

	s := NewServer(serverDeps)
	if s == nil {
		t.Fatalf("expected non-nil server")
	}
	if s.MCPServer() == nil {
		t.Fatalf("expected non-nil mcpServer")
	}
}

func TestHTTPHandler_AuthenticationAndPerCallerRBAC(t *testing.T) {
	projRepo := &testProjectRepo{
		members: map[string]*model.ProjectMember{
			"proj-1:uid-viewer": {
				UserID: "uid-viewer",
				Email:  "viewer@company.com",
				Role:   model.RoleViewer,
			},
		},
	}

	appCfg := &config.Config{
		Env:                "prd",
		RawEnv:             "prd",
		IAPAudience:        "/projects/123/apps/jynko-mcp",
		IAPRequireJWT:      true,
		CORSAllowedOrigins: []string{"https://jynko.company.com"},
	}

	mockValidator := func(ctx context.Context, token, audience string) (*idtoken.Payload, error) {
		if token == "viewer-jwt" && audience == "/projects/123/apps/jynko-mcp" {
			return &idtoken.Payload{
				Subject: "accounts.google.com:uid-viewer",
				Claims:  map[string]interface{}{"email": "viewer@company.com"},
			}, nil
		}
		return nil, errors.New("invalid token")
	}

	s := NewServer(ServerDeps{
		Config:       Config{DefaultProjectID: "proj-1"}, // No static User in prd SSE!
		AppConfig:    appCfg,
		JWTValidator: mockValidator,
		ProjectRepo:  projRepo,
		ProjectSvc:   service.NewProjectService(projRepo),
	})
	handler := s.HTTPHandler()

	// 1. Unauthenticated POST to /mcp -> 401 Unauthorized
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated SSE/HTTP MCP request, got %d", w.Code)
	}

	// 2. Disallowed cross-origin Origin header -> 403 Forbidden
	req = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	req.Header.Set("Origin", "https://evil.attacker.com")
	req.Header.Set("X-Goog-IAP-JWT-Assertion", "viewer-jwt")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed Origin header, got %d", w.Code)
	}

	// 3. Per-request context overrides static User in CheckRole
	reqCtx := WithUserContext(context.Background(), model.UserContext{
		UserID: "uid-viewer",
		Email:  "viewer@company.com",
	})
	if err := s.resolver.CheckRole(reqCtx, "proj-1", model.RoleViewer); err != nil {
		t.Fatalf("expected viewer context to pass RoleViewer: %v", err)
	}
	err := s.resolver.CheckRole(reqCtx, "proj-1", model.RoleMember)
	if err == nil || !strings.Contains(err.Error(), "insufficient permissions") {
		t.Fatalf("expected viewer context to fail RoleMember, got %v", err)
	}
}

// Suppress unused imports
var _ = time.Now
