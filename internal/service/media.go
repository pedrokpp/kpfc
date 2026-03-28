package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
	"kpp.dev/kpfc/internal/storage"
)

// allowedContentTypes lists the MIME types accepted for media uploads.
var allowedContentTypes = map[string]string{
	"image/png":      "png",
	"image/jpeg":     "jpg",
	"image/gif":      "gif",
	"image/webp":     "webp",
	"image/svg+xml":  "svg",
	"audio/mpeg":     "mp3",
	"audio/mp4":      "m4a",
}

const maxMediaSize = 10 * 1024 * 1024 // 10 MB

// MediaService handles upload, retrieval, and deletion of media files.
type MediaService struct {
	repo    repository.MediaRepository
	store   storage.Storage
	idGen   func() string
}

// NewMediaService wires up the media service with the given repository and storage backend.
func NewMediaService(repo repository.MediaRepository, store storage.Storage, idGen func() string) *MediaService {
	return &MediaService{repo: repo, store: store, idGen: idGen}
}

// Upload validates, stores, and records a media file. Returns the created Media record.
func (s *MediaService) Upload(ctx context.Context, userID uint, filename, contentType string, size int64, r io.Reader) (*model.Media, error) {
	ext, ok := allowedContentTypes[contentType]
	if !ok {
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
	if size > maxMediaSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", size, maxMediaSize)
	}

	storagePath := fmt.Sprintf("%d/%s.%s", userID, s.idGen(), ext)

	if err := s.store.Store(ctx, storagePath, r); err != nil {
		return nil, fmt.Errorf("media upload store: %w", err)
	}

	m := &model.Media{
		PublicID:    s.idGen(),
		UserID:      userID,
		Filename:    sanitizeFilename(filename),
		ContentType: contentType,
		Size:        size,
		StoragePath: storagePath,
	}
	if err := s.repo.Create(m); err != nil {
		// best-effort cleanup; ignore secondary error
		_ = s.store.Delete(ctx, storagePath)
		return nil, fmt.Errorf("media upload record: %w", err)
	}
	return m, nil
}

// GetByPublicID returns the Media record and an open reader for its content.
// The caller must close the reader.
func (s *MediaService) GetByPublicID(ctx context.Context, publicID string) (*model.Media, io.ReadCloser, error) {
	m, err := s.repo.FindByPublicID(publicID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, repository.ErrNotFound
		}
		return nil, nil, fmt.Errorf("media get: %w", err)
	}
	rc, err := s.store.Fetch(ctx, m.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("media fetch storage: %w", err)
	}
	return m, rc, nil
}

// Delete removes the media file and its DB record. Returns ErrForbidden if
// the requesting user does not own the file.
func (s *MediaService) Delete(ctx context.Context, userID uint, publicID string) error {
	m, err := s.repo.FindByPublicID(publicID)
	if err != nil {
		return err
	}
	if m.UserID != userID {
		return ErrForbidden
	}
	if err := s.store.Delete(ctx, m.StoragePath); err != nil {
		return fmt.Errorf("media delete storage: %w", err)
	}
	return s.repo.Delete(m.ID)
}

// sanitizeFilename strips directory components from an uploaded filename.
func sanitizeFilename(name string) string {
	name = filepath.Base(name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." {
		return "upload"
	}
	return name
}
