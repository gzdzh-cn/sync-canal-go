# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main .

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates and timezone data
RUN apk add --no-cache ca-certificates tzdata

# Copy binary and resources
COPY --from=builder /app/main .
COPY --from=builder /app/resource ./resource
COPY --from=builder /app/manifest ./manifest

# Expose port
EXPOSE 8000

# Run
CMD ["./main"]
