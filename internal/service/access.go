package service

import (
	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
)

// AccessService loads resources and applies domain access rules.
type AccessService struct {
	decks repository.DeckRepository
	cards repository.CardRepository
	media repository.MediaRepository
}

func NewAccessService(
	decks repository.DeckRepository,
	cards repository.CardRepository,
	media repository.MediaRepository,
) *AccessService {
	return &AccessService{
		decks: decks,
		cards: cards,
		media: media,
	}
}

func (s *AccessService) EditableDeck(userID, deckID uint) (*model.Deck, error) {
	return s.ownedDeck(userID, deckID)
}

func (s *AccessService) UpvotableDeck(userID, deckID uint) (*model.Deck, error) {
	deck, err := s.decks.FindByID(deckID)
	if err != nil {
		return nil, err
	}
	if !deck.IsPublic || deck.UserID == userID {
		return nil, ErrForbidden
	}
	return deck, nil
}

func (s *AccessService) ReadableCard(userID, cardID uint) (*model.Card, error) {
	card, _, err := s.ownedCard(userID, cardID)
	return card, err
}

func (s *AccessService) EditableCard(userID, cardID uint) (*model.Card, error) {
	card, _, err := s.ownedCard(userID, cardID)
	return card, err
}

func (s *AccessService) StudyableDeck(userID, deckID uint) (*model.Deck, error) {
	return s.ownedDeck(userID, deckID)
}

func (s *AccessService) ReviewableCard(userID, cardID uint) (*model.Card, error) {
	card, _, err := s.ownedCard(userID, cardID)
	return card, err
}

func (s *AccessService) DeletableMedia(userID uint, publicID string) (*model.Media, error) {
	return s.ownedMedia(userID, publicID)
}

func (s *AccessService) ownedDeck(userID, deckID uint) (*model.Deck, error) {
	deck, err := s.decks.FindByID(deckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, ErrForbidden
	}
	return deck, nil
}

func (s *AccessService) ownedCard(userID, cardID uint) (*model.Card, *model.Deck, error) {
	card, err := s.cards.FindByID(cardID)
	if err != nil {
		return nil, nil, err
	}
	deck, err := s.ownedDeck(userID, card.DeckID)
	if err != nil {
		return nil, nil, err
	}
	return card, deck, nil
}

func (s *AccessService) ownedMedia(userID uint, publicID string) (*model.Media, error) {
	media, err := s.media.FindByPublicID(publicID)
	if err != nil {
		return nil, err
	}
	if media.UserID != userID {
		return nil, ErrForbidden
	}
	return media, nil
}
