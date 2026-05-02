package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"kpp.dev/kpfc/internal/storage"
)

func newTemp(t *testing.T) *storage.LocalStorage {
	t.Helper()
	return storage.NewLocalStorage(t.TempDir())
}

func TestLocalStorage_StoreAndFetch(t *testing.T) {
	s := newTemp(t)
	ctx := context.Background()
	content := "hello world"

	if err := s.Store(ctx, "a/b/file.txt", strings.NewReader(content)); err != nil {
		t.Fatalf("Store: %v", err)
	}

	rc, err := s.Fetch(ctx, "a/b/file.txt")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != content {
		t.Errorf("content: got %q, want %q", got, content)
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	s := newTemp(t)
	ctx := context.Background()

	if err := s.Store(ctx, "del.txt", bytes.NewReader([]byte("x"))); err != nil {
		t.Fatalf("Store: %v", err)
	}
	if err := s.Delete(ctx, "del.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := s.Delete(ctx, "del.txt"); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("second Delete err = %v, want ErrNotFound", err)
	}
}

func TestLocalStorage_FetchNonExistent(t *testing.T) {
	s := newTemp(t)
	_, err := s.Fetch(context.Background(), "does/not/exist.txt")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Fetch err = %v, want ErrNotFound", err)
	}
}

func TestLocalStorage_NestedPathCreation(t *testing.T) {
	s := newTemp(t)
	ctx := context.Background()

	// deeply nested path — directories must be created automatically
	path := "a/b/c/d/e/deep.bin"
	if err := s.Store(ctx, path, bytes.NewReader([]byte{1, 2, 3})); err != nil {
		t.Fatalf("Store nested: %v", err)
	}
	rc, err := s.Fetch(ctx, path)
	if err != nil {
		t.Fatalf("Fetch nested: %v", err)
	}
	rc.Close()
}
