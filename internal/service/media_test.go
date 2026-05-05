package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
	"kpp.dev/kpfc/internal/storage"
)

var testPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
	0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
	0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
	0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc,
	0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
	0x44, 0xae, 0x42, 0x60, 0x82,
}

type fakeMediaRepo struct {
	createFn         func(*model.Media) error
	findByPublicIDFn func(string) (*model.Media, error)
	deleteFn         func(uint) error
	created          *model.Media
	deletedID        uint
}

func (f *fakeMediaRepo) Create(media *model.Media) error {
	f.created = media
	if f.createFn != nil {
		return f.createFn(media)
	}
	return nil
}

func (f *fakeMediaRepo) FindByID(id uint) (*model.Media, error) {
	return nil, repository.ErrNotFound
}

func (f *fakeMediaRepo) FindByPublicID(publicID string) (*model.Media, error) {
	if f.findByPublicIDFn != nil {
		return f.findByPublicIDFn(publicID)
	}
	return nil, repository.ErrNotFound
}

func (f *fakeMediaRepo) FindByUserID(userID uint) ([]model.Media, error) {
	return nil, nil
}

func (f *fakeMediaRepo) Delete(id uint) error {
	f.deletedID = id
	if f.deleteFn != nil {
		return f.deleteFn(id)
	}
	return nil
}

type fakeStorage struct {
	storeFn       func(context.Context, string, io.Reader) error
	fetchFn       func(context.Context, string) (io.ReadCloser, error)
	deleteFn      func(context.Context, string) error
	storedPath    string
	storedContent []byte
	deletedPath   string
}

func (f *fakeStorage) Store(ctx context.Context, path string, r io.Reader) error {
	f.storedPath = path
	if f.storeFn != nil {
		return f.storeFn(ctx, path, r)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.storedContent = data
	return nil
}

func (f *fakeStorage) Fetch(ctx context.Context, path string) (io.ReadCloser, error) {
	if f.fetchFn != nil {
		return f.fetchFn(ctx, path)
	}
	return io.NopCloser(bytes.NewReader(f.storedContent)), nil
}

func (f *fakeStorage) Delete(ctx context.Context, path string) error {
	f.deletedPath = path
	if f.deleteFn != nil {
		return f.deleteFn(ctx, path)
	}
	return nil
}

func newMediaServiceForTest(repo repository.MediaRepository, store storage.Storage, ids ...string) *MediaService {
	i := 0
	idGen := func() string {
		if i >= len(ids) {
			return "generated"
		}
		id := ids[i]
		i++
		return id
	}
	return NewMediaService(repo, store, idGen, NewAccessService(nil, nil, repo))
}

func TestMediaServiceUploadSniffsContentType(t *testing.T) {
	repo := &fakeMediaRepo{}
	store := &fakeStorage{}
	svc := newMediaServiceForTest(repo, store, "stored-file", "public-file")

	media, err := svc.Upload(context.Background(), 42, MediaUploadInput{
		Filename: "photo.png",
		Size:     int64(len(testPNG)),
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if media.ContentType != "image/png" {
		t.Fatalf("ContentType = %q, want image/png", media.ContentType)
	}
	if media.StoragePath != "42/stored-file.png" {
		t.Fatalf("StoragePath = %q", media.StoragePath)
	}
	if media.PublicID != "public-file" {
		t.Fatalf("PublicID = %q", media.PublicID)
	}
	if !bytes.Equal(store.storedContent, testPNG) {
		t.Fatal("stored content does not match upload content")
	}
}

func TestMediaServiceUploadRejectsUnsupportedType(t *testing.T) {
	repo := &fakeMediaRepo{}
	store := &fakeStorage{}
	svc := newMediaServiceForTest(repo, store)

	_, err := svc.Upload(context.Background(), 1, MediaUploadInput{
		Filename: "file.exe",
		Size:     int64(len("MZ fake exe")),
		Reader:   strings.NewReader("MZ fake exe"),
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported content type") {
		t.Fatalf("err = %v, want unsupported content type", err)
	}
}

func TestValidateUploadRejectsSVG(t *testing.T) {
	_, err := validateUpload("image/svg+xml", 1)
	if err == nil || !strings.Contains(err.Error(), "unsupported content type") {
		t.Fatalf("err = %v, want unsupported content type", err)
	}
}

func TestMediaServiceUploadRejectsOversize(t *testing.T) {
	repo := &fakeMediaRepo{}
	store := &fakeStorage{}
	svc := newMediaServiceForTest(repo, store)

	_, err := svc.Upload(context.Background(), 1, MediaUploadInput{
		Filename: "photo.png",
		Size:     maxMediaSize + 1,
		Reader:   bytes.NewReader(testPNG),
	})
	if err == nil || !strings.Contains(err.Error(), "file too large") {
		t.Fatalf("err = %v, want file too large", err)
	}
}

func TestMediaServiceUploadEmptyFilenameBecomesUpload(t *testing.T) {
	repo := &fakeMediaRepo{}
	store := &fakeStorage{}
	svc := newMediaServiceForTest(repo, store, "stored-file", "public-file")

	media, err := svc.Upload(context.Background(), 1, MediaUploadInput{
		Filename: "  ",
		Size:     int64(len(testPNG)),
		Reader:   bytes.NewReader(testPNG),
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if media.Filename != "upload" {
		t.Fatalf("Filename = %q, want upload", media.Filename)
	}
}

func TestMediaServiceUploadCreateFailureCleansStorage(t *testing.T) {
	repo := &fakeMediaRepo{
		createFn: func(*model.Media) error { return errors.New("db down") },
	}
	store := &fakeStorage{}
	svc := newMediaServiceForTest(repo, store, "stored-file", "public-file")

	_, err := svc.Upload(context.Background(), 1, MediaUploadInput{
		Filename: "photo.png",
		Size:     int64(len(testPNG)),
		Reader:   bytes.NewReader(testPNG),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if store.deletedPath != "1/stored-file.png" {
		t.Fatalf("cleanup path = %q", store.deletedPath)
	}
}

func TestMediaServiceOpenSuccess(t *testing.T) {
	repo := &fakeMediaRepo{
		findByPublicIDFn: func(publicID string) (*model.Media, error) {
			return &model.Media{PublicID: publicID, StoragePath: "1/path.png", ContentType: "image/png", Size: 3}, nil
		},
	}
	store := &fakeStorage{
		fetchFn: func(context.Context, string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("png")), nil
		},
	}
	svc := newMediaServiceForTest(repo, store)

	media, rc, err := svc.Open(context.Background(), "pub1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer rc.Close()
	body, _ := io.ReadAll(rc)
	if media.PublicID != "pub1" || string(body) != "png" {
		t.Fatalf("unexpected open result: media=%+v body=%q", media, body)
	}
}

func TestMediaServiceOpenMissingRecordReturnsNotFound(t *testing.T) {
	repo := &fakeMediaRepo{
		findByPublicIDFn: func(string) (*model.Media, error) { return nil, repository.ErrNotFound },
	}
	svc := newMediaServiceForTest(repo, &fakeStorage{})

	_, _, err := svc.Open(context.Background(), "missing")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestMediaServiceOpenMissingFileReturnsNotFound(t *testing.T) {
	repo := &fakeMediaRepo{
		findByPublicIDFn: func(publicID string) (*model.Media, error) {
			return &model.Media{PublicID: publicID, StoragePath: "1/path.png"}, nil
		},
	}
	store := &fakeStorage{
		fetchFn: func(context.Context, string) (io.ReadCloser, error) {
			return nil, storage.ErrNotFound
		},
	}
	svc := newMediaServiceForTest(repo, store)

	_, _, err := svc.Open(context.Background(), "orphan")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestMediaServiceDeleteOwnerExistingFileRemovesFileAndRecord(t *testing.T) {
	repo := &fakeMediaRepo{
		findByPublicIDFn: func(publicID string) (*model.Media, error) {
			return &model.Media{ID: 9, PublicID: publicID, UserID: 7, StoragePath: "7/file.png"}, nil
		},
	}
	store := &fakeStorage{}
	svc := newMediaServiceForTest(repo, store)

	if err := svc.Delete(context.Background(), 7, "pub1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if store.deletedPath != "7/file.png" {
		t.Fatalf("deleted storage path = %q", store.deletedPath)
	}
	if repo.deletedID != 9 {
		t.Fatalf("deleted record id = %d", repo.deletedID)
	}
}

func TestMediaServiceDeleteOwnerMissingFileStillRemovesRecord(t *testing.T) {
	repo := &fakeMediaRepo{
		findByPublicIDFn: func(publicID string) (*model.Media, error) {
			return &model.Media{ID: 9, PublicID: publicID, UserID: 7, StoragePath: "7/file.png"}, nil
		},
	}
	store := &fakeStorage{
		deleteFn: func(context.Context, string) error { return storage.ErrNotFound },
	}
	svc := newMediaServiceForTest(repo, store)

	if err := svc.Delete(context.Background(), 7, "pub1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if repo.deletedID != 9 {
		t.Fatalf("deleted record id = %d", repo.deletedID)
	}
}

func TestMediaServiceDeleteMissingAssetReturnsNotFound(t *testing.T) {
	repo := &fakeMediaRepo{
		findByPublicIDFn: func(string) (*model.Media, error) { return nil, repository.ErrNotFound },
	}
	svc := newMediaServiceForTest(repo, &fakeStorage{})

	err := svc.Delete(context.Background(), 7, "missing")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestMediaServiceDeleteOtherUsersAssetReturnsForbidden(t *testing.T) {
	repo := &fakeMediaRepo{
		findByPublicIDFn: func(publicID string) (*model.Media, error) {
			return &model.Media{ID: 9, PublicID: publicID, UserID: 8, StoragePath: "8/file.png"}, nil
		},
	}
	svc := newMediaServiceForTest(repo, &fakeStorage{})

	err := svc.Delete(context.Background(), 7, "pub1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestMediaServiceDeleteStorageFailureAborts(t *testing.T) {
	repo := &fakeMediaRepo{
		findByPublicIDFn: func(publicID string) (*model.Media, error) {
			return &model.Media{ID: 9, PublicID: publicID, UserID: 7, StoragePath: "7/file.png"}, nil
		},
	}
	store := &fakeStorage{
		deleteFn: func(context.Context, string) error { return errors.New("disk failure") },
	}
	svc := newMediaServiceForTest(repo, store)

	err := svc.Delete(context.Background(), 7, "pub1")
	if err == nil || !strings.Contains(err.Error(), "media delete storage") {
		t.Fatalf("err = %v, want storage error", err)
	}
	if repo.deletedID != 0 {
		t.Fatalf("record delete should not have happened, got %d", repo.deletedID)
	}
}
