package service

import (
	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
)

// CardService handles card management operations.
type CardService struct {
	cards repository.CardRepository
	decks repository.DeckRepository
}

func NewCardService(cards repository.CardRepository, decks repository.DeckRepository) *CardService {
	return &CardService{cards: cards, decks: decks}
}

// ownerDeck fetches deck deckID and returns ErrForbidden if it doesn't belong to userID.
func (s *CardService) ownerDeck(userID, deckID uint) (*model.Deck, error) {
	deck, err := s.decks.FindByID(deckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, ErrForbidden
	}
	return deck, nil
}

// ownerCard fetches card cardID and returns ErrForbidden if its deck doesn't belong to userID.
func (s *CardService) ownerCard(userID, cardID uint) (*model.Card, error) {
	card, err := s.cards.FindByID(cardID)
	if err != nil {
		return nil, err
	}
	if _, err := s.ownerDeck(userID, card.DeckID); err != nil {
		return nil, err
	}
	return card, nil
}

// Create adds a new card to deckID. Returns ErrForbidden if userID doesn't own the deck.
func (s *CardService) Create(userID, deckID uint, front, back string) (*model.Card, error) {
	if _, err := s.ownerDeck(userID, deckID); err != nil {
		return nil, err
	}
	card := &model.Card{
		DeckID:     deckID,
		Front:      front,
		Back:       back,
		EaseFactor: 2.5,
		Interval:   1,
	}
	if err := s.cards.Create(card); err != nil {
		return nil, err
	}
	return card, nil
}

// GetByID returns the card. Returns ErrForbidden if the card's deck doesn't belong to userID.
func (s *CardService) GetByID(userID, cardID uint) (*model.Card, error) {
	return s.ownerCard(userID, cardID)
}

// ListByDeck returns all cards in deckID. Returns ErrForbidden if userID doesn't own the deck.
func (s *CardService) ListByDeck(userID, deckID uint) ([]model.Card, error) {
	if _, err := s.ownerDeck(userID, deckID); err != nil {
		return nil, err
	}
	return s.cards.FindByDeckID(deckID)
}

// Update changes front/back of a card. Returns ErrForbidden if userID doesn't own the card's deck.
func (s *CardService) Update(userID, cardID uint, front, back string) (*model.Card, error) {
	card, err := s.ownerCard(userID, cardID)
	if err != nil {
		return nil, err
	}
	card.Front = front
	card.Back = back
	if err := s.cards.Update(card); err != nil {
		return nil, err
	}
	return card, nil
}

// Delete removes a card. Returns ErrForbidden if userID doesn't own the card's deck.
func (s *CardService) Delete(userID, cardID uint) error {
	if _, err := s.ownerCard(userID, cardID); err != nil {
		return err
	}
	return s.cards.Delete(cardID)
}
