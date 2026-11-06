package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// LabelHandler handles HTTP requests for project Labels.
type LabelHandler struct {
	svc *service.LabelService
}

// NewLabelHandler creates a new LabelHandler.
func NewLabelHandler(svc *service.LabelService) *LabelHandler {
	return &LabelHandler{svc: svc}
}

func (h *LabelHandler) ListLabels(c *gin.Context) {
	projectID := c.Param("projectId")
	labels, err := h.svc.List(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, labels)
}

func (h *LabelHandler) CreateLabel(c *gin.Context) {
	projectID := c.Param("projectId")
	var input service.CreateLabelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lbl, err := h.svc.Create(c.Request.Context(), projectID, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, lbl)
}

func (h *LabelHandler) UpdateLabel(c *gin.Context) {
	projectID := c.Param("projectId")
	labelID := c.Param("labelId")
	var input service.UpdateLabelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lbl, err := h.svc.Update(c.Request.Context(), projectID, labelID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "label not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, lbl)
}

func (h *LabelHandler) DeleteLabel(c *gin.Context) {
	projectID := c.Param("projectId")
	labelID := c.Param("labelId")
	if err := h.svc.Delete(c.Request.Context(), projectID, labelID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
