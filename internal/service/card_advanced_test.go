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

func seedBasicCard(t *testing.T, db *gorm.DB, deckID uint) *model.Card {
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
		t.Fatalf("create basic card: %v", err)
	}
	return card
}

func seedClozeCard(t *testing.T, db *gorm.DB, deckID uint) *model.Card {
	t.Helper()
	card := &model.Card{
		DeckID:     deckID,
		Title:      "Cloze",
		Front:      "{{c2::Paris}} is the capital of France",
		Back:       "",
		CardType:   "cloze",
		ClozeIndex: 2,
		Extra:      "geo",
		EaseFactor: 2.5,
		Interval:   1,
	}
	if err := db.Create(card).Error; err != nil {
		t.Fatalf("create cloze card: %v", err)
	}
	return card
}

func TestCardService_CreateAuthored_DefaultsToBasic(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)

	got, err := svc.CreateAuthored(1, deck.ID, CardAuthoringInput{
		Title: stringPtr("Capitals"),
		Front: stringPtr("Q"),
		Back:  stringPtr("A"),
	})
	if err != nil {
		t.Fatalf("CreateAuthored: %v", err)
	}
	if got.CardType != "basic" {
		t.Fatalf("card_type = %q, want basic", got.CardType)
	}
	if got.ClozeIndex != 0 {
		t.Fatalf("cloze_index = %d, want 0", got.ClozeIndex)
	}
	if got.Extra != "" {
		t.Fatalf("extra = %q, want empty", got.Extra)
	}
}

func TestCardService_CreateAuthored_ClozeNormalizesShape(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)

	got, err := svc.CreateAuthored(1, deck.ID, CardAuthoringInput{
		Title:      stringPtr("Capitals"),
		Front:      stringPtr("{{c2::Paris}} is the capital of France"),
		Back:       stringPtr("ignored"),
		CardType:   stringPtr("cloze"),
		ClozeIndex: intPtr(2),
		Extra:      stringPtr("geo"),
	})
	if err != nil {
		t.Fatalf("CreateAuthored: %v", err)
	}
	if got.CardType != "cloze" {
		t.Fatalf("card_type = %q, want cloze", got.CardType)
	}
	if got.Back != "" {
		t.Fatalf("back = %q, want empty", got.Back)
	}
	if got.ClozeIndex != 2 {
		t.Fatalf("cloze_index = %d, want 2", got.ClozeIndex)
	}
	if got.Extra != "geo" {
		t.Fatalf("extra = %q, want geo", got.Extra)
	}
}

func TestCardService_CreateAuthored_BasicClearsIncompatibleFields(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)

	got, err := svc.CreateAuthored(1, deck.ID, CardAuthoringInput{
		Front:      stringPtr("Q"),
		Back:       stringPtr("A"),
		CardType:   stringPtr("basic"),
		ClozeIndex: intPtr(7),
		Extra:      stringPtr("ignored"),
	})
	if err != nil {
		t.Fatalf("CreateAuthored: %v", err)
	}
	if got.ClozeIndex != 0 {
		t.Fatalf("cloze_index = %d, want 0", got.ClozeIndex)
	}
	if got.Extra != "" {
		t.Fatalf("extra = %q, want empty", got.Extra)
	}
}

func TestCardService_CreateAuthored_InvalidInputs(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)

	tests := []struct {
		name  string
		input CardAuthoringInput
		want  error
	}{
		{
			name: "invalid type",
			input: CardAuthoringInput{
				Front:    stringPtr("Q"),
				Back:     stringPtr("A"),
				CardType: stringPtr("essay"),
			},
			want: ErrInvalidCardType,
		},
		{
			name: "basic missing back",
			input: CardAuthoringInput{
				Front: stringPtr("Q"),
			},
			want: ErrInvalidBasicCard,
		},
		{
			name: "cloze missing index",
			input: CardAuthoringInput{
				Front:    stringPtr("{{c1::Paris}}"),
				CardType: stringPtr("cloze"),
			},
			want: ErrInvalidCloze,
		},
		{
			name: "cloze mismatched index",
			input: CardAuthoringInput{
				Front:      stringPtr("{{c1::Paris}}"),
				CardType:   stringPtr("cloze"),
				ClozeIndex: intPtr(2),
			},
			want: ErrInvalidCloze,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateAuthored(1, deck.ID, tc.input)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCardService_UpdateAuthored_PreservesTypeWhenOmitted(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)
	basic := seedBasicCard(t, db, deck.ID)
	cloze := seedClozeCard(t, db, deck.ID)

	gotBasic, err := svc.UpdateAuthored(1, basic.ID, CardAuthoringInput{
		Title: stringPtr("Updated"),
		Front: stringPtr("Updated Q"),
		Back:  stringPtr("Updated A"),
	})
	if err != nil {
		t.Fatalf("UpdateAuthored basic: %v", err)
	}
	if gotBasic.CardType != "basic" {
		t.Fatalf("basic card_type = %q, want basic", gotBasic.CardType)
	}

	gotCloze, err := svc.UpdateAuthored(1, cloze.ID, CardAuthoringInput{
		Title: stringPtr("Updated"),
		Front: stringPtr("{{c2::Paris}} remains the capital of France"),
	})
	if err != nil {
		t.Fatalf("UpdateAuthored cloze: %v", err)
	}
	if gotCloze.CardType != "cloze" {
		t.Fatalf("cloze card_type = %q, want cloze", gotCloze.CardType)
	}
	if gotCloze.ClozeIndex != 2 {
		t.Fatalf("cloze_index = %d, want 2", gotCloze.ClozeIndex)
	}
}

func TestCardService_UpdateAuthored_BasicToCloze(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)
	card := seedBasicCard(t, db, deck.ID)

	got, err := svc.UpdateAuthored(1, card.ID, CardAuthoringInput{
		Title:      stringPtr("Capitals"),
		Front:      stringPtr("{{c1::Paris}} is the capital of France"),
		CardType:   stringPtr("cloze"),
		ClozeIndex: intPtr(1),
		Extra:      stringPtr("geo"),
	})
	if err != nil {
		t.Fatalf("UpdateAuthored: %v", err)
	}
	if got.CardType != "cloze" {
		t.Fatalf("card_type = %q, want cloze", got.CardType)
	}
	if got.Back != "" {
		t.Fatalf("back = %q, want empty", got.Back)
	}
	if got.Extra != "geo" {
		t.Fatalf("extra = %q, want geo", got.Extra)
	}
}

func TestCardService_UpdateAuthored_BasicToCloze_RequiresIndex(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)
	card := seedBasicCard(t, db, deck.ID)

	_, err := svc.UpdateAuthored(1, card.ID, CardAuthoringInput{
		Front:    stringPtr("{{c1::Paris}} is the capital of France"),
		CardType: stringPtr("cloze"),
	})
	if !errors.Is(err, ErrInvalidCloze) {
		t.Fatalf("err = %v, want ErrInvalidCloze", err)
	}
}

func TestCardService_UpdateAuthored_ClozeToBasic(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)
	card := seedClozeCard(t, db, deck.ID)

	got, err := svc.UpdateAuthored(1, card.ID, CardAuthoringInput{
		Title:    stringPtr("Basic"),
		Front:    stringPtr("Question"),
		Back:     stringPtr("Answer"),
		CardType: stringPtr("basic"),
		Extra:    stringPtr("ignored"),
	})
	if err != nil {
		t.Fatalf("UpdateAuthored: %v", err)
	}
	if got.CardType != "basic" {
		t.Fatalf("card_type = %q, want basic", got.CardType)
	}
	if got.ClozeIndex != 0 {
		t.Fatalf("cloze_index = %d, want 0", got.ClozeIndex)
	}
	if got.Extra != "" {
		t.Fatalf("extra = %q, want empty", got.Extra)
	}
}

func TestCardService_UpdateAuthored_ClozeToBasic_RequiresBack(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)
	card := seedClozeCard(t, db, deck.ID)

	_, err := svc.UpdateAuthored(1, card.ID, CardAuthoringInput{
		Front:    stringPtr("Question"),
		CardType: stringPtr("basic"),
	})
	if !errors.Is(err, ErrInvalidBasicCard) {
		t.Fatalf("err = %v, want ErrInvalidBasicCard", err)
	}
}

func TestCardService_UpdateAuthored_ClozeOmitsPreserveClozeFields(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)
	card := seedClozeCard(t, db, deck.ID)

	got, err := svc.UpdateAuthored(1, card.ID, CardAuthoringInput{
		Title: stringPtr("Updated"),
		Front: stringPtr("{{c2::Paris}} is still the capital of France"),
	})
	if err != nil {
		t.Fatalf("UpdateAuthored: %v", err)
	}
	if got.ClozeIndex != 2 {
		t.Fatalf("cloze_index = %d, want 2", got.ClozeIndex)
	}
	if got.Extra != "geo" {
		t.Fatalf("extra = %q, want geo", got.Extra)
	}
}

func TestCardService_UpdateAuthored_BasicAlwaysClearsClozeFields(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)
	card := seedBasicCard(t, db, deck.ID)

	if err := db.Model(card).Updates(map[string]any{
		"cloze_index": 3,
		"extra":       "stale",
	}).Error; err != nil {
		t.Fatalf("seed stray fields: %v", err)
	}

	got, err := svc.UpdateAuthored(1, card.ID, CardAuthoringInput{
		Title: stringPtr("Updated"),
		Front: stringPtr("Updated Q"),
		Back:  stringPtr("Updated A"),
	})
	if err != nil {
		t.Fatalf("UpdateAuthored: %v", err)
	}
	if got.ClozeIndex != 0 {
		t.Fatalf("cloze_index = %d, want 0", got.ClozeIndex)
	}
	if got.Extra != "" {
		t.Fatalf("extra = %q, want empty", got.Extra)
	}
}

func TestCardService_UpdateAdvanced_Forbidden(t *testing.T) {
	svc, db := newAdvancedCardService(t)
	deck := seedAdvancedDeck(t, db, 1)
	card := seedBasicCard(t, db, deck.ID)

	_, err := svc.UpdateAdvanced(2, card.ID, CardCreateOpts{
		Front:      "{{c1::Paris}}",
		CardType:   "cloze",
		ClozeIndex: 1,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}
