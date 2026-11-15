package service

import (
	"context"
	"errors"
	"strings"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

// ErrPageLocked is returned when attempting to update or delete a locked Wiki Page.
var ErrPageLocked = errors.New("page is locked and cannot be edited or deleted")

const maxPageHierarchyDepth = 50

// PageService handles business logic for Wiki Pages, Versions, and Collections.
type PageService struct {
	repo repository.PageRepository
}

// NewPageService creates a new PageService.
func NewPageService(repo repository.PageRepository) *PageService {
	return &PageService{repo: repo}
}

type CreatePageInput struct {
	Title        string   `json:"title" binding:"required"`
	Content      string   `json:"content"`
	CollectionID string   `json:"collection_id"`
	ParentPageID string   `json:"parent_page_id"`
	Order        int      `json:"order"`
	LabelIDs     []string `json:"label_ids"`
}

type UpdatePageInput struct {
	Title         *string   `json:"title"`
	Content       *string   `json:"content"`
	CollectionID  *string   `json:"collection_id"`
	ParentPageID  *string   `json:"parent_page_id"`
	Order         *int      `json:"order"`
	LabelIDs      *[]string `json:"label_ids"`
	CreateVersion *bool     `json:"create_version"`
}

type CreateCollectionInput struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type UpdateCollectionInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (s *PageService) ListPages(ctx context.Context, projectID, collectionID string) ([]model.Page, error) {
	return s.repo.ListPages(ctx, projectID, collectionID)
}

func (s *PageService) GetPage(ctx context.Context, projectID, pageID string) (*model.Page, error) {
	return s.repo.GetPage(ctx, projectID, pageID)
}

func (s *PageService) CreatePage(ctx context.Context, projectID string, author model.UserContext, input CreatePageInput) (*model.Page, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrInvalidInput
	}
	parentID := strings.TrimSpace(input.ParentPageID)
	if parentID != "" {
		if err := s.validateNoPageCycle(ctx, projectID, "", parentID); err != nil {
			return nil, err
		}
	}

	labelIDs := input.LabelIDs
	if labelIDs == nil {
		labelIDs = []string{}
	}

	page := &model.Page{
		ProjectID:    projectID,
		CollectionID: strings.TrimSpace(input.CollectionID),
		ParentPageID: parentID,
		Title:        title,
		Content:      input.Content,
		Order:        input.Order,
		IsLocked:     false,
		LabelIDs:     labelIDs,
		CreatedBy:    author.UserID,
		UpdatedBy:    author.UserID,
	}

	if err := s.repo.CreatePage(ctx, page); err != nil {
		return nil, err
	}
	return page, nil
}

func (s *PageService) UpdatePage(ctx context.Context, projectID, pageID string, editor model.UserContext, input UpdatePageInput) (*model.Page, error) {
	page, err := s.repo.GetPage(ctx, projectID, pageID)
	if err != nil {
		return nil, err
	}

	if page.IsLocked {
		return nil, ErrPageLocked
	}

	contentChanged := false
	if input.Title != nil {
		trimmed := strings.TrimSpace(*input.Title)
		if trimmed == "" {
			return nil, ErrInvalidInput
		}
		if trimmed != page.Title {
			contentChanged = true
		}
		page.Title = trimmed
	}
	if input.Content != nil {
		if *input.Content != page.Content {
			contentChanged = true
		}
		page.Content = *input.Content
	}
	if input.CollectionID != nil {
		page.CollectionID = strings.TrimSpace(*input.CollectionID)
	}
	if input.ParentPageID != nil {
		parentID := strings.TrimSpace(*input.ParentPageID)
		if parentID != "" {
			if err := s.validateNoPageCycle(ctx, projectID, page.ID, parentID); err != nil {
				return nil, err
			}
		}
		page.ParentPageID = parentID
	}
	if input.Order != nil {
		page.Order = *input.Order
	}
	if input.LabelIDs != nil {
		if *input.LabelIDs == nil {
			page.LabelIDs = []string{}
		} else {
			page.LabelIDs = *input.LabelIDs
		}
	}

	createVersion := contentChanged
	if input.CreateVersion != nil && !*input.CreateVersion {
		createVersion = false
	}

	page.UpdatedBy = editor.UserID
	if err := s.repo.UpdatePage(ctx, page, createVersion); err != nil {
		if errors.Is(err, repository.ErrConflictLocked) {
			return nil, ErrPageLocked
		}
		return nil, err
	}
	return page, nil
}

func (s *PageService) validateNoPageCycle(ctx context.Context, projectID, pageID, candidateParentID string) error {
	if candidateParentID == "" {
		return nil
	}
	if pageID != "" && candidateParentID == pageID {
		return errors.New("a page cannot be its own parent")
	}

	visited := make(map[string]bool)
	if pageID != "" {
		visited[pageID] = true
	}

	curr := candidateParentID
	for depth := 0; curr != "" && depth < maxPageHierarchyDepth; depth++ {
		if visited[curr] {
			return errors.New("circular page hierarchy detected")
		}
		visited[curr] = true

		ancestor, err := s.repo.GetPage(ctx, projectID, curr)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return errors.New("parent page not found in this project")
			}
			return err
		}
		curr = strings.TrimSpace(ancestor.ParentPageID)
	}
	if curr != "" {
		return errors.New("page hierarchy exceeds maximum nesting depth")
	}
	return nil
}

func (s *PageService) SetPageLock(
	ctx context.Context,
	projectID, pageID string,
	editor model.UserContext,
	editorRole model.ProjectRole,
	isLocked bool,
) (*model.Page, error) {
	page, err := s.repo.GetPage(ctx, projectID, pageID)
	if err != nil {
		return nil, err
	}
	// Unlocking a locked page requires project ADMIN or the original page creator
	if !isLocked && page.IsLocked && editorRole != model.RoleAdmin && !isSameActor(page.CreatedBy, editor) {
		return nil, ErrForbidden
	}
	return s.repo.SetPageLock(ctx, projectID, pageID, editor.UserID, isLocked)
}

func (s *PageService) DeletePage(ctx context.Context, projectID, pageID string) error {
	page, err := s.repo.GetPage(ctx, projectID, pageID)
	if err != nil {
		return err
	}
	if page.IsLocked {
		return ErrPageLocked
	}
	return s.repo.DeletePage(ctx, projectID, pageID)
}

func (s *PageService) ListVersions(ctx context.Context, projectID, pageID string) ([]model.PageVersion, error) {
	return s.repo.ListVersions(ctx, projectID, pageID)
}

func (s *PageService) RestoreVersion(ctx context.Context, projectID, pageID, versionID string, editor model.UserContext) (*model.Page, error) {
	page, err := s.repo.GetPage(ctx, projectID, pageID)
	if err != nil {
		return nil, err
	}
	if page.IsLocked {
		return nil, ErrPageLocked
	}
	ver, err := s.repo.GetVersion(ctx, projectID, pageID, versionID)
	if err != nil {
		return nil, err
	}
	page.Title = ver.Title
	page.Content = ver.Content
	page.UpdatedBy = editor.UserID
	if err := s.repo.UpdatePage(ctx, page, true); err != nil {
		if errors.Is(err, repository.ErrConflictLocked) {
			return nil, ErrPageLocked
		}
		return nil, err
	}
	return page, nil
}

func (s *PageService) ListCollections(ctx context.Context, projectID string) ([]model.PageCollection, error) {
	return s.repo.ListCollections(ctx, projectID)
}

func (s *PageService) CreateCollection(ctx context.Context, projectID string, input CreateCollectionInput) (*model.PageCollection, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	col := &model.PageCollection{
		ProjectID:   projectID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
	}
	if err := s.repo.CreateCollection(ctx, col); err != nil {
		return nil, err
	}
	return col, nil
}

func (s *PageService) UpdateCollection(ctx context.Context, projectID, collectionID string, input UpdateCollectionInput) (*model.PageCollection, error) {
	col, err := s.repo.GetCollection(ctx, projectID, collectionID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			return nil, ErrInvalidInput
		}
		col.Name = trimmed
	}
	if input.Description != nil {
		col.Description = strings.TrimSpace(*input.Description)
	}
	if err := s.repo.UpdateCollection(ctx, col); err != nil {
		return nil, err
	}
	return col, nil
}

func (s *PageService) DeleteCollection(ctx context.Context, projectID, collectionID string) error {
	pages, err := s.repo.ListPages(ctx, projectID, collectionID)
	if err != nil {
		return err
	}
	for _, p := range pages {
		if p.IsLocked {
			return ErrPageLocked
		}
	}
	return s.repo.DeleteCollection(ctx, projectID, collectionID)
}
