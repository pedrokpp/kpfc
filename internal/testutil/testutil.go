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

package testutil

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"kpp.dev/kpfc/internal/domain"
)

// NewTestUser cria um usuário de teste com valores padrão
func NewTestUser(email, name string) *domain.User {
	return &domain.User{
		ID:        uuid.New().String(),
		Email:     email,
		Name:      name,
		Password:  "$2a$10$dummy.hashed.password.value",
		Provider:  domain.AuthProviderLocal,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// NewTestDeck cria um deck de teste com valores padrão
func NewTestDeck(userID, title string, isPublic bool) *domain.Deck {
	return &domain.Deck{
		ID:          uuid.New().String(),
		UserID:      userID,
		Title:       title,
		Description: "Test deck description",
		IsPublic:    isPublic,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// NewTestCard cria um card de teste com valores padrão
func NewTestCard(deckID, front, back string) *domain.Card {
	return &domain.Card{
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
}

// AssertTimeNear verifica se dois timestamps estão próximos (dentro de delta)
func AssertTimeNear(t *testing.T, expected, actual time.Time, delta time.Duration) {
	t.Helper()
	diff := actual.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	if diff > delta {
		t.Errorf("times not near: expected %v, got %v (diff: %v > %v)", expected, actual, diff, delta)
	}
}
