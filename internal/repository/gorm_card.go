package repository

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"kpp.dev/kpfc/internal/model"
)

type GORMCardRepository struct {
	db *gorm.DB
}

func NewGORMCardRepository(db *gorm.DB) CardRepository {
	return &GORMCardRepository{db: db}
}

func (r *GORMCardRepository) Create(card *model.Card) error {
	if err := r.db.Create(card).Error; err != nil {
		return fmt.Errorf("card create: %w", err)
	}
	return nil
}

func (r *GORMCardRepository) FindByID(id uint) (*model.Card, error) {
	var card model.Card
	if err := r.db.First(&card, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("card find by id: %w", err)
	}
	return &card, nil
}

func (r *GORMCardRepository) FindByDeckID(deckID uint) ([]model.Card, error) {
	var cards []model.Card
	if err := r.db.Where("deck_id = ?", deckID).Find(&cards).Error; err != nil {
		return nil, fmt.Errorf("card find by deck: %w", err)
	}
	return cards, nil
}

func (r *GORMCardRepository) FindDueCards(deckID uint, now time.Time) ([]model.Card, error) {
	var cards []model.Card
	if err := r.db.Where("deck_id = ? AND next_review_at <= ?", deckID, now).
		Find(&cards).Error; err != nil {
		return nil, fmt.Errorf("card find due: %w", err)
	}
	return cards, nil
}

func (r *GORMCardRepository) Update(card *model.Card) error {
	if err := r.db.Save(card).Error; err != nil {
		return fmt.Errorf("card update: %w", err)
	}
	return nil
}

func (r *GORMCardRepository) Delete(id uint) error {
	if err := r.db.Delete(&model.Card{}, id).Error; err != nil {
		return fmt.Errorf("card delete: %w", err)
	}
	return nil
}
