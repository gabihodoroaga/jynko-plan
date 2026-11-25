package repository

import (
	"context"
	"errors"
	"io"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
)

var (
	// ErrNotFound is returned when a requested Firestore document does not exist.
	ErrNotFound = errors.New("resource not found")
	// ErrLastAdmin is returned when attempting to remove or demote the sole ADMIN of a project.
	ErrLastAdmin = errors.New("cannot remove or demote the last project ADMIN")
	// ErrConflictLocked is returned when attempting to edit a locked resource inside a transaction.
	ErrConflictLocked = errors.New("resource is locked")
)

// BlobDeleter is satisfied by storage.StorageProvider to clean up attachment blobs without circular imports.
type BlobDeleter interface {
	Delete(ctx context.Context, storagePath string) error
	Open(ctx context.Context, storagePath string) (io.ReadCloser, error)
}

// ProjectRepository defines persistence operations for Projects and ProjectMembers.
type ProjectRepository interface {
	Create(ctx context.Context, project *model.Project, initialAdmin *model.ProjectMember) error
	GetByID(ctx context.Context, projectID string) (*model.Project, error)
	ListByUser(ctx context.Context, userID string, userEmail string) ([]model.Project, error)
	Update(ctx context.Context, project *model.Project) error
	Delete(ctx context.Context, projectID string) error
	// SetDeleted soft-deletes (deletedAt != nil) or restores (deletedAt == nil) a project (PLAN-13).
	SetDeleted(ctx context.Context, projectID string, deletedAt *time.Time, deletedBy string) error
	// ListDeletedBefore returns soft-deleted projects whose deleted_at is at or before cutoff.
	ListDeletedBefore(ctx context.Context, cutoff time.Time) ([]model.Project, error)

	GetMember(ctx context.Context, projectID, userID string) (*model.ProjectMember, error)
	GetMemberByEmail(ctx context.Context, projectID, email string) (*model.ProjectMember, error)
	ListMembers(ctx context.Context, projectID string) ([]model.ProjectMember, error)
	UpsertMember(ctx context.Context, projectID string, member *model.ProjectMember) error
	RemoveMember(ctx context.Context, projectID, userID string) error
}

type firestoreProjectRepository struct {
	fc      *FirestoreClient
	blobs   BlobDeleter
}

// NewProjectRepository creates a Firestore-backed ProjectRepository.
func NewProjectRepository(fc *FirestoreClient) ProjectRepository {
	return &firestoreProjectRepository{fc: fc}
}

// NewProjectRepositoryWithBlobs creates a Firestore-backed ProjectRepository that also cleans up attachment blobs on project deletion.
func NewProjectRepositoryWithBlobs(fc *FirestoreClient, blobs BlobDeleter) ProjectRepository {
	return &firestoreProjectRepository{fc: fc, blobs: blobs}
}

func (r *firestoreProjectRepository) Create(ctx context.Context, project *model.Project, initialAdmin *model.ProjectMember) error {
	projRef := r.fc.Projects().NewDoc()
	if project.ID == "" {
		project.ID = projRef.ID
	} else {
		projRef = r.fc.ProjectDoc(project.ID)
	}

	now := time.Now().UTC()
	project.CreatedAt = now
	project.UpdatedAt = now
	if project.MemberIDs == nil {
		project.MemberIDs = []string{}
	}
	if initialAdmin != nil {
		initialAdmin.Email = strings.ToLower(strings.TrimSpace(initialAdmin.Email))
		if initialAdmin.UserID != "" && !containsString(project.MemberIDs, initialAdmin.UserID) {
			project.MemberIDs = append(project.MemberIDs, initialAdmin.UserID)
		}
		if initialAdmin.Email != "" && !containsString(project.MemberIDs, initialAdmin.Email) {
			project.MemberIDs = append(project.MemberIDs, initialAdmin.Email)
		}
	}

	return r.fc.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if err := tx.Set(projRef, project); err != nil {
			return err
		}
		if initialAdmin != nil {
			initialAdmin.JoinedAt = now
			memberRef := projRef.Collection("members").Doc(initialAdmin.UserID)
			if err := tx.Set(memberRef, initialAdmin); err != nil {
				return err
			}
		}

		// Automatically create a corresponding Wiki PageCollection & starter Page for the new project
		colID := "col-" + project.ID
		colRef := projRef.Collection("collections").Doc(colID)
		projectCollection := &model.PageCollection{
			ID:          colID,
			ProjectID:   project.ID,
			Name:        project.Name,
			Description: project.Description,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := tx.Set(colRef, projectCollection); err != nil {
			return err
		}

		pageID := "page-" + project.ID + "-home"
		pageRef := projRef.Collection("pages").Doc(pageID)
		starterPage := &model.Page{
			ID:           pageID,
			ProjectID:    project.ID,
			CollectionID: colID,
			Title:        project.Name + " — Project Overview",
			Content:      "# " + project.Name + " — Project Overview\n\nWelcome to the **" + project.Name + "** project knowledge base collection.\n\nUse this space to document architecture decisions, specifications, and runbooks for `" + project.KeyPrefix + "`.",
			Order:        1,
			IsLocked:     false,
			LabelIDs:     []string{},
			CreatedBy:    project.CreatedBy,
			UpdatedBy:    project.CreatedBy,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Set(pageRef, starterPage); err != nil {
			return err
		}

		verRef := pageRef.Collection("versions").NewDoc()
		initialVer := &model.PageVersion{
			ID:        verRef.ID,
			PageID:    pageID,
			Title:     starterPage.Title,
			Content:   starterPage.Content,
			CreatedBy: project.CreatedBy,
			CreatedAt: now,
		}
		return tx.Set(verRef, initialVer)
	})
}

func (r *firestoreProjectRepository) GetByID(ctx context.Context, projectID string) (*model.Project, error) {
	if strings.TrimSpace(projectID) == "" || strings.Contains(projectID, "/") {
		return nil, ErrNotFound
	}
	snap, err := r.fc.ProjectDoc(projectID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var proj model.Project
	if err := snap.DataTo(&proj); err != nil {
		return nil, err
	}
	if proj.ID == "" {
		proj.ID = snap.Ref.ID
	}
	if proj.MemberIDs == nil {
		proj.MemberIDs = []string{}
	}
	if proj.Settings == nil {
		proj.Settings = model.DefaultProjectSettings()
	}
	return &proj, nil
}

func (r *firestoreProjectRepository) ListByUser(ctx context.Context, userID string, userEmail string) ([]model.Project, error) {
	return r.queryProjectsByUser(ctx, userID, userEmail)
}

func (r *firestoreProjectRepository) queryProjectsByUser(ctx context.Context, userID string, userEmail string) ([]model.Project, error) {
	projects := []model.Project{}
	seen := make(map[string]bool)

	if userID != "" {
		iter := r.fc.Projects().Where("member_ids", "array-contains", userID).Documents(ctx)
		defer iter.Stop()
		for {
			doc, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return nil, err
			}
			var p model.Project
			if err := doc.DataTo(&p); err != nil {
				return nil, err
			}
			if p.ID == "" {
				p.ID = doc.Ref.ID
			}
			if p.MemberIDs == nil {
				p.MemberIDs = []string{}
			}
			if p.Settings == nil {
				p.Settings = model.DefaultProjectSettings()
			}
			seen[p.ID] = true
			projects = append(projects, p)
		}
	}

	normalizedEmail := strings.ToLower(strings.TrimSpace(userEmail))
	if normalizedEmail != "" && normalizedEmail != userID {
		iter := r.fc.Projects().Where("member_ids", "array-contains", normalizedEmail).Documents(ctx)
		defer iter.Stop()
		for {
			doc, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return nil, err
			}
			var p model.Project
			if err := doc.DataTo(&p); err != nil {
				return nil, err
			}
			if p.ID == "" {
				p.ID = doc.Ref.ID
			}
			if !seen[p.ID] {
				if p.MemberIDs == nil {
					p.MemberIDs = []string{}
				}
				if p.Settings == nil {
					p.Settings = model.DefaultProjectSettings()
				}
				seen[p.ID] = true
				projects = append(projects, p)
			}
		}
	}

	sort.Slice(projects, func(i, j int) bool {
		if !projects[i].CreatedAt.Equal(projects[j].CreatedAt) {
			return projects[i].CreatedAt.Before(projects[j].CreatedAt)
		}
		return projects[i].ID < projects[j].ID
	})

	return projects, nil
}

// Update atomically updates mutable project metadata/settings while preserving live ItemSeq and MemberIDs.
func (r *firestoreProjectRepository) Update(ctx context.Context, project *model.Project) error {
	projRef := r.fc.ProjectDoc(project.ID)
	now := time.Now().UTC()

	return r.fc.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(projRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var live model.Project
		if err := snap.DataTo(&live); err != nil {
			return err
		}
		if live.ID == "" {
			live.ID = snap.Ref.ID
		}

		live.Name = project.Name
		live.KeyPrefix = project.KeyPrefix
		live.Description = project.Description
		live.Settings = project.Settings
		live.UpdatedAt = now
		if live.MemberIDs == nil {
			live.MemberIDs = []string{}
		}

		// Sync live authoritative counters back to caller struct
		project.ItemSeq = live.ItemSeq
		project.MemberIDs = live.MemberIDs
		project.CreatedAt = live.CreatedAt
		project.CreatedBy = live.CreatedBy
		project.UpdatedAt = now

		return tx.Set(projRef, &live)
	})
}

// SetDeleted marks a project as deleted (or clears the marker when deletedAt is nil) without touching
// its data, so it can later be restored or permanently purged.
func (r *firestoreProjectRepository) SetDeleted(ctx context.Context, projectID string, deletedAt *time.Time, deletedBy string) error {
	if strings.TrimSpace(projectID) == "" || strings.Contains(projectID, "/") {
		return ErrNotFound
	}
	projRef := r.fc.ProjectDoc(projectID)
	now := time.Now().UTC()
	return r.fc.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, err := tx.Get(projRef); err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		updates := []firestore.Update{{Path: "updated_at", Value: now}}
		if deletedAt != nil {
			at := deletedAt.UTC()
			updates = append(updates,
				firestore.Update{Path: "deleted_at", Value: at},
				firestore.Update{Path: "deleted_by", Value: deletedBy},
			)
		} else {
			updates = append(updates,
				firestore.Update{Path: "deleted_at", Value: firestore.Delete},
				firestore.Update{Path: "deleted_by", Value: firestore.Delete},
			)
		}
		return tx.Update(projRef, updates)
	})
}

// ListDeletedBefore returns all soft-deleted projects deleted at or before cutoff (single-field index on
// deleted_at; projects without the field are never matched).
func (r *firestoreProjectRepository) ListDeletedBefore(ctx context.Context, cutoff time.Time) ([]model.Project, error) {
	projects := []model.Project{}
	iter := r.fc.Projects().Where("deleted_at", "<=", cutoff.UTC()).Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var p model.Project
		if err := doc.DataTo(&p); err != nil {
			return nil, err
		}
		if p.ID == "" {
			p.ID = doc.Ref.ID
		}
		if p.MemberIDs == nil {
			p.MemberIDs = []string{}
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (r *firestoreProjectRepository) Delete(ctx context.Context, projectID string) error {
	projRef := r.fc.ProjectDoc(projectID)
	bw := r.fc.Client.BulkWriter(ctx)

	// 1. Delete nested subcollections under work_items (sub_items, comments)
	wiIter := projRef.Collection("work_items").Documents(ctx)
	for {
		wiDoc, err := wiIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			wiIter.Stop()
			return err
		}
		for _, nested := range []string{"sub_items", "comments"} {
			if err := enqueueCollectionDeletes(ctx, bw, wiDoc.Ref.Collection(nested)); err != nil {
				wiIter.Stop()
				return err
			}
		}
		_, _ = bw.Delete(wiDoc.Ref)
	}
	wiIter.Stop()

	// 2. Delete nested versions under pages
	pgIter := projRef.Collection("pages").Documents(ctx)
	for {
		pgDoc, err := pgIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			pgIter.Stop()
			return err
		}
		if err := enqueueCollectionDeletes(ctx, bw, pgDoc.Ref.Collection("versions")); err != nil {
			pgIter.Stop()
			return err
		}
		_, _ = bw.Delete(pgDoc.Ref)
	}
	pgIter.Stop()

	// 3. Delete attachments and underlying storage blobs
	attIter := projRef.Collection("attachments").Documents(ctx)
	for {
		attDoc, err := attIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			attIter.Stop()
			return err
		}
		if r.blobs != nil {
			var att model.Attachment
			if err := attDoc.DataTo(&att); err == nil && att.StoragePath != "" {
				_ = r.blobs.Delete(ctx, att.StoragePath)
			}
		}
		_, _ = bw.Delete(attDoc.Ref)
	}
	attIter.Stop()

	// 4. Delete remaining flat subcollections
	for _, sub := range []string{"members", "modules", "epics", "cycles", "labels", "collections", "stickies"} {
		if err := enqueueCollectionDeletes(ctx, bw, projRef.Collection(sub)); err != nil {
			return err
		}
	}

	_, _ = bw.Delete(projRef)
	bw.End()
	return nil
}

func enqueueCollectionDeletes(ctx context.Context, bw *firestore.BulkWriter, col *firestore.CollectionRef) error {
	iter := col.Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			return nil
		}
		if err != nil {
			return err
		}
		_, _ = bw.Delete(doc.Ref)
	}
}

func (r *firestoreProjectRepository) GetMember(ctx context.Context, projectID, userID string) (*model.ProjectMember, error) {
	if strings.TrimSpace(userID) == "" || strings.Contains(userID, "/") {
		return nil, ErrNotFound
	}
	snap, err := r.fc.ProjectDoc(projectID).Collection("members").Doc(userID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var m model.ProjectMember
	if err := snap.DataTo(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *firestoreProjectRepository) GetMemberByEmail(ctx context.Context, projectID, email string) (*model.ProjectMember, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return nil, ErrNotFound
	}
	iter := r.fc.ProjectDoc(projectID).Collection("members").Where("email", "==", normalized).Limit(1).Documents(ctx)
	defer iter.Stop()
	doc, err := iter.Next()
	if errors.Is(err, iterator.Done) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var m model.ProjectMember
	if err := doc.DataTo(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *firestoreProjectRepository) ListMembers(ctx context.Context, projectID string) ([]model.ProjectMember, error) {
	members := []model.ProjectMember{}
	iter := r.fc.ProjectDoc(projectID).Collection("members").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var m model.ProjectMember
		if err := doc.DataTo(&m); err != nil {
			return nil, err
		}
		if m.UserID == "" {
			m.UserID = doc.Ref.ID
		}
		members = append(members, m)
	}
	return members, nil
}

// UpsertMember atomically adds/updates a member, deduplicates prior email-invited placeholder records,
// prevents demoting the sole project ADMIN, and rebuilds Project.MemberIDs cleanly.
func (r *firestoreProjectRepository) UpsertMember(ctx context.Context, projectID string, member *model.ProjectMember) error {
	projRef := r.fc.ProjectDoc(projectID)
	membersCol := projRef.Collection("members")
	memberRef := membersCol.Doc(member.UserID)

	now := time.Now().UTC()
	if member.JoinedAt.IsZero() {
		member.JoinedAt = now
	}
	member.Email = strings.ToLower(strings.TrimSpace(member.Email))

	return r.fc.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(projRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var proj model.Project
		if err := snap.DataTo(&proj); err != nil {
			return err
		}

		// Read existing members inside transaction to deduplicate by email and enforce >= 1 ADMIN
		iter := tx.Documents(membersCol)
		defer iter.Stop()

		var activeMembers []model.ProjectMember
		var staleRefs []*firestore.DocumentRef
		for {
			doc, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return err
			}
			var existing model.ProjectMember
			if err := doc.DataTo(&existing); err != nil {
				return err
			}
			if existing.UserID == "" {
				existing.UserID = doc.Ref.ID
			}
			existingEmail := strings.ToLower(strings.TrimSpace(existing.Email))

			if doc.Ref.ID == member.UserID || existing.UserID == member.UserID {
				if !existing.JoinedAt.IsZero() {
					member.JoinedAt = existing.JoinedAt
				}
				continue
			}
			// Deduplicate email-invited record when real IAP numeric UserID is upserted/reconciled
			if member.Email != "" && existingEmail == member.Email {
				if !existing.JoinedAt.IsZero() {
					member.JoinedAt = existing.JoinedAt
				}
				staleRefs = append(staleRefs, doc.Ref)
				continue
			}
			activeMembers = append(activeMembers, existing)
		}

		activeMembers = append(activeMembers, *member)

		adminCount := 0
		for _, m := range activeMembers {
			if m.Role == model.RoleAdmin {
				adminCount++
			}
		}
		if adminCount == 0 {
			return ErrLastAdmin
		}

		proj.MemberIDs = buildMemberIDsSlice(activeMembers)
		proj.UpdatedAt = now

		if err := tx.Set(projRef, &proj); err != nil {
			return err
		}
		for _, staleRef := range staleRefs {
			if err := tx.Delete(staleRef); err != nil {
				return err
			}
		}
		return tx.Set(memberRef, member)
	})
}

// RemoveMember atomically deletes all member records matching userID (by UserID or Email),
// prevents removing the sole project ADMIN, and rebuilds Project.MemberIDs cleanly.
func (r *firestoreProjectRepository) RemoveMember(ctx context.Context, projectID, userID string) error {
	projRef := r.fc.ProjectDoc(projectID)
	membersCol := projRef.Collection("members")
	targetLower := strings.ToLower(strings.TrimSpace(userID))

	return r.fc.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(projRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return ErrNotFound
			}
			return err
		}
		var proj model.Project
		if err := snap.DataTo(&proj); err != nil {
			return err
		}

		iter := tx.Documents(membersCol)
		defer iter.Stop()

		var remaining []model.ProjectMember
		var deleteRefs []*firestore.DocumentRef
		for {
			doc, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				return err
			}
			var existing model.ProjectMember
			if err := doc.DataTo(&existing); err != nil {
				return err
			}
			if existing.UserID == "" {
				existing.UserID = doc.Ref.ID
			}
			existingEmail := strings.ToLower(strings.TrimSpace(existing.Email))

			if doc.Ref.ID == userID || existing.UserID == userID || (targetLower != "" && existingEmail == targetLower) {
				deleteRefs = append(deleteRefs, doc.Ref)
				continue
			}
			remaining = append(remaining, existing)
		}

		if len(deleteRefs) == 0 {
			return ErrNotFound
		}

		adminCount := 0
		for _, m := range remaining {
			if m.Role == model.RoleAdmin {
				adminCount++
			}
		}
		if adminCount == 0 {
			return ErrLastAdmin
		}

		proj.MemberIDs = buildMemberIDsSlice(remaining)
		proj.UpdatedAt = time.Now().UTC()

		if err := tx.Set(projRef, &proj); err != nil {
			return err
		}
		for _, ref := range deleteRefs {
			if err := tx.Delete(ref); err != nil {
				return err
			}
		}
		return nil
	})
}

func buildMemberIDsSlice(members []model.ProjectMember) []string {
	out := make([]string, 0, len(members)*2)
	for _, m := range members {
		uid := strings.TrimSpace(m.UserID)
		if uid != "" && !containsString(out, uid) {
			out = append(out, uid)
		}
		email := strings.ToLower(strings.TrimSpace(m.Email))
		if email != "" && !containsString(out, email) {
			out = append(out, email)
		}
	}
	return out
}

func containsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}
