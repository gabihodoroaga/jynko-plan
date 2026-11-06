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

// ModuleRepository defines persistence operations for project Modules.
type ModuleRepository interface {
	List(ctx context.Context, projectID string) ([]model.Module, error)
	GetByID(ctx context.Context, projectID, moduleID string) (*model.Module, error)
	Create(ctx context.Context, module *model.Module) error
	Update(ctx context.Context, module *model.Module) error
	Delete(ctx context.Context, projectID, moduleID string) error
}

type firestoreModuleRepository struct {
	fc *FirestoreClient
}

// NewModuleRepository creates a Firestore-backed ModuleRepository.
func NewModuleRepository(fc *FirestoreClient) ModuleRepository {
	return &firestoreModuleRepository{fc: fc}
}

func (r *firestoreModuleRepository) List(ctx context.Context, projectID string) ([]model.Module, error) {
	modules := []model.Module{}
	iter := r.fc.ProjectDoc(projectID).Collection("modules").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var m model.Module
		if err := doc.DataTo(&m); err != nil {
			return nil, err
		}
		if m.ID == "" {
			m.ID = doc.Ref.ID
		}
		modules = append(modules, m)
	}
	return modules, nil
}

func (r *firestoreModuleRepository) GetByID(ctx context.Context, projectID, moduleID string) (*model.Module, error) {
	snap, err := r.fc.ProjectDoc(projectID).Collection("modules").Doc(moduleID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var m model.Module
	if err := snap.DataTo(&m); err != nil {
		return nil, err
	}
	if m.ID == "" {
		m.ID = snap.Ref.ID
	}
	return &m, nil
}

func (r *firestoreModuleRepository) Create(ctx context.Context, module *model.Module) error {
	col := r.fc.ProjectDoc(module.ProjectID).Collection("modules")
	docRef := col.NewDoc()
	if module.ID == "" {
		module.ID = docRef.ID
	} else {
		docRef = col.Doc(module.ID)
	}
	now := time.Now().UTC()
	module.CreatedAt = now
	module.UpdatedAt = now
	_, err := docRef.Set(ctx, module)
	return err
}

func (r *firestoreModuleRepository) Update(ctx context.Context, module *model.Module) error {
	module.UpdatedAt = time.Now().UTC()
	_, err := r.fc.ProjectDoc(module.ProjectID).Collection("modules").Doc(module.ID).Set(ctx, module)
	return err
}

func (r *firestoreModuleRepository) Delete(ctx context.Context, projectID, moduleID string) error {
	_, err := r.fc.ProjectDoc(projectID).Collection("modules").Doc(moduleID).Delete(ctx)
	return err
}
