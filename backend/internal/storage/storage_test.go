package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStorageProvider(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "jynko_storage_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	provider, err := NewLocalStorageProvider(tempDir)
	if err != nil {
		t.Fatalf("failed to create local storage provider: %v", err)
	}

	ctx := context.Background()
	testPath := "projects/test-proj/attachments/att_123/hello.txt"
	testContent := []byte("Hello, Jynko Attachments!")

	// 1. Save
	if err := provider.Save(ctx, testPath, "text/plain", bytes.NewReader(testContent)); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists on disk
	fullPath := filepath.Join(tempDir, filepath.Clean("/"+testPath))
	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("expected file to exist at %s: %v", fullPath, err)
	}

	// 2. Open
	rc, err := provider.Open(ctx, testPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	readBytes, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if !bytes.Equal(readBytes, testContent) {
		t.Fatalf("content mismatch: got %q, want %q", string(readBytes), string(testContent))
	}

	// 3. Delete
	if err := provider.Delete(ctx, testPath); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify file removed
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		t.Fatalf("expected file to be removed, got err: %v", err)
	}
}
