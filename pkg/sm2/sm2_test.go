package sm2_test

import (
	"math"
	"testing"
	"time"

	"kpp.dev/kpfc/pkg/sm2"
)

var baseTime = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func TestCalculate_FirstRepetition_GoodQuality(t *testing.T) {
	r := sm2.Calculate(4, 0, 2.5, 1, baseTime)
	if r.Repetitions != 1 {
		t.Errorf("repetitions = %d, want 1", r.Repetitions)
	}
	if r.Interval != 1 {
		t.Errorf("interval = %d, want 1", r.Interval)
	}
	wantDate := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if !r.NextReviewAt.Equal(wantDate) {
		t.Errorf("NextReviewAt = %v, want %v", r.NextReviewAt, wantDate)
	}
}

func TestCalculate_SecondRepetition(t *testing.T) {
	r := sm2.Calculate(4, 1, 2.5, 1, baseTime)
	if r.Repetitions != 2 {
		t.Errorf("repetitions = %d, want 2", r.Repetitions)
	}
	if r.Interval != 6 {
		t.Errorf("interval = %d, want 6", r.Interval)
	}
}

func TestCalculate_ThirdRepetition_UsesEF(t *testing.T) {
	// rep=2, interval=6, EF=2.5, quality=4
	// newEF = 2.5 + (0.1 - 1*(0.08 + 1*0.02)) = 2.5 + (0.1 - 0.10) = 2.5
	// newInterval = round(6 * 2.5) = 15
	r := sm2.Calculate(4, 2, 2.5, 6, baseTime)
	if r.Repetitions != 3 {
		t.Errorf("repetitions = %d, want 3", r.Repetitions)
	}
	if r.Interval != 15 {
		t.Errorf("interval = %d, want 15", r.Interval)
	}
}

func TestCalculate_QualityLessThan3_Resets(t *testing.T) {
	for _, q := range []int{0, 1, 2} {
		r := sm2.Calculate(q, 5, 2.5, 30, baseTime)
		if r.Repetitions != 0 {
			t.Errorf("quality=%d: repetitions = %d, want 0", q, r.Repetitions)
		}
		if r.Interval != 1 {
			t.Errorf("quality=%d: interval = %d, want 1", q, r.Interval)
		}
	}
}

func TestCalculate_QualityAtThreshold(t *testing.T) {
	// quality=3 should advance, not reset
	r := sm2.Calculate(3, 0, 2.5, 1, baseTime)
	if r.Repetitions != 1 {
		t.Errorf("repetitions = %d, want 1", r.Repetitions)
	}
}

func TestCalculate_EaseFactorDecreasesOnBadQuality(t *testing.T) {
	r := sm2.Calculate(2, 3, 2.5, 10, baseTime)
	if r.EaseFactor >= 2.5 {
		t.Errorf("EF = %f, want < 2.5 after quality=2", r.EaseFactor)
	}
}

func TestCalculate_EaseFactorIncreasesOnPerfectQuality(t *testing.T) {
	r := sm2.Calculate(5, 3, 2.5, 10, baseTime)
	if r.EaseFactor <= 2.5 {
		t.Errorf("EF = %f, want > 2.5 after quality=5", r.EaseFactor)
	}
}

func TestCalculate_EaseFactorClampedAt1_3(t *testing.T) {
	// Repeatedly applying quality=0 should clamp at 1.3.
	ef := 1.4
	for i := 0; i < 10; i++ {
		r := sm2.Calculate(0, 0, ef, 1, baseTime)
		ef = r.EaseFactor
	}
	if ef < 1.3 {
		t.Errorf("EF = %f, want >= 1.3 (clamped)", ef)
	}
	if math.Abs(ef-1.3) > 1e-9 {
		t.Errorf("EF = %f, want exactly 1.3 after clamping", ef)
	}
}

func TestCalculate_AllQualities_EFFormula(t *testing.T) {
	for q := 0; q <= 5; q++ {
		ef0 := 2.5
		r := sm2.Calculate(q, 0, ef0, 1, baseTime)
		delta := 0.1 - float64(5-q)*(0.08+float64(5-q)*0.02)
		wantEF := ef0 + delta
		if wantEF < 1.3 {
			wantEF = 1.3
		}
		if math.Abs(r.EaseFactor-wantEF) > 1e-9 {
			t.Errorf("quality=%d: EF = %f, want %f", q, r.EaseFactor, wantEF)
		}
	}
}

func TestCalculate_NextReviewAt_TruncatedToMidnight(t *testing.T) {
	// regardless of time-of-day in 'now', NextReviewAt is midnight UTC
	noon := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)
	r := sm2.Calculate(4, 0, 2.5, 1, noon)
	if r.NextReviewAt.Hour() != 0 || r.NextReviewAt.Minute() != 0 || r.NextReviewAt.Second() != 0 {
		t.Errorf("NextReviewAt not at midnight: %v", r.NextReviewAt)
	}
}

func TestCalculate_IntervalProgression(t *testing.T) {
	// Simulate several perfect-quality reviews and track interval growth.
	rep, interval := 0, 1
	ef := 2.5
	for i := 0; i < 5; i++ {
		r := sm2.Calculate(5, rep, ef, interval, baseTime)
		rep = r.Repetitions
		interval = r.Interval
		ef = r.EaseFactor
	}
	// After 5 perfect reviews the interval should be well above 1.
	if interval <= 6 {
		t.Errorf("interval after 5 perfect reviews = %d, want > 6", interval)
	}
}
