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

func newAccessTestService(t *testing.T) (*AccessService, *gorm.DB) {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Deck{}, &model.Card{}, &model.Media{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return NewAccessService(
		repository.NewGORMDeckRepository(db),
		repository.NewGORMCardRepository(db),
		repository.NewGORMMediaRepository(db),
	), db
}

func seedDeck(t *testing.T, db *gorm.DB, userID uint, isPublic bool) *model.Deck {
	t.Helper()
	deck := &model.Deck{UserID: userID, Title: "Deck", IsPublic: isPublic}
	if err := db.Create(deck).Error; err != nil {
		t.Fatalf("create deck: %v", err)
	}
	return deck
}

func seedCard(t *testing.T, db *gorm.DB, deckID uint) *model.Card {
	t.Helper()
	card := &model.Card{DeckID: deckID, Front: "Front", Back: "Back"}
	if err := db.Create(card).Error; err != nil {
		t.Fatalf("create card: %v", err)
	}
	return card
}

func seedMedia(t *testing.T, db *gorm.DB, userID uint) *model.Media {
	t.Helper()
	media := &model.Media{
		PublicID:    "pubid",
		UserID:      userID,
		Filename:    "file.png",
		ContentType: "image/png",
		Size:        10,
		StoragePath: "1/file.png",
	}
	if err := db.Create(media).Error; err != nil {
		t.Fatalf("create media: %v", err)
	}
	return media
}

func TestAccessService_EditableDeck(t *testing.T) {
	access, db := newAccessTestService(t)
	deck := seedDeck(t, db, 1, false)

	got, err := access.EditableDeck(1, deck.ID)
	if err != nil {
		t.Fatalf("EditableDeck: %v", err)
	}
	if got.ID != deck.ID {
		t.Fatalf("deck id = %d, want %d", got.ID, deck.ID)
	}

	if _, err := access.EditableDeck(2, deck.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong user err = %v, want ErrForbidden", err)
	}
	if _, err := access.EditableDeck(1, 99999); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing deck err = %v, want ErrNotFound", err)
	}
}

func TestAccessService_UpvotableDeck(t *testing.T) {
	access, db := newAccessTestService(t)
	publicDeck := seedDeck(t, db, 1, true)
	privateDeck := seedDeck(t, db, 1, false)

	got, err := access.UpvotableDeck(2, publicDeck.ID)
	if err != nil {
		t.Fatalf("UpvotableDeck: %v", err)
	}
	if got.ID != publicDeck.ID {
		t.Fatalf("deck id = %d, want %d", got.ID, publicDeck.ID)
	}

	if _, err := access.UpvotableDeck(2, privateDeck.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("private deck err = %v, want ErrForbidden", err)
	}
	if _, err := access.UpvotableDeck(1, publicDeck.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("own deck err = %v, want ErrForbidden", err)
	}
	if _, err := access.UpvotableDeck(2, 99999); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing deck err = %v, want ErrNotFound", err)
	}
}

func TestAccessService_CardActions(t *testing.T) {
	access, db := newAccessTestService(t)
	deck := seedDeck(t, db, 1, false)
	card := seedCard(t, db, deck.ID)

	for name, fn := range map[string]func(uint, uint) (*model.Card, error){
		"ReadableCard":   access.ReadableCard,
		"EditableCard":   access.EditableCard,
		"ReviewableCard": access.ReviewableCard,
	} {
		t.Run(name, func(t *testing.T) {
			got, err := fn(1, card.ID)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if got.ID != card.ID {
				t.Fatalf("card id = %d, want %d", got.ID, card.ID)
			}
			if _, err := fn(2, card.ID); !errors.Is(err, ErrForbidden) {
				t.Fatalf("wrong user err = %v, want ErrForbidden", err)
			}
			if _, err := fn(1, 99999); !errors.Is(err, repository.ErrNotFound) {
				t.Fatalf("missing card err = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestAccessService_StudyableDeck(t *testing.T) {
	access, db := newAccessTestService(t)
	deck := seedDeck(t, db, 1, false)

	if _, err := access.StudyableDeck(1, deck.ID); err != nil {
		t.Fatalf("StudyableDeck: %v", err)
	}
	if _, err := access.StudyableDeck(2, deck.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong user err = %v, want ErrForbidden", err)
	}
}

func TestAccessService_DeletableMedia(t *testing.T) {
	access, db := newAccessTestService(t)
	media := seedMedia(t, db, 1)

	got, err := access.DeletableMedia(1, media.PublicID)
	if err != nil {
		t.Fatalf("DeletableMedia: %v", err)
	}
	if got.ID != media.ID {
		t.Fatalf("media id = %d, want %d", got.ID, media.ID)
	}
	if _, err := access.DeletableMedia(2, media.PublicID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("wrong user err = %v, want ErrForbidden", err)
	}
	if _, err := access.DeletableMedia(1, "missing"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("missing media err = %v, want ErrNotFound", err)
	}
}
