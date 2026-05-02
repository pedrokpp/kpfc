package service

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
)

func newStudyPointsService(t *testing.T) (*StudyService, repository.UserRepository) {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	userRepo := repository.NewGORMUserRepository(db)
	return NewStudyService(nil, userRepo, nil), userRepo
}

func TestStudyService_CompleteSession_AddsPoints(t *testing.T) {
	svc, users := newStudyPointsService(t)
	user := &model.User{Email: "points@example.com", PasswordHash: "hash", TotalPoints: 7}
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	if err := svc.CompleteSession(user.ID, 15); err != nil {
		t.Fatalf("CompleteSession: %v", err)
	}

	got, err := users.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.TotalPoints != 22 {
		t.Fatalf("total_points = %d, want 22", got.TotalPoints)
	}
}

func TestStudyService_CompleteSession_UserNotFound(t *testing.T) {
	svc, _ := newStudyPointsService(t)
	if err := svc.CompleteSession(99999, 10); err != repository.ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
