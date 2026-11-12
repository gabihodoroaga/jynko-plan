package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/storage"
)

const (
	// MaxAttachmentSize is 25 Megabytes
	MaxAttachmentSize = 25 * 1024 * 1024
)

var (
	ErrAttachmentTooLarge = errors.New("file exceeds maximum size of 25MB")
	ErrInvalidFileName    = errors.New("invalid or empty file name")
)

// AttachmentService handles business logic for project attachments.
type AttachmentService struct {
	repo    repository.AttachmentRepository
	storage storage.StorageProvider
}

// NewAttachmentService creates a new AttachmentService.
func NewAttachmentService(repo repository.AttachmentRepository, storage storage.StorageProvider) *AttachmentService {
	return &AttachmentService{
		repo:    repo,
		storage: storage,
	}
}

func generateAttachmentID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return fmt.Sprintf("att_%d_%s", time.Now().Unix(), hex.EncodeToString(b))
}

func sanitizeFileName(name string) string {
	base := filepath.Base(name)
	base = strings.ReplaceAll(base, "/", "_")
	base = strings.ReplaceAll(base, "\\", "_")
	base = strings.TrimSpace(base)
	if base == "" || base == "." {
		return "attachment"
	}
	return base
}

// PopulateURLAndMarkdown sets URL and Markdown fields on an Attachment.
func (s *AttachmentService) PopulateURLAndMarkdown(att *model.Attachment) {
	if att == nil {
		return
	}
	url := fmt.Sprintf("/api/v1/projects/%s/attachments/%s/content", att.ProjectID, att.ID)
	att.URL = url
	if strings.HasPrefix(strings.ToLower(att.ContentType), "image/") {
		att.Markdown = fmt.Sprintf("![%s](%s)", att.FileName, url)
	} else {
		att.Markdown = fmt.Sprintf("[%s](%s)", att.FileName, url)
	}
}

// UploadAttachment validates, stores the blob, and saves attachment metadata.
func (s *AttachmentService) UploadAttachment(
	ctx context.Context,
	projectID string,
	uploader model.UserContext,
	workItemID string,
	fileName string,
	contentType string,
	sizeBytes int64,
	reader io.Reader,
) (*model.Attachment, error) {
	if sizeBytes > MaxAttachmentSize {
		return nil, ErrAttachmentTooLarge
	}

	cleanFileName := sanitizeFileName(fileName)
	if contentType == "" || contentType == "application/octet-stream" {
		ext := filepath.Ext(cleanFileName)
		if detected := mime.TypeByExtension(ext); detected != "" {
			contentType = detected
		} else {
			contentType = "application/octet-stream"
		}
	}

	attachmentID := generateAttachmentID()
	storagePath := fmt.Sprintf("projects/%s/attachments/%s/%s", projectID, attachmentID, cleanFileName)

	if err := s.storage.Save(ctx, storagePath, contentType, reader); err != nil {
		return nil, fmt.Errorf("failed to save attachment to storage: %w", err)
	}

	att := &model.Attachment{
		ID:           attachmentID,
		ProjectID:    projectID,
		WorkItemID:   workItemID,
		FileName:     cleanFileName,
		ContentType:  contentType,
		SizeBytes:    sizeBytes,
		StoragePath:  storagePath,
		UploadedBy:   uploader.UserID,
		UploadedName: uploader.Name,
	}

	if err := s.repo.Create(ctx, att); err != nil {
		// Attempt rollback in storage if DB fails
		_ = s.storage.Delete(ctx, storagePath)
		return nil, fmt.Errorf("failed to save attachment metadata: %w", err)
	}

	s.PopulateURLAndMarkdown(att)
	return att, nil
}

// GetAttachment returns the attachment metadata and a stream to its content.
func (s *AttachmentService) GetAttachment(ctx context.Context, projectID, attachmentID string) (*model.Attachment, io.ReadCloser, error) {
	att, err := s.repo.GetByID(ctx, projectID, attachmentID)
	if err != nil {
		return nil, nil, err
	}
	s.PopulateURLAndMarkdown(att)

	rc, err := s.storage.Open(ctx, att.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open attachment file: %w", err)
	}
	return att, rc, nil
}

// ListAttachments returns project attachments, optionally filtered by work item.
func (s *AttachmentService) ListAttachments(ctx context.Context, projectID string, itemID string) ([]model.Attachment, error) {
	attachments, err := s.repo.List(ctx, projectID, itemID)
	if err != nil {
		return nil, err
	}
	for i := range attachments {
		s.PopulateURLAndMarkdown(&attachments[i])
	}
	return attachments, nil
}

// DeleteAttachment removes the blob from storage and removes metadata from repository.
func (s *AttachmentService) DeleteAttachment(ctx context.Context, projectID, attachmentID string) error {
	att, err := s.repo.GetByID(ctx, projectID, attachmentID)
	if err != nil {
		return err
	}

	if err := s.storage.Delete(ctx, att.StoragePath); err != nil {
		// Log error, continue removing Firestore metadata
	}

	return s.repo.Delete(ctx, projectID, attachmentID)
}
