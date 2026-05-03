# Stage 1: Builder
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Copy dependency files first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and VERSION.
COPY . .

# SQLite via go-sqlite3 requires CGO during build.
RUN apk --no-cache add build-base
RUN go build -ldflags "-X main.version=$(cat VERSION)" -o /app/kpfc .

# Stage 2: Runner
FROM alpine:latest

RUN apk --no-cache add ca-certificates

# Run as non-root user.
RUN adduser -D appuser \
	&& mkdir -p /app /data /media \
	&& chown -R appuser:appuser /app /data /media

WORKDIR /app

COPY --from=builder /app/kpfc /app/kpfc

USER appuser

EXPOSE 8080

CMD ["./kpfc"]
