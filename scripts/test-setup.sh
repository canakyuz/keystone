#!/bin/bash

# Keystone API Test Environment Setup Script
# This script sets up the complete test environment

set -e

echo "🚀 Keystone API - Test Environment Setup"
echo "=========================================="

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Check if Docker is running
echo ""
echo "📦 Step 1: Checking Docker..."
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}✗ Docker is not running. Please start Docker first.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Docker is running${NC}"

# Step 2: Stop existing containers
echo ""
echo "🛑 Step 2: Stopping existing containers..."
docker-compose down -v || true
echo -e "${GREEN}✓ Containers stopped${NC}"

# Step 3: Start PostgreSQL and Redis
echo ""
echo "🐘 Step 3: Starting PostgreSQL and Redis..."
docker-compose up -d postgres redis
echo "Waiting for database to be ready..."
sleep 5

# Wait for postgres to be healthy
echo "Checking PostgreSQL health..."
until docker exec keystone-postgres pg_isready -U postgres > /dev/null 2>&1; do
    echo -n "."
    sleep 1
done
echo ""
echo -e "${GREEN}✓ PostgreSQL is ready${NC}"

# Step 4: Run migrations
echo ""
echo "🔄 Step 4: Running database migrations..."
if command -v migrate &> /dev/null; then
    migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/keystone_dev?sslmode=disable" up
    echo -e "${GREEN}✓ Migrations completed${NC}"
else
    echo -e "${YELLOW}⚠ migrate tool not found. Installing...${NC}"
    echo "Run: brew install golang-migrate (macOS) or check https://github.com/golang-migrate/migrate"
    echo ""
    echo "Alternative: Use docker to run migrations:"
    echo 'docker run -v $(pwd)/migrations:/migrations --network host migrate/migrate -path=/migrations/ -database "postgresql://postgres:postgres@localhost:5432/keystone_dev?sslmode=disable" up'
    exit 1
fi

# Step 5: Seed test data
echo ""
echo "🌱 Step 5: Seeding test data..."
docker exec -i keystone-postgres psql -U postgres -d keystone_dev < scripts/seed/01_test_data.sql
echo -e "${GREEN}✓ Test data seeded${NC}"

# Step 6: Build Go application
echo ""
echo "🔨 Step 6: Building Go application..."
go build -o bin/server cmd/server/main.go
echo -e "${GREEN}✓ Application built${NC}"

# Step 7: Show test credentials
echo ""
echo "=========================================="
echo -e "${GREEN}✓ Test environment is ready!${NC}"
echo "=========================================="
echo ""
echo "📝 Test Credentials:"
echo "   Database: postgresql://postgres:postgres@localhost:5432/keystone_dev"
echo "   Redis: redis://localhost:6379/0"
echo ""
echo "   Tenant: canakyuz.co"
echo "   Admin User: admin@canakyuz.co / Admin123!"
echo "   Regular User: user@canakyuz.co / User123!"
echo ""
echo "🚀 Next Steps:"
echo "   1. Start API: go run cmd/server/main.go"
echo "   2. API will be available at: http://localhost:8080"
echo "   3. Health check: curl http://localhost:8080/health"
echo "   4. Import Postman collection from: docs/postman/"
echo ""
echo "📚 API Documentation:"
echo "   Swagger UI: http://localhost:8080/swagger (coming soon)"
echo "   Postman Collection: docs/postman/Keystone-API.postman_collection.json"
echo ""
