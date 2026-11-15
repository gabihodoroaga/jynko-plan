package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

type mockItemRepo struct {
	items    map[string]*model.WorkItem
	comments map[string]*model.Comment
}

func (m *mockItemRepo) List(_ context.Context, _ string, _ model.WorkItemFilter) ([]model.WorkItem, error) {
	return []model.WorkItem{}, nil
}
func (m *mockItemRepo) GetByID(_ context.Context, _, itemID string) (*model.WorkItem, error) {
	it, ok := m.items[itemID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *it
	return &cp, nil
}
func (m *mockItemRepo) GetByKey(_ context.Context, _, _ string) (*model.WorkItem, error) {
	return nil, repository.ErrNotFound
}
func (m *mockItemRepo) Create(_ context.Context, item *model.WorkItem) error {
	if item.ID == "" {
		item.ID = "item-1"
	}
	cp := *item
	m.items[item.ID] = &cp
	return nil
}
func (m *mockItemRepo) Update(_ context.Context, item *model.WorkItem) error {
	cp := *item
	m.items[item.ID] = &cp
	return nil
}
func (m *mockItemRepo) Delete(_ context.Context, _, itemID string) error {
	delete(m.items, itemID)
	return nil
}
func (m *mockItemRepo) ListSubItems(_ context.Context, _, _ string) ([]model.SubWorkItem, error) {
	return []model.SubWorkItem{}, nil
}
func (m *mockItemRepo) GetSubItemByID(_ context.Context, _, _, _ string) (*model.SubWorkItem, error) {
	return nil, repository.ErrNotFound
}
func (m *mockItemRepo) CreateSubItem(_ context.Context, _ string, _ *model.SubWorkItem) error {
	return nil
}
func (m *mockItemRepo) UpdateSubItem(_ context.Context, _ string, _ *model.SubWorkItem) error {
	return nil
}
func (m *mockItemRepo) DeleteSubItem(_ context.Context, _, _, _ string) error {
	return nil
}
func (m *mockItemRepo) ListComments(_ context.Context, _, _ string) ([]model.Comment, error) {
	return []model.Comment{}, nil
}
func (m *mockItemRepo) GetCommentByID(_ context.Context, _, _, commentID string) (*model.Comment, error) {
	c, ok := m.comments[commentID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *c
	return &cp, nil
}
func (m *mockItemRepo) CreateComment(_ context.Context, _ string, comment *model.Comment) error {
	if comment.ID == "" {
		comment.ID = "c-1"
	}
	cp := *comment
	m.comments[comment.ID] = &cp
	return nil
}
func (m *mockItemRepo) UpdateComment(_ context.Context, _, _, commentID, body string) (*model.Comment, error) {
	c, ok := m.comments[commentID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	c.Body = body
	cp := *c
	return &cp, nil
}
func (m *mockItemRepo) DeleteComment(_ context.Context, _, _, commentID string) error {
	delete(m.comments, commentID)
	return nil
}

func TestItemService_EstimateClearDateParseAndCommentOwnership(t *testing.T) {
	repo := &mockItemRepo{
		items:    map[string]*model.WorkItem{},
		comments: map[string]*model.Comment{},
	}
	svc := NewItemService(repo)
	ctx := context.Background()
	author := model.UserContext{UserID: "u-author", Email: "author@jynko.dev", Name: "Author"}
	other := model.UserContext{UserID: "u-other", Email: "other@jynko.dev", Name: "Other"}
	admin := model.UserContext{UserID: "u-admin", Email: "admin@jynko.dev", Name: "Admin"}

	five := 5
	due := "2026-06-15"
	created, err := svc.Create(ctx, "proj-1", author, CreateWorkItemInput{
		Title:    "Support date strings and estimate clearing",
		Estimate: &five,
		DueDate:  &due,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.Estimate == nil || *created.Estimate != 5 || created.DueDate == nil {
		t.Fatalf("expected estimate=5 and parsed DueDate, got estimate=%v due=%v", created.Estimate, created.DueDate)
	}

	// Unmarshal JSON {"estimate": null} to clear estimate
	var patch UpdateWorkItemInput
	if err := json.Unmarshal([]byte(`{"estimate": null}`), &patch); err != nil {
		t.Fatalf("Unmarshal null estimate failed: %v", err)
	}
	updated, err := svc.Update(ctx, "proj-1", created.ID, patch)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Estimate != nil {
		t.Fatalf("expected Estimate to be cleared to nil, got %v", *updated.Estimate)
	}

	// Comment ownership
	comment, err := svc.CreateComment(ctx, "proj-1", created.ID, author, CreateCommentInput{Body: "Initial note"})
	if err != nil {
		t.Fatalf("CreateComment failed: %v", err)
	}
	if _, err := svc.UpdateComment(ctx, "proj-1", created.ID, comment.ID, other, model.RoleMember, UpdateCommentInput{Body: "Hijack"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden when non-author edits comment, got %v", err)
	}
	if err := svc.DeleteComment(ctx, "proj-1", created.ID, comment.ID, other, model.RoleMember); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden when non-author deletes comment, got %v", err)
	}
	if err := svc.DeleteComment(ctx, "proj-1", created.ID, comment.ID, admin, model.RoleAdmin); err != nil {
		t.Fatalf("expected ADMIN to delete comment, got %v", err)
	}
}

type mockPageRepo struct {
	pages map[string]*model.Page
}

func (m *mockPageRepo) ListPages(_ context.Context, _, _ string) ([]model.Page, error) {
	return []model.Page{}, nil
}
func (m *mockPageRepo) GetPage(_ context.Context, _, pageID string) (*model.Page, error) {
	p, ok := m.pages[pageID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (m *mockPageRepo) CreatePage(_ context.Context, page *model.Page) error {
	cp := *page
	m.pages[page.ID] = &cp
	return nil
}
func (m *mockPageRepo) UpdatePage(_ context.Context, page *model.Page, _ bool) error {
	if existing, ok := m.pages[page.ID]; ok && existing.IsLocked {
		return repository.ErrConflictLocked
	}
	cp := *page
	m.pages[page.ID] = &cp
	return nil
}
func (m *mockPageRepo) SetPageLock(_ context.Context, _, pageID, updatedBy string, isLocked bool) (*model.Page, error) {
	p, ok := m.pages[pageID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	p.IsLocked = isLocked
	p.UpdatedBy = updatedBy
	cp := *p
	return &cp, nil
}
func (m *mockPageRepo) DeletePage(_ context.Context, _, pageID string) error {
	delete(m.pages, pageID)
	return nil
}
func (m *mockPageRepo) ListVersions(_ context.Context, _, _ string) ([]model.PageVersion, error) {
	return []model.PageVersion{}, nil
}
func (m *mockPageRepo) GetVersion(_ context.Context, _, _, _ string) (*model.PageVersion, error) {
	return nil, repository.ErrNotFound
}
func (m *mockPageRepo) ListCollections(_ context.Context, _ string) ([]model.PageCollection, error) {
	return []model.PageCollection{}, nil
}
func (m *mockPageRepo) GetCollection(_ context.Context, _, _ string) (*model.PageCollection, error) {
	return nil, repository.ErrNotFound
}
func (m *mockPageRepo) CreateCollection(_ context.Context, _ *model.PageCollection) error { return nil }
func (m *mockPageRepo) UpdateCollection(_ context.Context, _ *model.PageCollection) error { return nil }
func (m *mockPageRepo) DeleteCollection(_ context.Context, _, _ string) error             { return nil }

func TestPageService_CycleDetectionAndLockProtection(t *testing.T) {
	repo := &mockPageRepo{
		pages: map[string]*model.Page{
			"p-a": {ID: "p-a", ProjectID: "proj-1", Title: "A", CreatedBy: "u-creator"},
			"p-b": {ID: "p-b", ProjectID: "proj-1", Title: "B", ParentPageID: "p-a", CreatedBy: "u-creator"},
			"p-c": {ID: "p-c", ProjectID: "proj-1", Title: "C", ParentPageID: "p-b", CreatedBy: "u-creator"},
		},
	}
	svc := NewPageService(repo)
	ctx := context.Background()
	creator := model.UserContext{UserID: "u-creator"}
	otherMember := model.UserContext{UserID: "u-member"}
	admin := model.UserContext{UserID: "u-admin"}

	// Setting A's parent to C creates A -> C -> B -> A cycle -> must fail
	cycleParent := "p-c"
	if _, err := svc.UpdatePage(ctx, "proj-1", "p-a", creator, UpdatePageInput{ParentPageID: &cycleParent}); err == nil {
		t.Fatalf("expected multi-node page parent cycle A->C->B->A to be rejected")
	}

	// Lock page A -> DeletePage must fail with ErrPageLocked, and non-creator MEMBER cannot unlock
	if _, err := svc.SetPageLock(ctx, "proj-1", "p-a", creator, model.RoleMember, true); err != nil {
		t.Fatalf("SetPageLock(true) failed: %v", err)
	}
	if err := svc.DeletePage(ctx, "proj-1", "p-a"); !errors.Is(err, ErrPageLocked) {
		t.Fatalf("expected ErrPageLocked on DeletePage of locked page, got %v", err)
	}
	if _, err := svc.SetPageLock(ctx, "proj-1", "p-a", otherMember, model.RoleMember, false); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden when non-creator MEMBER unlocks page, got %v", err)
	}
	if _, err := svc.SetPageLock(ctx, "proj-1", "p-a", admin, model.RoleAdmin, false); err != nil {
		t.Fatalf("expected ADMIN to unlock page, got %v", err)
	}
}
