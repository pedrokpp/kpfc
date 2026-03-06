# KPFC - Flashcards API

REST API de flashcards com spaced repetition (algoritmo SM-2), construída em Go com Clean Architecture.

## Stack

- **Go 1.22+**
- **Chi Router** - HTTP router leve e idiomático
- **DynamoDB** - Multi-table design (users, decks, cards)
- **S3** - Storage para profile pictures
- **JWT** - Autenticação stateless
- **OAuth2** - Google e GitHub
- **LocalStack** - Simulação AWS para desenvolvimento
- **Docker** - Containerização multi-stage

## Features

- ✅ CRUD de usuários com autenticação local (email/password)
- ✅ OAuth2 via Google e GitHub
- ✅ CRUD de decks (públicos ou privados)
- ✅ Clonagem de decks públicos
- ✅ CRUD de cards com algoritmo SM-2 para spaced repetition
- ✅ Upload de avatar para S3
- ✅ Clean Architecture (domain, usecase, repository, handler)
- ✅ RFC 7807 Problem Details para respostas de erro

## Estrutura do Projeto

```
kpfc/
├── cmd/api/main.go              # Entry point
├── internal/
│   ├── domain/                  # Entidades e interfaces (sem deps externas)
│   ├── usecase/                 # Lógica de negócio
│   ├── repository/              # Adaptadores DynamoDB e S3
│   ├── handler/                 # HTTP handlers
│   ├── middleware/              # JWT, CORS, logger, recovery
│   └── config/                  # Carregamento de .env
├── docs/                        # OpenAPI spec
├── docker/localstack/           # Scripts de inicialização LocalStack
├── Dockerfile
├── docker-compose.yml
└── AGENTS.md                    # Guia para agentes de IA
```

## Quick Start

### 1. Clone e configure

```bash
git clone <repo-url>
cd kpfc
cp .env.example .env
# Edite .env conforme necessário
```

### 2. Inicie com Docker Compose

```bash
docker-compose up
```

A API estará disponível em `http://localhost:8080/api/v1`

### 3. Desenvolvimento local (sem Docker)

```bash
# Inicie apenas o LocalStack
docker-compose up localstack

# Em outro terminal, rode a API
go run cmd/api/main.go
```

## Comandos Úteis

```bash
# Build
go build -o bin/api cmd/api/main.go

# Run
go run cmd/api/main.go

# Tests
go test ./...
go test -v -run TestCreateUser ./internal/usecase/user

# Lint
golangci-lint run
```

## Endpoints (Resumo)

Ver documentação completa em `docs/openapi.yaml`

### Auth
- `POST /api/v1/auth/register` - Registrar
- `POST /api/v1/auth/login` - Login

### Users (autenticado)
- `GET /api/v1/users/me` - Perfil
- `PUT /api/v1/users/me` - Atualizar perfil
- `DELETE /api/v1/users/me` - Deletar conta

### Decks (autenticado)
- `GET /api/v1/decks` - Listar meus decks
- `POST /api/v1/decks` - Criar deck
- `GET /api/v1/decks/public` - Listar decks públicos
- `POST /api/v1/decks/{deckId}/clone` - Clonar deck público

### Cards (autenticado)
- `GET /api/v1/decks/{deckId}/cards` - Listar cards
- `POST /api/v1/decks/{deckId}/cards` - Criar card
- `GET /api/v1/decks/{deckId}/cards/due` - Cards pendentes
- `POST /api/v1/decks/{deckId}/cards/{cardId}/review` - Revisar (SM-2)

## Algoritmo SM-2

O algoritmo SM-2 calcula o próximo intervalo de revisão baseado na qualidade da resposta:

- **0 (Again)**: Reseta intervalo para 1 dia
- **1 (Hard)**: Aumenta intervalo moderadamente
- **2 (Good)**: Progressão normal
- **3 (Easy)**: Aumenta intervalo rapidamente

## Licença

Este projeto está licenciado sob a **GNU Affero General Public License v3.0 (AGPL-3.0)**.

**Copyright 2026 kpp.dev**

Este é software livre: você pode redistribuir e/ou modificar sob os termos da GNU Affero General Public License conforme publicada pela Free Software Foundation, versão 3 da Licença, ou (a seu critério) qualquer versão posterior.

Este programa é distribuído na esperança de que seja útil, mas SEM QUALQUER GARANTIA; sem mesmo a garantia implícita de COMERCIALIZAÇÃO ou ADEQUAÇÃO A UM PROPÓSITO ESPECÍFICO. Veja a GNU Affero General Public License para mais detalhes.

Você deve ter recebido uma cópia da GNU Affero General Public License junto com este programa (veja arquivo `COPYING`). Caso contrário, veja <https://www.gnu.org/licenses/>.

### Código-fonte (AGPL §13)

Como este software interage com usuários via rede, a AGPL exige que o código-fonte seja disponibilizado. O código completo está disponível em:

**<https://github.com/pedrokpp/kpfc>**

## Contribuindo

Ver `AGENTS.md` para guia de código e style guidelines.
