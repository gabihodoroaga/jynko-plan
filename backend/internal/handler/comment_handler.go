package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/scion"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// CommentHandler handles HTTP requests for WorkItem comments.
type CommentHandler struct {
	svc      *service.ItemService
	scionSvc *service.ScionOrchestrator
}

// NewCommentHandler creates a new CommentHandler.
func NewCommentHandler(svc *service.ItemService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

// NewCommentHandlerWithScion creates a CommentHandler with Scion slash-command orchestration.
func NewCommentHandlerWithScion(svc *service.ItemService, scionSvc *service.ScionOrchestrator) *CommentHandler {
	return &CommentHandler{svc: svc, scionSvc: scionSvc}
}

// CreateCommentResponse preserves all model.Comment fields at the top level while
// optionally returning the system dispatch comment and Scion execution summary.
type CreateCommentResponse struct {
	*model.Comment
	DispatchComment *model.Comment           `json:"dispatch_comment,omitempty"`
	Dispatch        *service.DispatchSummary `json:"dispatch,omitempty"`
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

	parsed := scion.ParseSlashCommand(input.Body)
	var precheck *service.PrecheckResult
	if parsed.IsCommand {
		if h.scionSvc == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Scion integration is not configured"})
			return
		}
		var err error
		precheck, err = h.scionSvc.Precheck(c.Request.Context(), projectID, itemID, parsed)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "work item not found"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		input.SlashCommand = parsed.Command
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
	if !parsed.IsCommand {
		c.JSON(http.StatusCreated, comment)
		return
	}

	dispatchComment, dispatchSummary := h.scionSvc.Execute(c.Request.Context(), precheck, userCtx, input.TargetAgentID)
	c.JSON(http.StatusCreated, CreateCommentResponse{
		Comment:         comment,
		DispatchComment: dispatchComment,
		Dispatch:        dispatchSummary,
	})
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
