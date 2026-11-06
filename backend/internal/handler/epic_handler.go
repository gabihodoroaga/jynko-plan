package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// EpicHandler handles HTTP requests for project Epics.
type EpicHandler struct {
	svc *service.EpicService
}

// NewEpicHandler creates a new EpicHandler.
func NewEpicHandler(svc *service.EpicService) *EpicHandler {
	return &EpicHandler{svc: svc}
}

func (h *EpicHandler) ListEpics(c *gin.Context) {
	projectID := c.Param("projectId")
	epics, err := h.svc.List(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, epics)
}

func (h *EpicHandler) CreateEpic(c *gin.Context) {
	projectID := c.Param("projectId")
	var input service.CreateEpicInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	epic, err := h.svc.Create(c.Request.Context(), projectID, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, epic)
}

func (h *EpicHandler) UpdateEpic(c *gin.Context) {
	projectID := c.Param("projectId")
	epicID := c.Param("epicId")
	var input service.UpdateEpicInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	epic, err := h.svc.Update(c.Request.Context(), projectID, epicID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "epic not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, epic)
}

func (h *EpicHandler) DeleteEpic(c *gin.Context) {
	projectID := c.Param("projectId")
	epicID := c.Param("epicId")
	if err := h.svc.Delete(c.Request.Context(), projectID, epicID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
