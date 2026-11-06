package service

import (
	"context"
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

func TestStickyService_CRUDAndRandomColor(t *testing.T) {
	repo := &mockStickyRepo{items: map[string]*model.Sticky{}}
	svc := NewStickyService(repo)
	ctx := context.Background()
	user := model.UserContext{UserID: "dev-user-1"}

	created, err := svc.CreateSticky(ctx, "proj-1", user, CreateStickyInput{
		Content: "Brainstorming Gemini agent prompts",
	})
	if err != nil {
		t.Fatalf("unexpected error creating sticky: %v", err)
	}
	if created.Color == "" {
		t.Fatalf("expected random funny color to be assigned when color is omitted")
	}

	pinned := true
	newColor := model.StickyColorFlamingo
	updated, err := svc.UpdateSticky(ctx, "proj-1", created.ID, UpdateStickyInput{
		Color:    &newColor,
		IsPinned: &pinned,
	})
	if err != nil {
		t.Fatalf("unexpected error updating sticky: %v", err)
	}
	if updated.Color != model.StickyColorFlamingo || !updated.IsPinned {
		t.Fatalf("expected updated color FLAMINGO and pinned=true, got color=%s pinned=%v", updated.Color, updated.IsPinned)
	}
}
