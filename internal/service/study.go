package service

import (
	"errors"
	"math/rand"
	"time"

	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
	"kpp.dev/kpfc/pkg/sm2"
)

// ErrInvalidQuality is returned when a review quality value is outside [0, 5].
var ErrInvalidQuality = errors.New("quality must be between 0 and 5")

// StudyService manages study sessions and SM-2 review submissions.
type StudyService struct {
	cards repository.CardRepository
	decks repository.DeckRepository
	users repository.UserRepository
}

func NewStudyService(
	cards repository.CardRepository,
	decks repository.DeckRepository,
	users repository.UserRepository,
) *StudyService {
	return &StudyService{cards: cards, decks: decks, users: users}
}

// StartSession returns the cards to review for the given mode.
//
// mode "spaced": cards whose NextReviewAt is on or before now.
// mode "random": all cards in the deck, shuffled.
//
// Returns ErrForbidden if userID does not own deckID.
func (s *StudyService) StartSession(userID, deckID uint, mode string) ([]model.Card, error) {
	deck, err := s.decks.FindByID(deckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, ErrForbidden
	}

	var cards []model.Card
	switch mode {
	case "spaced":
		cards, err = s.cards.FindDueCards(deckID, time.Now())
		if err != nil {
			return nil, err
		}
	case "random":
		cards, err = s.cards.FindByDeckID(deckID)
		if err != nil {
			return nil, err
		}
		rand.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	default:
		return nil, errors.New("mode must be \"spaced\" or \"random\"")
	}

	return cards, nil
}

// SubmitReview applies SM-2 to the card and persists the updated state.
// Returns ErrForbidden if userID does not own the card's deck.
// Returns ErrInvalidQuality if quality is outside [0, 5].
func (s *StudyService) SubmitReview(userID, cardID uint, quality int) (*model.Card, error) {
	if quality < 0 || quality > 5 {
		return nil, ErrInvalidQuality
	}

	card, err := s.cards.FindByID(cardID)
	if err != nil {
		return nil, err
	}
	deck, err := s.decks.FindByID(card.DeckID)
	if err != nil {
		return nil, err
	}
	if deck.UserID != userID {
		return nil, ErrForbidden
	}

	result := sm2.Calculate(quality, card.Repetitions, card.EaseFactor, card.Interval, time.Now())
	card.Repetitions = result.Repetitions
	card.EaseFactor = result.EaseFactor
	card.Interval = result.Interval
	card.NextReviewAt = result.NextReviewAt

	if err := s.cards.Update(card); err != nil {
		return nil, err
	}
	return card, nil
}

// CompleteSession adds points to the user's total score.
func (s *StudyService) CompleteSession(userID uint, points int) error {
	user, err := s.users.FindByID(userID)
	if err != nil {
		return err
	}
	user.TotalPoints += points
	return s.users.Update(user)
}
