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
	"io"

	"github.com/stretchr/testify/mock"

	"kpp.dev/kpfc/internal/domain"
)

// MockUserRepository é um mock do UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(id string) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*domain.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) Update(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// MockDeckRepository é um mock do DeckRepository
type MockDeckRepository struct {
	mock.Mock
}

func (m *MockDeckRepository) Create(deck *domain.Deck) error {
	args := m.Called(deck)
	return args.Error(0)
}

func (m *MockDeckRepository) GetByID(id string) (*domain.Deck, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Deck), args.Error(1)
}

func (m *MockDeckRepository) GetByUserID(userID string) ([]*domain.Deck, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Deck), args.Error(1)
}

func (m *MockDeckRepository) GetPublic() ([]*domain.Deck, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Deck), args.Error(1)
}

func (m *MockDeckRepository) Update(deck *domain.Deck) error {
	args := m.Called(deck)
	return args.Error(0)
}

func (m *MockDeckRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDeckRepository) Clone(deckID, newUserID string) (*domain.Deck, error) {
	args := m.Called(deckID, newUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Deck), args.Error(1)
}

// MockCardRepository é um mock do CardRepository
type MockCardRepository struct {
	mock.Mock
}

func (m *MockCardRepository) Create(card *domain.Card) error {
	args := m.Called(card)
	return args.Error(0)
}

func (m *MockCardRepository) GetByID(id string) (*domain.Card, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Card), args.Error(1)
}

func (m *MockCardRepository) GetByDeckID(deckID string) ([]*domain.Card, error) {
	args := m.Called(deckID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Card), args.Error(1)
}

func (m *MockCardRepository) GetDueCards(deckID string) ([]*domain.Card, error) {
	args := m.Called(deckID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Card), args.Error(1)
}

func (m *MockCardRepository) Update(card *domain.Card) error {
	args := m.Called(card)
	return args.Error(0)
}

func (m *MockCardRepository) Delete(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

// MockStorageRepository é um mock do StorageRepository
type MockStorageRepository struct {
	mock.Mock
}

func (m *MockStorageRepository) Upload(key string, data io.Reader, contentType string) (string, error) {
	args := m.Called(key, data, contentType)
	return args.String(0), args.Error(1)
}

func (m *MockStorageRepository) Delete(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockStorageRepository) GetURL(key string) string {
	args := m.Called(key)
	return args.String(0)
}
