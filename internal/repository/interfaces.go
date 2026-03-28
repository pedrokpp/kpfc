package repository

import (
	"time"

	"kpp.dev/kpfc/internal/model"
)

// UserRepository defines data access operations for User.
type UserRepository interface {
	Create(user *model.User) error
	FindByID(id uint) (*model.User, error)
	FindByEmail(email string) (*model.User, error)
	Update(user *model.User) error
	Delete(id uint) error
}

// DeckRepository defines data access operations for Deck.
type DeckRepository interface {
	Create(deck *model.Deck) error
	FindByID(id uint) (*model.Deck, error)
	FindByUserID(userID uint) ([]model.Deck, error)
	// FindPublic returns public decks. sortBy "upvotes" orders by UpvoteCount DESC;
	// any other value orders by CreatedAt DESC.
	FindPublic(sortBy string) ([]model.Deck, error)
	Update(deck *model.Deck) error
	Delete(id uint) error
	// HasUpvoted reports whether userID has upvoted deckID.
	HasUpvoted(userID, deckID uint) (bool, error)
	// AddUpvote records an upvote and increments UpvoteCount atomically.
	AddUpvote(userID, deckID uint) error
	// RemoveUpvote removes an upvote and decrements UpvoteCount atomically.
	RemoveUpvote(userID, deckID uint) error
}

// MediaRepository defines data access operations for Media.
type MediaRepository interface {
	Create(media *model.Media) error
	FindByID(id uint) (*model.Media, error)
	FindByPublicID(publicID string) (*model.Media, error)
	FindByUserID(userID uint) ([]model.Media, error)
	Delete(id uint) error
}

// CardRepository defines data access operations for Card.
type CardRepository interface {
	Create(card *model.Card) error
	FindByID(id uint) (*model.Card, error)
	FindByDeckID(deckID uint) ([]model.Card, error)
	// FindDueCards returns cards in deckID where NextReviewAt <= now.
	FindDueCards(deckID uint, now time.Time) ([]model.Card, error)
	Update(card *model.Card) error
	Delete(id uint) error
}
