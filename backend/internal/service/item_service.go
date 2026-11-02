package service

import (
	"context"
	"strings"
	"time"

	"jynko/internal/model"
	"jynko/internal/repository"
)

// ItemService handles business logic for WorkItems, SubWorkItems, and Comments.
type ItemService struct {
	repo repository.ItemRepository
}

// NewItemService creates a new ItemService.
func NewItemService(repo repository.ItemRepository) *ItemService {
	return &ItemService{repo: repo}
}

type CreateWorkItemInput struct {
	Title       string     `json:"title" binding:"required"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`
	ModuleID    string     `json:"module_id"`
	EpicID      string     `json:"epic_id"`
	CycleID     string     `json:"cycle_id"`
	AssigneeID  string     `json:"assignee_id"`
	LabelIDs    []string   `json:"label_ids"`
	Estimate    *int       `json:"estimate"`
	StartDate   *time.Time `json:"start_date"`
	DueDate     *time.Time `json:"due_date"`
}

type UpdateWorkItemInput struct {
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	Status      *string   `json:"status"`
	Priority    *string   `json:"priority"`
	ModuleID    *string   `json:"module_id"`
	EpicID      *string   `json:"epic_id"`
	CycleID     *string   `json:"cycle_id"`
	AssigneeID  *string   `json:"assignee_id"`
	LabelIDs    *[]string `json:"label_ids"`
	Estimate    *int      `json:"estimate"`
	StartDate   *string   `json:"start_date"`
	DueDate     *string   `json:"due_date"`
}

type CreateSubItemInput struct {
	Title       string `json:"title" binding:"required"`
	AssigneeID  string `json:"assignee_id"`
	IsCompleted bool   `json:"is_completed"`
}

type UpdateSubItemInput struct {
	Title       *string `json:"title"`
	AssigneeID  *string `json:"assignee_id"`
	IsCompleted *bool   `json:"is_completed"`
}

type CreateCommentInput struct {
	Body string `json:"body" binding:"required"`
}

func (s *ItemService) List(ctx context.Context, projectID string, filter model.WorkItemFilter) ([]model.WorkItem, error) {
	return s.repo.List(ctx, projectID, filter)
}

func (s *ItemService) GetByID(ctx context.Context, projectID, itemID string) (*model.WorkItem, error) {
	return s.repo.GetByID(ctx, projectID, itemID)
}

func (s *ItemService) Create(ctx context.Context, projectID string, reporter model.UserContext, input CreateWorkItemInput) (*model.WorkItem, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrInvalidInput
	}

	status := normalizeStatus(input.Status)
	priority := normalizePriority(input.Priority)

	labelIDs := input.LabelIDs
	if labelIDs == nil {
		labelIDs = []string{}
	}

	item := &model.WorkItem{
		ProjectID:   projectID,
		Title:       title,
		Description: strings.TrimSpace(input.Description),
		Status:      status,
		Priority:    priority,
		ModuleID:    strings.TrimSpace(input.ModuleID),
		EpicID:      strings.TrimSpace(input.EpicID),
		CycleID:     strings.TrimSpace(input.CycleID),
		AssigneeID:  strings.TrimSpace(input.AssigneeID),
		ReporterID:  reporter.UserID,
		LabelIDs:    labelIDs,
		Estimate:    input.Estimate,
		StartDate:   input.StartDate,
		DueDate:     input.DueDate,
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ItemService) Update(ctx context.Context, projectID, itemID string, input UpdateWorkItemInput) (*model.WorkItem, error) {
	item, err := s.repo.GetByID(ctx, projectID, itemID)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		trimmed := strings.TrimSpace(*input.Title)
		if trimmed == "" {
			return nil, ErrInvalidInput
		}
		item.Title = trimmed
	}
	if input.Description != nil {
		item.Description = *input.Description
	}
	if input.Status != nil {
		item.Status = normalizeStatus(*input.Status)
	}
	if input.Priority != nil {
		item.Priority = normalizePriority(*input.Priority)
	}
	if input.ModuleID != nil {
		item.ModuleID = strings.TrimSpace(*input.ModuleID)
	}
	if input.EpicID != nil {
		item.EpicID = strings.TrimSpace(*input.EpicID)
	}
	if input.CycleID != nil {
		item.CycleID = strings.TrimSpace(*input.CycleID)
	}
	if input.AssigneeID != nil {
		item.AssigneeID = strings.TrimSpace(*input.AssigneeID)
	}
	if input.LabelIDs != nil {
		item.LabelIDs = *input.LabelIDs
	}
	if input.Estimate != nil {
		item.Estimate = input.Estimate
	}
	if input.StartDate != nil {
		parsed, err := parseOptionalDate(*input.StartDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		item.StartDate = parsed
	}
	if input.DueDate != nil {
		parsed, err := parseOptionalDate(*input.DueDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		item.DueDate = parsed
	}

	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *ItemService) Delete(ctx context.Context, projectID, itemID string) error {
	return s.repo.Delete(ctx, projectID, itemID)
}

func (s *ItemService) ListSubItems(ctx context.Context, projectID, itemID string) ([]model.SubWorkItem, error) {
	return s.repo.ListSubItems(ctx, projectID, itemID)
}

func (s *ItemService) CreateSubItem(ctx context.Context, projectID, itemID string, input CreateSubItemInput) (*model.SubWorkItem, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrInvalidInput
	}
	sub := &model.SubWorkItem{
		ParentItemID: itemID,
		Title:        title,
		AssigneeID:   strings.TrimSpace(input.AssigneeID),
		IsCompleted:  input.IsCompleted,
	}
	if err := s.repo.CreateSubItem(ctx, projectID, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *ItemService) UpdateSubItem(ctx context.Context, projectID, itemID, subItemID string, input UpdateSubItemInput) (*model.SubWorkItem, error) {
	sub, err := s.repo.GetSubItemByID(ctx, projectID, itemID, subItemID)
	if err != nil {
		return nil, err
	}
	if input.Title != nil {
		trimmed := strings.TrimSpace(*input.Title)
		if trimmed == "" {
			return nil, ErrInvalidInput
		}
		sub.Title = trimmed
	}
	if input.AssigneeID != nil {
		sub.AssigneeID = strings.TrimSpace(*input.AssigneeID)
	}
	if input.IsCompleted != nil {
		sub.IsCompleted = *input.IsCompleted
	}
	if err := s.repo.UpdateSubItem(ctx, projectID, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *ItemService) DeleteSubItem(ctx context.Context, projectID, itemID, subItemID string) error {
	return s.repo.DeleteSubItem(ctx, projectID, itemID, subItemID)
}

func (s *ItemService) ListComments(ctx context.Context, projectID, itemID string) ([]model.Comment, error) {
	return s.repo.ListComments(ctx, projectID, itemID)
}

func (s *ItemService) CreateComment(ctx context.Context, projectID, itemID string, author model.UserContext, input CreateCommentInput) (*model.Comment, error) {
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return nil, ErrInvalidInput
	}
	comment := &model.Comment{
		WorkItemID: itemID,
		AuthorID:   author.UserID,
		AuthorName: author.Name,
		Body:       body,
	}
	if err := s.repo.CreateComment(ctx, projectID, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *ItemService) DeleteComment(ctx context.Context, projectID, itemID, commentID string) error {
	return s.repo.DeleteComment(ctx, projectID, itemID, commentID)
}

func normalizeStatus(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case model.StatusBacklog:
		return model.StatusBacklog
	case model.StatusTodo:
		return model.StatusTodo
	case model.StatusInProgress:
		return model.StatusInProgress
	case model.StatusInReview:
		return model.StatusInReview
	case model.StatusDone:
		return model.StatusDone
	case model.StatusCancelled:
		return model.StatusCancelled
	default:
		return model.StatusTodo
	}
}

func normalizePriority(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case model.PriorityNone:
		return model.PriorityNone
	case model.PriorityLow:
		return model.PriorityLow
	case model.PriorityMedium:
		return model.PriorityMedium
	case model.PriorityHigh:
		return model.PriorityHigh
	case model.PriorityUrgent:
		return model.PriorityUrgent
	default:
		return model.PriorityMedium
	}
}

func parseOptionalDate(raw string) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02"} {
		if t, err := time.Parse(layout, trimmed); err == nil {
			utc := t.UTC()
			return &utc, nil
		}
	}
	return nil, ErrInvalidInput
}
