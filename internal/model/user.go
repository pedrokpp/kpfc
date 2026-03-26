package model

import "time"

type User struct {
	ID            uint      `gorm:"primaryKey"`
	Email         string    `gorm:"uniqueIndex;not null"`
	PasswordHash  string    `gorm:"not null"`
	DisplayName   string
	LoginStreak   int       `gorm:"default:0"`
	LastLoginDate time.Time
	TotalPoints   int       `gorm:"default:0"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
