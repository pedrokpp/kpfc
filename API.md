# KPFC API Reference

**Version:** 0.1.12
**Base URL:** `http://localhost:8080/api/v1`

KPFC is an Anki-like spaced-repetition flashcard backend. It uses JWT Bearer tokens for authentication, PostgreSQL for storage, and the SM-2 algorithm for study scheduling.

## Configuration

| Env Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Server listen port |
| `DATABASE_URL` | (required) | PostgreSQL connection string |
| `JWT_SECRET` | (required) | Secret for JWT signing |
| `MEDIA_ROOT` | `./media` | Local directory for uploaded media files |

---

## Authentication

Authenticated endpoints require a JWT token in the `Authorization` header:

```
Authorization: Bearer <token>
```

Tokens are obtained via the login endpoint. Unauthenticated requests to protected endpoints return `401`.

---

## Error Format

All errors follow this structure:

```json
{"error": "error message"}
```

---

## Endpoints Summary

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | No | Health check |
| GET | `/version` | No | Version info |
| POST | `/auth/register` | No | Register new user |
| POST | `/auth/login` | No | Login and get JWT |
| GET | `/public/decks` | No | List public decks |
| GET | `/media/{public_id}` | No | Serve uploaded media file |
| GET | `/users/me` | Yes | Get current user profile |
| PUT | `/users/me` | Yes | Update current user profile |
| GET | `/decks` | Yes | List user's decks |
| POST | `/decks` | Yes | Create deck |
| GET | `/decks/{id}` | Yes | Get deck by ID |
| PUT | `/decks/{id}` | Yes | Update deck |
| DELETE | `/decks/{id}` | Yes | Delete deck |
| POST | `/decks/{id}/upvote` | Yes | Toggle upvote on deck |
| GET | `/decks/{id}/cards` | Yes | List cards in deck |
| POST | `/decks/{id}/cards` | Yes | Create card in deck |
| GET | `/cards/{id}` | Yes | Get card by ID |
| PUT | `/cards/{id}` | Yes | Update card |
| DELETE | `/cards/{id}` | Yes | Delete card |
| POST | `/decks/{id}/study` | Yes | Start study session |
| POST | `/cards/{id}/review` | Yes | Submit card review |
| POST | `/media` | Yes | Upload media file |
| DELETE | `/media/{id}` | Yes | Delete media file |

---

## Public Endpoints

### GET /health

Returns server health status.

**Response 200:**
```json
{"status": "ok", "version": "0.1.11"}
```

### GET /version

Returns server version.

**Response 200:**
```json
{"version": "0.1.11"}
```

---

## Auth

### POST /auth/register

Register a new user account.

**Request body:**
```json
{
  "email": "user@example.com",
  "password": "securepass",
  "display_name": "John"
}
```

| Field | Type | Required |
|---|---|---|
| `email` | string | Yes |
| `password` | string | Yes |
| `display_name` | string | No |

**Response 201:**
```json
{
  "id": 1,
  "email": "user@example.com",
  "display_name": "John"
}
```

**Errors:**
- `400` — `"invalid request body"` / `"email and password are required"`
- `409` — `"email already registered"`
- `500` — `"registration failed"`

---

### POST /auth/login

Authenticate and receive a JWT token.

**Request body:**
```json
{
  "email": "user@example.com",
  "password": "securepass"
}
```

**Response 200:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "display_name": "John",
    "login_streak": 3,
    "total_points": 150
  }
}
```

**Errors:**
- `400` — `"invalid request body"`
- `401` — `"invalid credentials"`
- `500` — `"login failed"`

---

## User Profile

### GET /users/me

Get the authenticated user's profile.

**Response 200:**
```json
{
  "id": 1,
  "email": "user@example.com",
  "display_name": "John",
  "login_streak": 3,
  "total_points": 150
}
```

**Errors:**
- `401` — `"unauthorized"`
- `500` — `"failed to get profile"`

---

### PUT /users/me

Update the authenticated user's profile.

**Request body:**
```json
{
  "display_name": "New Name"
}
```

**Response 200:** Same format as GET /users/me.

**Errors:**
- `400` — `"invalid request body"`
- `401` — `"unauthorized"`
- `500` — `"failed to update profile"`

---

## Decks

### Deck Object

```json
{
  "id": 1,
  "user_id": 1,
  "title": "Go Basics",
  "description": "Fundamental Go concepts",
  "is_public": false,
  "upvote_count": 0,
  "created_at": "2026-03-26T10:00:00Z",
  "updated_at": "2026-03-26T10:00:00Z"
}
```

---

### GET /decks

List all decks owned by the authenticated user.

**Response 200:** Array of deck objects.

**Errors:**
- `401` — `"unauthorized"`
- `500` — `"failed to list decks"`

---

### POST /decks

Create a new deck.

**Request body:**
```json
{
  "title": "Go Basics",
  "description": "Fundamental Go concepts",
  "is_public": false
}
```

| Field | Type | Required |
|---|---|---|
| `title` | string | Yes |
| `description` | string | No |
| `is_public` | bool | No (default: false) |

**Response 201:** Deck object.

**Errors:**
- `400` — `"invalid request body"` / `"title is required"`
- `401` — `"unauthorized"`
- `500` — `"failed to create deck"`

---

### GET /decks/{id}

Get a specific deck. Must be owned by the authenticated user.

**Path params:** `id` (uint) — deck ID

**Response 200:** Deck object.

**Errors:**
- `400` — `"invalid deck id"`
- `401` — `"unauthorized"`
- `403` — `"forbidden"`
- `404` — `"deck not found"`

---

### PUT /decks/{id}

Update a deck. Must be owned by the authenticated user.

**Path params:** `id` (uint) — deck ID

**Request body:**
```json
{
  "title": "Updated Title",
  "description": "Updated description",
  "is_public": true
}
```

| Field | Type | Required |
|---|---|---|
| `title` | string | Yes |
| `description` | string | No |
| `is_public` | bool | No |

**Response 200:** Deck object.

**Errors:**
- `400` — `"invalid request body"` / `"invalid deck id"` / `"title is required"`
- `401` — `"unauthorized"`
- `403` — `"forbidden"`
- `404` — `"deck not found"`

---

### DELETE /decks/{id}

Delete a deck. Must be owned by the authenticated user.

**Path params:** `id` (uint) — deck ID

**Response:** `204 No Content` (empty body)

**Errors:**
- `400` — `"invalid deck id"`
- `401` — `"unauthorized"`
- `403` — `"forbidden"`
- `404` — `"deck not found"`

---

### POST /decks/{id}/upvote

Toggle an upvote on a deck. Calling again removes the upvote.

**Path params:** `id` (uint) — deck ID

**Request:** No body.

**Response:** `204 No Content` (empty body)

**Errors:**
- `400` — `"invalid deck id"`
- `401` — `"unauthorized"`
- `403` — `"forbidden"`
- `404` — `"deck not found"`

---

### GET /public/decks

List all public decks. No authentication required.

**Query params:**
| Param | Type | Description |
|---|---|---|
| `sort` | string | Optional. `"upvotes"` sorts by upvote count descending. Default: newest first. |

**Response 200:** Array of deck objects.

---

## Cards

### Card Object

```json
{
  "id": 1,
  "deck_id": 1,
  "title": "Optional preview title",
  "front": "What is a goroutine?",
  "back": "A lightweight thread managed by the Go runtime",
  "card_type": "basic",
  "cloze_index": 0,
  "extra": "",
  "interval": 1,
  "repetitions": 0,
  "ease_factor": 2.5,
  "next_review_at": "2026-03-27T10:00:00Z",
  "created_at": "2026-03-26T10:00:00Z",
  "updated_at": "2026-03-26T10:00:00Z"
}
```

| Field | Description |
|---|---|
| `title` | Optional preview title for the card. Empty string if not set. |
| `card_type` | `"basic"` or `"cloze"`. Default: `"basic"`. |
| `cloze_index` | For cloze cards: which deletion number this card represents (1-based). `0` for basic cards. |
| `extra` | Optional extra context. Used by Anki cloze imports for the "Extra" field. Empty for basic cards. |

---

### GET /decks/{id}/cards

List all cards in a deck. Deck must be owned by the authenticated user.

**Path params:** `id` (uint) — deck ID

**Response 200:** Array of card objects.

**Errors:**
- `400` — `"invalid deck id"`
- `401` — `"unauthorized"`
- `403` — `"forbidden"`
- `404` — `"card not found"`

---

### POST /decks/{id}/cards

Create a new card in a deck.

**Path params:** `id` (uint) — deck ID

**Request body (basic card):**
```json
{
  "front": "What is a goroutine?",
  "back": "A lightweight thread managed by the Go runtime"
}
```

**Request body (cloze card):**
```json
{
  "front": "The {{c1::Go}} runtime schedules goroutines.",
  "card_type": "cloze",
  "cloze_index": 1,
  "extra": "Optional extra context"
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `title` | string | No | Optional preview title for the card. |
| `front` | string | Yes | For cloze: must contain at least one `{{cN::...}}` pattern. |
| `back` | string | Yes for basic | Ignored for cloze cards (rendered dynamically). |
| `card_type` | string | No | `"basic"` (default) or `"cloze"`. |
| `cloze_index` | int | Yes for cloze | Which deletion number this card represents (≥ 1). |
| `extra` | string | No | Extra context field. Used for cloze cards. |

**Response 201:** Card object.

**Errors:**
- `400` — `"invalid request body"` / `"invalid deck id"` / `"front and back are required"` / `"front must contain at least one cloze deletion"` / `"cloze_index must be greater than 0"`
- `401` — `"unauthorized"`
- `403` — `"forbidden"`

---

### GET /cards/{id}

Get a specific card. The card's deck must be owned by the authenticated user.

**Path params:** `id` (uint) — card ID

**Response 200:** Card object.

**Errors:**
- `400` — `"invalid card id"`
- `401` — `"unauthorized"`
- `403` — `"forbidden"`
- `404` — `"card not found"`

---

### PUT /cards/{id}

Update a card.

**Path params:** `id` (uint) — card ID

**Request body:**
```json
{
  "front": "Updated question",
  "back": "Updated answer",
  "extra": "Optional extra context"
}
```

| Field | Type | Required | Notes |
|---|---|---|---|
| `title` | string | No | Optional preview title. |
| `front` | string | Yes | |
| `back` | string | Yes | |
| `card_type` | string | No | If omitted, preserves existing card type. |
| `extra` | string | No | |

**Response 200:** Card object.

**Errors:**
- `400` — `"invalid request body"` / `"invalid card id"` / `"front and back are required"`
- `401` — `"unauthorized"`
- `403` — `"forbidden"`
- `404` — `"card not found"`

---

### DELETE /cards/{id}

Delete a card.

**Path params:** `id` (uint) — card ID

**Response:** `204 No Content` (empty body)

**Errors:**
- `400` — `"invalid card id"`
- `401` — `"unauthorized"`
- `403` — `"forbidden"`
- `404` — `"card not found"`

---

## Study Sessions

The study system uses the **SM-2 spaced repetition algorithm**. There are two study modes:

- **`spaced`** — Returns cards due for review (where `next_review_at <= now`), sorted by due date.
- **`random`** — Returns all cards in the deck in random order.

After viewing a card, submit a quality rating to update the card's scheduling parameters.

### POST /decks/{id}/study

Start a study session for a deck.

**Path params:** `id` (uint) — deck ID

**Request body:**
```json
{
  "mode": "spaced"
}
```

| Field | Type | Required | Values |
|---|---|---|---|
| `mode` | string | Yes | `"spaced"` or `"random"` |

**Response 200:** Array of card objects (the cards to study in this session).

**Errors:**
- `400` — `"invalid request body"` / `"invalid deck id"` / `"mode is required"` / `"mode must be \"spaced\" or \"random\""`
- `401` — `"unauthorized"`
- `403` — `"forbidden"`
- `404` — `"not found"`

---

### POST /cards/{id}/review

Submit a quality rating after reviewing a card. Updates the card's SM-2 scheduling parameters (`interval`, `repetitions`, `ease_factor`, `next_review_at`).

**Path params:** `id` (uint) — card ID

**Request body:**
```json
{
  "quality": 4
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `quality` | int | Yes | SM-2 quality rating (0-5). 0=complete blackout, 5=perfect response. |

**Quality scale:**
| Value | Meaning |
|---|---|
| 0 | Complete blackout — no recall at all |
| 1 | Incorrect, but upon seeing the answer it felt familiar |
| 2 | Incorrect, but the answer seemed easy to recall once seen |
| 3 | Correct, but required significant effort |
| 4 | Correct, with some hesitation |
| 5 | Perfect response — instant recall |

**Response 200:** Card object (with updated scheduling fields).

**Errors:**
- `400` — `"invalid request body"` / `"invalid card id"` / `"quality is required"` / invalid quality range
- `401` — `"unauthorized"`
- `403` — `"forbidden"`
- `404` — `"not found"`

---

## Media

Uploaded media files are stored server-side and served by ID. This enables embedding images and audio in card content using standard HTML tags.

### Media Object

```json
{
  "id": 1,
  "public_id": "a3f9e2b1c4d87e6f0123456789abcdef",
  "filename": "diagram.png",
  "content_type": "image/png",
  "size": 20480,
  "url": "http://localhost:8080/api/v1/media/a3f9e2b1c4d87e6f0123456789abcdef"
}
```

The `public_id` is a random 32-character hex token. Media URLs use this token instead of the sequential database ID to prevent enumeration (IDOR).

### Supported content types

| Type | Extensions |
|---|---|
| `image/png` | `.png` |
| `image/jpeg` | `.jpg`, `.jpeg` |
| `image/gif` | `.gif` |
| `image/webp` | `.webp` |
| `image/svg+xml` | `.svg` |
| `audio/mpeg` | `.mp3` |
| `audio/mp4` | `.m4a` |

Maximum file size: **10 MB**.

---

### POST /media

Upload a media file. Requires authentication.

**Request:** `multipart/form-data` with field name `file`.

```
POST /api/v1/media
Authorization: Bearer <token>
Content-Type: multipart/form-data; boundary=...

--boundary
Content-Disposition: form-data; name="file"; filename="diagram.png"
Content-Type: image/png

<binary data>
--boundary--
```

**Response 201:** Media object.

**Errors:**
- `400` — `"request too large or invalid multipart form"` / `"missing form file field 'file'"` / `"unsupported content type: ..."` / `"file too large: ..."`
- `401` — `"unauthorized"`

---

### GET /media/{id}

Serve a media file directly. No authentication required. Suitable for use in `<img src="...">` or `<audio src="...">` tags.

**Path params:** `id` (string) — media `public_id` token (from the media object returned by upload)

**Response 200:** Raw file bytes with appropriate `Content-Type` and `Cache-Control: public, max-age=31536000, immutable` headers.

**Errors:**
- `404` — `"media not found"`

---

### DELETE /media/{id}

Delete a media file. Requires authentication. Only the owning user may delete their own files.

**Path params:** `id` (string) — media `public_id` token

**Response:** `204 No Content` (empty body)

**Errors:**
- `401` — `"unauthorized"`
- `403` — `"forbidden"`
- `404` — `"media not found"`

---

## Typical Frontend Flow

1. **Register** or **Login** to get a JWT token.
2. **Create decks** and **add cards** to them. Cards may be `basic` (front/back) or `cloze` (fill-in-the-blank with `{{cN::answer}}` syntax).
3. Optionally **upload media** (images, audio) and embed the returned URLs in card `front`/`back` content.
4. **Start a study session** (spaced or random mode).
5. For each card returned, display the front, let the user reveal the back, then **submit a review** with a quality rating.
6. The backend updates the card's scheduling — the next time a spaced session is started, only due cards appear.
7. **User profile** shows login streak and total points for gamification.
8. Decks can be made **public** and browsed/upvoted by other users.
