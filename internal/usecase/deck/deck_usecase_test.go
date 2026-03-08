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
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"kpp.dev/kpfc/internal/domain"
	"kpp.dev/kpfc/internal/testutil"
)

func TestCreateDeck_EmptyTitle(t *testing.T) {
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(deckRepo)

	_, err := uc.CreateDeck("user1", "", "description", false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput), "empty title should return ErrInvalidInput")
}

func TestCreateDeck_TitleTooLong(t *testing.T) {
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(deckRepo)

	longTitle := strings.Repeat("a", 1001)
	_, err := uc.CreateDeck("user1", longTitle, "description", false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput), "title > 1000 chars should return ErrInvalidInput")
}

func TestCreateDeck_Success(t *testing.T) {
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(deckRepo)

	deckRepo.On("Create", mock.MatchedBy(func(d *domain.Deck) bool {
		return d.UserID == "user1" && d.Title == "My Deck"
	})).Return(nil)

	deck, err := uc.CreateDeck("user1", "My Deck", "description", false)

	assert.NoError(t, err)
	assert.NotNil(t, deck)
	assert.Equal(t, "My Deck", deck.Title)
	assert.Equal(t, "user1", deck.UserID)
	deckRepo.AssertExpectations(t)
}

func TestGetDeck_Forbidden(t *testing.T) {
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(deckRepo)

	deck := testutil.NewTestDeck("user2", "Deck", false)
	deckRepo.On("GetByID", deck.ID).Return(deck, nil)

	_, err := uc.GetDeck("user1", deck.ID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrForbidden), "getting another user's private deck should return ErrForbidden")
	deckRepo.AssertExpectations(t)
}

func TestUpdateDeck_EmptyTitle(t *testing.T) {
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(deckRepo)

	_, err := uc.UpdateDeck("user1", "deck1", "", "description", false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidInput))
}

func TestUpdateDeck_Forbidden(t *testing.T) {
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(deckRepo)

	deck := testutil.NewTestDeck("user2", "Deck", false)
	deckRepo.On("GetByID", deck.ID).Return(deck, nil)

	_, err := uc.UpdateDeck("user1", deck.ID, "New Title", "description", false)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrForbidden), "updating another user's deck should return ErrForbidden")
	deckRepo.AssertExpectations(t)
}

func TestDeleteDeck_Forbidden(t *testing.T) {
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(deckRepo)

	deck := testutil.NewTestDeck("user2", "Deck", false)
	deckRepo.On("GetByID", deck.ID).Return(deck, nil)

	err := uc.DeleteDeck("user1", deck.ID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrForbidden), "deleting another user's deck should return ErrForbidden")
	deckRepo.AssertExpectations(t)
}

func TestCloneDeck_PublicDeckSuccess(t *testing.T) {
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(deckRepo)

	publicDeck := testutil.NewTestDeck("user2", "Public Deck", true)
	clonedDeck := testutil.NewTestDeck("user1", "Public Deck", false)

	deckRepo.On("GetByID", publicDeck.ID).Return(publicDeck, nil)
	deckRepo.On("Clone", publicDeck.ID, "user1").Return(clonedDeck, nil)

	result, err := uc.CloneDeck("user1", publicDeck.ID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "user1", result.UserID, "cloned deck should belong to user1")
	deckRepo.AssertExpectations(t)
}

func TestCloneDeck_PrivateDeckForbidden(t *testing.T) {
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(deckRepo)

	privateDeck := testutil.NewTestDeck("user2", "Private Deck", false)
	deckRepo.On("GetByID", privateDeck.ID).Return(privateDeck, nil)

	_, err := uc.CloneDeck("user1", privateDeck.ID)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrForbidden), "cloning private deck should return ErrForbidden")
	deckRepo.AssertExpectations(t)
}

func TestGetPublicDecks_Success(t *testing.T) {
	deckRepo := &testutil.MockDeckRepository{}
	uc := NewUseCase(deckRepo)

	publicDecks := []*domain.Deck{
		testutil.NewTestDeck("user1", "Public 1", true),
		testutil.NewTestDeck("user2", "Public 2", true),
	}

	deckRepo.On("GetPublic").Return(publicDecks, nil)

	decks, err := uc.GetPublicDecks()

	assert.NoError(t, err)
	assert.Len(t, decks, 2)
	deckRepo.AssertExpectations(t)
}
