package service

import (
	"errors"
	"slices"

	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
	"kpp.dev/kpfc/pkg/cloze"
)

var (
	ErrInvalidCardType = errors.New("invalid card type: must be 'basic' or 'cloze'")
	ErrInvalidCloze    = errors.New("cloze card front must contain at least one {{cN::...}} deletion with a matching cloze_index")
)

// CardCreateOpts holds parameters for CreateAdvanced and UpdateAdvanced.
type CardCreateOpts struct {
	Title      string
	Front      string
	Back       string
	CardType   string
	ClozeIndex int
	Extra      string
}

func validateCardOpts(opts CardCreateOpts) error {
	switch opts.CardType {
	case "basic":
		if opts.Front == "" || opts.Back == "" {
			return errors.New("front and back are required")
		}
	case "cloze":
		if opts.ClozeIndex <= 0 || !slices.Contains(cloze.Indices(opts.Front), opts.ClozeIndex) {
			return ErrInvalidCloze
		}
	default:
		return ErrInvalidCardType
	}
	return nil
}

// CardService handles card management operations.
type CardService struct {
	cards  repository.CardRepository
	access *AccessService
}

func NewCardService(cards repository.CardRepository, access *AccessService) *CardService {
	return &CardService{cards: cards, access: access}
}

// Create adds a new card to deckID. Returns ErrForbidden if userID doesn't own the deck.
func (s *CardService) Create(userID, deckID uint, front, back, title string) (*model.Card, error) {
	if _, err := s.access.EditableDeck(userID, deckID); err != nil {
		return nil, err
	}
	card := &model.Card{
		DeckID:     deckID,
		Title:      title,
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
	return s.access.ReadableCard(userID, cardID)
}

// ListByDeck returns all cards in deckID. Returns ErrForbidden if userID doesn't own the deck.
func (s *CardService) ListByDeck(userID, deckID uint) ([]model.Card, error) {
	if _, err := s.access.EditableDeck(userID, deckID); err != nil {
		return nil, err
	}
	return s.cards.FindByDeckID(deckID)
}

// Update changes front/back/title of a card. Returns ErrForbidden if userID doesn't own the card's deck.
func (s *CardService) Update(userID, cardID uint, front, back, title string) (*model.Card, error) {
	card, err := s.access.EditableCard(userID, cardID)
	if err != nil {
		return nil, err
	}
	card.Title = title
	card.Front = front
	card.Back = back
	if err := s.cards.Update(card); err != nil {
		return nil, err
	}
	return card, nil
}

// Delete removes a card. Returns ErrForbidden if userID doesn't own the card's deck.
func (s *CardService) Delete(userID, cardID uint) error {
	if _, err := s.access.EditableCard(userID, cardID); err != nil {
		return err
	}
	return s.cards.Delete(cardID)
}

// CreateAdvanced adds a new card with explicit card type and extra fields.
// Returns ErrForbidden if userID doesn't own the deck.
// Returns ErrInvalidCardType or ErrInvalidCloze on invalid opts.
func (s *CardService) CreateAdvanced(userID, deckID uint, opts CardCreateOpts) (*model.Card, error) {
	if _, err := s.access.EditableDeck(userID, deckID); err != nil {
		return nil, err
	}
	if err := validateCardOpts(opts); err != nil {
		return nil, err
	}
	card := &model.Card{
		DeckID:     deckID,
		Title:      opts.Title,
		Front:      opts.Front,
		Back:       opts.Back,
		CardType:   opts.CardType,
		ClozeIndex: opts.ClozeIndex,
		Extra:      opts.Extra,
		EaseFactor: 2.5,
		Interval:   1,
	}
	if err := s.cards.Create(card); err != nil {
		return nil, err
	}
	return card, nil
}

// UpdateAdvanced changes a card's content with explicit card type and extra fields.
// Returns ErrForbidden if userID doesn't own the card's deck.
// Returns ErrInvalidCardType or ErrInvalidCloze on invalid opts.
func (s *CardService) UpdateAdvanced(userID, cardID uint, opts CardCreateOpts) (*model.Card, error) {
	card, err := s.access.EditableCard(userID, cardID)
	if err != nil {
		return nil, err
	}
	if err := validateCardOpts(opts); err != nil {
		return nil, err
	}
	card.Title = opts.Title
	card.Front = opts.Front
	card.Back = opts.Back
	card.CardType = opts.CardType
	card.ClozeIndex = opts.ClozeIndex
	card.Extra = opts.Extra
	if err := s.cards.Update(card); err != nil {
		return nil, err
	}
	return card, nil
}
