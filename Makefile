# NexSpaces API

.PHONY: help dev prod stop logs clean

help:
	@echo "🚀 NexSpaces API Commands"
	@echo ""
	@echo "  dev     - Start development (with hot reload)"
	@echo "  prod    - Start production"
	@echo "  stop    - Stop services"
	@echo "  logs    - Show logs"
	@echo "  clean   - Clean containers"
	@echo "  health  - Check health"

dev:
	@echo "🔧 Starting development..."
	@cd infra/docker && docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d --build
	@echo "✅ Started at http://localhost:8080"

prod:
	@echo "🚀 Starting production..."
	@cd infra/docker && docker-compose up -d --build
	@echo "✅ Started at http://localhost:8080"

stop:
	@cd infra/docker && docker-compose -f docker-compose.yml -f docker-compose.dev.yml down

logs:
	@cd infra/docker && docker-compose -f docker-compose.yml -f docker-compose.dev.yml logs -f

clean:
	@cd infra/docker && docker-compose -f docker-compose.yml -f docker-compose.dev.yml down -v
	@docker system prune -f

health:
	@curl -s http://localhost:8080/health | jq . || echo "API not responding"