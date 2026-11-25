package mcp

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/testutil"
)

// PLAN-13: MCP tools must not reach soft-deleted projects, even for their admins.
func TestResolverCheckRole_RejectsDeletedProject(t *testing.T) {
	repo := testutil.NewProjectRepo()
	admin := model.ProjectMember{UserID: "u-1", Email: "u1@jynko.dev", Role: model.RoleAdmin}
	repo.AddProject(model.Project{ID: "proj-1"}, admin)
	r := NewResolver(Config{}, repo)
	ctx := WithUserContext(context.Background(), model.UserContext{UserID: "u-1", Email: "u1@jynko.dev"})

	if err := r.CheckRole(ctx, "proj-1", model.RoleViewer); err != nil {
		t.Fatalf("active project: unexpected error %v", err)
	}
	at := time.Now().UTC()
	_ = repo.SetDeleted(ctx, "proj-1", &at, "u1@jynko.dev")
	err := r.CheckRole(ctx, "proj-1", model.RoleViewer)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("deleted project: expected 'not found' error, got %v", err)
	}
}
