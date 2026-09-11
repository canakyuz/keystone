APP_ENV ?= development

.DEFAULT_GOAL := help
.SILENT:
.PHONY: help dev build run up down restart rebuild logs ps shell db-shell \
        migrate-up migrate-down migrate-status migrate-bootstrap test fmt lint vet check clean-all console

# ==============================================================================
# Help
# ==============================================================================
help: ## Show the list of commands
	@awk 'BEGIN {FS=":.*##"; print "\n\033[36mAvailable Commands:\033[0m"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-18s\033[0m%s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ==============================================================================
# Development
# ==============================================================================
dev: ## Run the development server
	go run cmd/server/main.go

build: ## Build both binaries
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/keystone cmd/server/main.go
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/keystone-worker cmd/worker/main.go

worker: ## Run the provisioning worker
	go run cmd/worker/main.go

console: ## Run the web console against a running API
	cd console && bun install && bun run dev

run: build ## Run the binary
	./bin/keystone

# ==============================================================================
# Docker
# ==============================================================================
up: ## Start the services
	docker compose up -d
	@echo "\033[32m✓ Services started\033[0m"
	@echo "→ API: http://localhost:8080"
	@echo "→ Swagger: http://localhost:8080/docs"

down: ## Stop the services
	docker compose down

restart: ## Restart the services
	docker compose down
	@echo ”\033[32m✓ Temizlendi”
	docker compose up -d

rebuild: ## Rebuild the API and restart it
	docker compose up -d --build api

logs: ## Follow the API logs
	docker compose logs -f api

ps: ## Show container status
	docker compose ps

shell: ## Open a shell in the API container
	docker compose exec api sh

db-shell: ## PostgreSQL shell
	docker compose exec -T postgres psql -U postgres -d keystone_dev

db-reset: ## Reset the development database and apply the migrations
	docker compose exec -T postgres psql -U postgres -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'keystone_dev' AND pid <> pg_backend_pid();"
	docker compose exec -T postgres psql -U postgres -d postgres -c "DROP DATABASE IF EXISTS keystone_dev;"
	docker compose exec -T postgres psql -U postgres -d postgres -c "CREATE DATABASE keystone_dev;"
	docker compose exec -T postgres psql -U postgres -d keystone_dev -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"
	docker compose exec -T postgres psql -U postgres -d keystone_dev -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"
	APP_ENV=$(APP_ENV) $(MAKE) migrate-up

seed-dev: ## Load the development tenant and user seeds
	@if [ ! -f scripts/seed/dev_seed.sql ]; then \
		echo "\033[31mSeed script not found: scripts/seed/dev_seed.sql\033[0m"; \
		exit 1; \
	fi
	docker compose exec -T postgres psql -U postgres -d keystone_dev < scripts/seed/dev_seed.sql

# ==============================================================================
# Migration
# ==============================================================================
migrate-up: ## Run the migrations (*.up.sql)
	@echo "\033[32mDone\033[0m"
	@APP_ENV=$(APP_ENV) scripts/run_migrations.sh

migrate-down: ## Roll the migrations back (*.down.sql)
	@for f in $(shell ls -r migrations/*.down.sql 2>/dev/null); do \
		echo "→ $$f" && cat $$f | docker compose exec -T postgres psql -U postgres -d keystone_dev; \
	done
	@echo "\033[33m✓ Rollback complete\033[0m"

migrate-status: ## List the migration files
	@echo "Up migrations:";   ls -1 migrations/*.up.sql 2>/dev/null || echo "  None"
	@echo ""; echo "Down migrations:"; ls -1 migrations/*.down.sql 2>/dev/null || echo "  None"

# ==============================================================================
# Test and quality
# ==============================================================================
proto: ## Regenerate the protobuf code
	@buf lint
	@buf generate
	@echo "\033[32mDone\033[0m"

proto-lint: ## Lint the protobuf definitions
	@buf lint

grpc-list: ## List the gRPC services on a running server
	@grpcurl -plaintext $${GRPC_ADDR:-127.0.0.1:9099} list

loadtest: ## Run the load profile against a freshly seeded instance
	@scripts/loadtest.sh

test: ## Run the tests
	go test -v -race ./...

fmt: ## Format the code
	gofmt -s -w .
	go mod tidy

lint: ## Run the linter
	golangci-lint run --timeout 5m

vet: ## go vet
	go vet ./...

check: fmt vet lint test ## Run every check

# ==============================================================================
# Cleanup
# ==============================================================================
clean-all: ## Clean everything
	rm -rf bin/ coverage.out coverage.html
migrate-bootstrap: ## Reconcile an existing database with schema_migrations
	@echo "\033[33mfilling schema_migrations in bootstrap mode\033[0m"
	@APP_ENV=$(APP_ENV) BOOTSTRAP_MIGRATIONS=1 scripts/run_migrations.sh
	@echo "\033[32m✓ Bootstrap tamamlandi\033[0m"
