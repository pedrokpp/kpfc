# KPFC — Flashcards Backend

An [Anki](https://apps.ankiweb.net)-inspired spaced repetition flashcards backend built with [Go](https://go.dev). Licensed under the [MIT License](LICENSE).

---

## Overview

KPFC is a REST API backend for a flashcard study platform. Its core premise is that the platform acts as a **helper tool**, not a prescriber of behavior. Users are in control — they can follow the spaced repetition schedule, review a deck again freely, or practice in random mode. The system always tracks the underlying SM-2 state regardless of how the user chooses to study.

Key characteristics:

- **[SM-2](https://www.supermemo.com/en/articles/twenty-rules) spaced repetition algorithm** for scheduled review
- **Random study mode** for free-form practice (SM-2 state is not modified)
- **Login streaks** tracked server-side to prevent client-side tampering
- **Points** awarded per completed study session
- **Public/private decks** with community upvoting (positive-only)
- **[Anki](https://apps.ankiweb.net) `.apkg` import** for easy migration
- Designed for ~10 users initially, architected for easy horizontal scaling

---

## Tech Stack

| Tool | Purpose |
|------|---------|
| [Go](https://go.dev) | Backend language — standard library preferred to minimize supply chain attack surface |
| [Chi](https://github.com/go-chi/chi) | HTTP router — lightweight, idiomatic, stdlib-compatible |
| [GORM](https://gorm.io) | ORM for database access |
| [SQLite](https://www.sqlite.org) | Initial database — will migrate to [PostgreSQL](https://www.postgresql.org) as scale requires |
| [Argon2](https://pkg.go.dev/golang.org/x/crypto/argon2) | Password hashing (Argon2id) |
| [Docker](https://www.docker.com) | Containerization via multi-stage builds |
| [Just](https://github.com/casey/just) | Command runner (Justfile recipes) |

> **Dependency policy**: Do NOT add external packages without explicit discussion. Prefer Go standard library implementations. Each dependency added is a supply chain risk.

---

## Architecture Principles

- **SOLID** — Single responsibility, open/closed, Liskov substitution, interface segregation, dependency inversion
- **DRY** — Avoid duplicating logic; consolidate shared behavior
- **YAGNI** — Do not build for hypothetical future needs
- **Repository pattern** — All database access goes through repository interfaces. This makes swapping SQLite for PostgreSQL seamless.
- **Dependency injection via interfaces** — Facilitates unit testing and implementation swapping
- **Layered architecture**: `handler → service → repository`
  - Handlers are thin: parse request, call service, write response
  - Services contain all business logic
  - Repositories contain all data access logic

---

## Data Models

### User
```
ID            uint      (primary key)
Email         string    (unique)
PasswordHash  string    (Argon2id)
DisplayName   string
LoginStreak   int       (consecutive login days)
LastLoginDate time.Time (date of last login, for streak calculation)
TotalPoints   int       (cumulative points across all sessions)
CreatedAt     time.Time
UpdatedAt     time.Time
```

### Deck
```
ID          uint
UserID      uint      (owner, foreign key → User)
Title       string
Description string
IsPublic    bool      (controls visibility in public endpoint)
UpvoteCount int       (denormalized count for query performance)
CreatedAt   time.Time
UpdatedAt   time.Time
```

### Card
```
ID           uint
DeckID       uint      (foreign key → Deck)
Front        string
Back         string
Interval     int       (SM-2: days until next review)
Repetitions  int       (SM-2: number of successful reviews)
EaseFactor   float64   (SM-2: difficulty multiplier, min 1.3)
NextReviewAt time.Time (SM-2: scheduled next review date)
CreatedAt    time.Time
UpdatedAt    time.Time
```

**Relationships:** `1 User : N Decks`, `1 Deck : N Cards`

---

## API Endpoints

All endpoints are prefixed with `/api/v1`.

### Auth
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/auth/register` | Register with email and password |
| `POST` | `/auth/login` | Login — returns JWT token |

### Users
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/users/me` | Get current user profile |
| `PUT` | `/users/me` | Update current user profile |

### Decks
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/decks` | List authenticated user's decks |
| `POST` | `/decks` | Create a new deck |
| `GET` | `/decks/{id}` | Get a deck by ID |
| `PUT` | `/decks/{id}` | Update a deck |
| `DELETE` | `/decks/{id}` | Delete a deck |
| `POST` | `/decks/{id}/upvote` | Toggle upvote on a public deck |

### Cards
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/decks/{id}/cards` | List cards in a deck |
| `POST` | `/decks/{id}/cards` | Add a card to a deck |
| `GET` | `/cards/{id}` | Get a card by ID |
| `PUT` | `/cards/{id}` | Update a card |
| `DELETE` | `/cards/{id}` | Delete a card |

### Study
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/decks/{id}/study` | Start a study session (`?mode=spaced\|random`) |
| `POST` | `/cards/{id}/review` | Submit a card review (SM-2 grade 0–5) |

### Public
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/public/decks` | List public decks (`?sort=upvotes`) |

### System
| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/version` | Returns the embedded build version |

---

## Authentication

- **Current**: Email/password registration and login
- **Password hashing**: [Argon2id](https://pkg.go.dev/golang.org/x/crypto/argon2) — resistant to brute force and side-channel attacks
- **Sessions**: JWT tokens signed with HMAC-SHA256, implemented using Go standard library (`crypto/hmac`, `encoding/base64`, `encoding/json`) — no external JWT library
- **Future (priority)**: [OAuth2](https://oauth.net/2/) with Google and GitHub — goal is to minimize password management burden. Email/password auth will remain as fallback.

---

## SM-2 Algorithm

The [SM-2 algorithm](https://www.supermemo.com/en/articles/twenty-rules) by Piotr Wozniak drives spaced repetition scheduling.

**Review grades**: 0–5 (0 = complete blackout, 5 = perfect recall with no hesitation)

**Update rules**:
- If `grade < 3`: reset `repetitions = 0`, `interval = 1`
- If `grade >= 3`: `interval` is calculated based on repetition count and ease factor
  - `repetitions == 1` → `interval = 1`
  - `repetitions == 2` → `interval = 6`
  - `repetitions > 2` → `interval = round(previous_interval × EaseFactor)`

**Ease factor update** (applied after every review):
```
EF = EF + (0.1 - (5 - grade) × (0.08 + (5 - grade) × 0.02))
EF = max(EF, 1.3)   // minimum ease factor
```

**NextReviewAt**: `now + interval days`

The SM-2 implementation lives in `pkg/sm2/` as a pure function with no external dependencies or side effects.

---

## Study Modes

### Spaced Repetition (`mode=spaced`)
Cards with `NextReviewAt <= now` are presented for review. After each review, SM-2 state is updated and `NextReviewAt` is rescheduled. If the user voluntarily reviews a deck outside of the schedule, SM-2 state is still updated — the platform respects user intent.

### Random (`mode=random`)
All cards in the deck are shuffled and presented in random order. **SM-2 state is not modified** — this mode is purely for free-form practice or self-testing.

---

## Streaks and Points

### Login Streaks
Calculated server-side on every successful login:
- If `LastLoginDate` is yesterday → `LoginStreak += 1`
- If `LastLoginDate` is today → no change (streak preserved)
- If gap > 1 day → `LoginStreak = 1` (reset)

Stored as `LoginStreak` (int) and `LastLoginDate` (date) on the User model.

### Points
Awarded as a flat amount per completed study session. Stored as cumulative `TotalPoints` on the User model.

---

## Deck Visibility and Upvotes

- Every deck has an `IsPublic` boolean field
- Public decks are listed at `GET /api/v1/public/decks`
- Public decks can be sorted by `UpvoteCount` descending (`?sort=upvotes`)
- Each user can upvote a public deck once (enforced via a `user_deck_upvotes` join table)
- `POST /api/v1/decks/{id}/upvote` is a toggle: upvotes if not yet upvoted, removes upvote if already upvoted
- **No downvote mechanism** — by design, to keep the community experience positive

---

## Anki `.apkg` Import

[Anki](https://apps.ankiweb.net) `.apkg` files are ZIP archives containing a [SQLite](https://www.sqlite.org) database.

The import process:
1. Accept the `.apkg` file via multipart upload at `POST /api/v1/import/apkg`
2. Extract the ZIP archive in a temporary directory
3. Open the embedded SQLite database
4. Read Anki's `notes` table — fields are separated by `\x1f` (unit separator)
5. Map each note to a Card: `Front = fields[0]`, `Back = fields[1]`
6. Create a new Deck for the user and insert all Cards

This enables a frictionless migration path from Anki to KPFC.

---

## Project Structure

```
kpfc/
├── main.go              # Entrypoint — flag parsing, logger init, router, server start
├── VERSION              # Semantic version string (embedded at build time via ldflags)
├── Justfile             # Task runner recipes (build, run, test, docker, lint)
├── docker/
│   └── Dockerfile       # Multi-stage: builder (golang:1.24-alpine) + runner (alpine)
├── docker-compose.yml   # Single-service compose for bootstrapping the server
├── go.mod
├── go.sum
├── README.md
├── TODO.md
├── LICENSE
├── internal/
│   ├── config/          # Configuration struct and loading (port, db path, etc.)
│   ├── handler/         # HTTP handlers — thin, delegate to services
│   ├── middleware/       # JWT auth middleware, request logging middleware
│   ├── model/           # GORM model definitions (User, Deck, Card)
│   ├── repository/      # Repository interfaces and GORM implementations
│   ├── service/         # Business logic (auth, user, deck, card, study, import)
│   └── logger/          # Logger interface + human-readable and JSON implementations
├── pkg/
│   └── sm2/             # Pure SM-2 algorithm — no external deps, easily testable
└── test/
    └── integration/     # Integration tests using real SQLite in-memory database
```

---

## Building and Running

### Prerequisites
- [Go](https://go.dev) 1.21+
- [Just](https://github.com/casey/just) (optional, for Justfile recipes)
- [Docker](https://www.docker.com) and [Docker Compose](https://docs.docker.com/compose/) (optional, for containerized runs)

### Local build
```bash
# Build binary with version embedded from VERSION file
go build -ldflags "-X main.version=$(cat VERSION)" -o kpfc .

# Run with default settings (human-readable logs)
./kpfc

# Run with flags
./kpfc -debug                   # Enable debug-level log output
./kpfc -verbose                 # Enable verbose output
./kpfc -log-format json         # Structured JSON logs
./kpfc -debug -log-format json  # Debug + JSON logs
```

### Using Just
```bash
just build        # Build binary with version embedding
just run          # Build and run locally
just test         # Run go test ./...
just lint         # Run go vet ./...
just docker-build # Build Docker image
just docker-up    # Run with docker-compose
just docker-down  # Stop docker-compose services
```

### Using Docker Compose
```bash
docker-compose up --build
```

### Docker Setup
`docker/Dockerfile` uses a two-stage build:
1. **Builder stage** (`golang:1.24-alpine`) — compiles the binary with `ldflags` for version embedding
2. **Runner stage** (`alpine:latest`) — minimal image, runs binary as a non-root user

`docker-compose.yml` mounts a local volume for SQLite data persistence.

---

## Versioning

- Version is stored in the `VERSION` file at the project root (semantic versioning: `MAJOR.MINOR.PATCH`)
- Embedded into the binary at build time: `go build -ldflags "-X main.version=$(cat VERSION)"`
- Exposed at runtime via `GET /api/v1/version`

---

## Testing

- **Unit tests**: Test individual functions — SM-2 calculations, service logic, streak computation, password hashing
- **Integration tests** (`test/integration/`): Test full HTTP request lifecycle with an in-memory SQLite database
- Prefer quality over quantity — cover critical paths, edge cases, and error scenarios
- Run all tests: `go test ./...`

---

## Contributing

### Commit Convention
This project uses [Conventional Commits](https://www.conventionalcommits.org):
```
feat:     new feature
fix:      bug fix
docs:     documentation changes
refactor: code refactoring (no behavior change)
test:     adding or improving tests
chore:    maintenance, tooling, dependencies
```

### Guidelines
- **Do not add external dependencies** without discussion — every dependency is a supply chain risk
- **Follow the layered architecture**: handler → service → repository
- **Handlers must be thin** — request parsing and response writing only; all logic in services
- **All database access through repository interfaces** — never call GORM directly from handlers or services
- **Write tests** for all new behavior, preferring integration tests for HTTP endpoints
- **All code, comments, API responses, and documentation must be in English**
- **Credit third-party tools** with markdown links on first mention in documentation

---

## License

This project is licensed under the [MIT License](LICENSE).
