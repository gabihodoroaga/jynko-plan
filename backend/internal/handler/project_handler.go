package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// ProjectHandler handles HTTP requests for Projects and RBAC membership.
type ProjectHandler struct {
	svc *service.ProjectService
	// DefaultRetentionDays is the minimum age (in days) of a soft-deleted project before the purge
	// API removes it when the request does not specify older_than_days (PLAN-13).
	DefaultRetentionDays int
}

// NewProjectHandler creates a new ProjectHandler.
func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc, DefaultRetentionDays: 30}
}

func (h *ProjectHandler) ListProjects(c *gin.Context) {
	userCtx, _ := middleware.GetUserContext(c)
	list := h.svc.ListProjectsForUser
	// GET /projects?deleted=true lists the caller's soft-deleted projects so they can be restored (PLAN-13).
	if deleted, _ := strconv.ParseBool(strings.TrimSpace(c.Query("deleted"))); deleted {
		list = h.svc.ListDeletedProjectsForUser
	}
	projects, err := list(c.Request.Context(), userCtx)
	if err != nil {
		respondInternalError(c, "failed to list projects", err)
		return
	}
	c.JSON(http.StatusOK, projects)
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	userCtx, _ := middleware.GetUserContext(c)
	var input service.CreateProjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proj, err := h.svc.CreateProject(c.Request.Context(), userCtx, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, proj)
}

func (h *ProjectHandler) GetProject(c *gin.Context) {
	projectID := c.Param("projectId")
	proj, err := h.svc.GetProject(c.Request.Context(), projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		respondInternalError(c, "failed to get project", err)
		return
	}
	c.JSON(http.StatusOK, proj)
}

func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	projectID := c.Param("projectId")
	var input service.UpdateProjectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proj, err := h.svc.UpdateProject(c.Request.Context(), projectID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, proj)
}

// DeleteProject soft-deletes the project (PLAN-13); see RestoreProject and PurgeDeletedProjects.
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	projectID := c.Param("projectId")
	userCtx, _ := middleware.GetUserContext(c)
	if err := h.svc.DeleteProject(c.Request.Context(), projectID, userCtx); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		respondInternalError(c, "failed to delete project", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// RestoreProject clears the soft-delete marker of a project (POST /projects/:projectId/restore).
func (h *ProjectHandler) RestoreProject(c *gin.Context) {
	projectID := c.Param("projectId")
	proj, err := h.svc.RestoreProject(c.Request.Context(), projectID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		case errors.Is(err, service.ErrProjectNotDeleted):
			c.JSON(http.StatusConflict, gin.H{"error": "project is not deleted"})
		default:
			respondInternalError(c, "failed to restore project", err)
		}
		return
	}
	c.JSON(http.StatusOK, proj)
}

// PurgeDeletedProjects permanently removes projects soft-deleted more than older_than_days ago
// (POST /admin/projects/purge-deleted?older_than_days=30&dry_run=true). Platform admins only.
func (h *ProjectHandler) PurgeDeletedProjects(c *gin.Context) {
	days := h.DefaultRetentionDays
	if raw := strings.TrimSpace(c.Query("older_than_days")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "older_than_days must be a non-negative integer"})
			return
		}
		days = n
	}
	dryRun := false
	if raw := strings.TrimSpace(c.Query("dry_run")); raw != "" {
		b, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dry_run must be a boolean"})
			return
		}
		dryRun = b
	}
	res, err := h.svc.PurgeDeletedProjects(c.Request.Context(), time.Duration(days)*24*time.Hour, dryRun)
	if err != nil {
		respondInternalError(c, "failed to purge deleted projects", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *ProjectHandler) ListMembers(c *gin.Context) {
	projectID := c.Param("projectId")
	members, err := h.svc.ListMembers(c.Request.Context(), projectID)
	if err != nil {
		respondInternalError(c, "failed to list project members", err)
		return
	}
	c.JSON(http.StatusOK, members)
}

func (h *ProjectHandler) UpsertMember(c *gin.Context) {
	projectID := c.Param("projectId")
	var input service.UpsertMemberInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member, err := h.svc.UpsertMember(c.Request.Context(), projectID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		if errors.Is(err, repository.ErrLastAdmin) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, member)
}

func (h *ProjectHandler) UpdateMember(c *gin.Context) {
	projectID := c.Param("projectId")
	userID := c.Param("userId")
	var input service.UpdateMemberInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member, err := h.svc.UpdateMember(c.Request.Context(), projectID, userID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
			return
		}
		if errors.Is(err, service.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name must be a non-empty string of at most 100 characters"})
			return
		}
		respondInternalError(c, "failed to update project member", err)
		return
	}
	c.JSON(http.StatusOK, member)
}

func (h *ProjectHandler) RemoveMember(c *gin.Context) {
	projectID := c.Param("projectId")
	userID := c.Param("userId")
	if err := h.svc.RemoveMember(c.Request.Context(), projectID, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
			return
		}
		if errors.Is(err, repository.ErrLastAdmin) || errors.Is(err, service.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		respondInternalError(c, "failed to remove project member", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "removed"})
}
