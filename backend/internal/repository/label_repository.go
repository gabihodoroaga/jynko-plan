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

// LabelRepository defines persistence operations for project Labels.
type LabelRepository interface {
	List(ctx context.Context, projectID string) ([]model.Label, error)
	GetByID(ctx context.Context, projectID, labelID string) (*model.Label, error)
	Create(ctx context.Context, label *model.Label) error
	Update(ctx context.Context, label *model.Label) error
	Delete(ctx context.Context, projectID, labelID string) error
}

type firestoreLabelRepository struct {
	fc *FirestoreClient
}

// NewLabelRepository creates a Firestore-backed LabelRepository.
func NewLabelRepository(fc *FirestoreClient) LabelRepository {
	return &firestoreLabelRepository{fc: fc}
}

func (r *firestoreLabelRepository) List(ctx context.Context, projectID string) ([]model.Label, error) {
	labels := []model.Label{}
	iter := r.fc.ProjectDoc(projectID).Collection("labels").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var l model.Label
		if err := doc.DataTo(&l); err != nil {
			return nil, err
		}
		if l.ID == "" {
			l.ID = doc.Ref.ID
		}
		labels = append(labels, l)
	}
	return labels, nil
}

func (r *firestoreLabelRepository) GetByID(ctx context.Context, projectID, labelID string) (*model.Label, error) {
	snap, err := r.fc.ProjectDoc(projectID).Collection("labels").Doc(labelID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var l model.Label
	if err := snap.DataTo(&l); err != nil {
		return nil, err
	}
	if l.ID == "" {
		l.ID = snap.Ref.ID
	}
	return &l, nil
}

func (r *firestoreLabelRepository) Create(ctx context.Context, label *model.Label) error {
	col := r.fc.ProjectDoc(label.ProjectID).Collection("labels")
	docRef := col.NewDoc()
	if label.ID == "" {
		label.ID = docRef.ID
	} else {
		docRef = col.Doc(label.ID)
	}
	label.CreatedAt = time.Now().UTC()
	_, err := docRef.Set(ctx, label)
	return err
}

func (r *firestoreLabelRepository) Update(ctx context.Context, label *model.Label) error {
	_, err := r.fc.ProjectDoc(label.ProjectID).Collection("labels").Doc(label.ID).Set(ctx, label)
	return err
}

func (r *firestoreLabelRepository) Delete(ctx context.Context, projectID, labelID string) error {
	_, err := r.fc.ProjectDoc(projectID).Collection("labels").Doc(labelID).Delete(ctx)
	return err
}
