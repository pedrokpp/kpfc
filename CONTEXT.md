# AI Repository Overview

Purpose: fast repository orientation for AI agents and humans. This file prefers concise, verifiable statements over complete prose.

## 1. Verified Facts

- Language/runtime: Go 1.26.1 (`go.mod`).
- Entry point: `main.go`.
- Main external dependencies:
  - `github.com/go-chi/chi/v5` for HTTP routing
  - `gorm.io/gorm` + `gorm.io/driver/sqlite` for persistence
  - `golang.org/x/crypto` for Argon2 password hashing
- Architecture is layered as:
  - `internal/handler`: HTTP layer
  - `internal/service`: business logic
  - `internal/repository`: persistence interfaces + GORM implementations
  - `internal/model`: GORM models
  - `internal/middleware`: auth and CORS middleware
  - `internal/config`: environment-driven config loader
  - `internal/storage`: file storage abstraction with local filesystem implementation
  - `pkg/sm2`: spaced repetition algorithm
  - `pkg/cloze`: cloze-related utility package

## 2. Read Order

When you need context quickly, read files in this order:

1. `main.go`
2. `internal/config/config.go`
3. `internal/repository/interfaces.go`
4. Relevant handler in `internal/handler/`
5. Matching service in `internal/service/`
6. Matching GORM repository in `internal/repository/`
7. Matching model in `internal/model/`

This usually gives enough context for a safe change.

## 3. Bootstrap and Runtime

Verified from `main.go`:

- Runtime flags:
  - `-debug`
  - `-verbose`
  - `-log-format=human|json`
- Required runtime constraint:
  - process exits if `JWT_SECRET` is empty
- Default config values:
  - `PORT=8080`
  - `DB_PATH=./data/kpfc.db`
  - `MEDIA_ROOT=./media`
  - `CORS_ALLOW_ORIGINS=http://localhost:5173`
- On startup the app:
  1. loads config
  2. creates the DB parent directory
  3. opens SQLite via GORM
  4. runs `AutoMigrate`
  5. creates `MEDIA_ROOT`
  6. wires repositories, services, handlers
  7. starts an HTTP server with graceful shutdown

## 4. HTTP Surface

Verified routes from `main.go`:

- Public:
  - `GET /api/v1/health`
  - `GET /api/v1/version`
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/login`
  - `GET /api/v1/public/decks`
  - `GET /api/v1/media/{id}`
- Authenticated:
  - `GET/PUT /api/v1/users/me`
  - CRUD for decks
  - CRUD for cards
  - `POST /api/v1/decks/{id}/study`
  - `POST /api/v1/cards/{id}/review`
  - `POST /api/v1/decks/{id}/upvote`
  - `POST /api/v1/media`
  - `DELETE /api/v1/media/{id}`

Auth middleware is applied at the router group level for authenticated routes.

## 5. Persistence and CGO

Verified from `go.mod` and startup code:

- SQLite is the active database.
- The project depends on `github.com/mattn/go-sqlite3` indirectly through GORM's SQLite driver.
- Because that driver uses CGO, tests and local builds that touch SQLite may require `CGO_ENABLED=1`.
- `Justfile` test recipe now exports `CGO_ENABLED=1` for consistency.

## 6. Domain Areas

Verified by package/file names and service wiring:

- auth: registration, login, token generation/validation, password hashing
- users: profile reads/updates
- decks: CRUD, visibility, upvotes
- cards: CRUD
- study: spaced and random modes, SM-2 review submission
- media: upload, serve, delete with local filesystem backing
- access control: centralized in `internal/service/access.go`

Verified behavioral details:

- JWT handling is implemented in-project in `internal/service/auth.go`; no external JWT package is used.
- SM-2 logic lives in `pkg/sm2/sm2.go`.
- Study mode dispatch happens in `internal/service/study.go`.
- File storage is abstracted behind `internal/storage/storage.go`.

## 7. Testing Shape

Verified from the file tree:

- Most packages have focused `_test.go` coverage.
- There are tests for handlers, services, repositories, config, logger, middleware, storage, `pkg/sm2`, and `pkg/cloze`.
- Root-level `main_test.go` exists, so some app-level behavior is covered.

Safe assumption:

- The intended development style is test-backed changes, because nearly every package has direct tests nearby.

## 8. Working Rules For Agents

Recommended operating assumptions for changes:

- Prefer changing the narrowest layer possible.
- If an HTTP behavior changes, inspect handler + service + tests together.
- If a rule depends on ownership/permissions, inspect `internal/service/access.go` before editing.
- If a change touches persistence shape, inspect both `internal/model/*` and the relevant GORM repository.
- Keep new dependencies rare; the repo documentation explicitly prefers standard library where practical.

## 9. Assumptions / Open Questions

These are intentionally not presented as facts:

- `pkg/cloze` appears to support card content processing, but its active use from handlers/services was not verified in this overview pass.
- `API.md`, `README.md`, and `TODO.md` are substantial, but they may contain plans or drift from code. Treat `main.go` and tests as source of truth when they disagree.
- The repository appears optimized for small-scale deployment first, but scaling expectations should be confirmed in product docs before architecture work.

## 10. Quick File Map

- App bootstrap: `main.go`
- Config: `internal/config/config.go`
- HTTP handlers: `internal/handler/`
- Services: `internal/service/`
- Repositories: `internal/repository/`
- Data models: `internal/model/`
- Middleware: `internal/middleware/`
- Storage abstraction: `internal/storage/storage.go`
- Algorithm packages: `pkg/sm2/`, `pkg/cloze/`
- Commands: `Justfile`
- Containerization: `docker/Dockerfile`, `docker-compose.yml`

## 11. Source Of Truth Policy

For implementation work, trust sources in this order:

1. tests
2. `main.go` wiring and route registration
3. package-local code
4. `README.md` / `API.md`
5. `TODO.md`

This order is an operational heuristic, not a project rule formally declared in code.
