package repository

import (
	"context"
	"errors"
	"time"

	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
)

// EpicRepository defines persistence operations for project Epics.
type EpicRepository interface {
	List(ctx context.Context, projectID string) ([]model.Epic, error)
	GetByID(ctx context.Context, projectID, epicID string) (*model.Epic, error)
	Create(ctx context.Context, epic *model.Epic) error
	Update(ctx context.Context, epic *model.Epic) error
	Delete(ctx context.Context, projectID, epicID string) error
}

type firestoreEpicRepository struct {
	fc *FirestoreClient
}

// NewEpicRepository creates a Firestore-backed EpicRepository.
func NewEpicRepository(fc *FirestoreClient) EpicRepository {
	return &firestoreEpicRepository{fc: fc}
}

func (r *firestoreEpicRepository) List(ctx context.Context, projectID string) ([]model.Epic, error) {
	epics := []model.Epic{}
	iter := r.fc.ProjectDoc(projectID).Collection("epics").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var e model.Epic
		if err := doc.DataTo(&e); err != nil {
			return nil, err
		}
		if e.ID == "" {
			e.ID = doc.Ref.ID
		}
		epics = append(epics, e)
	}
	return epics, nil
}

func (r *firestoreEpicRepository) GetByID(ctx context.Context, projectID, epicID string) (*model.Epic, error) {
	snap, err := r.fc.ProjectDoc(projectID).Collection("epics").Doc(epicID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var e model.Epic
	if err := snap.DataTo(&e); err != nil {
		return nil, err
	}
	if e.ID == "" {
		e.ID = snap.Ref.ID
	}
	return &e, nil
}

func (r *firestoreEpicRepository) Create(ctx context.Context, epic *model.Epic) error {
	col := r.fc.ProjectDoc(epic.ProjectID).Collection("epics")
	docRef := col.NewDoc()
	if epic.ID == "" {
		epic.ID = docRef.ID
	} else {
		docRef = col.Doc(epic.ID)
	}
	now := time.Now().UTC()
	epic.CreatedAt = now
	epic.UpdatedAt = now
	_, err := docRef.Set(ctx, epic)
	return err
}

func (r *firestoreEpicRepository) Update(ctx context.Context, epic *model.Epic) error {
	epic.UpdatedAt = time.Now().UTC()
	_, err := r.fc.ProjectDoc(epic.ProjectID).Collection("epics").Doc(epic.ID).Set(ctx, epic)
	return err
}

func (r *firestoreEpicRepository) Delete(ctx context.Context, projectID, epicID string) error {
	_, err := r.fc.ProjectDoc(projectID).Collection("epics").Doc(epicID).Delete(ctx)
	return err
}
