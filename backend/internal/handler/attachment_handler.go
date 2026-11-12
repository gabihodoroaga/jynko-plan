package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabihodoroaga/jynko-plan/internal/middleware"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/service"
)

// AttachmentHandler handles HTTP requests for project attachments.
type AttachmentHandler struct {
	svc *service.AttachmentService
}

// NewAttachmentHandler creates a new AttachmentHandler.
func NewAttachmentHandler(svc *service.AttachmentService) *AttachmentHandler {
	return &AttachmentHandler{svc: svc}
}

// UploadAttachment handles multipart file upload.
func (h *AttachmentHandler) UploadAttachment(c *gin.Context) {
	projectID := c.Param("projectId")
	user, _ := middleware.GetUserContext(c)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file field is required"})
		return
	}

	if fileHeader.Size > service.MaxAttachmentSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file exceeds maximum allowed size of 25MB"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
		return
	}
	defer src.Close()

	workItemID := c.PostForm("itemId")
	if workItemID == "" {
		workItemID = c.Query("itemId")
	}

	contentType := fileHeader.Header.Get("Content-Type")

	att, err := h.svc.UploadAttachment(
		c.Request.Context(),
		projectID,
		user,
		workItemID,
		fileHeader.Filename,
		contentType,
		fileHeader.Size,
		src,
	)
	if err != nil {
		if errors.Is(err, service.ErrAttachmentTooLarge) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, att)
}

// GetAttachmentContent streams the attachment file with security headers.
func (h *AttachmentHandler) GetAttachmentContent(c *gin.Context) {
	projectID := c.Param("projectId")
	attachmentID := c.Param("attachmentId")

	att, reader, err := h.svc.GetAttachment(c.Request.Context(), projectID, attachmentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "attachment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	c.Header("Content-Type", att.ContentType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", att.FileName))
	c.Header("Cache-Control", "private, max-age=3600")
	if att.SizeBytes > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", att.SizeBytes))
	}

	_, _ = io.Copy(c.Writer, reader)
}

// ListAttachments returns project attachments, optionally filtered by item.
func (h *AttachmentHandler) ListAttachments(c *gin.Context) {
	projectID := c.Param("projectId")
	workItemID := c.Query("itemId")

	attachments, err := h.svc.ListAttachments(c.Request.Context(), projectID, workItemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, attachments)
}

// DeleteAttachment removes an attachment.
func (h *AttachmentHandler) DeleteAttachment(c *gin.Context) {
	projectID := c.Param("projectId")
	attachmentID := c.Param("attachmentId")

	if err := h.svc.DeleteAttachment(c.Request.Context(), projectID, attachmentID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "attachment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
