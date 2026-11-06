package service

import (
	"context"
	"strings"
	"time"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

// CycleService handles business logic for project sprint Cycles.
type CycleService struct {
	repo repository.CycleRepository
}

// NewCycleService creates a new CycleService.
func NewCycleService(repo repository.CycleRepository) *CycleService {
	return &CycleService{repo: repo}
}

type CreateCycleInput struct {
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	IsActive    bool      `json:"is_active"`
}

type UpdateCycleInput struct {
	Name        *string    `json:"name"`
	Description *string    `json:"description"`
	StartDate   *time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
	IsActive    *bool      `json:"is_active"`
}

func (s *CycleService) List(ctx context.Context, projectID string) ([]model.Cycle, error) {
	return s.repo.List(ctx, projectID)
}

func (s *CycleService) GetByID(ctx context.Context, projectID, cycleID string) (*model.Cycle, error) {
	return s.repo.GetByID(ctx, projectID, cycleID)
}

func (s *CycleService) Create(ctx context.Context, projectID string, input CreateCycleInput) (*model.Cycle, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	start := input.StartDate
	if start.IsZero() {
		start = time.Now().UTC()
	}
	end := input.EndDate
	if end.IsZero() {
		end = start.AddDate(0, 0, 14)
	}

	cycle := &model.Cycle{
		ProjectID:   projectID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		StartDate:   start,
		EndDate:     end,
		IsActive:    input.IsActive,
	}
	if err := s.repo.Create(ctx, cycle); err != nil {
		return nil, err
	}
	return cycle, nil
}

func (s *CycleService) Update(ctx context.Context, projectID, cycleID string, input UpdateCycleInput) (*model.Cycle, error) {
	cycle, err := s.repo.GetByID(ctx, projectID, cycleID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			return nil, ErrInvalidInput
		}
		cycle.Name = trimmed
	}
	if input.Description != nil {
		cycle.Description = strings.TrimSpace(*input.Description)
	}
	if input.StartDate != nil {
		cycle.StartDate = *input.StartDate
	}
	if input.EndDate != nil {
		cycle.EndDate = *input.EndDate
	}
	if input.IsActive != nil {
		cycle.IsActive = *input.IsActive
	}
	if err := s.repo.Update(ctx, cycle); err != nil {
		return nil, err
	}
	return cycle, nil
}

func (s *CycleService) Delete(ctx context.Context, projectID, cycleID string) error {
	return s.repo.Delete(ctx, projectID, cycleID)
}
