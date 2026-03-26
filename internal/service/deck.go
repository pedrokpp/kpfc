package service

import (
	"errors"

	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
)

// ErrForbidden is returned when a user attempts to access a resource they do not own.
var ErrForbidden = errors.New("forbidden")

// DeckService handles deck management operations.
type DeckService struct {
	decks repository.DeckRepository
}

func NewDeckService(decks repository.DeckRepository) *DeckService {
	return &DeckService{decks: decks}
}

// Create creates a new deck owned by userID.
func (s *DeckService) Create(userID uint, title, description string, isPublic bool) (*model.Deck, error) {
	deck := &model.Deck{
		UserID:      userID,
		Title:       title,
		Description: description,
		IsPublic:    isPublic,
	}
	if err := s.decks.Create(deck); err != nil {
		return nil, err
	}
	return deck, nil
}

// GetByID returns the deck with the given ID. Returns ErrForbidden if the deck
// exists but does not belong to userID. Returns repository.ErrNotFound if absent.
func (s *DeckService) GetByID(userID, deckID uint) (*model.Deck, error) {
	deck, err := s.decks.FindByID(deckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, ErrForbidden
	}
	return deck, nil
}

// ListByUser returns all decks owned by userID.
func (s *DeckService) ListByUser(userID uint) ([]model.Deck, error) {
	return s.decks.FindByUserID(userID)
}

// Update changes the mutable fields of a deck. Returns ErrForbidden if the deck
// does not belong to userID.
func (s *DeckService) Update(userID, deckID uint, title, description string, isPublic bool) (*model.Deck, error) {
	deck, err := s.decks.FindByID(deckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, ErrForbidden
	}
	deck.Title = title
	deck.Description = description
	deck.IsPublic = isPublic
	if err := s.decks.Update(deck); err != nil {
		return nil, err
	}
	return deck, nil
}

// Delete removes the deck. Returns ErrForbidden if the deck does not belong to userID.
func (s *DeckService) Delete(userID, deckID uint) error {
	deck, err := s.decks.FindByID(deckID)
	if err != nil {
		return err
	}
	if deck.UserID != userID {
		return ErrForbidden
	}
	return s.decks.Delete(deckID)
}
