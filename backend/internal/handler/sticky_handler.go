package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// StickyHandler handles HTTP requests for project Sticky notes.
type StickyHandler struct {
	svc *service.StickyService
}

// NewStickyHandler creates a new StickyHandler.
func NewStickyHandler(svc *service.StickyService) *StickyHandler {
	return &StickyHandler{svc: svc}
}

func (h *StickyHandler) ListStickies(c *gin.Context) {
	projectID := c.Param("projectId")
	stickies, err := h.svc.ListStickies(c.Request.Context(), projectID)
	if err != nil {
		respondInternalError(c, "failed to list stickies", err)
		return
	}
	c.JSON(http.StatusOK, stickies)
}

func (h *StickyHandler) CreateSticky(c *gin.Context) {
	projectID := c.Param("projectId")
	user, _ := middleware.GetUserContext(c)

	var input service.CreateStickyInput
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&input); err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	sticky, err := h.svc.CreateSticky(c.Request.Context(), projectID, user, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sticky)
}

func (h *StickyHandler) UpdateSticky(c *gin.Context) {
	projectID := c.Param("projectId")
	stickyID := c.Param("stickyId")
	user, _ := middleware.GetUserContext(c)
	role, _ := middleware.GetProjectRole(c)

	var input service.UpdateStickyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sticky, err := h.svc.UpdateSticky(c.Request.Context(), projectID, stickyID, user, role, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "sticky not found"})
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the sticky author or a project ADMIN can edit this sticky"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sticky)
}

func (h *StickyHandler) DeleteSticky(c *gin.Context) {
	projectID := c.Param("projectId")
	stickyID := c.Param("stickyId")
	user, _ := middleware.GetUserContext(c)
	role, _ := middleware.GetProjectRole(c)

	if err := h.svc.DeleteSticky(c.Request.Context(), projectID, stickyID, user, role); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "sticky not found"})
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the sticky author or a project ADMIN can delete this sticky"})
			return
		}
		respondInternalError(c, "failed to delete sticky", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
