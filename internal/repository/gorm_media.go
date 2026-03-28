package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"kpp.dev/kpfc/internal/model"
)

type GORMMediaRepository struct {
	db *gorm.DB
}

func NewGORMMediaRepository(db *gorm.DB) MediaRepository {
	return &GORMMediaRepository{db: db}
}

func (r *GORMMediaRepository) Create(media *model.Media) error {
	if err := r.db.Create(media).Error; err != nil {
		return fmt.Errorf("media create: %w", err)
	}
	return nil
}

func (r *GORMMediaRepository) FindByID(id uint) (*model.Media, error) {
	var m model.Media
	if err := r.db.First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("media find by id: %w", err)
	}
	return &m, nil
}

func (r *GORMMediaRepository) FindByPublicID(publicID string) (*model.Media, error) {
	var m model.Media
	if err := r.db.Where("public_id = ?", publicID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("media find by public id: %w", err)
	}
	return &m, nil
}

func (r *GORMMediaRepository) FindByUserID(userID uint) ([]model.Media, error) {
	var items []model.Media
	if err := r.db.Where("user_id = ?", userID).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("media find by user: %w", err)
	}
	return items, nil
}

func (r *GORMMediaRepository) Delete(id uint) error {
	if err := r.db.Delete(&model.Media{}, id).Error; err != nil {
		return fmt.Errorf("media delete: %w", err)
	}
	return nil
}
