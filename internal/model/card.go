package model

import "time"

type Card struct {
	ID           uint      `gorm:"primaryKey"`
	DeckID       uint      `gorm:"not null;index"`
	Title        string
	Front        string    `gorm:"not null"`
	Back         string    `gorm:"not null"`
	CardType     string    `gorm:"default:'basic';not null"`
	ClozeIndex   int       `gorm:"default:0"`
	Extra        string
	Interval     int       `gorm:"default:1"`
	Repetitions  int       `gorm:"default:0"`
	EaseFactor   float64   `gorm:"default:2.5"`
	NextReviewAt time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
