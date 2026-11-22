package service

import (
	"context"
	"strings"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

// ModuleService handles business logic for project Modules.
type ModuleService struct {
	repo repository.ModuleRepository
}

// NewModuleService creates a new ModuleService.
func NewModuleService(repo repository.ModuleRepository) *ModuleService {
	return &ModuleService{repo: repo}
}

type CreateModuleInput struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	LeadUserID  string `json:"lead_user_id"`
	GitRepoURL  string `json:"git_repo_url"`
	BaseBranch  string `json:"base_branch"`
}

type UpdateModuleInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	LeadUserID  *string `json:"lead_user_id"`
	GitRepoURL  *string `json:"git_repo_url"`
	BaseBranch  *string `json:"base_branch"`
}

func (s *ModuleService) List(ctx context.Context, projectID string) ([]model.Module, error) {
	return s.repo.List(ctx, projectID)
}

func (s *ModuleService) GetByID(ctx context.Context, projectID, moduleID string) (*model.Module, error) {
	return s.repo.GetByID(ctx, projectID, moduleID)
}

func (s *ModuleService) Create(ctx context.Context, projectID string, input CreateModuleInput) (*model.Module, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	mod := &model.Module{
		ProjectID:   projectID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		LeadUserID:  strings.TrimSpace(input.LeadUserID),
		GitRepoURL:  strings.TrimSpace(input.GitRepoURL),
		BaseBranch:  strings.TrimSpace(input.BaseBranch),
	}
	if err := s.repo.Create(ctx, mod); err != nil {
		return nil, err
	}
	return mod, nil
}

func (s *ModuleService) Update(ctx context.Context, projectID, moduleID string, input UpdateModuleInput) (*model.Module, error) {
	mod, err := s.repo.GetByID(ctx, projectID, moduleID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			return nil, ErrInvalidInput
		}
		mod.Name = trimmed
	}
	if input.Description != nil {
		mod.Description = strings.TrimSpace(*input.Description)
	}
	if input.LeadUserID != nil {
		mod.LeadUserID = strings.TrimSpace(*input.LeadUserID)
	}
	if input.GitRepoURL != nil {
		mod.GitRepoURL = strings.TrimSpace(*input.GitRepoURL)
	}
	if input.BaseBranch != nil {
		mod.BaseBranch = strings.TrimSpace(*input.BaseBranch)
	}
	if err := s.repo.Update(ctx, mod); err != nil {
		return nil, err
	}
	return mod, nil
}

func (s *ModuleService) Delete(ctx context.Context, projectID, moduleID string) error {
	return s.repo.Delete(ctx, projectID, moduleID)
}
