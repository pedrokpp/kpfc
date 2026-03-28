package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Storage abstracts file persistence. Implementations may target the local
// filesystem, S3, GCS, or any other backend without touching service code.
type Storage interface {
	Store(ctx context.Context, path string, r io.Reader) error
	Fetch(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
}

// LocalStorage persists files under a root directory on the local filesystem.
type LocalStorage struct {
	root string
}

// NewLocalStorage returns a LocalStorage that stores files under root.
func NewLocalStorage(root string) *LocalStorage {
	return &LocalStorage{root: root}
}

// Store writes the contents of r to root/path, creating parent directories as needed.
func (s *LocalStorage) Store(_ context.Context, path string, r io.Reader) error {
	full := filepath.Join(s.root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("storage store mkdir: %w", err)
	}
	f, err := os.Create(full)
	if err != nil {
		return fmt.Errorf("storage store create: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("storage store write: %w", err)
	}
	return nil
}

// Fetch opens and returns the file at root/path. The caller must close the reader.
func (s *LocalStorage) Fetch(_ context.Context, path string) (io.ReadCloser, error) {
	full := filepath.Join(s.root, filepath.FromSlash(path))
	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("storage fetch: %w", err)
	}
	return f, nil
}

// Delete removes the file at root/path.
func (s *LocalStorage) Delete(_ context.Context, path string) error {
	full := filepath.Join(s.root, filepath.FromSlash(path))
	if err := os.Remove(full); err != nil {
		return fmt.Errorf("storage delete: %w", err)
	}
	return nil
}
