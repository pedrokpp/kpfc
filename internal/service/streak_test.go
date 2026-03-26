package service

import (
	"testing"
	"time"
)

func TestCalculateStreak(t *testing.T) {
	now := time.Date(2026, 3, 25, 14, 0, 0, 0, time.UTC)

	tests := []struct {
		name            string
		lastLoginDate   time.Time
		currentStreak   int
		wantStreak      int
		wantDateChanged bool // true if returned date should differ from lastLoginDate
	}{
		{
			name:            "first login (zero time)",
			lastLoginDate:   time.Time{},
			currentStreak:   0,
			wantStreak:      1,
			wantDateChanged: true,
		},
		{
			name:            "same calendar day",
			lastLoginDate:   now.Add(-2 * time.Hour),
			currentStreak:   3,
			wantStreak:      3,
			wantDateChanged: false,
		},
		{
			name:            "yesterday — streak increments",
			lastLoginDate:   now.AddDate(0, 0, -1),
			currentStreak:   4,
			wantStreak:      5,
			wantDateChanged: true,
		},
		{
			name:            "two-day gap — streak resets to 1",
			lastLoginDate:   now.AddDate(0, 0, -2),
			currentStreak:   7,
			wantStreak:      1,
			wantDateChanged: true,
		},
		{
			name:            "multi-day gap — streak resets to 1",
			lastLoginDate:   now.AddDate(0, 0, -30),
			currentStreak:   100,
			wantStreak:      1,
			wantDateChanged: true,
		},
		{
			name:            "consecutive day from streak 1",
			lastLoginDate:   now.AddDate(0, 0, -1),
			currentStreak:   1,
			wantStreak:      2,
			wantDateChanged: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotStreak, gotDate := CalculateStreak(tc.lastLoginDate, tc.currentStreak, now)

			if gotStreak != tc.wantStreak {
				t.Errorf("streak = %d, want %d", gotStreak, tc.wantStreak)
			}

			if tc.wantDateChanged && gotDate.Equal(tc.lastLoginDate) {
				t.Errorf("expected date to change but got same lastLoginDate %v", gotDate)
			}
			if !tc.wantDateChanged && !gotDate.Equal(tc.lastLoginDate) {
				t.Errorf("expected date unchanged but got %v (was %v)", gotDate, tc.lastLoginDate)
			}
		})
	}
}
