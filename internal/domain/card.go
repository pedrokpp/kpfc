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

package domain

import "time"

type Card struct {
	ID         string    `json:"id"`
	DeckID     string    `json:"deck_id"`
	Front      string    `json:"front"`
	Back       string    `json:"back"`
	Interval   int       `json:"interval"`
	EaseFactor float64   `json:"ease_factor"`
	DueDate    time.Time `json:"due_date"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ReviewQuality int

const (
	ReviewQualityAgain ReviewQuality = iota
	ReviewQualityHard
	ReviewQualityGood
	ReviewQualityEasy
)

type CardRepository interface {
	Create(card *Card) error
	GetByID(id string) (*Card, error)
	GetByDeckID(deckID string) ([]*Card, error)
	GetDueCards(deckID string) ([]*Card, error)
	Update(card *Card) error
	Delete(id string) error
}
