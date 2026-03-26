# Stage 1: Builder
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy dependency files first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and VERSION.
COPY . .

# Build with version embedded from VERSION file.
RUN go build -ldflags "-X main.version=$(cat VERSION)" -o /app/kpfc .

# Stage 2: Runner
FROM alpine:latest

# Install CA certificates for HTTPS calls (e.g., future OAuth2 providers).
RUN apk --no-cache add ca-certificates

# Run as non-root user.
RUN adduser -D appuser
USER appuser

WORKDIR /app

COPY --from=builder /app/kpfc .

EXPOSE 8080

CMD ["./kpfc"]
