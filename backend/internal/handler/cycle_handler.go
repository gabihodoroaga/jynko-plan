package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// CycleHandler handles HTTP requests for project sprint Cycles.
type CycleHandler struct {
	svc *service.CycleService
}

// NewCycleHandler creates a new CycleHandler.
func NewCycleHandler(svc *service.CycleService) *CycleHandler {
	return &CycleHandler{svc: svc}
}

func (h *CycleHandler) ListCycles(c *gin.Context) {
	projectID := c.Param("projectId")
	cycles, err := h.svc.List(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cycles)
}

func (h *CycleHandler) CreateCycle(c *gin.Context) {
	projectID := c.Param("projectId")
	var input service.CreateCycleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cycle, err := h.svc.Create(c.Request.Context(), projectID, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, cycle)
}

func (h *CycleHandler) UpdateCycle(c *gin.Context) {
	projectID := c.Param("projectId")
	cycleID := c.Param("cycleId")
	var input service.UpdateCycleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cycle, err := h.svc.Update(c.Request.Context(), projectID, cycleID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "cycle not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cycle)
}

func (h *CycleHandler) DeleteCycle(c *gin.Context) {
	projectID := c.Param("projectId")
	cycleID := c.Param("cycleId")
	if err := h.svc.Delete(c.Request.Context(), projectID, cycleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
