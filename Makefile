.PHONY: help install dev build up down logs clean sqlc migrate test

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install dependencies
	@echo "Installing API dependencies..."
	cd api && go mod download
	@echo "Installing Web dependencies..."
	cd web && npm install
	@echo "Done!"

dev: ## Start development environment
	docker-compose up db -d
	@echo "Waiting for database to be ready..."
	sleep 5
	@echo "Development environment ready!"
	@echo "Run 'cd api && go run cmd/api/main.go' to start the API"
	@echo "Run 'cd web && npm run dev' to start the web app"

build: ## Build all Docker images
	docker-compose build

up: ## Start all services
	docker-compose up -d

down: ## Stop all services
	docker-compose down

logs: ## Show logs from all services
	docker-compose logs -f

clean: ## Clean up Docker volumes and build artifacts
	docker-compose down -v
	rm -rf api/internal/db/*
	rm -rf web/dist
	rm -rf web/node_modules

clean-coverage: ## Clean up test coverage files
	rm -f api/coverage*.out api/coverage.html api/benchmark-results.txt
	rm -rf web/coverage

sqlc: ## Generate sqlc code
	@echo "Generating sqlc code..."
	sqlc generate
	@echo "Done!"

migrate: ## Run database migrations
	@echo "Running migrations..."
	docker-compose exec db psql -U postgres -d loyalty -f /docker-entrypoint-initdb.d/001_initial_schema.sql
	@echo "Done!"

test: test-all ## Run all tests

test-unit: ## Run unit tests
	@echo "Running unit tests..."
	cd api && DATABASE_URL=postgres://postgres:postgres@localhost:5432/loyalty_test?sslmode=disable go test ./internal/... -v -coverprofile=coverage-unit.out

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	cd api && DATABASE_URL=postgres://postgres:postgres@localhost:5432/loyalty_test?sslmode=disable go test ./tests/integration/... -v -coverprofile=coverage-integration.out

test-api: ## Run API tests
	@echo "Running API tests..."
	cd api && DATABASE_URL=postgres://postgres:postgres@localhost:5432/loyalty_test?sslmode=disable go test ./tests/api/... -v -coverprofile=coverage-api.out

test-performance: ## Run performance tests
	@echo "Running performance tests..."
	cd api && DATABASE_URL=postgres://postgres:postgres@localhost:5432/loyalty_test?sslmode=disable go test ./tests/performance/... -bench=. -benchmem -benchtime=5s

test-backend: test-unit test-integration test-api ## Run all backend tests

test-frontend: ## Run frontend tests
	@echo "Running frontend tests..."
	cd web && npm test

test-all: test-backend ## Run all tests (backend and frontend)

coverage: ## Generate coverage report
	@echo "Generating coverage report..."
	cd api && go install github.com/wadey/gocovmerge@latest || true
	cd api && gocovmerge coverage-*.out > coverage.out 2>/dev/null || true
	cd api && go tool cover -func=coverage.out
	cd api && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: api/coverage.html"

coverage-check: coverage ## Check coverage meets 80% threshold
	@echo "Checking coverage threshold..."
	@COVERAGE=$$(cd api && go tool cover -func=coverage.out | grep total | awk '{print substr($$3, 1, length($$3)-1)}'); \
	echo "Total coverage: $${COVERAGE}%"; \
	if [ $$(echo "$${COVERAGE} < 80" | bc -l) -eq 1 ]; then \
		echo "Coverage $${COVERAGE}% is below 80% threshold"; \
		exit 1; \
	fi; \
	echo "Coverage $${COVERAGE}% meets the 80% threshold"

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	cd api && DATABASE_URL=postgres://postgres:postgres@localhost:5432/loyalty_test?sslmode=disable go test ./internal/rules/... -bench=. -benchmem -benchtime=10s | tee benchmark-results.txt

reset-db: ## Reset database (WARNING: This will delete all data)
	docker-compose down -v
	docker-compose up db -d
	@echo "Waiting for database to be ready..."
	sleep 5
	@echo "Database reset complete!"

api-shell: ## Open a shell in the API container
	docker-compose exec api sh

db-shell: ## Open a PostgreSQL shell
	docker-compose exec db psql -U postgres -d loyalty

status: ## Show status of all services
	docker-compose ps
