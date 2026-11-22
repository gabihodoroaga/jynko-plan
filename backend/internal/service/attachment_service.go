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
	// MaxAttachmentSize is 25 Megabytes.
	MaxAttachmentSize = 25 * 1024 * 1024
)

var (
	ErrAttachmentTooLarge = errors.New("file exceeds maximum size of 25MB")
	ErrInvalidFileName    = errors.New("invalid or empty file name")
)

// SafeInlineMIMETypes lists non-scriptable MIME types safe to render inline (excludes text/html and image/svg+xml).
var safeInlineMIMETypes = map[string]bool{
	"image/png":       true,
	"image/jpeg":      true,
	"image/gif":       true,
	"image/webp":      true,
	"image/avif":      true,
	"application/pdf": true,
	"text/plain":      true,
}

// safeMarkdownImageMIMETypes lists raster image MIME types safe to embed via ![alt](url).
var safeMarkdownImageMIMETypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
	"image/avif": true,
}

// IsSafeInlineMIME returns true if contentType (ignoring charset parameters) can be safely rendered inline.
func IsSafeInlineMIME(contentType string) bool {
	mediaType := normalizeMediaType(contentType)
	return safeInlineMIMETypes[mediaType]
}

func normalizeMediaType(contentType string) string {
	mt, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil || mt == "" {
		base := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
		return base
	}
	return strings.ToLower(mt)
}

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
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		"\r", "",
		"\n", "",
		"\x00", "",
		`"`, "_",
	)
	base = strings.TrimSpace(replacer.Replace(base))
	if base == "" || base == "." || base == ".." {
		return "attachment"
	}
	return base
}

func escapeMarkdownLabel(label string) string {
	replacer := strings.NewReplacer(
		"[", "(",
		"]", ")",
		"\r", "",
		"\n", " ",
	)
	return replacer.Replace(label)
}

// PopulateURLAndMarkdown sets URL and Markdown fields on an Attachment.
func (s *AttachmentService) PopulateURLAndMarkdown(att *model.Attachment) {
	if att == nil {
		return
	}
	url := fmt.Sprintf("/api/v1/projects/%s/attachments/%s/content", att.ProjectID, att.ID)
	att.URL = url
	safeLabel := escapeMarkdownLabel(att.FileName)
	mediaType := normalizeMediaType(att.ContentType)
	if safeMarkdownImageMIMETypes[mediaType] {
		att.Markdown = fmt.Sprintf("![%s](%s)", safeLabel, url)
	} else {
		att.Markdown = fmt.Sprintf("[%s](%s)", safeLabel, url)
	}
}

type countingLimitReader struct {
	r     io.Reader
	limit int64
	n     int64
}

func (c *countingLimitReader) Read(p []byte) (int, error) {
	if c.n > c.limit {
		return 0, ErrAttachmentTooLarge
	}
	maxRead := int64(len(p))
	remaining := (c.limit + 1) - c.n
	if maxRead > remaining {
		p = p[:remaining]
	}
	n, err := c.r.Read(p)
	c.n += int64(n)
	if c.n > c.limit {
		return n, ErrAttachmentTooLarge
	}
	return n, err
}

// UploadAttachment validates, stores the blob with strict stream size enforcement, and saves metadata.
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
	ext := strings.ToLower(filepath.Ext(cleanFileName))
	detectedByExt := mime.TypeByExtension(ext)

	mediaType := normalizeMediaType(contentType)
	if mediaType == "" || mediaType == "application/octet-stream" {
		if detectedByExt != "" {
			contentType = detectedByExt
		} else {
			contentType = "application/octet-stream"
		}
	} else if (ext == ".html" || ext == ".htm" || ext == ".svg" || ext == ".xml" || ext == ".xhtml") && safeInlineMIMETypes[mediaType] {
		// Prevent extension/MIME mismatch tricks (e.g., uploading payload.svg with Content-Type: image/png)
		contentType = "application/octet-stream"
	}

	attachmentID := generateAttachmentID()
	storagePath := fmt.Sprintf("projects/%s/attachments/%s/%s", projectID, attachmentID, cleanFileName)

	limitedReader := &countingLimitReader{
		r:     reader,
		limit: MaxAttachmentSize,
	}

	if err := s.storage.Save(ctx, storagePath, contentType, limitedReader); err != nil {
		_ = s.storage.Delete(ctx, storagePath)
		if errors.Is(err, ErrAttachmentTooLarge) {
			return nil, ErrAttachmentTooLarge
		}
		return nil, fmt.Errorf("failed to save attachment to storage: %w", err)
	}

	actualSize := limitedReader.n
	if actualSize == 0 && sizeBytes > 0 {
		actualSize = sizeBytes
	}

	att := &model.Attachment{
		ID:           attachmentID,
		ProjectID:    projectID,
		WorkItemID:   strings.TrimSpace(workItemID),
		FileName:     cleanFileName,
		ContentType:  contentType,
		SizeBytes:    actualSize,
		StoragePath:  storagePath,
		UploadedBy:   uploader.UserID,
		UploadedName: uploader.Name,
	}

	if err := s.repo.Create(ctx, att); err != nil {
		_ = s.storage.Delete(ctx, storagePath)
		return nil, fmt.Errorf("failed to save attachment metadata: %w", err)
	}

	s.PopulateURLAndMarkdown(att)
	return att, nil
}

// GetAttachmentMetadata returns the attachment metadata without opening the stored blob.
func (s *AttachmentService) GetAttachmentMetadata(ctx context.Context, projectID, attachmentID string) (*model.Attachment, error) {
	att, err := s.repo.GetByID(ctx, projectID, attachmentID)
	if err != nil {
		return nil, err
	}
	s.PopulateURLAndMarkdown(att)
	return att, nil
}

// GetAttachment returns the attachment metadata and a stream to its content.
func (s *AttachmentService) GetAttachment(ctx context.Context, projectID, attachmentID string) (*model.Attachment, io.ReadCloser, error) {
	att, err := s.GetAttachmentMetadata(ctx, projectID, attachmentID)
	if err != nil {
		return nil, nil, err
	}

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

	_ = s.storage.Delete(ctx, att.StoragePath)
	return s.repo.Delete(ctx, projectID, attachmentID)
}
