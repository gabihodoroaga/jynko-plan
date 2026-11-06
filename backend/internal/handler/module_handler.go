package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// ModuleHandler handles HTTP requests for project Modules.
type ModuleHandler struct {
	svc *service.ModuleService
}

// NewModuleHandler creates a new ModuleHandler.
func NewModuleHandler(svc *service.ModuleService) *ModuleHandler {
	return &ModuleHandler{svc: svc}
}

func (h *ModuleHandler) ListModules(c *gin.Context) {
	projectID := c.Param("projectId")
	modules, err := h.svc.List(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, modules)
}

func (h *ModuleHandler) CreateModule(c *gin.Context) {
	projectID := c.Param("projectId")
	var input service.CreateModuleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mod, err := h.svc.Create(c.Request.Context(), projectID, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, mod)
}

func (h *ModuleHandler) UpdateModule(c *gin.Context) {
	projectID := c.Param("projectId")
	moduleID := c.Param("moduleId")
	var input service.UpdateModuleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mod, err := h.svc.Update(c.Request.Context(), projectID, moduleID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "module not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mod)
}

func (h *ModuleHandler) DeleteModule(c *gin.Context) {
	projectID := c.Param("projectId")
	moduleID := c.Param("moduleId")
	if err := h.svc.Delete(c.Request.Context(), projectID, moduleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
