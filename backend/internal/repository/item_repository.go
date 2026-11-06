package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
)

// ItemRepository defines persistence operations for WorkItems, SubWorkItems, and Comments.
type ItemRepository interface {
	List(ctx context.Context, projectID string, filter model.WorkItemFilter) ([]model.WorkItem, error)
	GetByID(ctx context.Context, projectID, itemID string) (*model.WorkItem, error)
	Create(ctx context.Context, item *model.WorkItem) error
	Update(ctx context.Context, item *model.WorkItem) error
	Delete(ctx context.Context, projectID, itemID string) error

	ListSubItems(ctx context.Context, projectID, itemID string) ([]model.SubWorkItem, error)
	GetSubItemByID(ctx context.Context, projectID, itemID, subItemID string) (*model.SubWorkItem, error)
	CreateSubItem(ctx context.Context, projectID string, subItem *model.SubWorkItem) error
	UpdateSubItem(ctx context.Context, projectID string, subItem *model.SubWorkItem) error
	DeleteSubItem(ctx context.Context, projectID, itemID, subItemID string) error

	ListComments(ctx context.Context, projectID, itemID string) ([]model.Comment, error)
	CreateComment(ctx context.Context, projectID string, comment *model.Comment) error
	DeleteComment(ctx context.Context, projectID, itemID, commentID string) error
}

type firestoreItemRepository struct {
	fc *FirestoreClient
}

// NewItemRepository creates a Firestore-backed ItemRepository.
func NewItemRepository(fc *FirestoreClient) ItemRepository {
	return &firestoreItemRepository{fc: fc}
}

func (r *firestoreItemRepository) List(ctx context.Context, projectID string, filter model.WorkItemFilter) ([]model.WorkItem, error) {
	col := r.fc.ProjectDoc(projectID).Collection("work_items")
	q := col.Query
	if filter.Status != "" {
		q = q.Where("status", "==", filter.Status)
	}
	if filter.Priority != "" {
		q = q.Where("priority", "==", filter.Priority)
	}
	if filter.ModuleID != "" {
		q = q.Where("module_id", "==", filter.ModuleID)
	}
	if filter.EpicID != "" {
		q = q.Where("epic_id", "==", filter.EpicID)
	}
	if filter.CycleID != "" {
		q = q.Where("cycle_id", "==", filter.CycleID)
	}
	if filter.AssigneeID != "" {
		q = q.Where("assignee_id", "==", filter.AssigneeID)
	}
	if filter.LabelID != "" {
		q = q.Where("label_ids", "array-contains", filter.LabelID)
	}

	items := []model.WorkItem{}
	iter := q.Documents(ctx)
	defer iter.Stop()

	searchLower := strings.ToLower(strings.TrimSpace(filter.Search))
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var item model.WorkItem
		if err := doc.DataTo(&item); err != nil {
			return nil, err
		}
		if item.ID == "" {
			item.ID = doc.Ref.ID
		}
		if item.LabelIDs == nil {
			item.LabelIDs = []string{}
		}
		if searchLower != "" {
			if !strings.Contains(strings.ToLower(item.Title), searchLower) &&
				!strings.Contains(strings.ToLower(item.Description), searchLower) &&
				!strings.Contains(strings.ToLower(item.Key), searchLower) {
				continue
			}
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].SequenceID != items[j].SequenceID {
			return items[i].SequenceID < items[j].SequenceID
		}
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})

	return items, nil
}

func (r *firestoreItemRepository) GetByID(ctx context.Context, projectID, itemID string) (*model.WorkItem, error) {
	snap, err := r.fc.ProjectDoc(projectID).Collection("work_items").Doc(itemID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var item model.WorkItem
	if err := snap.DataTo(&item); err != nil {
		return nil, err
	}
	if item.ID == "" {
		item.ID = snap.Ref.ID
	}
	if item.LabelIDs == nil {
		item.LabelIDs = []string{}
	}
	return &item, nil
}

func (r *firestoreItemRepository) Create(ctx context.Context, item *model.WorkItem) error {
	projRef := r.fc.ProjectDoc(item.ProjectID)
	col := projRef.Collection("work_items")
	itemRef := col.NewDoc()
	if item.ID == "" {
		item.ID = itemRef.ID
	} else {
		itemRef = col.Doc(item.ID)
	}

	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	if item.LabelIDs == nil {
		item.LabelIDs = []string{}
	}

	return r.fc.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		projSnap, err := tx.Get(projRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var proj model.Project
		if err := projSnap.DataTo(&proj); err != nil {
			return err
		}

		proj.ItemSeq++
		prefix := proj.KeyPrefix
		if prefix == "" {
			prefix = "JYNKO"
		}
		item.SequenceID = proj.ItemSeq
		item.Key = fmt.Sprintf("%s-%d", prefix, proj.ItemSeq)
		proj.UpdatedAt = now

		if err := tx.Set(projRef, proj); err != nil {
			return err
		}
		return tx.Set(itemRef, item)
	})
}

func (r *firestoreItemRepository) Update(ctx context.Context, item *model.WorkItem) error {
	item.UpdatedAt = time.Now().UTC()
	if item.LabelIDs == nil {
		item.LabelIDs = []string{}
	}
	_, err := r.fc.ProjectDoc(item.ProjectID).Collection("work_items").Doc(item.ID).Set(ctx, item)
	return err
}

func (r *firestoreItemRepository) Delete(ctx context.Context, projectID, itemID string) error {
	_, err := r.fc.ProjectDoc(projectID).Collection("work_items").Doc(itemID).Delete(ctx)
	return err
}

func (r *firestoreItemRepository) ListSubItems(ctx context.Context, projectID, itemID string) ([]model.SubWorkItem, error) {
	subItems := []model.SubWorkItem{}
	iter := r.fc.ProjectDoc(projectID).Collection("work_items").Doc(itemID).Collection("sub_items").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var sub model.SubWorkItem
		if err := doc.DataTo(&sub); err != nil {
			return nil, err
		}
		if sub.ID == "" {
			sub.ID = doc.Ref.ID
		}
		subItems = append(subItems, sub)
	}
	return subItems, nil
}

func (r *firestoreItemRepository) GetSubItemByID(ctx context.Context, projectID, itemID, subItemID string) (*model.SubWorkItem, error) {
	snap, err := r.fc.ProjectDoc(projectID).Collection("work_items").Doc(itemID).Collection("sub_items").Doc(subItemID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var sub model.SubWorkItem
	if err := snap.DataTo(&sub); err != nil {
		return nil, err
	}
	if sub.ID == "" {
		sub.ID = snap.Ref.ID
	}
	return &sub, nil
}

func (r *firestoreItemRepository) CreateSubItem(ctx context.Context, projectID string, subItem *model.SubWorkItem) error {
	itemRef := r.fc.ProjectDoc(projectID).Collection("work_items").Doc(subItem.ParentItemID)
	subCol := itemRef.Collection("sub_items")
	subRef := subCol.NewDoc()
	if subItem.ID == "" {
		subItem.ID = subRef.ID
	} else {
		subRef = subCol.Doc(subItem.ID)
	}

	now := time.Now().UTC()
	subItem.CreatedAt = now
	subItem.UpdatedAt = now

	return r.fc.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(itemRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var item model.WorkItem
		if err := snap.DataTo(&item); err != nil {
			return err
		}
		item.SubItemCount++
		if subItem.IsCompleted {
			item.CompletedSubCount++
		}
		item.UpdatedAt = now
		if err := tx.Set(itemRef, item); err != nil {
			return err
		}
		return tx.Set(subRef, subItem)
	})
}

func (r *firestoreItemRepository) UpdateSubItem(ctx context.Context, projectID string, subItem *model.SubWorkItem) error {
	itemRef := r.fc.ProjectDoc(projectID).Collection("work_items").Doc(subItem.ParentItemID)
	subRef := itemRef.Collection("sub_items").Doc(subItem.ID)
	now := time.Now().UTC()

	return r.fc.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		oldSnap, err := tx.Get(subRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var oldSub model.SubWorkItem
		if err := oldSnap.DataTo(&oldSub); err != nil {
			return err
		}

		itemSnap, err := tx.Get(itemRef)
		if err == nil {
			var item model.WorkItem
			if err := itemSnap.DataTo(&item); err == nil {
				if !oldSub.IsCompleted && subItem.IsCompleted {
					item.CompletedSubCount++
				} else if oldSub.IsCompleted && !subItem.IsCompleted && item.CompletedSubCount > 0 {
					item.CompletedSubCount--
				}
				item.UpdatedAt = now
				_ = tx.Set(itemRef, item)
			}
		}

		subItem.UpdatedAt = now
		return tx.Set(subRef, subItem)
	})
}

func (r *firestoreItemRepository) DeleteSubItem(ctx context.Context, projectID, itemID, subItemID string) error {
	itemRef := r.fc.ProjectDoc(projectID).Collection("work_items").Doc(itemID)
	subRef := itemRef.Collection("sub_items").Doc(subItemID)
	now := time.Now().UTC()

	return r.fc.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		oldSnap, err := tx.Get(subRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var oldSub model.SubWorkItem
		_ = oldSnap.DataTo(&oldSub)

		if itemSnap, err := tx.Get(itemRef); err == nil {
			var item model.WorkItem
			if err := itemSnap.DataTo(&item); err == nil {
				if item.SubItemCount > 0 {
					item.SubItemCount--
				}
				if oldSub.IsCompleted && item.CompletedSubCount > 0 {
					item.CompletedSubCount--
				}
				item.UpdatedAt = now
				_ = tx.Set(itemRef, item)
			}
		}
		return tx.Delete(subRef)
	})
}

func (r *firestoreItemRepository) ListComments(ctx context.Context, projectID, itemID string) ([]model.Comment, error) {
	comments := []model.Comment{}
	iter := r.fc.ProjectDoc(projectID).Collection("work_items").Doc(itemID).Collection("comments").OrderBy("created_at", firestore.Asc).Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var c model.Comment
		if err := doc.DataTo(&c); err != nil {
			return nil, err
		}
		if c.ID == "" {
			c.ID = doc.Ref.ID
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func (r *firestoreItemRepository) CreateComment(ctx context.Context, projectID string, comment *model.Comment) error {
	col := r.fc.ProjectDoc(projectID).Collection("work_items").Doc(comment.WorkItemID).Collection("comments")
	docRef := col.NewDoc()
	if comment.ID == "" {
		comment.ID = docRef.ID
	} else {
		docRef = col.Doc(comment.ID)
	}
	now := time.Now().UTC()
	comment.CreatedAt = now
	comment.UpdatedAt = now
	_, err := docRef.Set(ctx, comment)
	return err
}

func (r *firestoreItemRepository) DeleteComment(ctx context.Context, projectID, itemID, commentID string) error {
	_, err := r.fc.ProjectDoc(projectID).Collection("work_items").Doc(itemID).Collection("comments").Doc(commentID).Delete(ctx)
	return err
}
