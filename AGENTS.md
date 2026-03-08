# AGENTS.md - Guia para Agentes de IA

Este documento fornece diretrizes para agentes de IA (como Claude, Cursor, Copilot) trabalharem no projeto **kpfc** (Flashcards API).

## Visão Geral do Projeto

**kpfc** é uma REST API de flashcards com spaced repetition (SM-2), construída em Go com Clean Architecture.

**Stack:** Go 1.22+, Chi Router, DynamoDB (multi-table), S3, JWT + OAuth2 (Google/GitHub), LocalStack (dev), Docker

**Domínio:**
- **Users**: autenticação local + OAuth2, profile picture (S3)
- **Decks**: coleções de cards, podem ser públicos ou privados, clonáveis
- **Cards**: frente/verso, algoritmo SM-2 para spaced repetition

**Relações:** User 1:N Decks, Deck 1:N Cards

---

## Comandos de Build/Lint/Test

> **Este projeto usa [Just](https://github.com/casey/just) como task runner.**  
> Execute `just` para ver todos os comandos disponíveis.

### Referência Rápida

#### Desenvolvimento
```bash
just run                    # Executar API local (requer .env)
just build                  # Compilar binário para produção
just clean                  # Limpar binários e artefatos
just setup                  # Setup inicial do projeto (primeira vez)
```

#### Testes
```bash
just test                   # Executar todos os testes
just test-race              # Testes com race detector
just test-coverage          # Testes com relatório de cobertura (HTML)
just test-package <pkg>     # Testar pacote específico (ex: just test-package user)
just test-watch             # Modo watch (requer gow)
```

#### Quality Assurance
```bash
just lint                   # Executar golangci-lint
just lint-fix               # Lint com auto-fix
just fmt                    # Formatar código (goimports + gofmt)
just fmt-check              # Verificar formatação sem modificar
just check                  # Pipeline completo: fmt + lint + test
just ci                     # Alias para check (simula CI)
```

#### Docker
```bash
just docker                 # Subir containers em background (padrão)
just docker-build           # Build da imagem Docker
just docker-up              # Subir containers em foreground
just docker-down            # Parar e remover containers
just docker-restart         # Reiniciar containers
just docker-logs [service]  # Ver logs (ex: just docker-logs api)
just localstack             # Apenas LocalStack standalone
```

#### Utilitários
```bash
just install-tools          # Instalar ferramentas de dev (golangci-lint, goimports, gow)
```

### Pre-commit Hook

O comando `just setup` instala automaticamente um **pre-commit hook** que executa `just check` antes de cada commit.

**Comportamento:**
- ✅ Formata código automaticamente
- ✅ Executa lint
- ✅ Roda todos os testes
- ❌ **Bloqueia o commit se alguma verificação falhar**

**Para pular o hook (use com cuidado):**
```bash
git commit --no-verify -m "WIP: work in progress"
```

### Comandos Go Nativos (Referência)

Os comandos abaixo são equivalentes diretos, úteis caso você não tenha o Just instalado:

#### Build
```bash
go build -o bin/api cmd/api/main.go
```

#### Run (development)
```bash
go run cmd/api/main.go
```

#### Lint
```bash
# Instalar golangci-lint se necessário
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Executar lint
golangci-lint run
```

#### Tests

##### Run all tests
```bash
go test ./...
```

##### Run tests with coverage
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

##### Run tests in a specific package
```bash
go test ./internal/usecase/user
go test ./internal/handler
```

##### Run a single test
```bash
go test -run TestCreateUser ./internal/usecase/user
go test -v -run TestUserHandler_Register ./internal/handler
```

##### Run tests with race detector
```bash
go test -race ./...
```

#### Docker

##### Build image
```bash
docker build -t kpfc:latest .
```

##### Run with LocalStack
```bash
docker-compose up
```

##### Run LocalStack standalone
```bash
docker-compose up localstack
```

---

## Estrutura do Projeto (Clean Architecture)

```
kpfc/
├── cmd/api/main.go              # Entry point, dependency injection
├── internal/
│   ├── domain/                  # Entidades, interfaces, erros de domínio (sem deps externas)
│   │   ├── user.go
│   │   ├── deck.go
│   │   ├── card.go
│   │   ├── storage.go
│   │   └── errors.go
│   ├── usecase/                 # Lógica de negócio (depende apenas de domain)
│   │   ├── user/
│   │   ├── deck/
│   │   └── card/
│   ├── repository/              # Adaptadores para persistência (implementam domain interfaces)
│   │   ├── dynamo/
│   │   └── s3/
│   ├── handler/                 # HTTP handlers (Chi)
│   ├── middleware/              # JWT, CORS, logger, recovery
│   └── config/                  # Carregamento de .env
├── docs/                        # OpenAPI spec
├── docker/                      # Docker configs (LocalStack init scripts)
├── Dockerfile
└── docker-compose.yml
```

**Princípios:** SOLID, DRY, KISS, Clean Code. Dependências sempre apontam para dentro (domain não depende de nada).

---

## Code Style Guidelines

### Imports

Ordem padrão:
1. Standard library
2. Third-party packages
3. Internal packages

```go
import (
	"context"
	"fmt"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"

	"kpp.dev/kpfc/internal/domain"
	"kpp.dev/kpfc/internal/middleware"
)
```

Use `goimports` para formatar automaticamente.

### Formatting

- Use **gofmt** (ou **goimports**) antes de commitar
- Tabs para indentação (padrão Go)
- Linha com no máximo 120 caracteres (soft limit)
- Evite trailing whitespace

### Types

- Use tipos nativos quando possível (`string`, `int`, `bool`)
- Use `time.Time` para timestamps
- Use `context.Context` como primeiro parâmetro de funções que fazem I/O
- Crie type aliases para melhorar legibilidade (ex: `type AuthProvider string`)
- Prefira structs com fields públicos (PascalCase) para DTOs/models
- Use tags JSON para serialização: `json:"field_name"`
- Use `json:"-"` para omitir fields sensíveis (ex: password)

### Naming Conventions

- **Packages**: lowercase, singular (ex: `user`, `card`, não `users`, `cards`)
- **Files**: snake_case (ex: `user_repository.go`, `deck_handler.go`)
- **Exported**: PascalCase (ex: `CreateUser`, `UserRepository`)
- **Unexported**: camelCase (ex: `validateEmail`, `hashPassword`)
- **Interfaces**: nome + sufixo ou apenas comportamento (ex: `UserRepository`, `Storage`)
- **Receiver names**: abreviação curta e consistente (ex: `func (u *UserUseCase)`, `func (h *Handler)`)

### Error Handling

- **SEMPRE** cheque erros explicitamente, nunca ignore com `_`
- Use `errors.New()` para erros simples
- Use `fmt.Errorf("context: %w", err)` para wrapping com contexto
- Defina erros de domínio em `internal/domain/errors.go` (ex: `ErrNotFound`, `ErrUnauthorized`)
- Retorne erros em vez de panic (exceto em casos irrecuperáveis no `main`)
- Handlers HTTP devem converter erros de domínio para respostas RFC 7807 (Problem Details)

```go
// Bom
user, err := repo.GetByID(ctx, id)
if err != nil {
    if errors.Is(err, domain.ErrNotFound) {
        return nil, fmt.Errorf("user not found: %w", err)
    }
    return nil, fmt.Errorf("failed to get user: %w", err)
}

// Ruim
user, _ := repo.GetByID(ctx, id)  // NUNCA faça isso
```

### Comentários

- **Evite comentários óbvios** — código deve ser auto-explicativo
- Use comentários apenas para:
  - **Documentar regras de negócio não triviais**
  - **Explicar "por quê", não "o quê"**
  - **Package-level doc comments** (obrigatório para packages exportados)
  - **Exported functions/types** (godoc)

```go
// Bom: explica regra de negócio SM-2
// CalculateNextInterval usa o algoritmo SM-2 para determinar
// o próximo intervalo baseado na qualidade da resposta.
// Qualidade 0-1: intervalo reseta para 1 dia
// Qualidade 2+: intervalo multiplicado pelo ease factor
func CalculateNextInterval(quality ReviewQuality, interval int, easeFactor float64) int {
    // ...
}

// Ruim: comentário óbvio
// GetByID retorna um usuário pelo ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
    // ...
}
```

### Testing

- Testes unitários: mock repositories, teste lógica isolada (use cases)
- Testes de integração: use `httptest` para handlers
- Nome dos testes: `TestFunctionName_Scenario` (ex: `TestCreateUser_Success`, `TestCreateUser_DuplicateEmail`)
- Use table-driven tests para múltiplos casos

```go
func TestCalculateNextInterval(t *testing.T) {
	tests := []struct {
		name       string
		quality    domain.ReviewQuality
		interval   int
		easeFactor float64
		want       int
	}{
		{"again resets to 1", domain.ReviewQualityAgain, 10, 2.5, 1},
		{"hard decreases interval", domain.ReviewQualityHard, 10, 2.5, 6},
		{"good maintains progression", domain.ReviewQualityGood, 10, 2.5, 25},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateNextInterval(tt.quality, tt.interval, tt.easeFactor)
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}
```

---

## Git Workflow

### Branches

- `main`: produção, sempre estável
- `develop`: branch de desenvolvimento
- `feature/<nome>`: novas features
- `fix/<nome>`: correções de bugs

### Conventional Commits

Formato: `<type>(<scope>): <subject>`

**Types:**
- `feat`: nova funcionalidade
- `fix`: correção de bug
- `refactor`: refatoração sem mudança de comportamento
- `test`: adicionar/modificar testes
- `docs`: documentação
- `chore`: tarefas de manutenção (deps, configs)
- `perf`: melhorias de performance

**Exemplos:**
```
feat(auth): add google oauth2 integration
fix(card): correct sm-2 interval calculation
refactor(user): extract validation logic to separate function
test(deck): add unit tests for clone functionality
docs(readme): update installation instructions
```

---

## API Endpoints (Referência)

**Base URL:** `/api/v1`

### Auth
- `POST /auth/register` - Registrar usuário local
- `POST /auth/login` - Login (email/password)
- `GET /auth/google` - Iniciar OAuth2 Google
- `GET /auth/google/callback` - Callback OAuth2 Google
- `GET /auth/github` - Iniciar OAuth2 GitHub
- `GET /auth/github/callback` - Callback OAuth2 GitHub

### Users (autenticado)
- `GET /users/me` - Perfil do usuário
- `PUT /users/me` - Atualizar perfil
- `DELETE /users/me` - Deletar conta
- `POST /users/me/avatar` - Upload avatar (S3)

### Decks (autenticado)
- `GET /decks` - Listar decks do usuário
- `POST /decks` - Criar deck
- `GET /decks/{deckId}` - Obter deck por ID
- `PUT /decks/{deckId}` - Atualizar deck
- `DELETE /decks/{deckId}` - Deletar deck
- `GET /decks/public` - Listar decks públicos
- `POST /decks/{deckId}/clone` - Clonar deck público

### Cards (autenticado)
- `GET /decks/{deckId}/cards` - Listar cards do deck
- `POST /decks/{deckId}/cards` - Criar card
- `GET /decks/{deckId}/cards/{cardId}` - Obter card por ID
- `PUT /decks/{deckId}/cards/{cardId}` - Atualizar card
- `DELETE /decks/{deckId}/cards/{cardId}` - Deletar card
- `POST /decks/{deckId}/cards/{cardId}/review` - Revisar card (SM-2)
- `GET /decks/{deckId}/cards/due` - Cards pendentes para revisão

---

## Notas Adicionais

- **AWS SDK:** Use `aws-sdk-go-v2` (não v1)
- **DynamoDB:** Multi-table design (kpfc_users, kpfc_decks, kpfc_cards)
- **LocalStack:** Endpoint `http://localhost:4566`, credentials `test`/`test`
- **Environment:** Configure via `.env` (copie `.env.example`)
