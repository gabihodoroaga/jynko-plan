package service

import (
	"context"
	"errors"
	"testing"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

// keyedItemRepo extends mockItemRepo with key lookups so GetByIDOrKey can be exercised.
type keyedItemRepo struct {
	mockItemRepo
}

func (m *keyedItemRepo) GetByKey(_ context.Context, _, key string) (*model.WorkItem, error) {
	for _, it := range m.items {
		if it.Key == key {
			cp := *it
			return &cp, nil
		}
	}
	return nil, repository.ErrNotFound
}

func TestGetByIDOrKey_ResolvesIDKeyAndMissing(t *testing.T) {
	repo := &keyedItemRepo{mockItemRepo{
		items: map[string]*model.WorkItem{
			"item-1": {ID: "item-1", ProjectID: "proj-1", Key: "PLAN-12", Title: "Improve refresh"},
		},
		comments: map[string]*model.Comment{},
	}}
	svc := NewItemService(repo)
	ctx := context.Background()

	byID, err := svc.GetByIDOrKey(ctx, "proj-1", "item-1")
	if err != nil || byID.ID != "item-1" {
		t.Fatalf("lookup by ID: got item=%v err=%v", byID, err)
	}

	byKey, err := svc.GetByIDOrKey(ctx, "proj-1", "PLAN-12")
	if err != nil || byKey.ID != "item-1" {
		t.Fatalf("lookup by key: got item=%v err=%v", byKey, err)
	}

	if _, err := svc.GetByIDOrKey(ctx, "proj-1", "PLAN-999"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing item: expected ErrNotFound, got %v", err)
	}

	if _, err := svc.GetByIDOrKey(ctx, "proj-1", "   "); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("blank id: expected ErrInvalidInput, got %v", err)
	}
}
