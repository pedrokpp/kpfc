package repository_test

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
)

func openMediaTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.Media{}); err != nil {
		t.Fatalf("automigrate media: %v", err)
	}
	return db
}

func TestMediaRepository_FindByID(t *testing.T) {
	db := openMediaTestDB(t)
	repo := repository.NewGORMMediaRepository(db)

	media := &model.Media{
		PublicID:    "m1",
		UserID:      10,
		Filename:    "one.png",
		ContentType: "image/png",
		Size:        123,
		StoragePath: "10/one.png",
	}
	if err := repo.Create(media); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.FindByID(media.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.PublicID != "m1" {
		t.Fatalf("public_id = %q, want m1", got.PublicID)
	}
}

func TestMediaRepository_FindByID_NotFound(t *testing.T) {
	db := openMediaTestDB(t)
	repo := repository.NewGORMMediaRepository(db)

	if _, err := repo.FindByID(99999); err != repository.ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestMediaRepository_FindByUserID(t *testing.T) {
	db := openMediaTestDB(t)
	repo := repository.NewGORMMediaRepository(db)

	items := []*model.Media{
		{PublicID: "m1", UserID: 1, Filename: "a.png", ContentType: "image/png", Size: 1, StoragePath: "1/a.png"},
		{PublicID: "m2", UserID: 1, Filename: "b.png", ContentType: "image/png", Size: 2, StoragePath: "1/b.png"},
		{PublicID: "m3", UserID: 2, Filename: "c.png", ContentType: "image/png", Size: 3, StoragePath: "2/c.png"},
	}
	for _, item := range items {
		if err := repo.Create(item); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	got, err := repo.FindByUserID(1)
	if err != nil {
		t.Fatalf("FindByUserID: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}
