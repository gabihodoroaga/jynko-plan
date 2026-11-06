package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, comment)
}

func (h *CommentHandler) DeleteComment(c *gin.Context) {
	projectID := c.Param("projectId")
	itemID := c.Param("itemId")
	commentID := c.Param("commentId")
	if err := h.svc.DeleteComment(c.Request.Context(), projectID, itemID, commentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
