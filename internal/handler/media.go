package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"kpp.dev/kpfc/internal/middleware"
	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
	"kpp.dev/kpfc/internal/service"
)

const maxUploadBytes = 10 * 1024 * 1024 // 10 MB

// mediaServiceIface is the subset of MediaService used by the handler.
type mediaServiceIface interface {
	Upload(ctx context.Context, userID uint, input service.MediaUploadInput) (*model.Media, error)
	Open(ctx context.Context, publicID string) (*model.Media, io.ReadCloser, error)
	Delete(ctx context.Context, userID uint, publicID string) error
}

// MediaHandler handles media upload/retrieval/deletion.
type MediaHandler struct {
	svc mediaServiceIface
}

func NewMediaHandler(svc mediaServiceIface) *MediaHandler {
	return &MediaHandler{svc: svc}
}

// Upload handles POST /api/v1/media (auth required, multipart form, field "file").
func (h *MediaHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "request too large or invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing form file field 'file'")
		return
	}
	defer file.Close()

	m, err := h.svc.Upload(r.Context(), userID, service.MediaUploadInput{
		Filename: header.Filename,
		Size:     header.Size,
		Reader:   file,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, mediaResponse(m, r))
}

// Serve handles GET /api/v1/media/{id} (no auth required).
func (h *MediaHandler) Serve(w http.ResponseWriter, r *http.Request) {
	publicID := chi.URLParam(r, "id")

	m, rc, err := h.svc.Open(r.Context(), publicID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "media not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", m.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(m.Size, 10))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	io.Copy(w, rc)
}

// Delete handles DELETE /api/v1/media/{id} (auth required).
func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	publicID := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), userID, publicID); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			writeError(w, http.StatusNotFound, "media not found")
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "forbidden")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func mediaResponse(m *model.Media, r *http.Request) map[string]any {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/api/v1/media/%s", scheme, r.Host, m.PublicID)
	return map[string]any{
		"id":           m.ID,
		"public_id":    m.PublicID,
		"filename":     m.Filename,
		"content_type": m.ContentType,
		"size":         m.Size,
		"url":          url,
	}
}
