package model

import "time"

type Card struct {
	ID           uint      `gorm:"primaryKey"`
	DeckID       uint      `gorm:"not null;index"`
	Front        string    `gorm:"not null"`
	Back         string    `gorm:"not null"`
	Interval     int       `gorm:"default:1"`
	Repetitions  int       `gorm:"default:0"`
	EaseFactor   float64   `gorm:"default:2.5"`
	NextReviewAt time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
