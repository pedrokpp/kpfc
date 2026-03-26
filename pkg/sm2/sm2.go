// Package sm2 implements the SM-2 spaced repetition algorithm.
package sm2

import (
	"math"
	"time"
)

// Result holds the updated SM-2 state after a review.
type Result struct {
	Repetitions  int
	EaseFactor   float64
	Interval     int
	NextReviewAt time.Time
}

// Calculate computes the new SM-2 state given a review quality (0–5).
//
// quality < 3: reset repetitions to 0 and interval to 1 (relearn).
// quality >= 3: advance the repetition counter and extend the interval.
// EaseFactor is always updated and clamped to a minimum of 1.3.
// NextReviewAt is set to midnight UTC of (now + interval days).
func Calculate(quality, repetitions int, easeFactor float64, interval int, now time.Time) Result {
	// Update ease factor (applies regardless of quality).
	delta := 0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02)
	newEF := easeFactor + delta
	if newEF < 1.3 {
		newEF = 1.3
	}

	var newRep, newInterval int

	if quality < 3 {
		newRep = 0
		newInterval = 1
	} else {
		newRep = repetitions + 1
		switch newRep {
		case 1:
			newInterval = 1
		case 2:
			newInterval = 6
		default:
			newInterval = int(math.Round(float64(interval) * newEF))
		}
	}

	nextReview := now.UTC().Truncate(24 * time.Hour).AddDate(0, 0, newInterval)

	return Result{
		Repetitions:  newRep,
		EaseFactor:   newEF,
		Interval:     newInterval,
		NextReviewAt: nextReview,
	}
}
