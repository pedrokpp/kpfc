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

package deck

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"kpp.dev/kpfc/internal/domain"
)

type UseCase struct {
	deckRepo domain.DeckRepository
}

func NewUseCase(deckRepo domain.DeckRepository) *UseCase {
	return &UseCase{
		deckRepo: deckRepo,
	}
}

func (uc *UseCase) CreateDeck(userID, title, description string, isPublic bool) (*domain.Deck, error) {
	if err := validateDeckTitle(title); err != nil {
		return nil, err
	}

	deck := &domain.Deck{
		ID:          uuid.New().String(),
		UserID:      userID,
		Title:       title,
		Description: description,
		IsPublic:    isPublic,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := uc.deckRepo.Create(deck); err != nil {
		return nil, fmt.Errorf("failed to create deck: %w", err)
	}

	return deck, nil
}

func (uc *UseCase) GetDeck(userID, deckID string) (*domain.Deck, error) {
	deck, err := uc.deckRepo.GetByID(deckID)
	if err != nil {
		return nil, err
	}

	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	return deck, nil
}

func (uc *UseCase) GetUserDecks(userID string) ([]*domain.Deck, error) {
	return uc.deckRepo.GetByUserID(userID)
}

func (uc *UseCase) GetPublicDecks() ([]*domain.Deck, error) {
	return uc.deckRepo.GetPublic()
}

func (uc *UseCase) UpdateDeck(userID, deckID, title, description string, isPublic bool) (*domain.Deck, error) {
	if err := validateDeckTitle(title); err != nil {
		return nil, err
	}

	deck, err := uc.deckRepo.GetByID(deckID)
	if err != nil {
		return nil, err
	}

	if deck.UserID != userID {
		return nil, domain.ErrForbidden
	}

	deck.Title = title
	deck.Description = description
	deck.IsPublic = isPublic
	deck.UpdatedAt = time.Now()

	if err := uc.deckRepo.Update(deck); err != nil {
		return nil, fmt.Errorf("failed to update deck: %w", err)
	}

	return deck, nil
}

func (uc *UseCase) DeleteDeck(userID, deckID string) error {
	deck, err := uc.deckRepo.GetByID(deckID)
	if err != nil {
		return err
	}

	if deck.UserID != userID {
		return domain.ErrForbidden
	}

	return uc.deckRepo.Delete(deckID)
}

func (uc *UseCase) CloneDeck(userID, deckID string) (*domain.Deck, error) {
	deck, err := uc.deckRepo.GetByID(deckID)
	if err != nil {
		return nil, err
	}

	if !deck.IsPublic {
		return nil, fmt.Errorf("cannot clone private deck: %w", domain.ErrForbidden)
	}

	return uc.deckRepo.Clone(deckID, userID)
}

func validateDeckTitle(title string) error {
	title = strings.TrimSpace(title)

	if title == "" {
		return fmt.Errorf("title cannot be empty: %w", domain.ErrInvalidInput)
	}

	if len(title) > 1000 {
		return fmt.Errorf("title exceeds 1000 characters: %w", domain.ErrInvalidInput)
	}

	return nil
}
