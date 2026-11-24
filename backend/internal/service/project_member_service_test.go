package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

type mockMemberProjectRepo struct {
	members     map[string]*model.ProjectMember
	upsertCalls int
}

func (m *mockMemberProjectRepo) Create(_ context.Context, _ *model.Project, _ *model.ProjectMember) error {
	return nil
}
func (m *mockMemberProjectRepo) GetByID(_ context.Context, _ string) (*model.Project, error) {
	return nil, repository.ErrNotFound
}
func (m *mockMemberProjectRepo) ListByUser(_ context.Context, _, _ string) ([]model.Project, error) {
	return []model.Project{}, nil
}
func (m *mockMemberProjectRepo) Update(_ context.Context, _ *model.Project) error { return nil }
func (m *mockMemberProjectRepo) Delete(_ context.Context, _ string) error         { return nil }
func (m *mockMemberProjectRepo) GetMember(_ context.Context, _, userID string) (*model.ProjectMember, error) {
	mem, ok := m.members[userID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *mem
	return &cp, nil
}
func (m *mockMemberProjectRepo) GetMemberByEmail(_ context.Context, _, _ string) (*model.ProjectMember, error) {
	return nil, repository.ErrNotFound
}
func (m *mockMemberProjectRepo) ListMembers(_ context.Context, _ string) ([]model.ProjectMember, error) {
	return []model.ProjectMember{}, nil
}
func (m *mockMemberProjectRepo) UpsertMember(_ context.Context, _ string, member *model.ProjectMember) error {
	m.upsertCalls++
	cp := *member
	m.members[member.UserID] = &cp
	return nil
}
func (m *mockMemberProjectRepo) RemoveMember(_ context.Context, _, _ string) error { return nil }

func strPtr(s string) *string { return &s }

func TestProjectService_UpdateMemberName(t *testing.T) {
	ctx := context.Background()
	joined := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	repo := &mockMemberProjectRepo{members: map[string]*model.ProjectMember{
		"u1": {UserID: "u1", Email: "alice@example.com", Name: "Alice", Role: model.RoleMember, JoinedAt: joined},
	}}
	svc := NewProjectService(repo)

	updated, err := svc.UpdateMember(ctx, "p1", "u1", UpdateMemberInput{Name: strPtr("  Alice Smith  ")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Alice Smith" {
		t.Fatalf("expected trimmed name 'Alice Smith', got %q", updated.Name)
	}
	stored := repo.members["u1"]
	if stored.Email != "alice@example.com" || stored.Role != model.RoleMember || !stored.JoinedAt.Equal(joined) {
		t.Fatalf("identity/role/joined_at must be preserved, got %+v", stored)
	}

	// Unchanged name does not trigger a write.
	calls := repo.upsertCalls
	if _, err := svc.UpdateMember(ctx, "p1", "u1", UpdateMemberInput{Name: strPtr("Alice Smith")}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.upsertCalls != calls {
		t.Fatalf("expected no upsert for unchanged name")
	}

	invalid := []UpdateMemberInput{{Name: nil}, {Name: strPtr("   ")}, {Name: strPtr(string(make([]rune, 101)))}}
	for i, in := range invalid {
		if _, err := svc.UpdateMember(ctx, "p1", "u1", in); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("case %d: expected ErrInvalidInput, got %v", i, err)
		}
	}
	if _, err := svc.UpdateMember(ctx, "p1", "a/b", UpdateMemberInput{Name: strPtr("X")}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for slash user id, got %v", err)
	}
	if _, err := svc.UpdateMember(ctx, "p1", "missing", UpdateMemberInput{Name: strPtr("X")}); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown member, got %v", err)
	}
}
