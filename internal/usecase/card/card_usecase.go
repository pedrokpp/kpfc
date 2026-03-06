// Copyright 2026 kpp.dev
//
// This file is part of kpfc.
//
// kpfc is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// kpfc is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with kpfc. If not, see <https://www.gnu.org/licenses/>.

package card

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"kpp.dev/kpfc/internal/domain"
)

type UseCase struct {
	cardRepo domain.CardRepository
	deckRepo domain.DeckRepository
}

func NewUseCase(cardRepo domain.CardRepository, deckRepo domain.DeckRepository) *UseCase {
	return &UseCase{
		cardRepo: cardRepo,
		deckRepo: deckRepo,
	}
}

func (uc *UseCase) CreateCard(userID, deckID, front, back string) (*domain.Card, error) {
	deck, err := uc.deckRepo.GetByID(deckID)
	if err != nil {
		return nil, err
	}

	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	card := &domain.Card{
		ID:         uuid.New().String(),
		DeckID:     deckID,
		Front:      front,
		Back:       back,
		Interval:   1,
		EaseFactor: 2.5,
		DueDate:    time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := uc.cardRepo.Create(card); err != nil {
		return nil, fmt.Errorf("failed to create card: %w", err)
	}

	return card, nil
}

func (uc *UseCase) GetCard(userID, cardID string) (*domain.Card, error) {
	card, err := uc.cardRepo.GetByID(cardID)
	if err != nil {
		return nil, err
	}

	deck, err := uc.deckRepo.GetByID(card.DeckID)
	if err != nil {
		return nil, err
	}

	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	return card, nil
}

func (uc *UseCase) GetCardsByDeck(userID, deckID string) ([]*domain.Card, error) {
	deck, err := uc.deckRepo.GetByID(deckID)
	if err != nil {
		return nil, err
	}

	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	return uc.cardRepo.GetByDeckID(deckID)
}

func (uc *UseCase) GetDueCards(userID, deckID string) ([]*domain.Card, error) {
	deck, err := uc.deckRepo.GetByID(deckID)
	if err != nil {
		return nil, err
	}

	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	return uc.cardRepo.GetDueCards(deckID)
}

func (uc *UseCase) UpdateCard(userID, cardID, front, back string) (*domain.Card, error) {
	card, err := uc.cardRepo.GetByID(cardID)
	if err != nil {
		return nil, err
	}

	deck, err := uc.deckRepo.GetByID(card.DeckID)
	if err != nil {
		return nil, err
	}

	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	card.Front = front
	card.Back = back
	card.UpdatedAt = time.Now()

	if err := uc.cardRepo.Update(card); err != nil {
		return nil, fmt.Errorf("failed to update card: %w", err)
	}

	return card, nil
}

func (uc *UseCase) DeleteCard(userID, cardID string) error {
	card, err := uc.cardRepo.GetByID(cardID)
	if err != nil {
		return err
	}

	deck, err := uc.deckRepo.GetByID(card.DeckID)
	if err != nil {
		return err
	}

	if deck.UserID != userID {
		return domain.ErrForbidden
	}

	return uc.cardRepo.Delete(cardID)
}

func (uc *UseCase) ReviewCard(userID, cardID string, quality domain.ReviewQuality) (*domain.Card, error) {
	card, err := uc.cardRepo.GetByID(cardID)
	if err != nil {
		return nil, err
	}

	deck, err := uc.deckRepo.GetByID(card.DeckID)
	if err != nil {
		return nil, err
	}

	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	CalculateNextReview(card, quality)

	if err := uc.cardRepo.Update(card); err != nil {
		return nil, fmt.Errorf("failed to update card after review: %w", err)
	}

	return card, nil
}
