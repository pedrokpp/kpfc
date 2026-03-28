package model

import "time"

// Media tracks a file uploaded by a user and stored via the Storage interface.
type Media struct {
	ID          uint      `gorm:"primaryKey"`
	UserID      uint      `gorm:"not null;index"`
	Filename    string    `gorm:"not null"`
	ContentType string    `gorm:"not null"`
	Size        int64     `gorm:"not null"`
	StoragePath string    `gorm:"not null"` // key passed to Storage.Store/Fetch/Delete
	CreatedAt   time.Time
}
