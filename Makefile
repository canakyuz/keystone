.DEFAULT_GOAL := help
.SILENT:
.PHONY: help dev build run up down restart rebuild logs ps shell db-shell \
        migrate-up migrate-down migrate-status migrate-bootstrap test fmt lint vet check clean-all

# ==============================================================================
# Yardım
# ==============================================================================
help: ## Komut listesini göster
	@awk 'BEGIN {FS=":.*##"; print "\n\033[36mAvailable Commands:\033[0m"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-18s\033[0m%s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ==============================================================================
# Geliştirme
# ==============================================================================
dev: ## Geliştirme sunucusunu çalıştır
	go run cmd/server/main.go

build: ## Binary oluştur
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/nexspaces-api cmd/server/main.go

run: build ## Binary’i çalıştır
	./bin/nexspaces-api

# ==============================================================================
# Docker
# ==============================================================================
up: ## Servisleri başlat
	docker compose up -d
	@echo "\033[32m✓ Services started\033[0m"
	@echo "→ API: http://localhost:8080"
	@echo "→ Swagger: http://localhost:8080/docs"

down: ## Servisleri durdur
	docker compose down

restart: ## Servisleri yeniden başlat
	docker compose down
	@echo ”\033[32m✓ Temizlendi”
	docker compose up -d

rebuild: ## API’yi yeniden build edip başlat
	docker compose up -d --build api

logs: ## API loglarını izle
	docker compose logs -f api

ps: ## Container durumlarını göster
	docker compose ps

shell: ## API container’ına shell
	docker compose exec api sh

db-shell: ## PostgreSQL shell
	docker compose exec -T postgres psql -U postgres -d nexspaces_dev

# ==============================================================================
# Migration
# ==============================================================================
migrate-up: ## Migrationları çalıştır (*.up.sql)
	@echo "\033[32m✓ Başlatıldı\033[0m"
	@scripts/run_migrations.sh

migrate-down: ## Migrationları geri al (*.down.sql)
	@for f in $(shell ls -r migrations/*.down.sql 2>/dev/null); do \
		echo "→ $$f" && cat $$f | docker compose exec -T postgres psql -U postgres -d nexspaces_dev; \
	done
	@echo "\033[33m✓ Rollback complete\033[0m"

migrate-status: ## Migration dosyalarını listele
	@echo "Up migrations:";   ls -1 migrations/*.up.sql 2>/dev/null || echo "  None"
	@echo ""; echo "Down migrations:"; ls -1 migrations/*.down.sql 2>/dev/null || echo "  None"

# ==============================================================================
# Test ve Kalite
# ==============================================================================
test: ## Testleri çalıştır
	go test -v -race ./...

fmt: ## Kod formatla
	gofmt -s -w .
	go mod tidy

lint: ## Linter çalıştır
	golangci-lint run --timeout 5m

vet: ## go vet
	go vet ./...

check: fmt vet lint test ## Tüm kontrolleri çalıştır

# ==============================================================================
# Temizlik
# ==============================================================================
clean-all: ## Her şeyi temizle
	rm -rf bin/ coverage.out coverage.html
migrate-bootstrap: ## Mevcut veritabanini schema_migrations ile esitle
	@echo "\033[33m⚠  Bootstrap modunda schema_migrations dolduruluyor\033[0m"
	@BOOTSTRAP_MIGRATIONS=1 scripts/run_migrations.sh
	@echo "\033[32m✓ Bootstrap tamamlandi\033[0m"
