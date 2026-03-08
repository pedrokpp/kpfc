# Documentação de Testes - kpfc

## 📊 Resumo de Cobertura

```
internal/usecase/card: 73.0% coverage
internal/usecase/deck: 68.1% coverage
internal/usecase/user: 76.4% coverage
```

**Foco:** Qualidade sobre quantidade - todos os testes têm propósito claro e cobrem regras de negócio críticas.

---

## 🧪 Testes Implementados

### ✅ Testes Unitários - Algoritmo SM-2 (`internal/usecase/card/sm2_test.go`)

**Cobertura:** Todas as regras do algoritmo Spaced Repetition.

| Teste | Descrição | Importância |
|-------|-----------|-------------|
| `TestCalculateNextReview_AgainResetsInterval` | Quality=0 reseta intervalo para 1 dia | 🔴 Crítico |
| `TestCalculateNextReview_GoodFirstReview` | Quality=2 no primeiro review vai para 6 dias | 🔴 Crítico |
| `TestCalculateNextReview_GoodProgression` | Quality=2 multiplica pelo ease factor | 🔴 Crítico |
| `TestCalculateEaseFactor_MinimumBound` | Ease factor nunca menor que 1.3 | 🔴 Crítico |
| `TestCalculateNextReview_InvalidQualityDefaultsAgain` | Quality fora de 0-3 tratado como 0 | 🟡 Importante |
| `TestCalculateNextReview_TableDriven` | Table-driven test com todos os casos | 🔴 Crítico |

**Comando:** `go test ./internal/usecase/card/sm2_test.go ./internal/usecase/card/sm2.go`

---

### ✅ Testes Unitários - Card UseCase (`internal/usecase/card/card_usecase_test.go`)

**Cobertura:** Validações de input, autorização (segurança crítica), e aplicação do SM-2.

| Teste | Descrição | Importância |
|-------|-----------|-------------|
| `TestCreateCard_EmptyFront` | Front vazio retorna ErrInvalidInput | 🟡 Importante |
| `TestCreateCard_EmptyBack` | Back vazio retorna ErrInvalidInput | 🟡 Importante |
| `TestCreateCard_FrontTooLong` | Front > 1000 chars retorna ErrInvalidInput | 🟡 Importante |
| `TestCreateCard_Forbidden` | Criar card em deck de outro user = ErrForbidden | 🔴 Crítico |
| `TestGetCard_Forbidden` | Ler card de outro user = ErrForbidden | 🔴 Crítico |
| `TestUpdateCard_Forbidden` | Atualizar card de outro user = ErrForbidden | 🔴 Crítico |
| `TestDeleteCard_Forbidden` | Deletar card de outro user = ErrForbidden | 🔴 Crítico |
| `TestReviewCard_Forbidden` | Revisar card de outro user = ErrForbidden | 🔴 Crítico |
| `TestReviewCard_AppliesSM2` | Review aplica SM-2 e persiste | 🔴 Crítico |

**Comando:** `go test ./internal/usecase/card/`

---

### ✅ Testes Unitários - Deck UseCase (`internal/usecase/deck/deck_usecase_test.go`)

**Cobertura:** Validações, autorização, e clonagem de decks públicos.

| Teste | Descrição | Importância |
|-------|-----------|-------------|
| `TestCreateDeck_EmptyTitle` | Title vazio retorna ErrInvalidInput | 🟡 Importante |
| `TestCreateDeck_TitleTooLong` | Title > 1000 chars retorna ErrInvalidInput | 🟡 Importante |
| `TestGetDeck_Forbidden` | Ler deck privado de outro user = ErrForbidden | 🔴 Crítico |
| `TestUpdateDeck_Forbidden` | Atualizar deck de outro user = ErrForbidden | 🔴 Crítico |
| `TestDeleteDeck_Forbidden` | Deletar deck de outro user = ErrForbidden | 🔴 Crítico |
| `TestCloneDeck_PublicDeckSuccess` | Clonar deck público funciona | 🔴 Crítico |
| `TestCloneDeck_PrivateDeckForbidden` | Clonar deck privado = ErrForbidden | 🔴 Crítico |

**Comando:** `go test ./internal/usecase/deck/`

---

### ✅ Testes Unitários - User UseCase (`internal/usecase/user/user_usecase_test.go`)

**Cobertura:** Validações de email/password, autenticação, OAuth2, e duplicação.

| Teste | Descrição | Importância |
|-------|-----------|-------------|
| `TestRegister_EmptyEmail` | Email vazio retorna ErrInvalidInput | 🟡 Importante |
| `TestRegister_InvalidEmailFormat` | Email inválido retorna ErrInvalidInput | 🟡 Importante |
| `TestRegister_EmptyPassword` | Password vazio retorna ErrInvalidInput | 🟡 Importante |
| `TestRegister_WeakPassword` | Password < 8 chars retorna ErrInvalidInput | 🟡 Importante |
| `TestRegister_DuplicateEmail` | Email duplicado retorna ErrAlreadyExists | 🔴 Crítico |
| `TestRegister_PasswordHashed` | Password hasheado com bcrypt | 🔴 Crítico |
| `TestLogin_InvalidPassword` | Senha errada retorna ErrInvalidCredentials | 🔴 Crítico |
| `TestLogin_UserNotFound` | Email inexistente retorna ErrInvalidCredentials (não vaza) | 🔴 Crítico |
| `TestCreateOrGetOAuthUser_NewUser` | Cria user OAuth2 se não existe | 🟡 Importante |
| `TestCreateOrGetOAuthUser_ExistingUser` | Retorna user existente | 🟡 Importante |

**Comando:** `go test ./internal/usecase/user/`

---

### ✅ Testes E2E - LocalStack (`internal/repository/integration/localstack_test.go`)

**Cobertura:** CRUD real com DynamoDB via LocalStack.

| Teste | Descrição | Importância |
|-------|-----------|-------------|
| `TestE2E_LocalStackHealthCheck` | Verifica conexão TCP com LocalStack | 🟢 Bom ter |
| `TestE2E_DynamoDBUserCRUD` | CRUD completo de users no DynamoDB | 🟡 Importante |
| `TestE2E_DynamoDBDeckCRUD` | CRUD completo de decks no DynamoDB | 🟡 Importante |
| `TestE2E_DynamoDBCardCRUD` | CRUD completo de cards no DynamoDB | 🟡 Importante |

**Comportamento:** Testes automaticamente **skipam** se LocalStack não estiver disponível em `localhost:4566` (conexão TCP).

**Como executar:**
```bash
# Iniciar LocalStack
docker-compose up localstack

# Rodar testes E2E
go test ./internal/repository/integration/ -v
```

---

## 🛠️ Validações Adicionadas aos Use Cases

### Card UseCase
- Front não vazio
- Back não vazio
- Front <= 1000 caracteres
- Back <= 1000 caracteres

### Deck UseCase
- Title não vazio
- Title <= 1000 caracteres
- Clonagem valida se deck é público

### User UseCase
- Email não vazio
- Email formato válido (regex: `^[^\s@]+@[^\s@]+\.[^\s@]+$`)
- Password não vazio
- Password >= 8 caracteres
- Name não vazio

---

## 📦 Infraestrutura de Testes

### Test Helpers (`internal/testutil/testutil.go`)
```go
NewTestUser(email, name string) *domain.User
NewTestDeck(userID, title string, isPublic bool) *domain.Deck
NewTestCard(deckID, front, back string) *domain.Card
AssertTimeNear(t *testing.T, expected, actual time.Time, delta time.Duration)
```

### Mocks (`internal/testutil/mocks.go`)
Implementados com `testify/mock`:
- `MockUserRepository`
- `MockDeckRepository`
- `MockCardRepository`
- `MockStorageRepository`

---

## 🚀 Comandos Úteis

### Executar todos os testes
```bash
go test ./...
```

### Executar testes com coverage
```bash
go test -cover ./internal/usecase/...
```

### Executar apenas testes unitários (skip E2E)
```bash
go test ./internal/usecase/...
```

### Executar testes de um package específico
```bash
go test ./internal/usecase/card/
go test ./internal/usecase/deck/
go test ./internal/usecase/user/
```

### Executar apenas testes críticos (SM-2)
```bash
go test ./internal/usecase/card/sm2_test.go ./internal/usecase/card/sm2.go -v
```

### Coverage detalhado por arquivo
```bash
go test -coverprofile=coverage.out ./internal/usecase/...
go tool cover -html=coverage.out
```

---

## ✅ Métricas de Sucesso

### Todas as regras de negócio críticas testadas
- ✅ SM-2 funcionando corretamente
- ✅ Autorização impedindo acessos não autorizados
- ✅ Clonagem de decks públicos funcionando
- ✅ Autenticação validando credenciais
- ✅ Validações de input em todos os use cases

### Todos os edge cases de segurança cobertos
- ✅ Forbidden em operações cross-user
- ✅ Password hashing com bcrypt
- ✅ Login não vaza existência de email
- ✅ Clonagem apenas de decks públicos

### Testes rápidos e confiáveis
- ✅ Testes unitários < 1s
- ✅ E2E skipam automaticamente se LocalStack indisponível
- ✅ Mocks com testify/mock para velocidade

### Testes fáceis de manter
- ✅ Table-driven tests onde aplicável
- ✅ Nomes descritivos (`TestFunction_Scenario`)
- ✅ Helpers para reduzir boilerplate
- ✅ Sem duplicação excessiva

---

## 🔜 Próximos Passos (Opcional)

Para expansão futura:

1. **Testes HTTP Handlers** - Adicionar testes de integração HTTP com `httptest` para validar:
   - Status codes corretos (200, 201, 400, 401, 403, 404)
   - Respostas JSON RFC 7807 (Problem Details)
   - Middleware JWT funcionando

2. **Testes de Performance** - Benchmarks para:
   - Algoritmo SM-2
   - Queries DynamoDB

3. **Testes de Carga** - Simular múltiplos usuários com `vegeta` ou `k6`

4. **Mutation Testing** - Validar qualidade dos testes com `go-mutesting`

---

## 📚 Convenções

### Nomenclatura de Testes
```
Test<FunctionName>_<Scenario>
```

Exemplos:
- `TestCalculateNextReview_AgainResetsInterval`
- `TestCreateCard_Forbidden`
- `TestRegister_DuplicateEmail`

### Estrutura de Teste (AAA Pattern)
```go
func TestFoo_Scenario(t *testing.T) {
    // Arrange - Setup
    repo := &testutil.MockRepository{}
    uc := NewUseCase(repo)
    
    // Act - Execute
    result, err := uc.DoSomething()
    
    // Assert - Verify
    assert.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

---

**Autor:** Implementado com foco em qualidade > quantidade  
**Data:** 2026-03-06  
**Versão:** 1.0
