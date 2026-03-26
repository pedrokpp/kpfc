package repository_test

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
)

// openTestDB opens an in-memory SQLite database and runs AutoMigrate.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Deck{},
		&model.Card{},
		&model.UserDeckUpvote{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	// Clean slate for each test.
	t.Cleanup(func() {
		db.Exec("DELETE FROM user_deck_upvotes")
		db.Exec("DELETE FROM cards")
		db.Exec("DELETE FROM decks")
		db.Exec("DELETE FROM users")
	})
	return db
}

// -- UserRepository -----------------------------------------------------------

func TestUserRepository_CreateAndFind(t *testing.T) {
	repo := repository.NewGORMUserRepository(openTestDB(t))

	user := &model.User{Email: "a@example.com", PasswordHash: "hash", DisplayName: "Alice"}
	if err := repo.Create(user); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	got, err := repo.FindByID(user.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Email != user.Email {
		t.Errorf("email: got %q, want %q", got.Email, user.Email)
	}

	got2, err := repo.FindByEmail("a@example.com")
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	if got2.ID != user.ID {
		t.Errorf("FindByEmail ID: got %d, want %d", got2.ID, user.ID)
	}
}

func TestUserRepository_FindByID_NotFound(t *testing.T) {
	repo := repository.NewGORMUserRepository(openTestDB(t))
	_, err := repo.FindByID(9999)
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepository_Update(t *testing.T) {
	repo := repository.NewGORMUserRepository(openTestDB(t))

	user := &model.User{Email: "b@example.com", PasswordHash: "h"}
	repo.Create(user)

	user.DisplayName = "Bob"
	if err := repo.Update(user); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.FindByID(user.ID)
	if got.DisplayName != "Bob" {
		t.Errorf("DisplayName: got %q, want %q", got.DisplayName, "Bob")
	}
}

func TestUserRepository_Delete(t *testing.T) {
	repo := repository.NewGORMUserRepository(openTestDB(t))

	user := &model.User{Email: "c@example.com", PasswordHash: "h"}
	repo.Create(user)

	if err := repo.Delete(user.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := repo.FindByID(user.ID)
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

// -- DeckRepository -----------------------------------------------------------

func TestDeckRepository_CRUD(t *testing.T) {
	db := openTestDB(t)
	userRepo := repository.NewGORMUserRepository(db)
	deckRepo := repository.NewGORMDeckRepository(db)

	user := &model.User{Email: "d@example.com", PasswordHash: "h"}
	userRepo.Create(user)

	deck := &model.Deck{UserID: user.ID, Title: "Go Basics", IsPublic: true}
	if err := deckRepo.Create(deck); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := deckRepo.FindByID(deck.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Title != "Go Basics" {
		t.Errorf("Title: got %q, want %q", got.Title, "Go Basics")
	}

	decks, err := deckRepo.FindByUserID(user.ID)
	if err != nil {
		t.Fatalf("FindByUserID: %v", err)
	}
	if len(decks) != 1 {
		t.Errorf("FindByUserID count: got %d, want 1", len(decks))
	}

	deck.Title = "Go Advanced"
	deckRepo.Update(deck)
	updated, _ := deckRepo.FindByID(deck.ID)
	if updated.Title != "Go Advanced" {
		t.Errorf("Update Title: got %q, want %q", updated.Title, "Go Advanced")
	}

	deckRepo.Delete(deck.ID)
	_, err = deckRepo.FindByID(deck.ID)
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDeckRepository_FindPublic_SortByUpvotes(t *testing.T) {
	db := openTestDB(t)
	userRepo := repository.NewGORMUserRepository(db)
	deckRepo := repository.NewGORMDeckRepository(db)

	user := &model.User{Email: "e@example.com", PasswordHash: "h"}
	userRepo.Create(user)

	d1 := &model.Deck{UserID: user.ID, Title: "Low", IsPublic: true, UpvoteCount: 1}
	d2 := &model.Deck{UserID: user.ID, Title: "High", IsPublic: true, UpvoteCount: 10}
	d3 := &model.Deck{UserID: user.ID, Title: "Private", IsPublic: false}
	deckRepo.Create(d1)
	deckRepo.Create(d2)
	deckRepo.Create(d3)

	decks, err := deckRepo.FindPublic("upvotes")
	if err != nil {
		t.Fatalf("FindPublic: %v", err)
	}
	if len(decks) != 2 {
		t.Fatalf("expected 2 public decks, got %d", len(decks))
	}
	if decks[0].Title != "High" {
		t.Errorf("expected first deck to be 'High' (most upvotes), got %q", decks[0].Title)
	}
}

func TestDeckRepository_Upvote_Toggle(t *testing.T) {
	db := openTestDB(t)
	userRepo := repository.NewGORMUserRepository(db)
	deckRepo := repository.NewGORMDeckRepository(db)

	user := &model.User{Email: "f@example.com", PasswordHash: "h"}
	userRepo.Create(user)
	deck := &model.Deck{UserID: user.ID, Title: "D", IsPublic: true}
	deckRepo.Create(deck)

	has, _ := deckRepo.HasUpvoted(user.ID, deck.ID)
	if has {
		t.Fatal("expected no upvote initially")
	}

	if err := deckRepo.AddUpvote(user.ID, deck.ID); err != nil {
		t.Fatalf("AddUpvote: %v", err)
	}
	has, _ = deckRepo.HasUpvoted(user.ID, deck.ID)
	if !has {
		t.Fatal("expected upvote after AddUpvote")
	}
	got, _ := deckRepo.FindByID(deck.ID)
	if got.UpvoteCount != 1 {
		t.Errorf("UpvoteCount after add: got %d, want 1", got.UpvoteCount)
	}

	if err := deckRepo.RemoveUpvote(user.ID, deck.ID); err != nil {
		t.Fatalf("RemoveUpvote: %v", err)
	}
	got, _ = deckRepo.FindByID(deck.ID)
	if got.UpvoteCount != 0 {
		t.Errorf("UpvoteCount after remove: got %d, want 0", got.UpvoteCount)
	}
}

// -- CardRepository -----------------------------------------------------------

func TestCardRepository_CRUD(t *testing.T) {
	db := openTestDB(t)
	userRepo := repository.NewGORMUserRepository(db)
	deckRepo := repository.NewGORMDeckRepository(db)
	cardRepo := repository.NewGORMCardRepository(db)

	user := &model.User{Email: "g@example.com", PasswordHash: "h"}
	userRepo.Create(user)
	deck := &model.Deck{UserID: user.ID, Title: "Cards Test"}
	deckRepo.Create(deck)

	card := &model.Card{DeckID: deck.ID, Front: "Q", Back: "A", EaseFactor: 2.5}
	if err := cardRepo.Create(card); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := cardRepo.FindByID(card.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Front != "Q" {
		t.Errorf("Front: got %q, want %q", got.Front, "Q")
	}

	cards, err := cardRepo.FindByDeckID(deck.ID)
	if err != nil {
		t.Fatalf("FindByDeckID: %v", err)
	}
	if len(cards) != 1 {
		t.Errorf("FindByDeckID count: got %d, want 1", len(cards))
	}

	card.Back = "Answer"
	cardRepo.Update(card)
	updated, _ := cardRepo.FindByID(card.ID)
	if updated.Back != "Answer" {
		t.Errorf("Update Back: got %q, want %q", updated.Back, "Answer")
	}

	cardRepo.Delete(card.ID)
	_, err = cardRepo.FindByID(card.ID)
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestCardRepository_FindDueCards(t *testing.T) {
	db := openTestDB(t)
	userRepo := repository.NewGORMUserRepository(db)
	deckRepo := repository.NewGORMDeckRepository(db)
	cardRepo := repository.NewGORMCardRepository(db)

	user := &model.User{Email: "h@example.com", PasswordHash: "h"}
	userRepo.Create(user)
	deck := &model.Deck{UserID: user.ID, Title: "Due Test"}
	deckRepo.Create(deck)

	now := time.Now()
	due := &model.Card{DeckID: deck.ID, Front: "Due", Back: "A", NextReviewAt: now.Add(-time.Hour)}
	future := &model.Card{DeckID: deck.ID, Front: "Future", Back: "B", NextReviewAt: now.Add(24 * time.Hour)}
	cardRepo.Create(due)
	cardRepo.Create(future)

	cards, err := cardRepo.FindDueCards(deck.ID, now)
	if err != nil {
		t.Fatalf("FindDueCards: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected 1 due card, got %d", len(cards))
	}
	if cards[0].Front != "Due" {
		t.Errorf("expected 'Due' card, got %q", cards[0].Front)
	}
}
