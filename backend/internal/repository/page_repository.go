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

// PageRepository defines persistence operations for Wiki Pages, PageVersions, and PageCollections.
type PageRepository interface {
	ListPages(ctx context.Context, projectID string, collectionID string) ([]model.Page, error)
	GetPage(ctx context.Context, projectID, pageID string) (*model.Page, error)
	CreatePage(ctx context.Context, page *model.Page) error
	UpdatePage(ctx context.Context, page *model.Page, createVersionSnapshot bool) error
	DeletePage(ctx context.Context, projectID, pageID string) error

	ListVersions(ctx context.Context, projectID, pageID string) ([]model.PageVersion, error)
	GetVersion(ctx context.Context, projectID, pageID, versionID string) (*model.PageVersion, error)

	ListCollections(ctx context.Context, projectID string) ([]model.PageCollection, error)
	GetCollection(ctx context.Context, projectID, collectionID string) (*model.PageCollection, error)
	CreateCollection(ctx context.Context, collection *model.PageCollection) error
	UpdateCollection(ctx context.Context, collection *model.PageCollection) error
	DeleteCollection(ctx context.Context, projectID, collectionID string) error
}

type firestorePageRepository struct {
	fc *FirestoreClient
}

// NewPageRepository creates a Firestore-backed PageRepository.
func NewPageRepository(fc *FirestoreClient) PageRepository {
	return &firestorePageRepository{fc: fc}
}

func (r *firestorePageRepository) ListPages(ctx context.Context, projectID string, collectionID string) ([]model.Page, error) {
	col := r.fc.ProjectDoc(projectID).Collection("pages")
	q := col.Query
	if collectionID != "" {
		q = q.Where("collection_id", "==", collectionID)
	}
	pages := []model.Page{}
	iter := q.Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var p model.Page
		if err := doc.DataTo(&p); err != nil {
			return nil, err
		}
		if p.ID == "" {
			p.ID = doc.Ref.ID
		}
		if p.LabelIDs == nil {
			p.LabelIDs = []string{}
		}
		pages = append(pages, p)
	}

	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Order != pages[j].Order {
			return pages[i].Order < pages[j].Order
		}
		return pages[i].CreatedAt.Before(pages[j].CreatedAt)
	})

	return pages, nil
}

func (r *firestorePageRepository) GetPage(ctx context.Context, projectID, pageID string) (*model.Page, error) {
	snap, err := r.fc.ProjectDoc(projectID).Collection("pages").Doc(pageID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var p model.Page
	if err := snap.DataTo(&p); err != nil {
		return nil, err
	}
	if p.ID == "" {
		p.ID = snap.Ref.ID
	}
	if p.LabelIDs == nil {
		p.LabelIDs = []string{}
	}
	return &p, nil
}

func (r *firestorePageRepository) CreatePage(ctx context.Context, page *model.Page) error {
	col := r.fc.ProjectDoc(page.ProjectID).Collection("pages")
	docRef := col.NewDoc()
	if page.ID == "" {
		page.ID = docRef.ID
	} else {
		docRef = col.Doc(page.ID)
	}
	now := time.Now().UTC()
	page.CreatedAt = now
	page.UpdatedAt = now
	if page.LabelIDs == nil {
		page.LabelIDs = []string{}
	}

	versionRef := docRef.Collection("versions").NewDoc()
	initialVersion := &model.PageVersion{
		ID:        versionRef.ID,
		PageID:    page.ID,
		Title:     page.Title,
		Content:   page.Content,
		CreatedBy: page.CreatedBy,
		CreatedAt: now,
	}

	return r.fc.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if err := tx.Set(docRef, page); err != nil {
			return err
		}
		return tx.Set(versionRef, initialVersion)
	})
}

func (r *firestorePageRepository) UpdatePage(ctx context.Context, page *model.Page, createVersionSnapshot bool) error {
	docRef := r.fc.ProjectDoc(page.ProjectID).Collection("pages").Doc(page.ID)
	now := time.Now().UTC()
	page.UpdatedAt = now
	if page.LabelIDs == nil {
		page.LabelIDs = []string{}
	}

	if !createVersionSnapshot {
		_, err := docRef.Set(ctx, page)
		return err
	}

	versionRef := docRef.Collection("versions").NewDoc()
	ver := &model.PageVersion{
		ID:        versionRef.ID,
		PageID:    page.ID,
		Title:     page.Title,
		Content:   page.Content,
		CreatedBy: page.UpdatedBy,
		CreatedAt: now,
	}

	return r.fc.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if err := tx.Set(docRef, page); err != nil {
			return err
		}
		return tx.Set(versionRef, ver)
	})
}

func (r *firestorePageRepository) DeletePage(ctx context.Context, projectID, pageID string) error {
	_, err := r.fc.ProjectDoc(projectID).Collection("pages").Doc(pageID).Delete(ctx)
	return err
}

func (r *firestorePageRepository) ListVersions(ctx context.Context, projectID, pageID string) ([]model.PageVersion, error) {
	versions := []model.PageVersion{}
	iter := r.fc.ProjectDoc(projectID).Collection("pages").Doc(pageID).Collection("versions").OrderBy("created_at", firestore.Desc).Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var v model.PageVersion
		if err := doc.DataTo(&v); err != nil {
			return nil, err
		}
		if v.ID == "" {
			v.ID = doc.Ref.ID
		}
		versions = append(versions, v)
	}
	return versions, nil
}

func (r *firestorePageRepository) GetVersion(ctx context.Context, projectID, pageID, versionID string) (*model.PageVersion, error) {
	snap, err := r.fc.ProjectDoc(projectID).Collection("pages").Doc(pageID).Collection("versions").Doc(versionID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var v model.PageVersion
	if err := snap.DataTo(&v); err != nil {
		return nil, err
	}
	if v.ID == "" {
		v.ID = snap.Ref.ID
	}
	return &v, nil
}

func (r *firestorePageRepository) ListCollections(ctx context.Context, projectID string) ([]model.PageCollection, error) {
	collections := []model.PageCollection{}
	iter := r.fc.ProjectDoc(projectID).Collection("collections").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var c model.PageCollection
		if err := doc.DataTo(&c); err != nil {
			return nil, err
		}
		if c.ID == "" {
			c.ID = doc.Ref.ID
		}
		collections = append(collections, c)
	}

	sort.Slice(collections, func(i, j int) bool {
		if !collections[i].CreatedAt.Equal(collections[j].CreatedAt) {
			return collections[i].CreatedAt.Before(collections[j].CreatedAt)
		}
		return collections[i].ID < collections[j].ID
	})

	return collections, nil
}

func (r *firestorePageRepository) GetCollection(ctx context.Context, projectID, collectionID string) (*model.PageCollection, error) {
	snap, err := r.fc.ProjectDoc(projectID).Collection("collections").Doc(collectionID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var c model.PageCollection
	if err := snap.DataTo(&c); err != nil {
		return nil, err
	}
	if c.ID == "" {
		c.ID = snap.Ref.ID
	}
	return &c, nil
}

func (r *firestorePageRepository) CreateCollection(ctx context.Context, collection *model.PageCollection) error {
	col := r.fc.ProjectDoc(collection.ProjectID).Collection("collections")
	docRef := col.NewDoc()
	if collection.ID == "" {
		collection.ID = docRef.ID
	} else {
		docRef = col.Doc(collection.ID)
	}
	now := time.Now().UTC()
	collection.CreatedAt = now
	collection.UpdatedAt = now
	_, err := docRef.Set(ctx, collection)
	return err
}

func (r *firestorePageRepository) UpdateCollection(ctx context.Context, collection *model.PageCollection) error {
	collection.UpdatedAt = time.Now().UTC()
	_, err := r.fc.ProjectDoc(collection.ProjectID).Collection("collections").Doc(collection.ID).Set(ctx, collection)
	return err
}

func (r *firestorePageRepository) DeleteCollection(ctx context.Context, projectID, collectionID string) error {
	_, err := r.fc.ProjectDoc(projectID).Collection("collections").Doc(collectionID).Delete(ctx)
	return err
}
