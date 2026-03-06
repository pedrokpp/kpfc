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
	"math"
	"time"

	"kpp.dev/kpfc/internal/domain"
)

// CalculateNextReview calcula o próximo intervalo e ease factor baseado no algoritmo SM-2
// quality 0 (Again): reseta intervalo para 1
// quality 1 (Hard): diminui intervalo
// quality 2 (Good): mantém progressão normal
// quality 3 (Easy): aumenta intervalo mais rapidamente
func CalculateNextReview(card *domain.Card, quality domain.ReviewQuality) {
	if quality < domain.ReviewQualityAgain || quality > domain.ReviewQualityEasy {
		quality = domain.ReviewQualityAgain
	}

	card.EaseFactor = calculateEaseFactor(card.EaseFactor, quality)

	switch quality {
	case domain.ReviewQualityAgain:
		card.Interval = 1
	case domain.ReviewQualityHard:
		card.Interval = int(math.Max(1, float64(card.Interval)*1.2))
	case domain.ReviewQualityGood:
		if card.Interval == 1 {
			card.Interval = 6
		} else {
			card.Interval = int(float64(card.Interval) * card.EaseFactor)
		}
	case domain.ReviewQualityEasy:
		if card.Interval == 1 {
			card.Interval = 6
		} else {
			card.Interval = int(float64(card.Interval) * card.EaseFactor * 1.3)
		}
	}

	card.DueDate = time.Now().Add(time.Duration(card.Interval) * 24 * time.Hour)
	card.UpdatedAt = time.Now()
}

func calculateEaseFactor(currentEF float64, quality domain.ReviewQuality) float64 {
	q := float64(quality)
	newEF := currentEF + (0.1 - (3.0-q)*(0.08+(3.0-q)*0.02))

	if newEF < 1.3 {
		return 1.3
	}

	return newEF
}
