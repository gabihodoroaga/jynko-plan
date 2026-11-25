package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/config"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

const projectRoleContextKey = "jynko_project_role"

// RequireProjectRole checks if the authenticated user has at least minRole in the target project (:projectId).
// Soft-deleted projects (PLAN-13) are reported as 404 so they are no longer reachable.
func RequireProjectRole(projectRepo repository.ProjectRepository, minRole model.ProjectRole) gin.HandlerFunc {
	return requireProjectRole(projectRepo, minRole, false)
}

// RequireProjectRoleIncludingDeleted is RequireProjectRole for routes that must reach soft-deleted
// projects, such as restoring one.
func RequireProjectRoleIncludingDeleted(projectRepo repository.ProjectRepository, minRole model.ProjectRole) gin.HandlerFunc {
	return requireProjectRole(projectRepo, minRole, true)
}

// CheckProjectNotDeleted returns repository.ErrNotFound when the project is soft-deleted. A project
// document that cannot be found is left to the membership check (preserving the 403 behaviour).
func CheckProjectNotDeleted(ctx context.Context, projectRepo repository.ProjectRepository, projectID string) error {
	proj, err := projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}
	if proj.IsDeleted() {
		return repository.ErrNotFound
	}
	return nil
}

func requireProjectRole(projectRepo repository.ProjectRepository, minRole model.ProjectRole, allowDeleted bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx, exists := GetUserContext(c)
		if !exists || userCtx.UserID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		projectID := c.Param("projectId")
		if projectID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing projectId parameter"})
			return
		}

		if !allowDeleted {
			if err := CheckProjectNotDeleted(c.Request.Context(), projectRepo, projectID); err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "project not found"})
					return
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to load project"})
				return
			}
		}

		member, err := projectRepo.GetMember(c.Request.Context(), projectID, userCtx.UserID)
		if err != nil || member == nil {
			member, err = projectRepo.GetMemberByEmail(c.Request.Context(), projectID, userCtx.Email)
			if err != nil || member == nil {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied: not a member of this project"})
				return
			}
			// Reconcile email-invited member document to canonical IAP UserID on first login
			if member.UserID != userCtx.UserID && !strings.Contains(userCtx.UserID, "/") {
				resolvedName := member.Name
				if userCtx.Name != "" && (resolvedName == "" || strings.EqualFold(resolvedName, member.Email)) {
					resolvedName = userCtx.Name
				}
				reconciled := &model.ProjectMember{
					UserID:   userCtx.UserID,
					Email:    strings.ToLower(strings.TrimSpace(userCtx.Email)),
					Name:     resolvedName,
					Role:     member.Role,
					JoinedAt: member.JoinedAt,
				}
				_ = projectRepo.UpsertMember(c.Request.Context(), projectID, reconciled)
				member = reconciled
			}
		}

		if !HasPermission(member.Role, minRole) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient project role permissions"})
			return
		}

		c.Set(projectRoleContextKey, member.Role)
		c.Next()
	}
}

// GetProjectRole returns the resolved ProjectRole for the current request if set by RequireProjectRole.
func GetProjectRole(c *gin.Context) (model.ProjectRole, bool) {
	val, exists := c.Get(projectRoleContextKey)
	if !exists {
		return "", false
	}
	role, ok := val.(model.ProjectRole)
	return role, ok
}

// HasPermission returns true when userRole is greater than or equal to requiredRole in the RBAC hierarchy.
func HasPermission(userRole, requiredRole model.ProjectRole) bool {
	roles := map[model.ProjectRole]int{
		model.RoleViewer: 1,
		model.RoleMember: 2,
		model.RoleAdmin:  3,
	}
	return roles[userRole] >= roles[requiredRole]
}

// RequirePlatformAdmin restricts cross-project maintenance routes (e.g. purging deleted projects) to the
// users listed in PLATFORM_ADMIN_EMAILS. With no platform admins configured the route is disabled.
func RequirePlatformAdmin(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx, exists := GetUserContext(c)
		if !exists || userCtx.UserID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if !cfg.IsPlatformAdmin(userCtx.Email) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "platform admin access required"})
			return
		}
		c.Next()
	}
}
