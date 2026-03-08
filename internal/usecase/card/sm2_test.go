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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"kpp.dev/kpfc/internal/domain"
)

func TestCalculateNextReview_AgainResetsInterval(t *testing.T) {
	card := &domain.Card{
		Interval:   10,
		EaseFactor: 2.5,
	}

	CalculateNextReview(card, domain.ReviewQualityAgain)

	assert.Equal(t, 1, card.Interval, "quality=Again should reset interval to 1")
	assert.True(t, card.DueDate.After(time.Now()), "due date should be in the future")
}

func TestCalculateNextReview_HardIncreasesSlightly(t *testing.T) {
	card := &domain.Card{
		Interval:   10,
		EaseFactor: 2.5,
	}

	CalculateNextReview(card, domain.ReviewQualityHard)

	assert.Equal(t, 12, card.Interval, "quality=Hard should increase interval by 1.2x")
}

func TestCalculateNextReview_GoodFirstReview(t *testing.T) {
	card := &domain.Card{
		Interval:   1,
		EaseFactor: 2.5,
	}

	CalculateNextReview(card, domain.ReviewQualityGood)

	assert.Equal(t, 6, card.Interval, "quality=Good on first review (interval=1) should go to 6 days")
}

func TestCalculateNextReview_GoodProgression(t *testing.T) {
	card := &domain.Card{
		Interval:   10,
		EaseFactor: 2.5,
	}

	CalculateNextReview(card, domain.ReviewQualityGood)

	assert.Equal(t, 25, card.Interval, "quality=Good should multiply interval by ease factor (10 * 2.5 = 25)")
}

func TestCalculateNextReview_Easy(t *testing.T) {
	card := &domain.Card{
		Interval:   10,
		EaseFactor: 2.5,
	}

	CalculateNextReview(card, domain.ReviewQualityEasy)

	assert.Equal(t, 33, card.Interval, "quality=Easy should multiply by EF * 1.3 (10 * 2.5 * 1.3 = 32.5 -> 33)")
}

func TestCalculateNextReview_EasyFirstReview(t *testing.T) {
	card := &domain.Card{
		Interval:   1,
		EaseFactor: 2.5,
	}

	CalculateNextReview(card, domain.ReviewQualityEasy)

	assert.Equal(t, 6, card.Interval, "quality=Easy on first review should go to 6 days")
}

func TestCalculateEaseFactor_IncreasesOnEasy(t *testing.T) {
	card := &domain.Card{
		Interval:   10,
		EaseFactor: 2.5,
	}

	CalculateNextReview(card, domain.ReviewQualityEasy)

	assert.Greater(t, card.EaseFactor, 2.5, "ease factor should increase on quality=Easy")
}

func TestCalculateEaseFactor_DecreasesOnAgain(t *testing.T) {
	card := &domain.Card{
		Interval:   10,
		EaseFactor: 2.5,
	}

	CalculateNextReview(card, domain.ReviewQualityAgain)

	assert.Less(t, card.EaseFactor, 2.5, "ease factor should decrease on quality=Again")
}

func TestCalculateEaseFactor_MinimumBound(t *testing.T) {
	card := &domain.Card{
		Interval:   10,
		EaseFactor: 1.35,
	}

	// Aplicar várias vezes quality=Again para tentar reduzir abaixo de 1.3
	for i := 0; i < 10; i++ {
		CalculateNextReview(card, domain.ReviewQualityAgain)
	}

	assert.GreaterOrEqual(t, card.EaseFactor, 1.3, "ease factor should never go below 1.3")
}

func TestCalculateNextReview_InvalidQualityDefaultsAgain(t *testing.T) {
	tests := []struct {
		name    string
		quality domain.ReviewQuality
	}{
		{"negative quality", domain.ReviewQuality(-1)},
		{"quality too high", domain.ReviewQuality(10)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := &domain.Card{
				Interval:   10,
				EaseFactor: 2.5,
			}

			CalculateNextReview(card, tt.quality)

			assert.Equal(t, 1, card.Interval, "invalid quality should be treated as Again (interval=1)")
		})
	}
}

func TestCalculateNextReview_DueDateUpdated(t *testing.T) {
	before := time.Now()
	card := &domain.Card{
		Interval:   5,
		EaseFactor: 2.5,
	}

	CalculateNextReview(card, domain.ReviewQualityGood)

	expectedDueDate := before.Add(time.Duration(card.Interval) * 24 * time.Hour)
	assert.WithinDuration(t, expectedDueDate, card.DueDate, 2*time.Second, "due date should be interval days in the future")
}

func TestCalculateNextReview_UpdatedAtSet(t *testing.T) {
	before := time.Now()
	card := &domain.Card{
		Interval:   5,
		EaseFactor: 2.5,
	}

	CalculateNextReview(card, domain.ReviewQualityGood)

	assert.WithinDuration(t, before, card.UpdatedAt, 2*time.Second, "updated_at should be set to now")
}

func TestCalculateNextReview_TableDriven(t *testing.T) {
	tests := []struct {
		name             string
		initialInterval  int
		initialEF        float64
		quality          domain.ReviewQuality
		expectedInterval int
	}{
		{"Again resets", 10, 2.5, domain.ReviewQualityAgain, 1},
		{"Hard slight increase", 10, 2.5, domain.ReviewQualityHard, 12},
		{"Good first review", 1, 2.5, domain.ReviewQualityGood, 6},
		{"Good progression", 10, 2.5, domain.ReviewQualityGood, 25},
		{"Easy first review", 1, 2.5, domain.ReviewQualityEasy, 6},
		{"Easy progression", 10, 2.5, domain.ReviewQualityEasy, 33},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := &domain.Card{
				Interval:   tt.initialInterval,
				EaseFactor: tt.initialEF,
			}

			CalculateNextReview(card, tt.quality)

			assert.Equal(t, tt.expectedInterval, card.Interval)
		})
	}
}
