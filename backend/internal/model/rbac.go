package model

import "time"

// ProjectRole represents a per-project RBAC role.
type ProjectRole string

const (
	RoleAdmin  ProjectRole = "ADMIN"
	RoleMember ProjectRole = "MEMBER"
	RoleViewer ProjectRole = "VIEWER"
)

// IsValidRole checks whether r is a valid ProjectRole.
func IsValidRole(r ProjectRole) bool {
	switch r {
	case RoleAdmin, RoleMember, RoleViewer:
		return true
	default:
		return false
	}
}

// UserContext represents the authenticated user identity extracted from Google Cloud IAP headers.
type UserContext struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	AvatarColor string `json:"avatar_color,omitempty"`
}

// ProjectMember defines a user's role assignment in a project.
type ProjectMember struct {
	UserID   string      `firestore:"user_id" json:"user_id"`
	Email    string      `firestore:"email" json:"email"`
	Name     string      `firestore:"name" json:"name"`
	Role     ProjectRole `firestore:"role" json:"role"`
	JoinedAt time.Time   `firestore:"joined_at" json:"joined_at"`
}
