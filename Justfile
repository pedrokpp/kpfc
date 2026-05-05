version := `cat VERSION`

# Build the binary locally with version embedding.
build:
    go build -ldflags "-X main.version={{version}}" -o kpfc .

# Build and run the server locally, sourcing .env.local for environment variables.
run: build
    set -a && source .env.local && set +a && ./kpfc

# Run all tests.
test:
    CGO_ENABLED=1 go test ./...

# Run static analysis.
lint:
    go vet ./...

# Build the Docker image.
docker-build:
    docker build -f docker/Dockerfile -t kpfc .

# Start the server via docker-compose.
docker-up:
    docker-compose up --build

# Stop docker-compose services.
docker-down:
    docker-compose down
