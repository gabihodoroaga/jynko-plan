package repository

import (
	"context"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserProfile stores durable user display preferences keyed by canonical UserID.
type UserProfile struct {
	UserID      string    `json:"user_id" firestore:"user_id"`
	Email       string    `json:"email" firestore:"email"`
	Name        string    `json:"name" firestore:"name"`
	AvatarURL   string    `json:"avatar_url" firestore:"avatar_url"`
	AvatarColor string    `json:"avatar_color" firestore:"avatar_color"`
	UpdatedAt   time.Time `json:"updated_at" firestore:"updated_at"`
}

// ProfileRepository defines persistence operations for user profiles.
type ProfileRepository interface {
	Get(ctx context.Context, userID string) (*UserProfile, error)
	Upsert(ctx context.Context, profile *UserProfile) error
}

type firestoreProfileRepository struct {
	fc *FirestoreClient
}

// NewProfileRepository creates a Firestore-backed ProfileRepository.
func NewProfileRepository(fc *FirestoreClient) ProfileRepository {
	return &firestoreProfileRepository{fc: fc}
}

func (r *firestoreProfileRepository) collectionName() string {
	base := strings.TrimSpace(r.fc.ProjectsCollection)
	if base == "" {
		base = "jp-dev-projects"
	}
	return base + "-profiles"
}

func (r *firestoreProfileRepository) Get(ctx context.Context, userID string) (*UserProfile, error) {
	id := strings.TrimSpace(userID)
	if id == "" || strings.Contains(id, "/") {
		return nil, ErrNotFound
	}
	snap, err := r.fc.Client.Collection(r.collectionName()).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	var p UserProfile
	if err := snap.DataTo(&p); err != nil {
		return nil, err
	}
	p.UserID = snap.Ref.ID
	return &p, nil
}

func (r *firestoreProfileRepository) Upsert(ctx context.Context, profile *UserProfile) error {
	id := strings.TrimSpace(profile.UserID)
	if id == "" || strings.Contains(id, "/") {
		return ErrNotFound
	}
	profile.UserID = id
	profile.UpdatedAt = time.Now().UTC()
	_, err := r.fc.Client.Collection(r.collectionName()).Doc(id).Set(ctx, profile)
	return err
}
