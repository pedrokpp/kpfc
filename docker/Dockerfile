# Stage 1: Builder
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Copy dependency files first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and VERSION.
COPY . .

# Build with CGO disabled (pure-Go postgres driver) and version embedded from VERSION file.
RUN CGO_ENABLED=0 go build -ldflags "-X main.version=$(cat VERSION)" -o /app/kpfc .

# Stage 2: Runner
FROM alpine:latest

RUN apk --no-cache add ca-certificates

# Run as non-root user.
RUN adduser -D appuser
USER appuser

WORKDIR /app

COPY --from=builder /app/kpfc .

EXPOSE 8080

CMD ["./kpfc"]
