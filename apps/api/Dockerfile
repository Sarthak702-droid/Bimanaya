# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/migrate ./cmd/migrate/main.go

# Run stage
FROM alpine:3.19

WORKDIR /app

# Copy binaries
COPY --from=builder /app/api /app/api
COPY --from=builder /app/migrate /app/migrate

# Copy migrations files
COPY db/migrations /app/db/migrations

# Copy entrypoint script
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/app/entrypoint.sh"]
