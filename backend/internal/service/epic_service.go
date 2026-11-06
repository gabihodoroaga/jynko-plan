package service

import (
	"context"
	"strings"
	"time"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

// EpicService handles business logic for project Epics.
type EpicService struct {
	repo repository.EpicRepository
}

// NewEpicService creates a new EpicService.
func NewEpicService(repo repository.EpicRepository) *EpicService {
	return &EpicService{repo: repo}
}

type CreateEpicInput struct {
	Name        string     `json:"name" binding:"required"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	StartDate   *time.Time `json:"start_date"`
	TargetDate  *time.Time `json:"target_date"`
}

type UpdateEpicInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	StartDate   *string `json:"start_date"`
	TargetDate  *string `json:"target_date"`
}

func (s *EpicService) List(ctx context.Context, projectID string) ([]model.Epic, error) {
	return s.repo.List(ctx, projectID)
}

func (s *EpicService) GetByID(ctx context.Context, projectID, epicID string) (*model.Epic, error) {
	return s.repo.GetByID(ctx, projectID, epicID)
}

func (s *EpicService) Create(ctx context.Context, projectID string, input CreateEpicInput) (*model.Epic, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	status := strings.ToUpper(strings.TrimSpace(input.Status))
	if status == "" {
		status = model.EpicStatusPlanned
	}

	epic := &model.Epic{
		ProjectID:   projectID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Status:      status,
		StartDate:   input.StartDate,
		TargetDate:  input.TargetDate,
	}
	if err := s.repo.Create(ctx, epic); err != nil {
		return nil, err
	}
	return epic, nil
}

func (s *EpicService) Update(ctx context.Context, projectID, epicID string, input UpdateEpicInput) (*model.Epic, error) {
	epic, err := s.repo.GetByID(ctx, projectID, epicID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			return nil, ErrInvalidInput
		}
		epic.Name = trimmed
	}
	if input.Description != nil {
		epic.Description = strings.TrimSpace(*input.Description)
	}
	if input.Status != nil {
		epic.Status = strings.ToUpper(strings.TrimSpace(*input.Status))
	}
	if input.StartDate != nil {
		parsed, err := parseOptionalDate(*input.StartDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		epic.StartDate = parsed
	}
	if input.TargetDate != nil {
		parsed, err := parseOptionalDate(*input.TargetDate)
		if err != nil {
			return nil, ErrInvalidInput
		}
		epic.TargetDate = parsed
	}
	if err := s.repo.Update(ctx, epic); err != nil {
		return nil, err
	}
	return epic, nil
}

func (s *EpicService) Delete(ctx context.Context, projectID, epicID string) error {
	return s.repo.Delete(ctx, projectID, epicID)
}
