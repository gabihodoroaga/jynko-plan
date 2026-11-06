package repository

import (
	"context"
	"errors"
	"sort"
	"time"

	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
)

// StickyRepository defines persistence operations for project Sticky notes.
type StickyRepository interface {
	List(ctx context.Context, projectID string) ([]model.Sticky, error)
	GetByID(ctx context.Context, projectID, stickyID string) (*model.Sticky, error)
	Create(ctx context.Context, sticky *model.Sticky) error
	Update(ctx context.Context, sticky *model.Sticky) error
	Delete(ctx context.Context, projectID, stickyID string) error
}

type firestoreStickyRepository struct {
	fc *FirestoreClient
}

// NewStickyRepository creates a Firestore-backed StickyRepository.
func NewStickyRepository(fc *FirestoreClient) StickyRepository {
	return &firestoreStickyRepository{fc: fc}
}

func (r *firestoreStickyRepository) List(ctx context.Context, projectID string) ([]model.Sticky, error) {
	stickies := []model.Sticky{}
	iter := r.fc.ProjectDoc(projectID).Collection("stickies").Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var s model.Sticky
		if err := doc.DataTo(&s); err != nil {
			return nil, err
		}
		if s.ID == "" {
			s.ID = doc.Ref.ID
		}
		stickies = append(stickies, s)
	}

	sort.Slice(stickies, func(i, j int) bool {
		if stickies[i].IsPinned != stickies[j].IsPinned {
			return stickies[i].IsPinned
		}
		return stickies[i].UpdatedAt.After(stickies[j].UpdatedAt)
	})

	return stickies, nil
}

func (r *firestoreStickyRepository) GetByID(ctx context.Context, projectID, stickyID string) (*model.Sticky, error) {
	snap, err := r.fc.ProjectDoc(projectID).Collection("stickies").Doc(stickyID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var s model.Sticky
	if err := snap.DataTo(&s); err != nil {
		return nil, err
	}
	if s.ID == "" {
		s.ID = snap.Ref.ID
	}
	return &s, nil
}

func (r *firestoreStickyRepository) Create(ctx context.Context, sticky *model.Sticky) error {
	col := r.fc.ProjectDoc(sticky.ProjectID).Collection("stickies")
	docRef := col.NewDoc()
	if sticky.ID == "" {
		sticky.ID = docRef.ID
	} else {
		docRef = col.Doc(sticky.ID)
	}
	now := time.Now().UTC()
	sticky.CreatedAt = now
	sticky.UpdatedAt = now
	_, err := docRef.Set(ctx, sticky)
	return err
}

func (r *firestoreStickyRepository) Update(ctx context.Context, sticky *model.Sticky) error {
	sticky.UpdatedAt = time.Now().UTC()
	_, err := r.fc.ProjectDoc(sticky.ProjectID).Collection("stickies").Doc(sticky.ID).Set(ctx, sticky)
	return err
}

func (r *firestoreStickyRepository) Delete(ctx context.Context, projectID, stickyID string) error {
	_, err := r.fc.ProjectDoc(projectID).Collection("stickies").Doc(stickyID).Delete(ctx)
	return err
}
