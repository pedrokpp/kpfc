package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kpp.dev/kpfc/internal/model"
)

type GORMDeckRepository struct {
	db *gorm.DB
}

func NewGORMDeckRepository(db *gorm.DB) DeckRepository {
	return &GORMDeckRepository{db: db}
}

func (r *GORMDeckRepository) Create(deck *model.Deck) error {
	if err := r.db.Create(deck).Error; err != nil {
		return fmt.Errorf("deck create: %w", err)
	}
	return nil
}

func (r *GORMDeckRepository) FindByID(id uint) (*model.Deck, error) {
	var deck model.Deck
	if err := r.db.First(&deck, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("deck find by id: %w", err)
	}
	return &deck, nil
}

func (r *GORMDeckRepository) FindByUserID(userID uint) ([]model.Deck, error) {
	var decks []model.Deck
	if err := r.db.Where("user_id = ?", userID).Find(&decks).Error; err != nil {
		return nil, fmt.Errorf("deck find by user: %w", err)
	}
	return decks, nil
}

func (r *GORMDeckRepository) FindPublic(sortBy string) ([]model.Deck, error) {
	var decks []model.Deck
	q := r.db.Where("is_public = ?", true)
	if sortBy == "upvotes" {
		q = q.Order("upvote_count DESC")
	} else {
		q = q.Order("created_at DESC")
	}
	if err := q.Find(&decks).Error; err != nil {
		return nil, fmt.Errorf("deck find public: %w", err)
	}
	return decks, nil
}

func (r *GORMDeckRepository) Update(deck *model.Deck) error {
	if err := r.db.Save(deck).Error; err != nil {
		return fmt.Errorf("deck update: %w", err)
	}
	return nil
}

func (r *GORMDeckRepository) Delete(id uint) error {
	if err := r.db.Delete(&model.Deck{}, id).Error; err != nil {
		return fmt.Errorf("deck delete: %w", err)
	}
	return nil
}

func (r *GORMDeckRepository) ToggleUpvote(userID, deckID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&model.Deck{}, deckID).Error; err != nil {
			return fmt.Errorf("deck toggle upvote lock: %w", err)
		}

		var count int64
		if err := tx.Model(&model.UserDeckUpvote{}).
			Where("user_id = ? AND deck_id = ?", userID, deckID).
			Count(&count).Error; err != nil {
			return fmt.Errorf("deck toggle upvote check: %w", err)
		}

		if count > 0 {
			if err := tx.Where("user_id = ? AND deck_id = ?", userID, deckID).
				Delete(&model.UserDeckUpvote{}).Error; err != nil {
				return fmt.Errorf("deck toggle remove upvote: %w", err)
			}
			return tx.Model(&model.Deck{}).Where("id = ?", deckID).
				UpdateColumn("upvote_count", gorm.Expr("upvote_count - 1")).Error
		}

		if err := tx.Create(&model.UserDeckUpvote{UserID: userID, DeckID: deckID}).Error; err != nil {
			return fmt.Errorf("deck toggle add upvote: %w", err)
		}
		return tx.Model(&model.Deck{}).Where("id = ?", deckID).
			UpdateColumn("upvote_count", gorm.Expr("upvote_count + 1")).Error
	})
}
