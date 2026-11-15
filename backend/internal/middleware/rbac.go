package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
)

const projectRoleContextKey = "jynko_project_role"

// RequireProjectRole checks if the authenticated user has at least minRole in the target project (:projectId).
func RequireProjectRole(projectRepo repository.ProjectRepository, minRole model.ProjectRole) gin.HandlerFunc {
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
