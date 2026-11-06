package repository

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"jynko/internal/model"
)

// ErrNotFound is returned when a requested Firestore document does not exist.
var ErrNotFound = errors.New("resource not found")

// ProjectRepository defines persistence operations for Projects and ProjectMembers.
type ProjectRepository interface {
	Create(ctx context.Context, project *model.Project, initialAdmin *model.ProjectMember) error
	GetByID(ctx context.Context, projectID string) (*model.Project, error)
	ListByUser(ctx context.Context, userID string, userEmail string) ([]model.Project, error)
	Update(ctx context.Context, project *model.Project) error
	Delete(ctx context.Context, projectID string) error

	GetMember(ctx context.Context, projectID, userID string) (*model.ProjectMember, error)
	GetMemberByEmail(ctx context.Context, projectID, email string) (*model.ProjectMember, error)
	ListMembers(ctx context.Context, projectID string) ([]model.ProjectMember, error)
	UpsertMember(ctx context.Context, projectID string, member *model.ProjectMember) error
	RemoveMember(ctx context.Context, projectID, userID string) error
}

type firestoreProjectRepository struct {
	fc *FirestoreClient
}

// NewProjectRepository creates a Firestore-backed ProjectRepository.
func NewProjectRepository(fc *FirestoreClient) ProjectRepository {
	return &firestoreProjectRepository{fc: fc}
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
	if initialAdmin != nil && !containsString(project.MemberIDs, initialAdmin.UserID) {
		project.MemberIDs = append(project.MemberIDs, initialAdmin.UserID)
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
	projects, err := r.queryProjectsByUser(ctx, userID, userEmail)
	if err != nil {
		return nil, err
	}

	if len(projects) == 0 && userID == "dev-user-1" {
		if seedErr := EnsureSeedData(ctx, r.fc); seedErr == nil {
			projects, err = r.queryProjectsByUser(ctx, userID, userEmail)
			if err != nil {
				return nil, err
			}
		}
	}

	return projects, nil
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

	// Also check if user was added by email (where user_id in member_ids might be their email)
	if userEmail != "" && userEmail != userID {
		iter := r.fc.Projects().Where("member_ids", "array-contains", userEmail).Documents(ctx)
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

func (r *firestoreProjectRepository) Update(ctx context.Context, project *model.Project) error {
	project.UpdatedAt = time.Now().UTC()
	_, err := r.fc.ProjectDoc(project.ID).Set(ctx, project)
	return err
}

func (r *firestoreProjectRepository) Delete(ctx context.Context, projectID string) error {
	projRef := r.fc.ProjectDoc(projectID)
	subcols := []string{"members", "modules", "epics", "cycles", "labels", "work_items", "collections", "pages", "stickies"}
	for _, sub := range subcols {
		iter := projRef.Collection(sub).Documents(ctx)
		for {
			doc, err := iter.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				break
			}
			_, _ = doc.Ref.Delete(ctx)
		}
		iter.Stop()
	}
	_, err := projRef.Delete(ctx)
	return err
}

func (r *firestoreProjectRepository) GetMember(ctx context.Context, projectID, userID string) (*model.ProjectMember, error) {
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
	if email == "" {
		return nil, ErrNotFound
	}
	iter := r.fc.ProjectDoc(projectID).Collection("members").Where("email", "==", strings.ToLower(strings.TrimSpace(email))).Limit(1).Documents(ctx)
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
		members = append(members, m)
	}
	return members, nil
}

func (r *firestoreProjectRepository) UpsertMember(ctx context.Context, projectID string, member *model.ProjectMember) error {
	projRef := r.fc.ProjectDoc(projectID)
	memberRef := projRef.Collection("members").Doc(member.UserID)

	if member.JoinedAt.IsZero() {
		member.JoinedAt = time.Now().UTC()
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
		if !containsString(proj.MemberIDs, member.UserID) {
			proj.MemberIDs = append(proj.MemberIDs, member.UserID)
		}
		if member.Email != "" && !containsString(proj.MemberIDs, member.Email) {
			proj.MemberIDs = append(proj.MemberIDs, member.Email)
		}
		proj.UpdatedAt = time.Now().UTC()
		if err := tx.Set(projRef, proj); err != nil {
			return err
		}
		return tx.Set(memberRef, member)
	})
}

func (r *firestoreProjectRepository) RemoveMember(ctx context.Context, projectID, userID string) error {
	projRef := r.fc.ProjectDoc(projectID)
	memberRef := projRef.Collection("members").Doc(userID)

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

		// Also check member's email to remove from MemberIDs if present
		var memberEmail string
		if mSnap, err := tx.Get(memberRef); err == nil {
			var m model.ProjectMember
			if err := mSnap.DataTo(&m); err == nil {
				memberEmail = m.Email
			}
		}

		filtered := make([]string, 0, len(proj.MemberIDs))
		for _, id := range proj.MemberIDs {
			if id != userID && (memberEmail == "" || id != memberEmail) {
				filtered = append(filtered, id)
			}
		}
		proj.MemberIDs = filtered
		proj.UpdatedAt = time.Now().UTC()

		if err := tx.Set(projRef, proj); err != nil {
			return err
		}
		return tx.Delete(memberRef)
	})
}

func containsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}
