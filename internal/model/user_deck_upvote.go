package model

// UserDeckUpvote is the join table enforcing one upvote per user per deck.
type UserDeckUpvote struct {
	UserID uint `gorm:"primaryKey"`
	DeckID uint `gorm:"primaryKey"`
}
