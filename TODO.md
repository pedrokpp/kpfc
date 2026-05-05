# TODO — KPFC Implementation Roadmap

This document tracks the phased implementation of the KPFC backend. Each phase builds on the previous. Tasks are written for contributors who are familiar with the project context — refer to `README.md` for architectural decisions and rationale.

> Mark tasks as `- [x]` when complete.

---

## Phase 0: Project Scaffolding

**Objective**: Bootstrap the Go project, logger, config, HTTP router, Docker, and task runner.

- [x] Initialize Go module: `go mod init github.com/kp/kpfc` (confirm module path with project owner if hosting changes)
- [x] Create `VERSION` file at project root with contents `0.1.0`
- [x] Create `main.go` at project root as the single entrypoint. Parse the following CLI flags using `flag` package:
  - `-debug` (bool, default `false`) — enables debug-level log output
  - `-verbose` (bool, default `false`) — enables verbose output
  - `-log-format` (string, default `"human"`) — accepts `"human"` or `"json"`
- [x] Embed the version string from `VERSION` into the binary at build time using `-ldflags "-X main.version=..."`. The `version` variable in `main.go` should be a package-level `var version string`.
- [x] Create `internal/logger/` package:
  - Define a `Logger` interface with methods: `Debug(msg string, args ...any)`, `Info(...)`, `Warn(...)`, `Error(...)`
  - Implement `HumanLogger` — colored, human-readable output to stdout. Use ANSI codes for level colors. Debug messages are suppressed unless `-debug` flag is set.
  - Implement `JSONLogger` — each log line is a JSON object with fields: `level`, `time` (RFC3339), `msg`, and any extra args as key-value pairs.
  - Expose a constructor `New(format string, debug bool) Logger` that returns the appropriate implementation.
- [x] Create `internal/config/` package:
  - Define a `Config` struct with fields: `Port` (int, default `8080`), `DBPath` (string, default `./kpfc.db`), `JWTSecret` (string)
  - Load config from environment variables with sensible defaults. No config file needed at this stage.
- [x] Add MIT `LICENSE` file to project root (standard MIT text, copyright holder: kp)
- [x] Set up [Chi](https://github.com/go-chi/chi) router in `main.go`. Register a health check at `GET /api/v1/health` returning `{"status":"ok","version":"<embedded version>"}`.
- [x] Create `docker/Dockerfile` with a two-stage build:
  - **Builder stage**: `FROM golang:1.24-alpine AS builder`. Copy source, run `go build -ldflags "-X main.version=$(cat VERSION)" -o /app/kpfc .`
  - **Runner stage**: `FROM alpine:latest`. Copy binary from builder. Create a non-root user (`adduser -D appuser`), run as that user. `EXPOSE 8080`. `CMD ["/app/kpfc"]`
- [x] Create `docker-compose.yml` at project root:
  - Service name: `kpfc`
  - Build: context `.`, dockerfile `docker/Dockerfile`
  - Ports: `"8080:8080"`
  - Volumes: `./data:/data` (for SQLite persistence)
  - Environment: `DB_PATH=/data/kpfc.db`, `PORT=8080`
- [x] Create `Justfile` at project root with these recipes:
  - `build`: `go build -ldflags "-X main.version=$(cat VERSION)" -o kpfc .`
  - `run`: depends on `build`, then `./kpfc`
  - `test`: `go test ./...`
  - `lint`: `go vet ./...`
  - `docker-build`: `docker build -f docker/Dockerfile -t kpfc .`
  - `docker-up`: `docker-compose up --build`
  - `docker-down`: `docker-compose down`

---

## Phase 1: Database and Models

**Objective**: Set up [GORM](https://gorm.io) with [SQLite](https://www.sqlite.org) and define the three core models.

- [x] Add dependencies: `gorm.io/gorm` and `gorm.io/driver/sqlite`
- [x] Create `internal/model/user.go` — `User` struct with GORM tags:
  ```go
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
  ```
- [x] Create `internal/model/deck.go` — `Deck` struct:
  ```go
  type Deck struct {
    ID          uint   `gorm:"primaryKey"`
    UserID      uint   `gorm:"not null;index"`
    Title       string `gorm:"not null"`
    Description string
    IsPublic    bool   `gorm:"default:false"`
    UpvoteCount int    `gorm:"default:0"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
  }
  ```
- [x] Create `internal/model/card.go` — `Card` struct:
  ```go
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
  ```
- [x] Create `internal/model/user_deck_upvote.go` — join table for upvote tracking:
  ```go
  type UserDeckUpvote struct {
    UserID uint `gorm:"primaryKey"`
    DeckID uint `gorm:"primaryKey"`
  }
  ```
- [x] Create `internal/repository/` package:
  - Define interface `UserRepository` with methods: `Create`, `FindByID`, `FindByEmail`, `Update`, `Delete`
  - Define interface `DeckRepository` with methods: `Create`, `FindByID`, `FindByUserID`, `FindPublic`, `Update`, `Delete`, `Upvote`, `RemoveUpvote`, `HasUpvoted`
  - Define interface `CardRepository` with methods: `Create`, `FindByID`, `FindByDeckID`, `FindDueCards`, `Update`, `Delete`
- [x] Implement GORM-backed structs:
  - `internal/repository/gorm_user.go` — `GORMUserRepository` implementing `UserRepository`
  - `internal/repository/gorm_deck.go` — `GORMDeckRepository` implementing `DeckRepository`
  - `internal/repository/gorm_card.go` — `GORMCardRepository` implementing `CardRepository`
- [x] Open SQLite database in `main.go` using `gorm.Open(sqlite.Open(config.DBPath), &gorm.Config{})`. Auto-migrate all models on startup: `db.AutoMigrate(&model.User{}, &model.Deck{}, &model.Card{}, &model.UserDeckUpvote{})`.
- [x] Write unit tests for repository layer using in-memory SQLite (`file::memory:?cache=shared` DSN). Test CRUD operations for each repository.

---

## Phase 2: Authentication

**Objective**: Implement email/password registration and login with [Argon2id](https://pkg.go.dev/golang.org/x/crypto/argon2) password hashing and JWT sessions.

- [x] Add `golang.org/x/crypto` for Argon2id (the only addition to `golang.org/x`)
- [x] Create `internal/service/auth.go` — `AuthService` with:
  - `Register(email, password, displayName string) (*model.User, error)` — validate email uniqueness, hash password, create user
  - `Login(email, password string) (token string, err error)` — verify credentials, compute streak, return JWT
- [x] Implement Argon2id hashing in a helper (`internal/service/auth.go` or `pkg/argon2/`):
  - Hash: `argon2.IDKey(password, salt, time=1, memory=64*1024, threads=4, keyLen=32)`. Store as `$argon2id$v=19$m=65536,t=1,p=4$<base64-salt>$<base64-hash>`
  - Verify: parse stored hash, recompute, compare with `subtle.ConstantTimeCompare`
- [x] Implement JWT using only stdlib (`crypto/hmac`, `crypto/sha256`, `encoding/base64`, `encoding/json`):
  - Token format: standard 3-part base64url-encoded header.payload.signature
  - Claims: `sub` (user ID), `exp` (expiration, 24h), `iat` (issued at)
  - Sign with `HMAC-SHA256` using `config.JWTSecret`
  - Expose: `GenerateToken(userID uint) (string, error)` and `ValidateToken(token string) (userID uint, err error)`
- [x] Create `internal/middleware/auth.go` — Chi middleware that extracts the Bearer token from `Authorization` header, validates it, and injects `userID` into the request context
- [x] Create `internal/handler/auth.go`:
  - `POST /api/v1/auth/register` — body: `{email, password, display_name}`, response: `{id, email, display_name}`
  - `POST /api/v1/auth/login` — body: `{email, password}`, response: `{token, user: {id, email, display_name, login_streak, total_points}}`
- [x] Streak logic in `Login` (see Phase 7 for detailed spec — can implement here directly since login is the trigger):
  - Compare `user.LastLoginDate` (date only, no time) with today's date
  - Yesterday → streak +1; today → no change; gap > 1 day → reset to 1; never logged in → set to 1
- [x] Write unit tests: password hashing and verification, JWT generation and validation (valid, expired, tampered)
- [x] Write integration tests: register flow (success, duplicate email), login flow (success, wrong password, unknown email)

---

## Phase 3: User Profile

**Objective**: Allow authenticated users to view and update their profile.

- [x] Create `internal/service/user.go` — `UserService`:
  - `GetProfile(userID uint) (*model.User, error)`
  - `UpdateProfile(userID uint, displayName string) (*model.User, error)`
- [x] Create `internal/handler/user.go`:
  - `GET /api/v1/users/me` — returns current user profile (never return `PasswordHash`)
  - `PUT /api/v1/users/me` — body: `{display_name}`, returns updated profile
- [x] Both endpoints require the JWT auth middleware
- [x] Write tests for both endpoints (authenticated, unauthenticated)

---

## Phase 4: Deck CRUD

**Objective**: Full deck management with ownership enforcement.

- [x] Create `internal/service/deck.go` — `DeckService`:
  - `Create(userID uint, title, description string, isPublic bool) (*model.Deck, error)`
  - `GetByID(userID, deckID uint) (*model.Deck, error)` — returns error if deck doesn't belong to user
  - `ListByUser(userID uint) ([]model.Deck, error)`
  - `Update(userID, deckID uint, title, description string, isPublic bool) (*model.Deck, error)` — ownership check
  - `Delete(userID, deckID uint) error` — ownership check
- [x] Create `internal/handler/deck.go` — all routes require JWT auth middleware:
  - `GET /api/v1/decks` — list user's decks
  - `POST /api/v1/decks` — create deck
  - `GET /api/v1/decks/{id}` — get single deck
  - `PUT /api/v1/decks/{id}` — update deck
  - `DELETE /api/v1/decks/{id}` — delete deck
- [x] Ownership enforcement: any operation on a deck by a user who is not the owner must return `403 Forbidden`
- [x] Write tests for all endpoints including ownership enforcement

---

## Phase 5: Card CRUD

**Objective**: Card management with deck ownership validation.

- [x] Create `internal/service/card.go` — `CardService`:
  - `Create(userID, deckID uint, front, back string) (*model.Card, error)` — verify user owns the deck
  - `GetByID(userID, cardID uint) (*model.Card, error)` — verify user owns the card's deck
  - `ListByDeck(userID, deckID uint) ([]model.Card, error)` — verify user owns the deck
  - `Update(userID, cardID uint, front, back string) (*model.Card, error)` — ownership check
  - `Delete(userID, cardID uint) error` — ownership check
- [x] Create `internal/handler/card.go` — all routes require JWT auth middleware:
  - `GET /api/v1/decks/{id}/cards` — list cards in a deck
  - `POST /api/v1/decks/{id}/cards` — add a card to a deck
  - `GET /api/v1/cards/{id}` — get single card
  - `PUT /api/v1/cards/{id}` — update card front/back
  - `DELETE /api/v1/cards/{id}` — delete card
- [x] Write tests for all card endpoints including deck ownership enforcement

---

## Phase 6: SM-2 Algorithm and Study Sessions

**Objective**: Implement the SM-2 spaced repetition algorithm and both study modes.

- [x] Create `pkg/sm2/sm2.go` — pure SM-2 function with no external dependencies:
  ```go
  type Result struct {
    Repetitions  int
    EaseFactor   float64
    Interval     int
    NextReviewAt time.Time
  }

  func Calculate(quality, repetitions int, easeFactor float64, interval int, now time.Time) Result
  ```
  - `quality` is 0–5. If `quality < 3`: reset `repetitions = 0`, `interval = 1`
  - If `quality >= 3`: `repetitions += 1`; interval: rep 1 → 1, rep 2 → 6, rep > 2 → `round(interval × EF)`
  - `EF = max(EF + (0.1 - (5-q)×(0.08 + (5-q)×0.02)), 1.3)`
  - `NextReviewAt = now + interval days` (truncated to day boundary)
- [x] Write thorough unit tests for `pkg/sm2/sm2.go`: test all quality grades (0–5), minimum EF clamping (1.3), reset behavior on grade < 3, interval progression
- [x] Create `internal/service/study.go` — `StudyService`:
  - `StartSession(userID, deckID uint, mode string) ([]model.Card, error)` — mode is `"spaced"` or `"random"`
    - Spaced: query cards where `NextReviewAt <= now` and card belongs to user's deck
    - Random: fetch all cards in deck, shuffle using `math/rand`
  - `SubmitReview(userID, cardID uint, quality int) (*model.Card, error)` — only updates SM-2 state, call `sm2.Calculate`, update card in DB. No-op for random sessions (caller responsibility to not call this in random mode).
  - `CompleteSession(userID uint, points int) error` — add points to `user.TotalPoints`
- [x] Create `internal/handler/study.go` — requires JWT auth middleware:
  - `POST /api/v1/decks/{id}/study` — body: `{mode: "spaced"|"random"}`, returns list of cards to review
  - `POST /api/v1/cards/{id}/review` — body: `{quality: 0-5}`, returns updated card with new SM-2 values
- [x] Write integration tests for both study modes, verifying SM-2 state updates in spaced mode and no state change in random mode

---

## Phase 7: Login Streaks

**Objective**: Server-side streak tracking on login (may already be implemented in Phase 2 — verify and extract if needed).

- [x] Ensure streak calculation is isolated in a pure function (e.g., `internal/service/streak.go`):
  ```go
  func CalculateStreak(lastLoginDate time.Time, currentStreak int, now time.Time) (newStreak int, newDate time.Time)
  ```
  - `lastLoginDate` is zero value → first login → streak = 1
  - Same calendar day as `now` → no change
  - Yesterday → streak + 1
  - Gap > 1 day → streak = 1
- [x] Write unit tests covering: first login, same day, consecutive day, gap, multi-day gap

---

## Phase 8: Public Decks and Upvotes

**Objective**: Enable deck discovery and community upvoting.

- [x] Implement `DeckRepository.FindPublic(sortBy string) ([]model.Deck, error)`:
  - Returns decks where `IsPublic = true`
  - `sortBy = "upvotes"` → order by `UpvoteCount DESC`; default → order by `CreatedAt DESC`
- [x] Implement upvote toggle in `DeckService`:
  - `ToggleUpvote(userID, deckID uint) error` — check `UserDeckUpvote` table; if exists, remove and decrement `UpvoteCount`; if not, insert and increment `UpvoteCount`
  - Use a DB transaction to ensure consistency between the join table and the counter
- [x] Create `internal/service/public.go` (or add to `DeckService`) — `ListPublicDecks(sortBy string) ([]model.Deck, error)`
- [x] Create `internal/handler/public.go` — **no auth required**:
  - `GET /api/v1/public/decks` — query param `?sort=upvotes`, returns public decks
- [x] Add to `internal/handler/deck.go` — requires JWT auth:
  - `POST /api/v1/decks/{id}/upvote` — toggle upvote; only works on public decks; user cannot upvote their own deck
- [x] Write tests: listing public decks, upvote toggle (add, remove, idempotent), upvote count accuracy

---

## Phase 9: Anki `.apkg` Import

**Objective**: Allow users to import Anki decks with support for multiple card types and media, enabling full migration from Anki to KPFC. Each sub-phase is additive — existing tests and API contracts must never break.

### Phase 9.1: Card Type Support

**Objective**: Extend the Card model to support multiple card types (basic, cloze). All changes are backward-compatible: new fields use defaults, existing service/handler signatures are preserved, existing tests pass unmodified.

- [x] Update `internal/model/card.go` — add **new fields** to `Card` struct (GORM AutoMigrate adds columns with defaults, no data loss):
  - `CardType string` with `gorm:"default:'basic';not null"` — values: `"basic"`, `"cloze"`
  - `ClozeIndex int` with `gorm:"default:0"` — for cloze cards: which cloze number this card represents (1-based, e.g. `c1` → 1). Ignored for basic cards.
  - `Extra string` — optional extra context field (maps to Anki's "Extra" field in cloze notes). Empty string for basic cards.
- [x] Create `pkg/cloze/cloze.go` — **new package**, pure cloze-parsing functions with no external dependencies:
  - `Deletion` struct: `Index int`, `Answer string`, `Hint string`
  - `Parse(text string) []Deletion` — extract all `{{cN::answer::hint}}` patterns
  - `Render(text string, activeIndex int) (front, back string)` — render cloze text for a specific card:
    - Active cloze (matching `activeIndex`): front replaces with `[...]` or `[hint]` if hint exists; back replaces with `<b>answer</b>`
    - Inactive clozes (different index): both sides show plain `answer` text
  - `Indices(text string) []int` — return sorted unique cloze numbers found in text
  - Regex pattern: `\{\{c(\d+)::([^}]*?)(?:::([^}]*?))?\}\}`
- [x] Write thorough unit tests for `pkg/cloze/`: single cloze, multiple clozes, cloze with hint, same-index multiple regions, non-sequential indices (c1, c3 but no c2), text with no clozes
- [x] **Keep `CardService.Create(userID, deckID, front, back)` signature unchanged** — it continues to create basic cards. Add a **new method** `CreateAdvanced(userID, deckID uint, opts CardCreateOpts) (*model.Card, error)` where `CardCreateOpts` struct contains `Front, Back, CardType, ClozeIndex, Extra string` with validation per card type:
  - Basic: requires non-empty `Front` and `Back`
  - Cloze: requires at least one `{{c1::...}}` pattern in `Front`; `Back` is ignored (rendered dynamically); `ClozeIndex` must be > 0
- [x] Update `internal/handler/card.go` **additively**:
  - `cardResponse` — include `"card_type"`, `"cloze_index"`, `"extra"` in JSON output (new keys added, no keys removed)
  - Create endpoint: accept **optional** `card_type` in request body. If absent or `"basic"` → use existing `CardService.Create` path. If `"cloze"` → use `CardService.CreateAdvanced`. This means existing API clients sending `{"front","back"}` continue to work with zero changes.
  - Update endpoint: accept **optional** `card_type`, `extra`. Preserve same backward-compat logic.
- [x] Run `go test ./...` — all existing tests must pass without modification. Write **new** tests in `internal/handler/card_test.go` covering:
  - Creating/updating cloze cards with valid and invalid cloze text
  - `card_type`, `cloze_index`, `extra` present in all card JSON responses
  - Default `card_type` is `"basic"` when omitted

### Phase 9.2: Media Model and Upload

**Objective**: Support image and audio uploads with a storage-abstraction layer. Local filesystem for now, designed with an interface to swap for S3/GCS later without touching service or handler code.

- [x] Create `internal/storage/storage.go` — **storage interface** and local implementation:
  ```go
  type Storage interface {
    Store(ctx context.Context, path string, r io.Reader) error
    Fetch(ctx context.Context, path string) (io.ReadCloser, error)
    Delete(ctx context.Context, path string) error
  }
  ```
  - Implement `LocalStorage` struct: `root string` (base directory). `Store` writes to `root/path` (creating parent dirs as needed). `Fetch` returns `os.Open`. `Delete` removes the file.
  - Future implementations (S3Storage, GCSStorage) will satisfy the same interface — service layer never imports `os` or touches the filesystem directly.
- [x] Write unit tests for `LocalStorage`: store/fetch/delete, non-existent path fetch returns error, nested path creation
- [x] Create `internal/model/media.go` — `Media` struct:
  ```go
  type Media struct {
    ID          uint      `gorm:"primaryKey"`
    UserID      uint      `gorm:"not null;index"`
    Filename    string    `gorm:"not null"`
    ContentType string    `gorm:"not null"`
    Size        int64     `gorm:"not null"`
    StoragePath string    `gorm:"not null"` // key passed to Storage interface
    CreatedAt   time.Time
  }
  ```
- [x] Add `MediaRoot string` to `internal/config/config.go` — env `MEDIA_ROOT`, default `"./media"`. **Additive change**: new field with default, existing config tests still pass.
- [x] Update `main.go` — auto-migrate `model.Media`, instantiate `LocalStorage` with `cfg.MediaRoot`, wire up media service and handler. Create `MediaRoot` directory on startup if it doesn't exist.
- [x] Update `docker-compose.yml` — add named volume `mediadata:/media` to `kpfc` service. Update `.env.docker` with `MEDIA_ROOT=/media`.
- [x] Define `MediaRepository` in `internal/repository/interfaces.go` (additive — new interface, existing ones untouched):
  - `Create(media *model.Media) error`
  - `FindByID(id uint) (*model.Media, error)`
  - `FindByUserID(userID uint) ([]model.Media, error)`
  - `Delete(id uint) error`
- [x] Implement `internal/repository/gorm_media.go` — `GORMMediaRepository`
- [x] Create `internal/service/media.go` — `MediaService` (receives `Storage` interface + `MediaRepository`):
  - `Upload(userID uint, filename string, contentType string, size int64, reader io.Reader) (*model.Media, error)`:
    - Validate content type (allow: `image/png`, `image/jpeg`, `image/gif`, `image/webp`, `audio/mpeg`, `audio/mp4`)
    - Validate size (max 10MB)
    - Generate storage path: `{userID}/{uuid}.{ext}`
    - Call `storage.Store(ctx, storagePath, reader)` — **never** `os.Create` directly
    - Create DB record
  - `GetByID(id uint) (*model.Media, io.ReadCloser, error)` — look up record, call `storage.Fetch`
  - `Delete(userID, mediaID uint) error` — ownership check, `storage.Delete`, delete DB record
- [x] Create `internal/handler/media.go` — requires JWT auth:
  - `POST /api/v1/media` — multipart form upload, field name `file`. Response: `{id, filename, content_type, size, url}`
  - `GET /api/v1/media/{id}` — serve the file directly (set `Content-Type`, `Cache-Control` headers). **No auth required** (files served by ID, enables embedding in cards).
  - `DELETE /api/v1/media/{id}` — requires auth, ownership check
- [x] Register routes in `main.go`:
  - Public: `GET /api/v1/media/{id}`
  - Auth-protected: `POST /api/v1/media`, `DELETE /api/v1/media/{id}`
- [x] Write tests for media upload (valid file, invalid type, size limit), retrieval, and deletion. Tests use `LocalStorage` with a temp directory.

### Phase 9.3: Basic Anki Import

**Objective**: Import Basic (front/back) cards from `.apkg` files, detecting note types via Anki's model metadata. Cloze notes are skipped (handled in 9.4).

- [ ] Create `internal/service/import.go` — `ImportService`:
  - `ImportAPKG(userID uint, fileData []byte, deckTitle string) (*ImportResult, error)`
  - `ImportResult` struct: `Deck *model.Deck`, `CardsImported int`, `CardsSkipped int`, `SkippedTypes []string`
  - Process:
    1. Open `fileData` as ZIP using `archive/zip` (via `bytes.Reader`, no temp file for the ZIP itself)
    2. Find SQLite file: prefer `collection.anki21`, fall back to `collection.anki2`. Return error if neither found. Skip `collection.anki21b` (protobuf format — not supported).
    3. Write SQLite file to a temp file (required for `go-sqlite3` to open it), defer cleanup
    4. Open with `gorm.io/driver/sqlite` (already a dependency — used in tests)
    5. Read `col` table → parse `models` JSON to build map: `modelID → {name, type, fields[], templates[]}`
    6. Query: `SELECT n.id, n.mid, n.flds FROM notes n`
    7. For each note, look up model by `mid`:
       - **Standard (type 0)**: split `flds` on `\x1f`, map to model's field names by ordinal. Find fields named `Front`/`Back` (case-insensitive). If model has 2+ templates, create one card per template (forward + reverse for "Basic and reversed"). Set `CardType = "basic"`.
       - **Cloze (type 1)**: skip for now. Increment `CardsSkipped`, add model name to `SkippedTypes`.
    8. Create `Deck` with `Title = deckTitle` (or Anki deck name from `col.decks` JSON if deckTitle is empty)
    9. Bulk-insert cards with default SM-2 values (`EaseFactor = 2.5`, `Interval = 1`, `NextReviewAt = now`)
- [ ] Add `BulkCreate(cards []model.Card) error` to `CardRepository` interface and implement in `gorm_card.go` — use `db.CreateInBatches(cards, 100)`. **Additive**: new method on existing interface, no existing code calls it.
- [ ] Create `internal/handler/import.go` — requires JWT auth:
  - `POST /api/v1/import/apkg` — multipart form upload, field name `file`, optional field `deck_title` (defaults to filename without extension)
  - Max upload size: 50MB (enforce with `http.MaxBytesReader`)
  - Response: `{deck_id, deck_title, cards_imported, cards_skipped, skipped_types}`
- [ ] Register route in `main.go`: `POST /api/v1/import/apkg` (auth-protected)
- [ ] Create test fixtures in `testdata/`:
  - `basic.apkg` — minimal valid Anki package with a few Basic front/back cards
  - `empty.apkg` — valid package with no notes
  - `invalid.zip` — a ZIP file that is not a valid `.apkg`
- [ ] Write integration tests: valid import (basic cards), empty deck, invalid ZIP, missing SQLite file, deck title override vs. Anki deck name

### Phase 9.4: Cloze Card Import

**Objective**: Extend the Anki importer to handle cloze deletion notes, generating one KPFC card per cloze number.

- [ ] Update `internal/service/import.go` — handle cloze notes (model `type == 1`):
  - For each cloze note: read the text field (first field by convention, or field referenced by `{{cloze:FieldName}}` in the model's template `qfmt`)
  - Use `pkg/cloze.Indices(text)` to find all unique cloze numbers
  - For each cloze number N, create one `Card`:
    - `CardType = "cloze"`
    - `Front = raw cloze text` (full text with `{{cN::...}}` markup preserved — frontend renders it using the cloze package logic)
    - `Back = ""` (rendered dynamically by frontend)
    - `ClozeIndex = N`
    - `Extra = second field value` (if present — Anki cloze notes typically have an "Extra" second field)
  - Remove cloze from `SkippedTypes`, update `CardsImported` count
- [ ] Create test fixture `testdata/cloze.apkg` — Anki package with cloze notes (single and multi-cloze)
- [ ] Write tests: cloze import (single c1, multi c1+c2+c3), cloze with hints, cloze with extra field

### Phase 9.5: Media Import from `.apkg`

**Objective**: Extract and store media files (images, audio) from `.apkg` archives and rewrite references in imported cards. Uses the `Storage` interface from Phase 9.2.

- [ ] Update `internal/service/import.go` — media extraction:
  - Read the `media` JSON file from the ZIP → build map: `numericKey → originalFilename`
  - For each media entry: read the file from the ZIP (named `0`, `1`, `2`, etc.)
  - Detect content type via `http.DetectContentType` on the first 512 bytes
  - Store via `MediaService.Upload` (reuses the `Storage` interface — works with local FS now, cloud buckets later)
  - Build a rewrite map: `originalFilename → /api/v1/media/{newID}`
- [ ] After storing all media, rewrite references in card `Front`, `Back`, and `Extra` fields before bulk insert:
  - HTML image tags: replace `src="originalFilename"` with `src="/api/v1/media/{id}"`
  - Anki audio syntax: convert `[sound:filename.mp3]` to `<audio src="/api/v1/media/{id}" controls></audio>`
- [ ] Update `ImportResult` — add `MediaImported int`
- [ ] Update handler response to include `media_imported` count
- [ ] Create test fixture `testdata/with_media.apkg` — Anki package with an embedded image and audio file
- [ ] Write tests: media extraction, URL rewriting in card content, import with mixed media types, media stored via `Storage` interface (not direct filesystem calls)

---

## Phase 10: OAuth2 Authentication

**Objective**: Add third-party login providers to reduce password management burden.

- [ ] Design the OAuth2 authorization code flow:
  - `GET /api/v1/auth/oauth/{provider}` — redirect to provider's authorization URL with state parameter
  - `GET /api/v1/auth/oauth/{provider}/callback` — exchange code for token, fetch user info, find or create User by email, return JWT
- [ ] Implement `internal/service/oauth.go` with a `OAuthProvider` interface:
  - `GetAuthURL(state string) string`
  - `ExchangeCode(code string) (email, name string, err error)`
- [ ] Implement `GoogleProvider` using Go stdlib `net/http` and `encoding/json` to call Google's OAuth2 and userinfo endpoints
- [ ] Implement `GitHubProvider` using the same approach for GitHub's OAuth2 and user API
- [ ] On callback: if a user with the same email already exists, link the OAuth login to that account and return a JWT. If not, create a new user with no password hash.
- [ ] Write integration tests for OAuth2 flows (may require mocking provider HTTP endpoints)

---

## Phase 11: PostgreSQL Migration

**Objective**: Support [PostgreSQL](https://www.postgresql.org) as an alternative database backend for production scale.

- [x] Add `gorm.io/driver/postgres` dependency
- [x] Update `internal/config/` — add `DBDriver` field (values: `"sqlite"`, `"postgres"`) and `DBConnectionString` for Postgres DSN
- [x] Update `main.go` — open DB connection based on `config.DBDriver`
- [x] Verify all repository operations work correctly with PostgreSQL (data types, index behavior, timestamp handling)
- [x] Ensure `docker-compose.yml` can be extended with a PostgreSQL service (document this in README)
- [x] Run full integration test suite against PostgreSQL and resolve any dialect-specific issues

---

## Phase 12: Media IDOR Fix + Card Title Field

**Objective**: Prevent IDOR on media endpoints by replacing sequential numeric IDs with unguessable public tokens in URLs. Add an optional title field to cards for preview purposes.

### Phase 12.1: Media Public Token (IDOR Fix)

**Objective**: The `GET /api/v1/media/{id}` endpoint is intentionally unauthenticated (media embedded in cards), but uses sequential auto-increment IDs, allowing enumeration of all uploaded files. Replace the numeric ID in URLs with a random 32-character hex token.

- [x] Update `internal/model/media.go` — add `PublicID string` field with `gorm:"uniqueIndex;not null;size:32"` tag
- [x] Update `internal/repository/interfaces.go` — add `FindByPublicID(publicID string) (*model.Media, error)` to `MediaRepository`
- [x] Implement `FindByPublicID` in `internal/repository/gorm_media.go` — query `WHERE public_id = ?`, return `ErrNotFound` if missing
- [x] Update `internal/service/media.go`:
  - `Upload()`: set `m.PublicID = s.idGen()` before calling `repo.Create()`
  - Rename `GetByID` → `GetByPublicID(ctx, publicID string)` — use `repo.FindByPublicID` instead of `FindByID`
  - Update `Delete(ctx, userID uint, publicID string)` — lookup via `FindByPublicID` instead of `FindByID`
- [x] Update `internal/handler/media.go`:
  - `mediaServiceIface`: change `GetByID` → `GetByPublicID(ctx, publicID string)`, change `Delete` to accept `publicID string` instead of `mediaID uint`
  - `Serve()`: read `chi.URLParam(r, "id")` as string directly (no uint parsing)
  - `Delete()`: read URL param as string directly
  - `mediaResponse()`: include `"public_id"` in response, build URL using `m.PublicID` instead of `m.ID`
  - Remove `parseMediaID()` helper (no longer needed)
- [x] Update `internal/handler/media_test.go` — use `public_id` from upload response for Serve/Delete URL paths instead of numeric `id`
- [x] Run `go test ./...` — all tests must pass

### Phase 12.2: Card Title Field

**Objective**: Add an optional `Title` field to the Card model for preview purposes (e.g., showing a card title before revealing front/back).

- [x] Update `internal/model/card.go` — add `Title string` field (no `not null`, optional, empty string default)
- [x] Update `internal/service/card.go`:
  - Add `Title string` to `CardCreateOpts`
  - `Create()`: add `title string` parameter, set `card.Title = title`
  - `CreateAdvanced()`: set `card.Title = opts.Title`
  - `Update()`: add `title string` parameter, set `card.Title = title`
  - `UpdateAdvanced()`: set `card.Title = opts.Title`
- [x] Update `internal/handler/card.go`:
  - `cardResponse()`: add `"title": c.Title` to response map
  - `Create` handler: read `title` from request JSON body, pass to service `Create()` / `CreateAdvanced()`
  - `Update` handler: read `title` from request JSON body, pass to service `Update()` / `UpdateAdvanced()`
- [x] Update `API.md` — document `title` field in card create/update/response
- [x] Run `go test ./...` — all tests must pass
