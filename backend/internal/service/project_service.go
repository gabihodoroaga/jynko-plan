package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"jynko/internal/model"
	"jynko/internal/repository"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	keyPrefixRegex  = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,9}$`)
)

// ProjectService handles business logic for Projects and RBAC membership.
type ProjectService struct {
	repo repository.ProjectRepository
}

// NewProjectService creates a new ProjectService.
func NewProjectService(repo repository.ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

// CreateProjectInput holds parameters for creating a project.
type CreateProjectInput struct {
	Name        string `json:"name" binding:"required"`
	KeyPrefix   string `json:"key_prefix"`
	Description string `json:"description"`
}

// UpdateProjectInput holds parameters for updating a project.
type UpdateProjectInput struct {
	Name        *string                `json:"name"`
	KeyPrefix   *string                `json:"key_prefix"`
	Description *string                `json:"description"`
	Settings    *model.ProjectSettings `json:"settings"`
}

// UpsertMemberInput holds parameters for adding or updating a project member.
type UpsertMemberInput struct {
	UserID string            `json:"user_id"`
	Email  string            `json:"email"`
	Name   string            `json:"name"`
	Role   model.ProjectRole `json:"role" binding:"required"`
}

func (s *ProjectService) CreateProject(ctx context.Context, creator model.UserContext, input CreateProjectInput) (*model.Project, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}

	prefix := strings.ToUpper(strings.TrimSpace(input.KeyPrefix))
	if prefix == "" {
		prefix = deriveKeyPrefix(name)
	}
	if !keyPrefixRegex.MatchString(prefix) {
		return nil, errors.New("key_prefix must be 2-10 uppercase alphanumeric characters starting with a letter")
	}

	proj := &model.Project{
		Name:        name,
		KeyPrefix:   prefix,
		Description: strings.TrimSpace(input.Description),
		ItemSeq:     0,
		MemberIDs:   []string{creator.UserID},
		Settings:    model.DefaultProjectSettings(),
		CreatedBy:   creator.UserID,
	}

	adminMember := &model.ProjectMember{
		UserID: creator.UserID,
		Email:  creator.Email,
		Name:   creator.Name,
		Role:   model.RoleAdmin,
	}

	if err := s.repo.Create(ctx, proj, adminMember); err != nil {
		return nil, err
	}
	return proj, nil
}

func (s *ProjectService) GetProject(ctx context.Context, projectID string) (*model.Project, error) {
	return s.repo.GetByID(ctx, projectID)
}

func (s *ProjectService) ListProjectsForUser(ctx context.Context, user model.UserContext) ([]model.Project, error) {
	return s.repo.ListByUser(ctx, user.UserID, user.Email)
}

func (s *ProjectService) UpdateProject(ctx context.Context, projectID string, input UpdateProjectInput) (*model.Project, error) {
	proj, err := s.repo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			return nil, ErrInvalidInput
		}
		proj.Name = trimmed
	}
	if input.KeyPrefix != nil {
		prefix := strings.ToUpper(strings.TrimSpace(*input.KeyPrefix))
		if !keyPrefixRegex.MatchString(prefix) {
			return nil, errors.New("key_prefix must be 2-10 uppercase alphanumeric characters starting with a letter")
		}
		proj.KeyPrefix = prefix
	}
	if input.Description != nil {
		proj.Description = strings.TrimSpace(*input.Description)
	}
	if input.Settings != nil {
		proj.Settings = input.Settings
	}

	if err := s.repo.Update(ctx, proj); err != nil {
		return nil, err
	}
	return proj, nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, projectID string) error {
	return s.repo.Delete(ctx, projectID)
}

func (s *ProjectService) ListMembers(ctx context.Context, projectID string) ([]model.ProjectMember, error) {
	return s.repo.ListMembers(ctx, projectID)
}

func (s *ProjectService) UpsertMember(ctx context.Context, projectID string, input UpsertMemberInput) (*model.ProjectMember, error) {
	if !model.IsValidRole(input.Role) {
		return nil, errors.New("role must be ADMIN, MEMBER, or VIEWER")
	}

	userID := strings.TrimSpace(input.UserID)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if userID == "" && email == "" {
		return nil, errors.New("either user_id or email is required")
	}
	if userID == "" {
		userID = email
	}
	if email == "" {
		email = userID
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = email
	}

	member := &model.ProjectMember{
		UserID: userID,
		Email:  email,
		Name:   name,
		Role:   input.Role,
	}

	if err := s.repo.UpsertMember(ctx, projectID, member); err != nil {
		return nil, err
	}
	return member, nil
}

func (s *ProjectService) RemoveMember(ctx context.Context, projectID, userID string) error {
	return s.repo.RemoveMember(ctx, projectID, userID)
}

func (s *ProjectService) GetMemberRole(ctx context.Context, projectID string, user model.UserContext) (model.ProjectRole, error) {
	member, err := s.repo.GetMember(ctx, projectID, user.UserID)
	if err != nil || member == nil {
		member, err = s.repo.GetMemberByEmail(ctx, projectID, user.Email)
		if err != nil || member == nil {
			return "", repository.ErrNotFound
		}
	}
	return member.Role, nil
}

func deriveKeyPrefix(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if (r >= 'A' && r <= 'Z') || (b.Len() > 0 && r >= '0' && r <= '9') {
			b.WriteRune(r)
			if b.Len() >= 5 {
				break
			}
		}
	}
	res := b.String()
	if len(res) < 2 {
		return "JYNKO"
	}
	return res
}
