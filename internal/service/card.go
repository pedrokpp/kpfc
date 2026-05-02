package service

import (
	"errors"
	"slices"

	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
	"kpp.dev/kpfc/pkg/cloze"
)

var (
	ErrInvalidCardType  = errors.New("invalid card type: must be 'basic' or 'cloze'")
	ErrInvalidBasicCard = errors.New("front and back are required")
	ErrInvalidCloze     = errors.New("cloze card front must contain at least one {{cN::...}} deletion with a matching cloze_index")
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

type CardAuthoringInput struct {
	Title      *string
	Front      *string
	Back       *string
	CardType   *string
	ClozeIndex *int
	Extra      *string
}

type normalizedCard struct {
	Title      string
	Front      string
	Back       string
	CardType   string
	ClozeIndex int
	Extra      string
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
	return s.CreateAuthored(userID, deckID, CardAuthoringInput{
		Title: stringPtr(title),
		Front: stringPtr(front),
		Back:  stringPtr(back),
	})
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
	return s.UpdateAuthored(userID, cardID, CardAuthoringInput{
		Title: stringPtr(title),
		Front: stringPtr(front),
		Back:  stringPtr(back),
	})
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
	return s.CreateAuthored(userID, deckID, CardAuthoringInput{
		Title:      stringPtr(opts.Title),
		Front:      stringPtr(opts.Front),
		Back:       stringPtr(opts.Back),
		CardType:   stringPtr(opts.CardType),
		ClozeIndex: intPtr(opts.ClozeIndex),
		Extra:      stringPtr(opts.Extra),
	})
}

// CreateAuthored adds a new authored card, selecting and normalizing the final persisted shape by card type.
func (s *CardService) CreateAuthored(userID, deckID uint, input CardAuthoringInput) (*model.Card, error) {
	if _, err := s.access.EditableDeck(userID, deckID); err != nil {
		return nil, err
	}
	normalized, err := normalizeCreateInput(input)
	if err != nil {
		return nil, err
	}
	card := &model.Card{
		DeckID:     deckID,
		Title:      normalized.Title,
		Front:      normalized.Front,
		Back:       normalized.Back,
		CardType:   normalized.CardType,
		ClozeIndex: normalized.ClozeIndex,
		Extra:      normalized.Extra,
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
	return s.UpdateAuthored(userID, cardID, CardAuthoringInput{
		Title:      stringPtr(opts.Title),
		Front:      stringPtr(opts.Front),
		Back:       stringPtr(opts.Back),
		CardType:   stringPtr(opts.CardType),
		ClozeIndex: intPtr(opts.ClozeIndex),
		Extra:      stringPtr(opts.Extra),
	})
}

// UpdateAuthored updates an authored card, preserving or changing the current type based on the authored input.
func (s *CardService) UpdateAuthored(userID, cardID uint, input CardAuthoringInput) (*model.Card, error) {
	card, err := s.access.EditableCard(userID, cardID)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeUpdateInput(card, input)
	if err != nil {
		return nil, err
	}
	card.Title = normalized.Title
	card.Front = normalized.Front
	card.Back = normalized.Back
	card.CardType = normalized.CardType
	card.ClozeIndex = normalized.ClozeIndex
	card.Extra = normalized.Extra
	if err := s.cards.Update(card); err != nil {
		return nil, err
	}
	return card, nil
}

func normalizeCreateInput(input CardAuthoringInput) (normalizedCard, error) {
	effectiveType := "basic"
	if input.CardType != nil {
		effectiveType = *input.CardType
	}
	if effectiveType != "basic" && effectiveType != "cloze" {
		return normalizedCard{}, ErrInvalidCardType
	}
	return normalizeByType(effectiveType, nil, input, true)
}

func normalizeUpdateInput(current *model.Card, input CardAuthoringInput) (normalizedCard, error) {
	effectiveType := currentCardType(current)
	if input.CardType != nil {
		effectiveType = *input.CardType
	}
	if effectiveType != "basic" && effectiveType != "cloze" {
		return normalizedCard{}, ErrInvalidCardType
	}
	return normalizeByType(effectiveType, current, input, false)
}

func normalizeByType(effectiveType string, current *model.Card, input CardAuthoringInput, isCreate bool) (normalizedCard, error) {
	normalized := normalizedCard{
		Title:    stringValue(input.Title),
		CardType: effectiveType,
	}

	front := stringValue(input.Front)
	switch effectiveType {
	case "basic":
		back := stringValue(input.Back)
		if front == "" || back == "" {
			return normalizedCard{}, ErrInvalidBasicCard
		}
		normalized.Front = front
		normalized.Back = back
		return normalized, nil
	case "cloze":
		if front == "" {
			return normalizedCard{}, ErrInvalidCloze
		}

		clozeIndex, ok := clozeIndexForInput(current, input, isCreate)
		if !ok || clozeIndex <= 0 || !slices.Contains(cloze.Indices(front), clozeIndex) {
			return normalizedCard{}, ErrInvalidCloze
		}

		normalized.Front = front
		normalized.Back = ""
		normalized.ClozeIndex = clozeIndex
		normalized.Extra = extraForInput(current, input, isCreate)
		return normalized, nil
	default:
		return normalizedCard{}, ErrInvalidCardType
	}
}

func currentCardType(card *model.Card) string {
	if card == nil || card.CardType == "" {
		return "basic"
	}
	return card.CardType
}

func clozeIndexForInput(current *model.Card, input CardAuthoringInput, isCreate bool) (int, bool) {
	if input.ClozeIndex != nil {
		return *input.ClozeIndex, true
	}
	if isCreate || current == nil {
		return 0, false
	}
	if currentCardType(current) == "cloze" {
		return current.ClozeIndex, true
	}
	return 0, false
}

func extraForInput(current *model.Card, input CardAuthoringInput, isCreate bool) string {
	if input.Extra != nil {
		return *input.Extra
	}
	if isCreate || current == nil {
		return ""
	}
	if currentCardType(current) == "cloze" {
		return current.Extra
	}
	return ""
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func stringPtr(v string) *string {
	return &v
}

func intPtr(v int) *int {
	return &v
}
