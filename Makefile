.PHONY: help dev build test lint fmt clean docker-up docker-down migrate-up migrate-down

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

dev: ## Run development server
	go run cmd/server/main.go

build: ## Build production binary
	CGO_ENABLED=0 go build -o bin/nexspaces-api cmd/server/main.go

test: ## Run all tests
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests with coverage report
	go tool cover -html=coverage.out

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	gofmt -s -w .
	go mod tidy

clean: ## Clean build artifacts
	rm -rf bin/ coverage.out

# Docker commands
docker-build: ## Build Docker image
	docker build -f docker/Dockerfile -t nexspaces-api:latest .

docker-up: ## Start docker-compose stack
	docker-compose up -d

docker-down: ## Stop docker-compose stack
	docker-compose down

docker-logs: ## View API logs
	docker-compose logs -f api

# Database migrations
migrate-up: ## Run all migrations
	@echo "Running migrations..."
	@for file in migrations/*.up.sql; do \
		echo "Applying $$file..."; \
		PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -U $(DB_USER) -d $(DB_NAME) -f $$file; \
	done

migrate-down: ## Rollback last migration
	@echo "Rolling back migrations..."
	@for file in migrations/*.down.sql; do \
		echo "Rolling back $$file..."; \
		PGPASSWORD=$(DB_PASSWORD) psql -h $(DB_HOST) -U $(DB_USER) -d $(DB_NAME) -f $$file; \
	done

# Database helpers
db-create: ## Create database
	createdb -h $(DB_HOST) -U $(DB_USER) $(DB_NAME)

db-drop: ## Drop database
	dropdb -h $(DB_HOST) -U $(DB_USER) $(DB_NAME)

db-reset: db-drop db-create migrate-up ## Reset database
