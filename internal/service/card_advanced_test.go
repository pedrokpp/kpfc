package service

import (
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
)

func newAdvancedCardService(t *testing.T) (*CardService, *gorm.DB) {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Deck{}, &model.Card{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	deckRepo := repository.NewGORMDeckRepository(db)
	cardRepo := repository.NewGORMCardRepository(db)
	accessSvc := NewAccessService(deckRepo, cardRepo, nil)
	return NewCardService(cardRepo, accessSvc), db
}

func seedAdvancedDeck(t *testing.T, db *gorm.DB, userID uint) *model.Deck {
	t.Helper()
	deck := &model.Deck{UserID: userID, Title: "Deck"}
	if err := db.Create(deck).Error; err != nil {
		t.Fatalf("create deck: %v", err)
	}
	return deck
}

func seedAdvancedCard(t *testing.T, db *gorm.DB, deckID uint) *model.Card {
	t.Helper()
	card := &model.Card{
		DeckID:     deckID,
		Title:      "Basic",
		Front:      "Front",
		Back:       "Back",
		CardType:   "basic",
		EaseFactor: 2.5,
		Interval:   1,
	}
	if err := db.Create(card).Error; err != nil {
		t.Fatalf("create card: %v", err)
	}
	return card
}

func TestCardService_UpdateAdvanced_Success(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)
	card := seedAdvancedCard(t, db, deck.ID)

	got, err := svc.UpdateAdvanced(1, card.ID, CardCreateOpts{
		Title:      "Capitals",
		Front:      "{{c2::Paris}} is the capital of France",
		CardType:   "cloze",
		ClozeIndex: 2,
		Extra:      "geo",
	})
	if err != nil {
		t.Fatalf("UpdateAdvanced: %v", err)
	}
	if got.Title != "Capitals" {
		t.Fatalf("title = %q, want Capitals", got.Title)
	}
	if got.CardType != "cloze" {
		t.Fatalf("card_type = %q, want cloze", got.CardType)
	}
	if got.ClozeIndex != 2 {
		t.Fatalf("cloze_index = %d, want 2", got.ClozeIndex)
	}
	if got.Extra != "geo" {
		t.Fatalf("extra = %q, want geo", got.Extra)
	}
	if got.Back != "" {
		t.Fatalf("back = %q, want empty for cloze update", got.Back)
	}
}

func TestCardService_UpdateAdvanced_InvalidCloze(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)
	card := seedAdvancedCard(t, db, deck.ID)

	_, err := svc.UpdateAdvanced(1, card.ID, CardCreateOpts{
		Front:      "plain text",
		CardType:   "cloze",
		ClozeIndex: 1,
	})
	if !errors.Is(err, ErrInvalidCloze) {
		t.Fatalf("err = %v, want ErrInvalidCloze", err)
	}
}

func TestCardService_UpdateAdvanced_InvalidCardType(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)
	card := seedAdvancedCard(t, db, deck.ID)

	_, err := svc.UpdateAdvanced(1, card.ID, CardCreateOpts{
		Front:    "Q",
		Back:     "A",
		CardType: "essay",
	})
	if !errors.Is(err, ErrInvalidCardType) {
		t.Fatalf("err = %v, want ErrInvalidCardType", err)
	}
}

func TestCardService_UpdateAdvanced_Forbidden(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)
	card := seedAdvancedCard(t, db, deck.ID)

	_, err := svc.UpdateAdvanced(2, card.ID, CardCreateOpts{
		Front:      "{{c1::Paris}}",
		CardType:   "cloze",
		ClozeIndex: 1,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}
