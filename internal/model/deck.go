package model

import "time"

type Deck struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uint      `gorm:"not null;index"`
	Title       string    `gorm:"not null"`
	Description string
	IsPublic    bool      `gorm:"default:false"`
	UpvoteCount int       `gorm:"default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
