.PHONY: all build clean run dev frontend backend test

# Default target
all: build

# Build both frontend and backend
build: frontend-build
	cd backend && go build -o openpost ./cmd/openpost

# Build frontend (SvelteKit -> static)
frontend-build:
	cd web && bun install && bun run build

# Development mode - runs frontend and backend separately
dev:
	@echo "Starting development servers..."
	@make -j2 dev-frontend dev-backend

dev-frontend:
	cd web && bun install && bun run dev

dev-backend:
	cd backend && go run ./cmd/openpost

# Run the binary
run:
	cd backend && ./openpost

# Clean build artifacts
clean:
	rm -rf backend/openpost
	rm -rf web/.svelte-kit
	rm -rf web/node_modules
	rm -f backend/*.db

# Run tests
test:
	cd backend && go test ./...
	cd web && bun run test

# Install dependencies
install:
	cd web && bun install
	cd backend && go mod download

# Create production .env from example
setup:
	cp backend/.env.example backend/.env
	@echo "Created backend/.env - edit with your OAuth credentials"

# Docker build
docker-build:
	docker build -t openpost:latest -f docker/Dockerfile .

# Docker run
docker-run:
	docker run -d -p 8080:8080 --name openpost openpost:latest