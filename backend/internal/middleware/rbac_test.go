package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

type mockProjectRepo struct {
	members map[string]*model.ProjectMember
}

func (m *mockProjectRepo) Create(ctx context.Context, project *model.Project, initialAdmin *model.ProjectMember) error {
	return nil
}
func (m *mockProjectRepo) GetByID(ctx context.Context, projectID string) (*model.Project, error) {
	return &model.Project{ID: projectID, Name: "Test", KeyPrefix: "TST"}, nil
}
func (m *mockProjectRepo) ListByUser(ctx context.Context, userID, userEmail string) ([]model.Project, error) {
	return nil, nil
}
func (m *mockProjectRepo) Update(ctx context.Context, project *model.Project) error { return nil }
func (m *mockProjectRepo) Delete(ctx context.Context, projectID string) error       { return nil }
func (m *mockProjectRepo) GetMember(ctx context.Context, projectID, userID string) (*model.ProjectMember, error) {
	if mem, ok := m.members[projectID+":"+userID]; ok {
		return mem, nil
	}
	return nil, repository.ErrNotFound
}
func (m *mockProjectRepo) GetMemberByEmail(ctx context.Context, projectID, email string) (*model.ProjectMember, error) {
	if mem, ok := m.members[projectID+":"+email]; ok {
		return mem, nil
	}
	return nil, repository.ErrNotFound
}
func (m *mockProjectRepo) ListMembers(ctx context.Context, projectID string) ([]model.ProjectMember, error) {
	return nil, nil
}
func (m *mockProjectRepo) UpsertMember(ctx context.Context, projectID string, member *model.ProjectMember) error {
	return nil
}
func (m *mockProjectRepo) RemoveMember(ctx context.Context, projectID, userID string) error {
	return nil
}

func TestIAPAuthAndRBACMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockProjectRepo{
		members: map[string]*model.ProjectMember{
			"proj-1:user-viewer": {UserID: "user-viewer", Email: "viewer@jynko.dev", Role: model.RoleViewer},
			"proj-1:user-member": {UserID: "user-member", Email: "member@jynko.dev", Role: model.RoleMember},
			"proj-1:user-admin":  {UserID: "user-admin", Email: "admin@jynko.dev", Role: model.RoleAdmin},
		},
	}

	cfg := &config.Config{Env: "production"}
	r := gin.New()
	r.Use(IAPAuthMiddleware(cfg))

	r.GET("/projects/:projectId/items", RequireProjectRole(repo, model.RoleViewer), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.POST("/projects/:projectId/items", RequireProjectRole(repo, model.RoleMember), func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})
	r.POST("/projects/:projectId/members", RequireProjectRole(repo, model.RoleAdmin), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 1. Missing IAP headers in production -> 401
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/projects/proj-1/items", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without IAP headers, got %d", w.Code)
	}

	// 2. Viewer can GET items -> 200
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/projects/proj-1/items", nil)
	req.Header.Set("X-Goog-Authenticated-User-Id", "accounts.google.com:user-viewer")
	req.Header.Set("X-Goog-Authenticated-User-Email", "accounts.google.com:viewer@jynko.dev")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for Viewer GET, got %d", w.Code)
	}

	// 3. Viewer cannot POST items -> 403
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/projects/proj-1/items", nil)
	req.Header.Set("X-Goog-Authenticated-User-Id", "accounts.google.com:user-viewer")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for Viewer POST item, got %d", w.Code)
	}

	// 4. Member can POST items -> 201, but cannot POST members -> 403
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/projects/proj-1/items", nil)
	req.Header.Set("X-Goog-Authenticated-User-Id", "accounts.google.com:user-member")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for Member POST item, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/projects/proj-1/members", nil)
	req.Header.Set("X-Goog-Authenticated-User-Id", "accounts.google.com:user-member")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for Member POST members, got %d", w.Code)
	}

	// 5. Admin can POST members -> 200
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/projects/proj-1/members", nil)
	req.Header.Set("X-Goog-Authenticated-User-Id", "accounts.google.com:user-admin")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for Admin POST members, got %d", w.Code)
	}
}
