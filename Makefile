.PHONY: help build run test clean migrate-up migrate-down docker-up docker-down

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building application..."
	@go build -o bin/api cmd/api/main.go cmd/api/helpers.go

run: ## Run the application
	@echo "Running application..."
	@go run cmd/api/main.go cmd/api/helpers.go

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

deps: ## Install dependencies
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

migrate-up: ## Run database migrations up
	@echo "Running migrations..."
	@psql -U $(DB_USER) -d $(DB_NAME) -f migrations/001_init_schema.up.sql

migrate-down: ## Run database migrations down
	@echo "Rolling back migrations..."
	@psql -U $(DB_USER) -d $(DB_NAME) -f migrations/001_init_schema.down.sql

docker-up: ## Start Docker containers
	@echo "Starting Docker containers..."
	@docker-compose up -d

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	@docker-compose down

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

.DEFAULT_GOAL := help
