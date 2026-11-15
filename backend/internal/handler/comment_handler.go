package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// CommentHandler handles HTTP requests for WorkItem comments.
type CommentHandler struct {
	svc *service.ItemService
}

// NewCommentHandler creates a new CommentHandler.
func NewCommentHandler(svc *service.ItemService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

func (h *CommentHandler) ListComments(c *gin.Context) {
	projectID := c.Param("projectId")
	itemID := c.Param("itemId")
	comments, err := h.svc.ListComments(c.Request.Context(), projectID, itemID)
	if err != nil {
		respondInternalError(c, "failed to list comments", err)
		return
	}
	c.JSON(http.StatusOK, comments)
}

func (h *CommentHandler) CreateComment(c *gin.Context) {
	projectID := c.Param("projectId")
	itemID := c.Param("itemId")
	userCtx, _ := middleware.GetUserContext(c)
	var input service.CreateCommentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment, err := h.svc.CreateComment(c.Request.Context(), projectID, itemID, userCtx, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "work item not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, comment)
}

func (h *CommentHandler) UpdateComment(c *gin.Context) {
	projectID := c.Param("projectId")
	itemID := c.Param("itemId")
	commentID := c.Param("commentId")
	userCtx, _ := middleware.GetUserContext(c)
	role, _ := middleware.GetProjectRole(c)

	var input service.UpdateCommentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment, err := h.svc.UpdateComment(c.Request.Context(), projectID, itemID, commentID, userCtx, role, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the comment author or a project ADMIN can edit this comment"})
			return
		}
		if errors.Is(err, service.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		respondInternalError(c, "failed to update comment", err)
		return
	}
	c.JSON(http.StatusOK, comment)
}

func (h *CommentHandler) DeleteComment(c *gin.Context) {
	projectID := c.Param("projectId")
	itemID := c.Param("itemId")
	commentID := c.Param("commentId")
	userCtx, _ := middleware.GetUserContext(c)
	role, _ := middleware.GetProjectRole(c)

	if err := h.svc.DeleteComment(c.Request.Context(), projectID, itemID, commentID, userCtx, role); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "comment not found"})
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the comment author or a project ADMIN can delete this comment"})
			return
		}
		respondInternalError(c, "failed to delete comment", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
