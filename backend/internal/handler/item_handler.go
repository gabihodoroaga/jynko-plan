package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// ItemHandler handles HTTP requests for WorkItems and SubWorkItems.
type ItemHandler struct {
	svc *service.ItemService
}

// NewItemHandler creates a new ItemHandler.
func NewItemHandler(svc *service.ItemService) *ItemHandler {
	return &ItemHandler{svc: svc}
}

func (h *ItemHandler) ListItems(c *gin.Context) {
	projectID := c.Param("projectId")
	filter := model.WorkItemFilter{
		Status:     c.Query("status"),
		Priority:   c.Query("priority"),
		ModuleID:   c.Query("module_id"),
		EpicID:     c.Query("epic_id"),
		CycleID:    c.Query("cycle_id"),
		AssigneeID: c.Query("assignee_id"),
		LabelID:    c.Query("label_id"),
		Search:     c.Query("search"),
	}

	items, err := h.svc.List(c.Request.Context(), projectID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *ItemHandler) GetItem(c *gin.Context) {
	projectID := c.Param("projectId")
	itemID := c.Param("itemId")
	item, err := h.svc.GetByID(c.Request.Context(), projectID, itemID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "work item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ItemHandler) CreateItem(c *gin.Context) {
	projectID := c.Param("projectId")
	userCtx, _ := middleware.GetUserContext(c)
	var input service.CreateWorkItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.svc.Create(c.Request.Context(), projectID, userCtx, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *ItemHandler) UpdateItem(c *gin.Context) {
	projectID := c.Param("projectId")
	itemID := c.Param("itemId")
	var input service.UpdateWorkItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.svc.Update(c.Request.Context(), projectID, itemID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "work item not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ItemHandler) DeleteItem(c *gin.Context) {
	projectID := c.Param("projectId")
	itemID := c.Param("itemId")
	if err := h.svc.Delete(c.Request.Context(), projectID, itemID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *ItemHandler) ListSubItems(c *gin.Context) {
	projectID := c.Param("projectId")
	itemID := c.Param("itemId")
	subItems, err := h.svc.ListSubItems(c.Request.Context(), projectID, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, subItems)
}

func (h *ItemHandler) CreateSubItem(c *gin.Context) {
	projectID := c.Param("projectId")
	itemID := c.Param("itemId")
	var input service.CreateSubItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.svc.CreateSubItem(c.Request.Context(), projectID, itemID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "parent work item not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sub)
}

func (h *ItemHandler) UpdateSubItem(c *gin.Context) {
	projectID := c.Param("projectId")
	itemID := c.Param("itemId")
	subItemID := c.Param("subItemId")
	var input service.UpdateSubItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.svc.UpdateSubItem(c.Request.Context(), projectID, itemID, subItemID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "sub-item not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func (h *ItemHandler) DeleteSubItem(c *gin.Context) {
	projectID := c.Param("projectId")
	itemID := c.Param("itemId")
	subItemID := c.Param("subItemId")
	if err := h.svc.DeleteSubItem(c.Request.Context(), projectID, itemID, subItemID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "sub-item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
