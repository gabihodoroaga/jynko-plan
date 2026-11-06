package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// PageHandler handles HTTP requests for Wiki Pages, Versions, and Collections.
type PageHandler struct {
	svc *service.PageService
}

// NewPageHandler creates a new PageHandler.
func NewPageHandler(svc *service.PageService) *PageHandler {
	return &PageHandler{svc: svc}
}

func (h *PageHandler) ListPages(c *gin.Context) {
	projectID := c.Param("projectId")
	collectionID := c.Query("collection_id")
	pages, err := h.svc.ListPages(c.Request.Context(), projectID, collectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pages)
}

func (h *PageHandler) GetPage(c *gin.Context) {
	projectID := c.Param("projectId")
	pageID := c.Param("pageId")
	page, err := h.svc.GetPage(c.Request.Context(), projectID, pageID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wiki page not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *PageHandler) CreatePage(c *gin.Context) {
	projectID := c.Param("projectId")
	userCtx, _ := middleware.GetUserContext(c)
	var input service.CreatePageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	page, err := h.svc.CreatePage(c.Request.Context(), projectID, userCtx, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, page)
}

func (h *PageHandler) UpdatePage(c *gin.Context) {
	projectID := c.Param("projectId")
	pageID := c.Param("pageId")
	userCtx, _ := middleware.GetUserContext(c)
	var input service.UpdatePageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	page, err := h.svc.UpdatePage(c.Request.Context(), projectID, pageID, userCtx, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wiki page not found"})
			return
		}
		if errors.Is(err, service.ErrPageLocked) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, page)
}

type lockInput struct {
	IsLocked bool `json:"is_locked"`
}

func (h *PageHandler) SetPageLock(c *gin.Context) {
	projectID := c.Param("projectId")
	pageID := c.Param("pageId")
	userCtx, _ := middleware.GetUserContext(c)
	var input lockInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	page, err := h.svc.SetPageLock(c.Request.Context(), projectID, pageID, userCtx, input.IsLocked)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wiki page not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *PageHandler) DeletePage(c *gin.Context) {
	projectID := c.Param("projectId")
	pageID := c.Param("pageId")
	if err := h.svc.DeletePage(c.Request.Context(), projectID, pageID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *PageHandler) ListVersions(c *gin.Context) {
	projectID := c.Param("projectId")
	pageID := c.Param("pageId")
	versions, err := h.svc.ListVersions(c.Request.Context(), projectID, pageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, versions)
}

func (h *PageHandler) RestoreVersion(c *gin.Context) {
	projectID := c.Param("projectId")
	pageID := c.Param("pageId")
	versionID := c.Param("versionId")
	userCtx, _ := middleware.GetUserContext(c)

	page, err := h.svc.RestoreVersion(c.Request.Context(), projectID, pageID, versionID, userCtx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "page or version not found"})
			return
		}
		if errors.Is(err, service.ErrPageLocked) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *PageHandler) ListCollections(c *gin.Context) {
	projectID := c.Param("projectId")
	collections, err := h.svc.ListCollections(c.Request.Context(), projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, collections)
}

func (h *PageHandler) CreateCollection(c *gin.Context) {
	projectID := c.Param("projectId")
	var input service.CreateCollectionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	col, err := h.svc.CreateCollection(c.Request.Context(), projectID, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, col)
}

func (h *PageHandler) UpdateCollection(c *gin.Context) {
	projectID := c.Param("projectId")
	collectionID := c.Param("collectionId")
	var input service.UpdateCollectionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	col, err := h.svc.UpdateCollection(c.Request.Context(), projectID, collectionID, input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "collection not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, col)
}

func (h *PageHandler) DeleteCollection(c *gin.Context) {
	projectID := c.Param("projectId")
	collectionID := c.Param("collectionId")
	if err := h.svc.DeleteCollection(c.Request.Context(), projectID, collectionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
