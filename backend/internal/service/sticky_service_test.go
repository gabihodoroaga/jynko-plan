package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

type mockStickyRepo struct {
	items map[string]*model.Sticky
}

func (m *mockStickyRepo) List(_ context.Context, projectID string) ([]model.Sticky, error) {
	out := []model.Sticky{}
	for _, s := range m.items {
		if s.ProjectID == projectID {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (m *mockStickyRepo) GetByID(_ context.Context, projectID, stickyID string) (*model.Sticky, error) {
	s, ok := m.items[stickyID]
	if !ok || s.ProjectID != projectID {
		return nil, repository.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (m *mockStickyRepo) Create(_ context.Context, sticky *model.Sticky) error {
	if sticky.ID == "" {
		sticky.ID = "sticky-1"
	}
	cp := *sticky
	m.items[sticky.ID] = &cp
	return nil
}

func (m *mockStickyRepo) Update(_ context.Context, sticky *model.Sticky) error {
	cp := *sticky
	m.items[sticky.ID] = &cp
	return nil
}

func (m *mockStickyRepo) Delete(_ context.Context, _, stickyID string) error {
	delete(m.items, stickyID)
	return nil
}

func TestStickyService_CRUDAndOwnership(t *testing.T) {
	repo := &mockStickyRepo{items: map[string]*model.Sticky{}}
	svc := NewStickyService(repo)
	ctx := context.Background()
	author := model.UserContext{UserID: "dev-user-1"}
	otherMember := model.UserContext{UserID: "dev-user-2"}
	admin := model.UserContext{UserID: "dev-admin"}

	created, err := svc.CreateSticky(ctx, "proj-1", author, CreateStickyInput{
		Content: "Brainstorming Gemini agent prompts",
	})
	if err != nil {
		t.Fatalf("unexpected error creating sticky: %v", err)
	}
	if created.Color == "" {
		t.Fatalf("expected random color to be assigned when color is omitted")
	}

	pinned := true
	newColor := model.StickyColorFlamingo
	updated, err := svc.UpdateSticky(ctx, "proj-1", created.ID, author, model.RoleMember, UpdateStickyInput{
		Color:    &newColor,
		IsPinned: &pinned,
	})
	if err != nil {
		t.Fatalf("unexpected error updating own sticky: %v", err)
	}
	if updated.Color != model.StickyColorFlamingo || !updated.IsPinned {
		t.Fatalf("expected updated color FLAMINGO and pinned=true, got color=%s pinned=%v", updated.Color, updated.IsPinned)
	}

	// Non-author MEMBER cannot edit or delete someone else's sticky
	if _, err := svc.UpdateSticky(ctx, "proj-1", created.ID, otherMember, model.RoleMember, UpdateStickyInput{IsPinned: &pinned}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden when non-author MEMBER edits sticky, got %v", err)
	}
	if err := svc.DeleteSticky(ctx, "proj-1", created.ID, otherMember, model.RoleMember); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden when non-author MEMBER deletes sticky, got %v", err)
	}

	// ADMIN can delete any sticky
	if err := svc.DeleteSticky(ctx, "proj-1", created.ID, admin, model.RoleAdmin); err != nil {
		t.Fatalf("expected ADMIN to delete sticky, got %v", err)
	}
}
