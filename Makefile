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

sqlc: ## Generate sqlc code
	@echo "Generating sqlc code..."
	sqlc generate
	@echo "Done!"

migrate: ## Run database migrations
	@echo "Running migrations..."
	docker-compose exec db psql -U postgres -d loyalty -f /docker-entrypoint-initdb.d/001_initial_schema.sql
	@echo "Done!"

test: ## Run tests
	@echo "Running API tests..."
	cd api && go test ./...
	@echo "Running Web tests..."
	cd web && npm test
	@echo "Done!"

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
