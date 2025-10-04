.PHONY: help
help: ## Show help
	@echo "\033[36m╔════════════════════════════════════════════╗\033[0m"
	@echo "\033[36m║  \033[35mNexSpaces API - Commands\033[36m               ║\033[0m"
	@echo "\033[36m╚════════════════════════════════════════════╝\033[0m"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
	@echo ""

# ==============================================================================
# Development
# ==============================================================================

dev: ## Run development server
	go run cmd/server/main.go

build: ## Build binary
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/nexspaces-api cmd/server/main.go

run: build ## Build and run
	./bin/nexspaces-api

# ==============================================================================
# Docker
# ==============================================================================

up: ## Start all services
	docker-compose up -d
	@echo "\n\033[32m✓ Services started\033[0m"
	@echo "\033[36m→ API:     http://localhost:8080\033[0m"
	@echo "\033[36m→ Swagger: http://localhost:8080/docs\033[0m"

down: ## Stop all services
	docker-compose down

restart: down up ## Restart services

rebuild: ## Rebuild and restart API
	docker-compose up -d --build api

logs: ## View API logs
	docker-compose logs -f api

logs-all: ## View all logs
	docker-compose logs -f

ps: ## Show containers
	docker-compose ps

shell: ## Shell into API container
	docker-compose exec api sh

db-shell: ## psql shell
	docker-compose exec postgres psql -U postgres -d nexspaces_dev

clean: ## Clean Docker resources
	docker-compose down -v --remove-orphans
	docker system prune -f

# ==============================================================================
# Database
# ==============================================================================

DB_HOST ?= localhost
DB_USER ?= postgres
DB_PASSWORD ?= postgres
DB_NAME ?= nexspaces_dev

migrate-up: ## Run migrations
	@echo "\033[32m▶ Running migrations...\033[0m"
	@for file in migrations/*.up.sql; do \
		[ -f "$$file" ] || continue; \
		echo "  → $$(basename $$file)"; \
		PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -U $(DB_USER) -d $(DB_NAME) -f $$file -q || exit 1; \
	done
	@echo "\033[32m✓ Migrations complete\033[0m"

migrate-down: ## Rollback migrations
	@echo "\033[33m▶ Rolling back migrations...\033[0m"
	@for file in $$(ls -r migrations/*.down.sql 2>/dev/null); do \
		echo "  → $$(basename $$file)"; \
		PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -U $(DB_USER) -d $(DB_NAME) -f $$file -q || exit 1; \
	done
	@echo "\033[33m✓ Rollback complete\033[0m"

migrate-status: ## Show migration files
	@echo "Up migrations:"
	@ls -1 migrations/*.up.sql 2>/dev/null || echo "  None"
	@echo "\nDown migrations:"
	@ls -1 migrations/*.down.sql 2>/dev/null || echo "  None"

docker-migrate: ## Run migrations in Docker
	docker-compose exec api make migrate-up DB_HOST=postgres

# ==============================================================================
# Testing
# ==============================================================================

test: ## Run tests
	go test -v -race ./...

test-coverage: ## Test with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "\033[32m✓ Coverage: coverage.html\033[0m"

test-short: ## Run short tests
	go test -v -short ./...

# ==============================================================================
# Code Quality
# ==============================================================================

fmt: ## Format code
	gofmt -s -w .
	go mod tidy

lint: ## Run linter
	golangci-lint run --timeout 5m

lint-fix: ## Fix linting issues
	golangci-lint run --fix --timeout 5m

vet: ## Run go vet
	go vet ./...

check: fmt vet lint test ## Run all checks

# ==============================================================================
# OpenAPI
# ==============================================================================

openapi-validate: ## Validate OpenAPI spec
	docker run --rm -v $(CURDIR):/work -w /work stoplight/spectral:6 lint api/openapi.yaml

openapi-diff: ## Compare with main branch
	docker run --rm -v $(CURDIR):/work -w /work redocly/cli:latest diff api/openapi.yaml --branch=origin/main || true

# ==============================================================================
# Utilities
# ==============================================================================

deps: ## Install dependencies
	go mod download
	go mod tidy

deps-upgrade: ## Upgrade dependencies
	go get -u ./...
	go mod tidy

clean-all: clean ## Clean everything
	rm -rf bin/ coverage.out coverage.html

version: ## Show versions
	@echo "Go:            $$(go version)"
	@echo "Docker:        $$(docker --version)"
	@echo "Docker Compose: $$(docker-compose --version)"

.DEFAULT_GOAL := help
