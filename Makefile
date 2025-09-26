# Makefile for NexSpaces API

# ====================================================================================
# VARIABLES
# ====================================================================================

# Docker-compose file
COMPOSE_FILE := -f infra/docker/docker-compose.yml

# Database connection string for migrate tool
# Note: This connects from the 'api' container to the 'postgres' container.
DB_USER := postgres
DB_PASSWORD := password
DB_HOST := postgres
DB_PORT := 5432
DB_NAME := nexspaces_dev
DB_URL := "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable"

.PHONY: help dev prod stop logs clean build db-create migrate-create migrate-up migrate-down

# ====================================================================================
# GENERAL COMMANDS
# ====================================================================================

help:
	@echo "🚀 NexSpaces API Commands"
	@echo ""
	@echo "  dev              - Start development environment (with hot reload)"
	@echo "  prod             - Start production environment"
	@echo "  stop             - Stop all services"
	@echo "  logs             - Show logs for development services"
	@echo "  clean            - Stop and remove all containers and volumes"
	@echo "  build            - Build the Go binary for production"
	@echo "  db-create        - Create the development database inside Docker"
	@echo "  migrate-create   - Create a new migration file. Usage: make migrate-create name=your_migration_name"
	@echo "  migrate-up       - Apply all pending database migrations"
	@echo "  migrate-down     - Revert the last applied database migration"

# ====================================================================================
# DEVELOPMENT & DOCKER COMMANDS
# ====================================================================================

dev:
	@echo "🔧 Starting development environment..."
	@docker-compose $(COMPOSE_FILE) up -d --build
	@echo "✅ API started at http://localhost:8080"
	@echo "✅ Swagger UI at http://localhost:8080/swagger/index.html"

prod:
	@echo "🚀 Starting production environment..."
	@docker-compose $(COMPOSE_FILE) up -d --build
	@echo "✅ API started at http://localhost:8080"

stop:
	@echo "🔥 Stopping all services..."
	@docker-compose $(COMPOSE_FILE) down

logs:
	@docker-compose $(COMPOSE_FILE) logs -f api

clean:
	@echo "🧹 Stopping and cleaning this project's containers and volumes..."
	@docker-compose $(COMPOSE_FILE) down -v

# ====================================================================================
# DATABASE & MIGRATION COMMANDS
# ====================================================================================

db-create:
	@echo "Creating database '$(DB_NAME)' (if it doesn't exist)..."
	-@docker-compose $(COMPOSE_FILE) exec -T postgres createdb --username=$(DB_USER) --owner=$(DB_USER) $(DB_NAME) 2>/dev/null || true
	@echo "Database ready."

migrate-create:
	@echo "Creating migration file: $(name)"
	@docker-compose $(COMPOSE_FILE) run --rm api /bin/sh -c "migrate create -ext sql -dir migrations -seq $(name)"

migrate-up:
	@echo "Applying database migrations..."
	@docker-compose $(COMPOSE_FILE) run --rm api /bin/sh -c "migrate -database $(DB_URL) -path migrations up"
	@echo "Migrations applied successfully."

migrate-down:
	@echo "Reverting last database migration..."
	@docker-compose $(COMPOSE_FILE) run --rm api /bin/sh -c "migrate -database $(DB_URL) -path migrations down 1"
	@echo "Last migration reverted."

# ====================================================================================
# BUILD COMMANDS
# ====================================================================================

build:
	@echo "📦 Building Go binary for production..."
	@GOOS=linux GOARCH=amd64 go build -o bin/nexspaces-api ./cmd/server/main.go
	@echo "✅ Built at ./bin/nexspaces-api"