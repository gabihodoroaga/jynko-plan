package service

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

// OptionalInt distinguishes an omitted JSON field (Set=false) from explicit null or integer (Set=true).
type OptionalInt struct {
	Set   bool
	Value *int
}

func (o *OptionalInt) UnmarshalJSON(data []byte) error {
	o.Set = true
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		o.Value = nil
		return nil
	}
	var v int
	if err := json.Unmarshal(trimmed, &v); err != nil {
		return err
	}
	if v < 0 {
		return ErrInvalidInput
	}
	if v == 0 {
		o.Value = nil
		return nil
	}
	o.Value = &v
	return nil
}

// ItemService handles business logic for WorkItems, SubWorkItems, and Comments.
type ItemService struct {
	repo repository.ItemRepository
}

// NewItemService creates a new ItemService.
func NewItemService(repo repository.ItemRepository) *ItemService {
	return &ItemService{repo: repo}
}

type CreateWorkItemInput struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	ModuleID    string   `json:"module_id"`
	EpicID      string   `json:"epic_id"`
	CycleID     string   `json:"cycle_id"`
	AssigneeID  string   `json:"assignee_id"`
	LabelIDs    []string `json:"label_ids"`
	Estimate    *int     `json:"estimate"`
	StartDate   *string  `json:"start_date"`
	DueDate     *string  `json:"due_date"`
}

type UpdateWorkItemInput struct {
	Title       *string     `json:"title"`
	Description *string     `json:"description"`
	Status      *string     `json:"status"`
	Priority    *string     `json:"priority"`
	ModuleID    *string     `json:"module_id"`
	EpicID      *string     `json:"epic_id"`
	CycleID     *string     `json:"cycle_id"`
	AssigneeID  *string     `json:"assignee_id"`
	LabelIDs    *[]string   `json:"label_ids"`
	Estimate    OptionalInt `json:"estimate"`
	StartDate   *string     `json:"start_date"`
	DueDate     *string     `json:"due_date"`
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
	Body          string `json:"body" binding:"required"`
	TargetAgentID string `json:"target_agent_id,omitempty"`
	IsSystem      bool   `json:"-"`
	SlashCommand  string `json:"-"`
	AgentID       string `json:"-"`
	AgentURL      string `json:"-"`
}

type UpdateCommentInput struct {
	Body string `json:"body" binding:"required"`
}

func (s *ItemService) List(ctx context.Context, projectID string, filter model.WorkItemFilter) ([]model.WorkItem, error) {
	return s.repo.List(ctx, projectID, filter)
}

func (s *ItemService) GetByID(ctx context.Context, projectID, itemID string) (*model.WorkItem, error) {
	return s.repo.GetByID(ctx, projectID, itemID)
}

func (s *ItemService) GetByKey(ctx context.Context, projectID, key string) (*model.WorkItem, error) {
	return s.repo.GetByKey(ctx, projectID, key)
}

func (s *ItemService) GetByIDOrKey(ctx context.Context, projectID, idOrKey string) (*model.WorkItem, error) {
	trimmed := strings.TrimSpace(idOrKey)
	if trimmed == "" {
		return nil, ErrInvalidInput
	}
	item, err := s.repo.GetByID(ctx, projectID, trimmed)
	if err == nil && item != nil {
		return item, nil
	}
	return s.repo.GetByKey(ctx, projectID, trimmed)
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

	var estimate *int
	if input.Estimate != nil {
		if *input.Estimate < 0 {
			return nil, ErrInvalidInput
		}
		if *input.Estimate > 0 {
			est := *input.Estimate
			estimate = &est
		}
	}

	var startDate *time.Time
	if input.StartDate != nil {
		parsed, err := parseOptionalDate(*input.StartDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		startDate = parsed
	}

	var dueDate *time.Time
	if input.DueDate != nil {
		parsed, err := parseOptionalDate(*input.DueDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		dueDate = parsed
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
		Estimate:    estimate,
		StartDate:   startDate,
		DueDate:     dueDate,
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
		if *input.LabelIDs == nil {
			item.LabelIDs = []string{}
		} else {
			item.LabelIDs = *input.LabelIDs
		}
	}
	if input.Estimate.Set {
		if input.Estimate.Value != nil && *input.Estimate.Value < 0 {
			return nil, ErrInvalidInput
		}
		item.Estimate = input.Estimate.Value
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
		WorkItemID:   itemID,
		AuthorID:     author.UserID,
		AuthorName:   author.Name,
		Body:         body,
		IsSystem:     input.IsSystem,
		SlashCommand: input.SlashCommand,
		AgentID:      input.AgentID,
		AgentURL:     input.AgentURL,
	}
	if err := s.repo.CreateComment(ctx, projectID, comment); err != nil {
		return nil, err
	}
	return comment, nil
}

// SaveItem persists direct modifications to a WorkItem (e.g. Scion agent bindings).
func (s *ItemService) SaveItem(ctx context.Context, item *model.WorkItem) error {
	if item == nil {
		return ErrInvalidInput
	}
	return s.repo.Update(ctx, item)
}

func (s *ItemService) UpdateComment(
	ctx context.Context,
	projectID, itemID, commentID string,
	actor model.UserContext,
	actorRole model.ProjectRole,
	input UpdateCommentInput,
) (*model.Comment, error) {
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return nil, ErrInvalidInput
	}
	existing, err := s.repo.GetCommentByID(ctx, projectID, itemID, commentID)
	if err != nil {
		return nil, err
	}
	if actorRole != model.RoleAdmin && !isSameActor(existing.AuthorID, actor) {
		return nil, ErrForbidden
	}
	return s.repo.UpdateComment(ctx, projectID, itemID, commentID, body)
}

func (s *ItemService) DeleteComment(
	ctx context.Context,
	projectID, itemID, commentID string,
	actor model.UserContext,
	actorRole model.ProjectRole,
) error {
	existing, err := s.repo.GetCommentByID(ctx, projectID, itemID, commentID)
	if err != nil {
		return err
	}
	if actorRole != model.RoleAdmin && !isSameActor(existing.AuthorID, actor) {
		return ErrForbidden
	}
	return s.repo.DeleteComment(ctx, projectID, itemID, commentID)
}

func isSameActor(authorID string, actor model.UserContext) bool {
	if authorID == "" {
		return false
	}
	if actor.UserID != "" && authorID == actor.UserID {
		return true
	}
	if actor.Email != "" && strings.EqualFold(authorID, actor.Email) {
		return true
	}
	return false
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
