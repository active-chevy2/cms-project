.PHONY: help build run dev test clean docker-build docker-up docker-down

help:
	@echo "Available commands:"
	@echo "  make dev              - Run locally with hot reload (requires air)"
	@echo "  make build            - Build binary"
	@echo "  make run              - Run compiled binary"
	@echo "  make test             - Run tests (when available)"
	@echo "  make clean            - Remove binaries"
	@echo "  make docker-build     - Build Docker image"
	@echo "  make docker-up        - Start with docker-compose"
	@echo "  make docker-down      - Stop docker-compose"
	@echo "  make docker-logs      - View docker logs"
	@echo "  make install-deps     - Install Go dependencies"

install-deps:
	go mod download
	go mod tidy

dev:
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air

build:
	go build -o cms

run: build
	./cms

test:
	go test ./...

clean:
	rm -f cms

docker-build:
	docker build -t cms:latest .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f app

docker-restart:
	docker-compose down
	docker-compose up -d

db-logs:
	docker-compose logs -f db

# Development setup
setup:
	cp .env.example .env
	go mod download
	mkdir -p static/uploads
	touch static/uploads/.gitkeep
	@echo "Setup complete! Edit .env and run 'make docker-up' or 'make dev'"
