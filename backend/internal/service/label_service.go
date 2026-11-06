package service

import (
	"context"
	"strings"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

// LabelService handles business logic for project Labels.
type LabelService struct {
	repo repository.LabelRepository
}

// NewLabelService creates a new LabelService.
func NewLabelService(repo repository.LabelRepository) *LabelService {
	return &LabelService{repo: repo}
}

type CreateLabelInput struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color"`
}

type UpdateLabelInput struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

func (s *LabelService) List(ctx context.Context, projectID string) ([]model.Label, error) {
	return s.repo.List(ctx, projectID)
}

func (s *LabelService) Create(ctx context.Context, projectID string, input CreateLabelInput) (*model.Label, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	color := strings.TrimSpace(input.Color)
	if color == "" {
		color = "#0284c7"
	}
	lbl := &model.Label{
		ProjectID: projectID,
		Name:      name,
		Color:     color,
	}
	if err := s.repo.Create(ctx, lbl); err != nil {
		return nil, err
	}
	return lbl, nil
}

func (s *LabelService) Update(ctx context.Context, projectID, labelID string, input UpdateLabelInput) (*model.Label, error) {
	lbl, err := s.repo.GetByID(ctx, projectID, labelID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			return nil, ErrInvalidInput
		}
		lbl.Name = trimmed
	}
	if input.Color != nil {
		lbl.Color = strings.TrimSpace(*input.Color)
	}
	if err := s.repo.Update(ctx, lbl); err != nil {
		return nil, err
	}
	return lbl, nil
}

func (s *LabelService) Delete(ctx context.Context, projectID, labelID string) error {
	return s.repo.Delete(ctx, projectID, labelID)
}
