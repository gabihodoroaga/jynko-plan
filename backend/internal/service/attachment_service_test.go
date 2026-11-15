package service

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gabihodoroaga/jynko-plan/internal/model"
	"github.com/gabihodoroaga/jynko-plan/internal/repository"
	"github.com/gabihodoroaga/jynko-plan/internal/storage"
)

type mockAttachmentRepo struct {
	items map[string]*model.Attachment
}

func newMockAttachmentRepo() *mockAttachmentRepo {
	return &mockAttachmentRepo{items: make(map[string]*model.Attachment)}
}

func (m *mockAttachmentRepo) List(ctx context.Context, projectID string, itemID string) ([]model.Attachment, error) {
	var res []model.Attachment
	for _, a := range m.items {
		if a.ProjectID == projectID {
			if itemID == "" || a.WorkItemID == itemID {
				res = append(res, *a)
			}
		}
	}
	return res, nil
}

func (m *mockAttachmentRepo) GetByID(ctx context.Context, projectID, attachmentID string) (*model.Attachment, error) {
	if a, ok := m.items[attachmentID]; ok && a.ProjectID == projectID {
		copy := *a
		return &copy, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockAttachmentRepo) Create(ctx context.Context, attachment *model.Attachment) error {
	now := time.Now().UTC()
	attachment.CreatedAt = now
	attachment.UpdatedAt = now
	m.items[attachment.ID] = attachment
	return nil
}

func (m *mockAttachmentRepo) Delete(ctx context.Context, projectID, attachmentID string) error {
	delete(m.items, attachmentID)
	return nil
}

func TestAttachmentService_UploadAndRetrieve(t *testing.T) {
	tempDir := t.TempDir()
	store, err := storage.NewLocalStorageProvider(tempDir)
	if err != nil {
		t.Fatalf("failed to create storage provider: %v", err)
	}

	repo := newMockAttachmentRepo()
	svc := NewAttachmentService(repo, store)
	ctx := context.Background()

	uploader := model.UserContext{
		UserID: "user-123",
		Name:   "Alex Test",
	}

	// 1. Upload image
	imgContent := []byte("fake-png-data")
	att, err := svc.UploadAttachment(
		ctx,
		"proj-1",
		uploader,
		"item-99",
		"screenshot.png",
		"image/png",
		int64(len(imgContent)),
		bytes.NewReader(imgContent),
	)
	if err != nil {
		t.Fatalf("UploadAttachment failed: %v", err)
	}

	if att.ID == "" {
		t.Errorf("expected generated ID")
	}
	if !strings.HasPrefix(att.Markdown, "![screenshot.png](") {
		t.Errorf("unexpected image markdown: %s", att.Markdown)
	}
	if !strings.Contains(att.URL, att.ID) {
		t.Errorf("unexpected url: %s", att.URL)
	}

	// 2. Upload document (non-image)
	docContent := []byte("specification notes")
	docAtt, err := svc.UploadAttachment(
		ctx,
		"proj-1",
		uploader,
		"item-99",
		"specs.pdf",
		"application/pdf",
		int64(len(docContent)),
		bytes.NewReader(docContent),
	)
	if err != nil {
		t.Fatalf("UploadAttachment for PDF failed: %v", err)
	}
	if !strings.HasPrefix(docAtt.Markdown, "[specs.pdf](") {
		t.Errorf("unexpected doc markdown: %s", docAtt.Markdown)
	}

	// 3. Size limit validation
	hugeReader := bytes.NewReader(make([]byte, 10))
	_, err = svc.UploadAttachment(
		ctx,
		"proj-1",
		uploader,
		"item-99",
		"huge.bin",
		"application/octet-stream",
		MaxAttachmentSize+1,
		hugeReader,
	)
	if !errors.Is(err, ErrAttachmentTooLarge) {
		t.Errorf("expected ErrAttachmentTooLarge, got: %v", err)
	}

	// 4. List attachments
	list, err := svc.ListAttachments(ctx, "proj-1", "item-99")
	if err != nil {
		t.Fatalf("ListAttachments failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 attachments, got %d", len(list))
	}

	// 5. Get and read attachment
	retrieved, rc, err := svc.GetAttachment(ctx, "proj-1", att.ID)
	if err != nil {
		t.Fatalf("GetAttachment failed: %v", err)
	}
	defer rc.Close()
	if retrieved.FileName != "screenshot.png" {
		t.Errorf("expected filename screenshot.png, got: %s", retrieved.FileName)
	}

	// 6. Delete attachment
	if err := svc.DeleteAttachment(ctx, "proj-1", att.ID); err != nil {
		t.Fatalf("DeleteAttachment failed: %v", err)
	}
	_, _, err = svc.GetAttachment(ctx, "proj-1", att.ID)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected ErrNotFound after deletion, got: %v", err)
	}

	// 7. SVG / HTML XSS protection and Markdown bracket escaping
	svgPayload := []byte(`<svg onload="alert(1)"></svg>`)
	svgAtt, err := svc.UploadAttachment(
		ctx,
		"proj-1",
		uploader,
		"item-99",
		"exploit[v2].svg",
		"image/png", // Spoofed raster MIME with .svg extension!
		int64(len(svgPayload)),
		bytes.NewReader(svgPayload),
	)
	if err != nil {
		t.Fatalf("SVG UploadAttachment failed: %v", err)
	}
	if IsSafeInlineMIME(svgAtt.ContentType) {
		t.Errorf("expected .svg with spoofed image/png MIME to NOT be safe inline, got ContentType=%q", svgAtt.ContentType)
	}
	if strings.HasPrefix(svgAtt.Markdown, "![") {
		t.Errorf("expected .svg to NOT generate inline image markdown, got %q", svgAtt.Markdown)
	}
	if strings.Contains(svgAtt.Markdown, "[v2]") {
		t.Errorf("expected brackets in filename to be escaped in markdown label, got %q", svgAtt.Markdown)
	}
	if IsSafeInlineMIME("text/html; charset=utf-8") || IsSafeInlineMIME("image/svg+xml") {
		t.Errorf("expected text/html and image/svg+xml to be rejected by IsSafeInlineMIME")
	}
	if !IsSafeInlineMIME("image/png") || !IsSafeInlineMIME("application/pdf") {
		t.Errorf("expected image/png and application/pdf to be allowed by IsSafeInlineMIME")
	}
}
