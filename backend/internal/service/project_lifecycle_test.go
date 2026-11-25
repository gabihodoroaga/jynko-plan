package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/testutil"
)

var lifecycleAdmin = model.UserContext{UserID: "u-admin", Email: "admin@jynko.dev"}

func newLifecycleService(t *testing.T, now time.Time) (*ProjectService, *testutil.ProjectRepo) {
	t.Helper()
	repo := testutil.NewProjectRepo()
	admin := model.ProjectMember{UserID: "u-admin", Email: "admin@jynko.dev", Role: model.RoleAdmin}
	repo.AddProject(model.Project{ID: "p-keep", Name: "Keep"}, admin)
	repo.AddProject(model.Project{ID: "p-del", Name: "Delete me"}, admin)
	svc := NewProjectService(repo)
	svc.now = func() time.Time { return now }
	return svc, repo
}

func projectIDs(list []model.Project) []string {
	ids := make([]string, 0, len(list))
	for _, p := range list {
		ids = append(ids, p.ID)
	}
	return ids
}

// PLAN-13: deleting marks the project as deleted (its data is kept), hides it from the project list and
// shows it in the deleted list; restoring brings it back.
func TestProjectService_SoftDeleteAndRestore(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	svc, repo := newLifecycleService(t, now)

	if err := svc.DeleteProject(ctx, "p-del", lifecycleAdmin); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	stored, err := repo.GetByID(ctx, "p-del")
	if err != nil {
		t.Fatalf("soft-deleted project must still exist in storage: %v", err)
	}
	if !stored.IsDeleted() || !stored.DeletedAt.Equal(now) || stored.DeletedBy != "admin@jynko.dev" {
		t.Fatalf("expected deleted_at=%v deleted_by=admin@jynko.dev, got %+v / %q", now, stored.DeletedAt, stored.DeletedBy)
	}
	if len(repo.HardDeleted) != 0 {
		t.Fatalf("DeleteProject must not hard-delete data, hard deleted: %v", repo.HardDeleted)
	}

	active, _ := svc.ListProjectsForUser(ctx, lifecycleAdmin)
	if ids := projectIDs(active); len(ids) != 1 || ids[0] != "p-keep" {
		t.Fatalf("active projects = %v, want [p-keep]", ids)
	}
	deleted, _ := svc.ListDeletedProjectsForUser(ctx, lifecycleAdmin)
	if ids := projectIDs(deleted); len(ids) != 1 || ids[0] != "p-del" {
		t.Fatalf("deleted projects = %v, want [p-del]", ids)
	}

	if err := svc.DeleteProject(ctx, "p-del", lifecycleAdmin); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("deleting an already-deleted project: expected ErrNotFound, got %v", err)
	}
	if err := svc.DeleteProject(ctx, "missing", lifecycleAdmin); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("deleting an unknown project: expected ErrNotFound, got %v", err)
	}

	restored, err := svc.RestoreProject(ctx, "p-del")
	if err != nil {
		t.Fatalf("RestoreProject: %v", err)
	}
	if restored.IsDeleted() || restored.DeletedBy != "" {
		t.Fatalf("restored project still marked deleted: %+v", restored)
	}
	active, _ = svc.ListProjectsForUser(ctx, lifecycleAdmin)
	if len(active) != 2 {
		t.Fatalf("after restore expected 2 active projects, got %v", projectIDs(active))
	}

	if _, err := svc.RestoreProject(ctx, "p-keep"); !errors.Is(err, ErrProjectNotDeleted) {
		t.Fatalf("restoring an active project: expected ErrProjectNotDeleted, got %v", err)
	}
	if _, err := svc.RestoreProject(ctx, "missing"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("restoring an unknown project: expected ErrNotFound, got %v", err)
	}
}

// PLAN-13: the cleanup only purges projects deleted at least olderThan ago; dry runs remove nothing;
// failures are reported without aborting the run.
func TestProjectService_PurgeDeletedProjects(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	svc, repo := newLifecycleService(t, now)
	admin := model.ProjectMember{UserID: "u-admin", Email: "admin@jynko.dev", Role: model.RoleAdmin}
	repo.AddProject(model.Project{ID: "p-old"}, admin)
	repo.AddProject(model.Project{ID: "p-old-broken"}, admin)
	old := now.Add(-40 * 24 * time.Hour)
	recent := now.Add(-2 * 24 * time.Hour)
	_ = repo.SetDeleted(ctx, "p-old", &old, "admin@jynko.dev")
	_ = repo.SetDeleted(ctx, "p-old-broken", &old, "admin@jynko.dev")
	_ = repo.SetDeleted(ctx, "p-del", &recent, "admin@jynko.dev")
	repo.DeleteErr["p-old-broken"] = testutil.ErrInjected

	retention := 30 * 24 * time.Hour

	dry, err := svc.PurgeDeletedProjects(ctx, retention, true)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if !dry.DryRun || dry.Candidates != 2 || len(dry.PurgedIDs) != 2 || len(repo.HardDeleted) != 0 {
		t.Fatalf("dry run should list 2 candidates and delete nothing, got %+v (hard deleted %v)", dry, repo.HardDeleted)
	}
	if !dry.Cutoff.Equal(now.Add(-retention)) {
		t.Fatalf("cutoff = %v, want %v", dry.Cutoff, now.Add(-retention))
	}

	res, err := svc.PurgeDeletedProjects(ctx, retention, false)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if len(res.PurgedIDs) != 1 || res.PurgedIDs[0] != "p-old" {
		t.Fatalf("purged = %v, want [p-old]", res.PurgedIDs)
	}
	if len(res.FailedIDs) != 1 || res.FailedIDs[0] != "p-old-broken" {
		t.Fatalf("failed = %v, want [p-old-broken]", res.FailedIDs)
	}
	for _, id := range []string{"p-keep", "p-del"} {
		if _, err := repo.GetByID(ctx, id); err != nil {
			t.Fatalf("%s must survive the purge (active or deleted too recently): %v", id, err)
		}
	}

	all, err := svc.PurgeDeletedProjects(ctx, 0, false)
	if err != nil {
		t.Fatalf("purge with zero retention: %v", err)
	}
	if len(all.PurgedIDs) != 1 || all.PurgedIDs[0] != "p-del" {
		t.Fatalf("zero retention purged = %v, want [p-del]", all.PurgedIDs)
	}
	if _, err := repo.GetByID(ctx, "p-keep"); err != nil {
		t.Fatalf("active project must never be purged: %v", err)
	}

	if _, err := svc.PurgeDeletedProjects(ctx, -time.Hour, false); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("negative retention: expected ErrInvalidInput, got %v", err)
	}
}
