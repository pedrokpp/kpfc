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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"kpp.dev/kpfc/internal/domain"
	"kpp.dev/kpfc/internal/testutil"
)

func TestCreateCard_EmptyFront(t *testing.T) {
	cardRepo := &testutil.MockCardRepository{}
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(cardRepo, deckRepo)

	_, err := uc.CreateCard("user1", "deck1", "", "back")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput), "empty front should return ErrInvalidInput")
}

func TestCreateCard_EmptyBack(t *testing.T) {
	cardRepo := &testutil.MockCardRepository{}
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(cardRepo, deckRepo)

	_, err := uc.CreateCard("user1", "deck1", "front", "")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput), "empty back should return ErrInvalidInput")
}

func TestCreateCard_FrontTooLong(t *testing.T) {
	cardRepo := &testutil.MockCardRepository{}
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(cardRepo, deckRepo)

	longFront := strings.Repeat("a", 1001)
	_, err := uc.CreateCard("user1", "deck1", longFront, "back")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput), "front > 1000 chars should return ErrInvalidInput")
}

func TestCreateCard_BackTooLong(t *testing.T) {
	cardRepo := &testutil.MockCardRepository{}
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(cardRepo, deckRepo)

	longBack := strings.Repeat("b", 1001)
	_, err := uc.CreateCard("user1", "deck1", "front", longBack)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput), "back > 1000 chars should return ErrInvalidInput")
}

func TestCreateCard_Forbidden(t *testing.T) {
	cardRepo := &testutil.MockCardRepository{}
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(cardRepo, deckRepo)

	deck := testutil.NewTestDeck("user2", "Deck", false)
	deckRepo.On("GetByID", deck.ID).Return(deck, nil)

	_, err := uc.CreateCard("user1", deck.ID, "front", "back")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrForbidden), "creating card in another user's deck should return ErrForbidden")
	deckRepo.AssertExpectations(t)
}

func TestCreateCard_Success(t *testing.T) {
	cardRepo := &testutil.MockCardRepository{}
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(cardRepo, deckRepo)

	deck := testutil.NewTestDeck("user1", "Deck", false)
	deckRepo.On("GetByID", deck.ID).Return(deck, nil)
	cardRepo.On("Create", mock.MatchedBy(func(c *domain.Card) bool {
		return c.DeckID == deck.ID && c.Front == "front" && c.Back == "back"
	})).Return(nil)

	card, err := uc.CreateCard("user1", deck.ID, "front", "back")

	assert.NoError(t, err)
	assert.NotNil(t, card)
	assert.Equal(t, "front", card.Front)
	assert.Equal(t, "back", card.Back)
	assert.Equal(t, 1, card.Interval)
	assert.Equal(t, 2.5, card.EaseFactor)
	deckRepo.AssertExpectations(t)
	cardRepo.AssertExpectations(t)
}

func TestGetCard_Forbidden(t *testing.T) {
	cardRepo := &testutil.MockCardRepository{}
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(cardRepo, deckRepo)

	deck := testutil.NewTestDeck("user2", "Deck", false)
	card := testutil.NewTestCard(deck.ID, "front", "back")

	cardRepo.On("GetByID", card.ID).Return(card, nil)
	deckRepo.On("GetByID", deck.ID).Return(deck, nil)

	_, err := uc.GetCard("user1", card.ID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrForbidden), "getting card from another user's deck should return ErrForbidden")
	cardRepo.AssertExpectations(t)
	deckRepo.AssertExpectations(t)
}

func TestUpdateCard_EmptyFront(t *testing.T) {
	cardRepo := &testutil.MockCardRepository{}
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(cardRepo, deckRepo)

	_, err := uc.UpdateCard("user1", "card1", "", "back")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
}

func TestUpdateCard_Forbidden(t *testing.T) {
	cardRepo := &testutil.MockCardRepository{}
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(cardRepo, deckRepo)

	deck := testutil.NewTestDeck("user2", "Deck", false)
	card := testutil.NewTestCard(deck.ID, "front", "back")

	cardRepo.On("GetByID", card.ID).Return(card, nil)
	deckRepo.On("GetByID", deck.ID).Return(deck, nil)

	_, err := uc.UpdateCard("user1", card.ID, "new front", "new back")

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrForbidden), "updating card from another user's deck should return ErrForbidden")
	cardRepo.AssertExpectations(t)
	deckRepo.AssertExpectations(t)
}

func TestDeleteCard_Forbidden(t *testing.T) {
	cardRepo := &testutil.MockCardRepository{}
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(cardRepo, deckRepo)

	deck := testutil.NewTestDeck("user2", "Deck", false)
	card := testutil.NewTestCard(deck.ID, "front", "back")

	cardRepo.On("GetByID", card.ID).Return(card, nil)
	deckRepo.On("GetByID", deck.ID).Return(deck, nil)

	err := uc.DeleteCard("user1", card.ID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrForbidden), "deleting card from another user's deck should return ErrForbidden")
	cardRepo.AssertExpectations(t)
	deckRepo.AssertExpectations(t)
}

func TestReviewCard_Forbidden(t *testing.T) {
	cardRepo := &testutil.MockCardRepository{}
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(cardRepo, deckRepo)

	deck := testutil.NewTestDeck("user2", "Deck", false)
	card := testutil.NewTestCard(deck.ID, "front", "back")

	cardRepo.On("GetByID", card.ID).Return(card, nil)
	deckRepo.On("GetByID", deck.ID).Return(deck, nil)

	_, err := uc.ReviewCard("user1", card.ID, domain.ReviewQualityGood)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrForbidden), "reviewing card from another user's deck should return ErrForbidden")
	cardRepo.AssertExpectations(t)
	deckRepo.AssertExpectations(t)
}

func TestReviewCard_AppliesSM2(t *testing.T) {
	cardRepo := &testutil.MockCardRepository{}
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(cardRepo, deckRepo)

	deck := testutil.NewTestDeck("user1", "Deck", false)
	card := testutil.NewTestCard(deck.ID, "front", "back")
	card.Interval = 1
	card.EaseFactor = 2.5

	cardRepo.On("GetByID", card.ID).Return(card, nil)
	deckRepo.On("GetByID", deck.ID).Return(deck, nil)
	cardRepo.On("Update", mock.MatchedBy(func(c *domain.Card) bool {
		return c.ID == card.ID && c.Interval == 6
	})).Return(nil)

	result, err := uc.ReviewCard("user1", card.ID, domain.ReviewQualityGood)

	assert.NoError(t, err)
	assert.Equal(t, 6, result.Interval, "quality=Good on first review should set interval to 6")
	cardRepo.AssertExpectations(t)
	deckRepo.AssertExpectations(t)
}

func TestGetDueCards_OnlyDueReturned(t *testing.T) {
	cardRepo := &testutil.MockCardRepository{}
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(cardRepo, deckRepo)

	deck := testutil.NewTestDeck("user1", "Deck", false)

	dueCard1 := testutil.NewTestCard(deck.ID, "due1", "back1")
	dueCard1.DueDate = time.Now().Add(-1 * time.Hour)

	dueCard2 := testutil.NewTestCard(deck.ID, "due2", "back2")
	dueCard2.DueDate = time.Now().Add(-1 * time.Minute)

	deckRepo.On("GetByID", deck.ID).Return(deck, nil)
	cardRepo.On("GetDueCards", deck.ID).Return([]*domain.Card{dueCard1, dueCard2}, nil)

	cards, err := uc.GetDueCards("user1", deck.ID)

	assert.NoError(t, err)
	assert.Len(t, cards, 2)
	deckRepo.AssertExpectations(t)
	cardRepo.AssertExpectations(t)
}
