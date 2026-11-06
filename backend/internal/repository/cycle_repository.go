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

// CycleRepository defines persistence operations for project Cycles.
type CycleRepository interface {
	List(ctx context.Context, projectID string) ([]model.Cycle, error)
	GetByID(ctx context.Context, projectID, cycleID string) (*model.Cycle, error)
	Create(ctx context.Context, cycle *model.Cycle) error
	Update(ctx context.Context, cycle *model.Cycle) error
	Delete(ctx context.Context, projectID, cycleID string) error
}

type firestoreCycleRepository struct {
	fc *FirestoreClient
}

// NewCycleRepository creates a Firestore-backed CycleRepository.
func NewCycleRepository(fc *FirestoreClient) CycleRepository {
	return &firestoreCycleRepository{fc: fc}
}

func (r *firestoreCycleRepository) List(ctx context.Context, projectID string) ([]model.Cycle, error) {
	cycles := []model.Cycle{}
	iter := r.fc.ProjectDoc(projectID).Collection("cycles").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var c model.Cycle
		if err := doc.DataTo(&c); err != nil {
			return nil, err
		}
		if c.ID == "" {
			c.ID = doc.Ref.ID
		}
		cycles = append(cycles, c)
	}

	sort.Slice(cycles, func(i, j int) bool {
		if cycles[i].IsActive != cycles[j].IsActive {
			return cycles[i].IsActive
		}
		return cycles[i].StartDate.After(cycles[j].StartDate)
	})

	return cycles, nil
}

func (r *firestoreCycleRepository) GetByID(ctx context.Context, projectID, cycleID string) (*model.Cycle, error) {
	snap, err := r.fc.ProjectDoc(projectID).Collection("cycles").Doc(cycleID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var c model.Cycle
	if err := snap.DataTo(&c); err != nil {
		return nil, err
	}
	if c.ID == "" {
		c.ID = snap.Ref.ID
	}
	return &c, nil
}

func (r *firestoreCycleRepository) Create(ctx context.Context, cycle *model.Cycle) error {
	col := r.fc.ProjectDoc(cycle.ProjectID).Collection("cycles")
	docRef := col.NewDoc()
	if cycle.ID == "" {
		cycle.ID = docRef.ID
	} else {
		docRef = col.Doc(cycle.ID)
	}
	now := time.Now().UTC()
	cycle.CreatedAt = now
	cycle.UpdatedAt = now
	_, err := docRef.Set(ctx, cycle)
	return err
}

func (r *firestoreCycleRepository) Update(ctx context.Context, cycle *model.Cycle) error {
	cycle.UpdatedAt = time.Now().UTC()
	_, err := r.fc.ProjectDoc(cycle.ProjectID).Collection("cycles").Doc(cycle.ID).Set(ctx, cycle)
	return err
}

func (r *firestoreCycleRepository) Delete(ctx context.Context, projectID, cycleID string) error {
	_, err := r.fc.ProjectDoc(projectID).Collection("cycles").Doc(cycleID).Delete(ctx)
	return err
}
