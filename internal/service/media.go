package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
	"kpp.dev/kpfc/internal/storage"
)

// allowedContentTypes lists the MIME types accepted for media uploads.
var allowedContentTypes = map[string]string{
	"image/png":     "png",
	"image/jpeg":    "jpg",
	"image/gif":     "gif",
	"image/webp":    "webp",
	"audio/mpeg":    "mp3",
	"audio/mp4":     "m4a",
}

const maxMediaSize = 10 * 1024 * 1024 // 10 MB

// MediaUploadInput captures the raw upload passed from the HTTP layer.
type MediaUploadInput struct {
	Filename string
	Size     int64
	Reader   io.Reader
}

// MediaService handles upload, retrieval, and deletion of media files.
type MediaService struct {
	repo   repository.MediaRepository
	store  storage.Storage
	idGen  func() string
	access *AccessService
}

// NewMediaService wires up the media service with the given repository and storage backend.
func NewMediaService(repo repository.MediaRepository, store storage.Storage, idGen func() string, access *AccessService) *MediaService {
	return &MediaService{repo: repo, store: store, idGen: idGen, access: access}
}

// Upload validates, stores, and records a media file. Returns the created Media record.
func (s *MediaService) Upload(ctx context.Context, userID uint, input MediaUploadInput) (*model.Media, error) {
	contentType, body, err := detectContentType(input.Reader)
	if err != nil {
		return nil, fmt.Errorf("detect content type: %w", err)
	}
	ext, err := validateUpload(contentType, input.Size)
	if err != nil {
		return nil, err
	}

	storagePath := s.buildStoragePath(userID, ext)

	if err := s.store.Store(ctx, storagePath, body); err != nil {
		return nil, fmt.Errorf("media upload store: %w", err)
	}

	m := s.buildMediaRecord(userID, input.Filename, contentType, storagePath, input.Size)
	if err := s.repo.Create(m); err != nil {
		// best-effort cleanup; ignore secondary error
		_ = s.store.Delete(ctx, storagePath)
		return nil, fmt.Errorf("media upload record: %w", err)
	}
	return m, nil
}

// Open returns the Media record and an open reader for its content.
// The caller must close the reader.
func (s *MediaService) Open(ctx context.Context, publicID string) (*model.Media, io.ReadCloser, error) {
	m, err := s.repo.FindByPublicID(publicID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, repository.ErrNotFound
		}
		return nil, nil, fmt.Errorf("media get: %w", err)
	}
	rc, err := s.store.Fetch(ctx, m.StoragePath)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, nil, repository.ErrNotFound
		}
		return nil, nil, fmt.Errorf("media fetch storage: %w", err)
	}
	return m, rc, nil
}

// Delete removes the media file and its DB record. Returns ErrForbidden if
// the requesting user does not own the file.
func (s *MediaService) Delete(ctx context.Context, userID uint, publicID string) error {
	m, err := s.access.DeletableMedia(userID, publicID)
	if err != nil {
		return err
	}
	if err := s.store.Delete(ctx, m.StoragePath); err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return fmt.Errorf("media delete storage: %w", err)
		}
	}
	if err := s.repo.Delete(m.ID); err != nil {
		return fmt.Errorf("media delete record: %w", err)
	}
	return nil
}

func detectContentType(r io.Reader) (contentType string, body io.Reader, err error) {
	sniff := make([]byte, 512)
	n, readErr := io.ReadFull(r, sniff)
	switch {
	case readErr == nil:
	case errors.Is(readErr, io.EOF), errors.Is(readErr, io.ErrUnexpectedEOF):
	default:
		return "", nil, readErr
	}

	sniff = sniff[:n]
	return http.DetectContentType(sniff), io.MultiReader(bytes.NewReader(sniff), r), nil
}

func validateUpload(contentType string, size int64) (string, error) {
	ext, ok := allowedContentTypes[contentType]
	if !ok {
		return "", fmt.Errorf("unsupported content type: %s", contentType)
	}
	if size > maxMediaSize {
		return "", fmt.Errorf("file too large: %d bytes (max %d)", size, maxMediaSize)
	}
	return ext, nil
}

func (s *MediaService) buildStoragePath(userID uint, ext string) string {
	return fmt.Sprintf("%d/%s.%s", userID, s.idGen(), ext)
}

func (s *MediaService) buildMediaRecord(userID uint, filename, contentType, storagePath string, size int64) *model.Media {
	return &model.Media{
		PublicID:    s.idGen(),
		UserID:      userID,
		Filename:    sanitizeFilename(filename),
		ContentType: contentType,
		Size:        size,
		StoragePath: storagePath,
	}
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
