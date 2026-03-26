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

- [ ] Implement `DeckRepository.FindPublic(sortBy string) ([]model.Deck, error)`:
  - Returns decks where `IsPublic = true`
  - `sortBy = "upvotes"` → order by `UpvoteCount DESC`; default → order by `CreatedAt DESC`
- [ ] Implement upvote toggle in `DeckService`:
  - `ToggleUpvote(userID, deckID uint) error` — check `UserDeckUpvote` table; if exists, remove and decrement `UpvoteCount`; if not, insert and increment `UpvoteCount`
  - Use a DB transaction to ensure consistency between the join table and the counter
- [ ] Create `internal/service/public.go` (or add to `DeckService`) — `ListPublicDecks(sortBy string) ([]model.Deck, error)`
- [ ] Create `internal/handler/public.go` — **no auth required**:
  - `GET /api/v1/public/decks` — query param `?sort=upvotes`, returns public decks
- [ ] Add to `internal/handler/deck.go` — requires JWT auth:
  - `POST /api/v1/decks/{id}/upvote` — toggle upvote; only works on public decks; user cannot upvote their own deck
- [ ] Write tests: listing public decks, upvote toggle (add, remove, idempotent), upvote count accuracy

---

## Phase 9: Anki `.apkg` Import

**Objective**: Allow users to import Anki decks, enabling migration to KPFC.

- [ ] Create `internal/service/import.go` — `ImportService`:
  - `ImportAPKG(userID uint, fileData []byte, deckTitle string) (*model.Deck, int, error)` — returns created deck and number of cards imported
  - Process:
    1. Write `fileData` to a temp file (or use `archive/zip` from memory)
    2. Open as ZIP using `archive/zip`
    3. Find and extract `collection.anki2` (or `collection.anki21`) — the embedded SQLite database
    4. Open extracted SQLite with GORM or `database/sql` + SQLite driver
    5. Query Anki's `notes` table: `SELECT flds FROM notes`
    6. Split `flds` on `\x1f` (U+001F, unit separator character) — `fields[0]` = Front, `fields[1]` = Back
    7. Create a new `Deck` for the user with `Title = deckTitle`
    8. Bulk-insert `Card` records, setting `EaseFactor = 2.5`, `Interval = 1`, `NextReviewAt = now`
    9. Clean up temp files
- [ ] Create `internal/handler/import.go` — requires JWT auth:
  - `POST /api/v1/import/apkg` — multipart form upload, field name `file`, optional field `deck_title` (defaults to filename without extension)
  - Max upload size: 50MB
  - Response: `{deck_id, deck_title, cards_imported}`
- [ ] Add a fixture `.apkg` file to `test/` for testing (a minimal valid Anki package)
- [ ] Write integration tests for import: valid `.apkg`, invalid ZIP, empty notes table

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

- [ ] Add `gorm.io/driver/postgres` dependency
- [ ] Update `internal/config/` — add `DBDriver` field (values: `"sqlite"`, `"postgres"`) and `DBConnectionString` for Postgres DSN
- [ ] Update `main.go` — open DB connection based on `config.DBDriver`
- [ ] Verify all repository operations work correctly with PostgreSQL (data types, index behavior, timestamp handling)
- [ ] Ensure `docker-compose.yml` can be extended with a PostgreSQL service (document this in README)
- [ ] Run full integration test suite against PostgreSQL and resolve any dialect-specific issues
