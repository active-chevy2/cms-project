# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install git (needed for go mod download)
RUN apk add --no-cache git

# Copy go.mod and go.sum first (to cache dependency download)
COPY go.mod go.sum* ./
RUN go mod download

# Now copy the rest of the source code
COPY . .

# Ensure go.sum is complete based on the actual imports in the source
RUN go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cms .

# Final lightweight stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary and assets
COPY --from=builder /build/cms .
COPY templates ./templates
COPY static ./static

# Create uploads directory
RUN mkdir -p ./static/uploads && chmod 755 ./static/uploads

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1

CMD ["./cms"]
