package service

import (
	"context"
	"math/rand"
	"strings"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

var validStickyColors = []string{
	model.StickyColorBanana,
	model.StickyColorFlamingo,
	model.StickyColorMint,
	model.StickyColorLagoon,
	model.StickyColorLavender,
	model.StickyColorPeach,
}

// StickyService handles business logic for project Sticky notes.
type StickyService struct {
	repo repository.StickyRepository
}

// NewStickyService creates a new StickyService.
func NewStickyService(repo repository.StickyRepository) *StickyService {
	return &StickyService{repo: repo}
}

// CreateStickyInput holds parameters for creating a sticky note.
type CreateStickyInput struct {
	Content  string `json:"content"`
	Color    string `json:"color"`
	IsPinned bool   `json:"is_pinned"`
}

// UpdateStickyInput holds parameters for updating a sticky note.
type UpdateStickyInput struct {
	Content  *string `json:"content"`
	Color    *string `json:"color"`
	IsPinned *bool   `json:"is_pinned"`
}

func (s *StickyService) ListStickies(ctx context.Context, projectID string) ([]model.Sticky, error) {
	return s.repo.List(ctx, projectID)
}

func (s *StickyService) CreateSticky(ctx context.Context, projectID string, author model.UserContext, input CreateStickyInput) (*model.Sticky, error) {
	color := normalizeStickyColor(input.Color)
	sticky := &model.Sticky{
		ProjectID: projectID,
		Content:   input.Content,
		Color:     color,
		IsPinned:  input.IsPinned,
		AuthorID:  author.UserID,
	}
	if err := s.repo.Create(ctx, sticky); err != nil {
		return nil, err
	}
	return sticky, nil
}

func (s *StickyService) UpdateSticky(ctx context.Context, projectID, stickyID string, input UpdateStickyInput) (*model.Sticky, error) {
	sticky, err := s.repo.GetByID(ctx, projectID, stickyID)
	if err != nil {
		return nil, err
	}
	if input.Content != nil {
		sticky.Content = *input.Content
	}
	if input.Color != nil {
		sticky.Color = normalizeStickyColor(*input.Color)
	}
	if input.IsPinned != nil {
		sticky.IsPinned = *input.IsPinned
	}
	if err := s.repo.Update(ctx, sticky); err != nil {
		return nil, err
	}
	return sticky, nil
}

func (s *StickyService) DeleteSticky(ctx context.Context, projectID, stickyID string) error {
	return s.repo.Delete(ctx, projectID, stickyID)
}

func normalizeStickyColor(raw string) string {
	upper := strings.ToUpper(strings.TrimSpace(raw))
	for _, c := range validStickyColors {
		if c == upper {
			return c
		}
	}
	return validStickyColors[rand.Intn(len(validStickyColors))]
}
