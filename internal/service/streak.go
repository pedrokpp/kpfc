package service

import "time"

// CalculateStreak computes a new login streak given the last login date,
// the current streak count, and the current time.
//
// Rules:
//   - Zero lastLoginDate (first ever login) → streak = 1
//   - Same calendar day as now → no change
//   - Yesterday → streak + 1
//   - Gap > 1 day → streak reset to 1
func CalculateStreak(lastLoginDate time.Time, currentStreak int, now time.Time) (newStreak int, newDate time.Time) {
	if lastLoginDate.IsZero() {
		return 1, now
	}

	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	lastDate := time.Date(lastLoginDate.Year(), lastLoginDate.Month(), lastLoginDate.Day(), 0, 0, 0, 0, time.UTC)

	diff := nowDate.Sub(lastDate)

	switch diff {
	case 0:
		return currentStreak, lastLoginDate
	case 24 * time.Hour:
		return currentStreak + 1, now
	default:
		return 1, now
	}
}
