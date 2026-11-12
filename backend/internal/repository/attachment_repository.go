package repository

import (
	"context"
	"errors"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
)

// AttachmentRepository defines persistence operations for project Attachments.
type AttachmentRepository interface {
	List(ctx context.Context, projectID string, itemID string) ([]model.Attachment, error)
	GetByID(ctx context.Context, projectID, attachmentID string) (*model.Attachment, error)
	Create(ctx context.Context, attachment *model.Attachment) error
	Delete(ctx context.Context, projectID, attachmentID string) error
}

type firestoreAttachmentRepository struct {
	fc *FirestoreClient
}

// NewAttachmentRepository creates a Firestore-backed AttachmentRepository.
func NewAttachmentRepository(fc *FirestoreClient) AttachmentRepository {
	return &firestoreAttachmentRepository{fc: fc}
}

func (r *firestoreAttachmentRepository) List(ctx context.Context, projectID string, itemID string) ([]model.Attachment, error) {
	attachments := []model.Attachment{}
	col := r.fc.ProjectDoc(projectID).Collection("attachments")
	var iter *firestore.DocumentIterator
	if itemID != "" {
		iter = col.Where("work_item_id", "==", itemID).Documents(ctx)
	} else {
		iter = col.Documents(ctx)
	}
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var a model.Attachment
		if err := doc.DataTo(&a); err != nil {
			return nil, err
		}
		if a.ID == "" {
			a.ID = doc.Ref.ID
		}
		attachments = append(attachments, a)
	}

	sort.Slice(attachments, func(i, j int) bool {
		return attachments[i].CreatedAt.After(attachments[j].CreatedAt)
	})

	return attachments, nil
}

func (r *firestoreAttachmentRepository) GetByID(ctx context.Context, projectID, attachmentID string) (*model.Attachment, error) {
	snap, err := r.fc.ProjectDoc(projectID).Collection("attachments").Doc(attachmentID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var a model.Attachment
	if err := snap.DataTo(&a); err != nil {
		return nil, err
	}
	if a.ID == "" {
		a.ID = snap.Ref.ID
	}
	return &a, nil
}

func (r *firestoreAttachmentRepository) Create(ctx context.Context, attachment *model.Attachment) error {
	col := r.fc.ProjectDoc(attachment.ProjectID).Collection("attachments")
	docRef := col.NewDoc()
	if attachment.ID == "" {
		attachment.ID = docRef.ID
	} else {
		docRef = col.Doc(attachment.ID)
	}
	now := time.Now().UTC()
	attachment.CreatedAt = now
	attachment.UpdatedAt = now
	_, err := docRef.Set(ctx, attachment)
	return err
}

func (r *firestoreAttachmentRepository) Delete(ctx context.Context, projectID, attachmentID string) error {
	_, err := r.fc.ProjectDoc(projectID).Collection("attachments").Doc(attachmentID).Delete(ctx)
	return err
}
