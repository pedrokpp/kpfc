# Justfile para kpfc (Flashcards API)
# Comandos unificados para build, test, lint e docker

# Configurações
set dotenv-load := true
set shell := ["bash", "-c"]

# Variáveis
go_module := "kpp.dev/kpfc"
bin_dir := "bin"
binary_name := "api"
coverage_file := "coverage.out"

# Receita padrão: mostrar lista de comandos
default:
    @just --list

# ============================================================================
# DESENVOLVIMENTO
# ============================================================================

# Executar API em modo desenvolvimento (requer .env)
run:
    go run cmd/api/main.go

# Compilar binário de produção
build:
    @echo "Compilando binário..."
    @mkdir -p {{bin_dir}}
    go build -ldflags="-s -w" -o {{bin_dir}}/{{binary_name}} cmd/api/main.go
    @echo "✓ Binário criado em {{bin_dir}}/{{binary_name}}"

# Limpar binários e artefatos de build
clean:
    @echo "Limpando artefatos..."
    rm -rf {{bin_dir}}
    rm -f {{coverage_file}}
    @echo "✓ Limpeza concluída"

# ============================================================================
# TESTES
# ============================================================================

# Executar todos os testes
test:
    go test ./...

# Executar testes com race detector
test-race:
    go test -race ./...

# Executar testes com cobertura e gerar HTML
test-coverage:
    @echo "Executando testes com cobertura..."
    go test -coverprofile={{coverage_file}} ./...
    go tool cover -html={{coverage_file}}
    @echo "✓ Cobertura gerada em {{coverage_file}}"

# Executar testes em modo watch (requer gow: go install github.com/mitranim/gow@latest)
test-watch:
    @command -v gow >/dev/null 2>&1 || (echo "gow não encontrado. Instale com: go install github.com/mitranim/gow@latest" && exit 1)
    gow test ./...

# Executar testes de um pacote específico
test-package pkg:
    go test -v ./internal/{{pkg}}/...

# ============================================================================
# QUALITY ASSURANCE
# ============================================================================

# Executar linter (golangci-lint)
lint:
    @command -v golangci-lint >/dev/null 2>&1 || (echo "golangci-lint não encontrado. Execute: just install-tools" && exit 1)
    golangci-lint run

# Executar linter com auto-fix
lint-fix:
    @command -v golangci-lint >/dev/null 2>&1 || (echo "golangci-lint não encontrado. Execute: just install-tools" && exit 1)
    golangci-lint run --fix

# Formatar código (goimports + gofmt)
fmt:
    @command -v goimports >/dev/null 2>&1 || (echo "goimports não encontrado. Execute: just install-tools" && exit 1)
    @echo "Formatando código..."
    goimports -w .
    gofmt -s -w .
    @echo "✓ Código formatado"

# Verificar formatação sem modificar
fmt-check:
    @command -v goimports >/dev/null 2>&1 || (echo "goimports não encontrado. Execute: just install-tools" && exit 1)
    @echo "Verificando formatação..."
    @test -z "$(gofmt -l .)" || (echo "Arquivos não formatados:" && gofmt -l . && exit 1)
    @echo "✓ Código está formatado corretamente"

# Pipeline completo: fmt + lint + test (simula CI)
check: fmt lint test
    @echo "✓ Todas as verificações passaram!"

# ============================================================================
# DOCKER
# ============================================================================

# Subir containers em background (padrão)
docker:
    docker-compose up -d --force-recreate

# Build da imagem Docker
docker-build:
    docker build -t kpfc:latest .

# Subir containers em foreground
docker-up:
    docker-compose up

# Parar e remover containers
docker-down:
    docker-compose down

# Reiniciar containers
docker-restart:
    docker-compose restart

# Ver logs dos containers (opcional: especificar serviço)
docker-logs service="":
    #!/usr/bin/env bash
    if [ -z "{{service}}" ]; then
        docker-compose logs -f
    else
        docker-compose logs -f {{service}}
    fi

# Apenas LocalStack standalone
localstack:
    docker-compose up localstack

# ============================================================================
# SETUP E UTILITÁRIOS
# ============================================================================

# Setup inicial do projeto
setup:
    #!/usr/bin/env bash
    set -euo pipefail
    
    echo "🚀 Configurando projeto kpfc..."
    
    # 1. Copiar .env.example se não existir .env
    if [ ! -f .env ]; then
        echo "📝 Criando .env a partir de .env.example..."
        cp .env.example .env
        echo "✓ .env criado. Configure as variáveis antes de rodar a API."
    else
        echo "ℹ️  .env já existe, pulando..."
    fi
    
    # 2. Instalar ferramentas de desenvolvimento
    echo "🔧 Instalando ferramentas de desenvolvimento..."
    just install-tools
    
    # 3. Baixar dependências Go
    echo "📦 Baixando dependências Go..."
    go mod download
    
    # 4. Iniciar LocalStack
    echo "🐳 Iniciando LocalStack..."
    docker-compose up -d localstack
    
    # 5. Aguardar LocalStack ficar pronto
    echo "⏳ Aguardando LocalStack inicializar..."
    sleep 5
    
    # 6. Configurar pre-commit hook
    echo "🪝 Configurando pre-commit hook..."
    if [ -f .git/hooks/pre-commit ]; then
        echo "ℹ️  Pre-commit hook já existe, pulando..."
    else
        echo '#!/usr/bin/env bash' > .git/hooks/pre-commit
        echo '# Pre-commit hook para kpfc' >> .git/hooks/pre-commit
        echo '# Executa verificações de qualidade antes de cada commit' >> .git/hooks/pre-commit
        echo '' >> .git/hooks/pre-commit
        echo 'set -e' >> .git/hooks/pre-commit
        echo '' >> .git/hooks/pre-commit
        echo 'echo "🔍 Executando verificações pre-commit..."' >> .git/hooks/pre-commit
        echo '' >> .git/hooks/pre-commit
        echo '# Verificar se Just está instalado' >> .git/hooks/pre-commit
        echo 'if ! command -v just &> /dev/null; then' >> .git/hooks/pre-commit
        echo '    echo "❌ Erro: '\''just'\'' não encontrado. Instale em: https://github.com/casey/just"' >> .git/hooks/pre-commit
        echo '    exit 1' >> .git/hooks/pre-commit
        echo 'fi' >> .git/hooks/pre-commit
        echo '' >> .git/hooks/pre-commit
        echo '# Executar pipeline de CI local' >> .git/hooks/pre-commit
        echo 'if just check; then' >> .git/hooks/pre-commit
        echo '    echo "✅ Verificações concluídas. Prosseguindo com commit..."' >> .git/hooks/pre-commit
        echo '    exit 0' >> .git/hooks/pre-commit
        echo 'else' >> .git/hooks/pre-commit
        echo '    echo "❌ Pre-commit falhou! Corrija os erros antes de commitar."' >> .git/hooks/pre-commit
        echo '    echo "💡 Dica: use '\''git commit --no-verify'\'' para pular as verificações (não recomendado)"' >> .git/hooks/pre-commit
        echo '    exit 1' >> .git/hooks/pre-commit
        echo 'fi' >> .git/hooks/pre-commit
        chmod +x .git/hooks/pre-commit
        echo "✓ Pre-commit hook configurado"
    fi
    
    echo ""
    echo "✅ Setup concluído!"
    echo ""
    echo "Próximos passos:"
    echo "  1. Configure o arquivo .env com suas credenciais OAuth2"
    echo "  2. Execute 'just run' para iniciar a API"
    echo "  3. Ou execute 'just docker up' para rodar com Docker"
    echo ""
    echo "⚠️  Pre-commit hook instalado: 'just check' será executado antes de cada commit"
    echo "   Para pular: git commit --no-verify (use com cuidado!)"

# Instalar ferramentas de desenvolvimento
install-tools:
    @echo "Instalando ferramentas de desenvolvimento..."
    @echo "→ golangci-lint..."
    @go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    @echo "→ goimports..."
    @go install golang.org/x/tools/cmd/goimports@latest
    @echo "→ gow (opcional - hot reload)..."
    @go install github.com/mitranim/gow@latest
    @echo "✓ Ferramentas instaladas"

# Alias para check (CI)
ci: check

deploy-dev: docker-build docker
