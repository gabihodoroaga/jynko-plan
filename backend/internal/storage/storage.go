package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/gabihodoroaga/jynko-plan/internal/config"
)

// StorageProvider abstracts persistence of raw file blobs (local disk vs GCS).
type StorageProvider interface {
	Save(ctx context.Context, storagePath string, contentType string, reader io.Reader) error
	Open(ctx context.Context, storagePath string) (io.ReadCloser, error)
	Delete(ctx context.Context, storagePath string) error
	Close() error
}

// LocalStorageProvider stores files on the local filesystem.
type LocalStorageProvider struct {
	baseDir string
}

// NewLocalStorageProvider creates a new LocalStorageProvider.
func NewLocalStorageProvider(baseDir string) (*LocalStorageProvider, error) {
	if baseDir == "" {
		baseDir = "data/attachments"
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create local storage dir: %w", err)
	}
	return &LocalStorageProvider{baseDir: baseDir}, nil
}

func (p *LocalStorageProvider) resolvePath(storagePath string) string {
	clean := filepath.Clean("/" + storagePath)
	return filepath.Join(p.baseDir, clean)
}

func (p *LocalStorageProvider) Save(ctx context.Context, storagePath string, contentType string, reader io.Reader) error {
	fullPath := p.resolvePath(storagePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func (p *LocalStorageProvider) Open(ctx context.Context, storagePath string) (io.ReadCloser, error) {
	fullPath := p.resolvePath(storagePath)
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (p *LocalStorageProvider) Delete(ctx context.Context, storagePath string) error {
	fullPath := p.resolvePath(storagePath)
	if err := os.Remove(fullPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (p *LocalStorageProvider) Close() error {
	return nil
}

// GCSStorageProvider stores files in a Google Cloud Storage bucket.
type GCSStorageProvider struct {
	client *storage.Client
	bucket string
}

// NewGCSStorageProvider creates a new GCSStorageProvider.
func NewGCSStorageProvider(ctx context.Context, bucket string) (*GCSStorageProvider, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcs client: %w", err)
	}
	return &GCSStorageProvider{
		client: client,
		bucket: bucket,
	}, nil
}

func (p *GCSStorageProvider) Save(ctx context.Context, storagePath string, contentType string, reader io.Reader) error {
	obj := p.client.Bucket(p.bucket).Object(storagePath)
	w := obj.NewWriter(ctx)
	if contentType != "" {
		w.ContentType = contentType
	}
	if _, err := io.Copy(w, reader); err != nil {
		_ = w.Close()
		return fmt.Errorf("failed to write to gcs: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close gcs writer: %w", err)
	}
	return nil
}

func (p *GCSStorageProvider) Open(ctx context.Context, storagePath string) (io.ReadCloser, error) {
	obj := p.client.Bucket(p.bucket).Object(storagePath)
	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (p *GCSStorageProvider) Delete(ctx context.Context, storagePath string) error {
	obj := p.client.Bucket(p.bucket).Object(storagePath)
	if err := obj.Delete(ctx); err != nil && !errors.Is(err, storage.ErrObjectNotExist) {
		return fmt.Errorf("failed to delete from gcs: %w", err)
	}
	return nil
}

func (p *GCSStorageProvider) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

// NewStorageProvider initializes the appropriate StorageProvider based on configuration.
func NewStorageProvider(ctx context.Context, cfg *config.Config) (StorageProvider, error) {
	if cfg.AttachmentsGCSBucket != "" {
		return NewGCSStorageProvider(ctx, cfg.AttachmentsGCSBucket)
	}
	return NewLocalStorageProvider(cfg.LocalStorageDir)
}
