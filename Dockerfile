# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies for CGO (required for SQLite)
RUN apk add --no-cache gcc musl-dev

# Download dependencies first (better caching)
COPY go.mod go.sum ./
RUN GOTOOLCHAIN=auto go mod download

# Copy source code and build
COPY . .
RUN CGO_ENABLED=1 GOTOOLCHAIN=auto go build -o server ./cmd/server

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /app

# Copy binary and web assets
COPY --from=builder /app/server .
COPY web/ ./web/

EXPOSE 3000

CMD ["./server"]
