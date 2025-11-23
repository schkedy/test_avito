.PHONY: help build run test generate sqlc migrate lint docker-up docker-down clean

# Default target
help:
	@echo "Available targets:"
	@echo "  build                  - Build the application"
	@echo "  run                    - Run the application locally"
	@echo "  test                   - Run tests"
	@echo "  test-e2e               - Run E2E tests (requires running service)"
	@echo "  test-integration       - Run integration tests only"
	@echo "  generate               - Generate code from OpenAPI spec and sqlc"
	@echo "  sqlc                   - Generate database code from SQL queries"
	@echo "  migrate                - Run database migrations (local DB)"
	@echo "  migrate-docker         - Run database migrations (Docker DB)"
	@echo "  lint                   - Run linter"
	@echo "  docker-up              - Start services with docker-compose"
	@echo "  docker-build-no-cache  - Build Docker image without cache"
	@echo "  docker-up-clean        - Start services with clean build (no cache)"
	@echo "  docker-down            - Stop services with docker-compose"
	@echo "  clean                  - Clean build artifacts"

# Build the application
build:
	@echo "Building application..."
	go build -o bin/server ./cmd/server

# Run the application locally
run:
	@echo "Running application..."
	go run ./cmd/server/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run E2E tests (requires running service at http://localhost:8080)
test-e2e:
	@echo "Running E2E tests..."
	@echo "Note: Service must be running at http://localhost:8080"
	@echo "Run 'make docker-up' first if not running"
	go test -v ./tests/e2e/...

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	go test -v ./tests/integration/...

# Generate database code from SQL queries
sqlc:
	@echo "Generating database code with sqlc..."
	sqlc generate

# Generate code from OpenAPI spec and sqlc
generate: sqlc
	@echo "Generating code from OpenAPI..."
	oapi-codegen --config oapi-codegen.yaml openapi/openapi.yml

# Run database migrations for local DB (port 5434)
migrate:
	@echo "Running migrations on localhost:5434..."
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate \
		-path ./migrations \
		-database "postgres://postgres:postgres@localhost:5434/pr_reviewer?sslmode=disable" \
		up

# Run database migrations for Docker DB (port 5432)
migrate-docker:
	@echo "Running migrations on localhost:5432..."
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate \
		-path ./migrations \
		-database "postgres://postgres:postgres@localhost:5432/pr_reviewer?sslmode=disable" \
		up

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Start services with docker-compose
docker-up:
	@echo "Starting services..."
	docker-compose up --build -d
	@echo "Services started. Application available at http://localhost:8080"

# Build Docker image without cache
docker-build-no-cache:
	@echo "Building Docker image without cache..."
	docker-compose build --no-cache
	@echo "Docker image built successfully"

# Start services with clean build (no cache)
docker-up-clean:
	@echo "Starting services with clean build..."
	docker-compose build --no-cache
	docker-compose up -d
	@echo "Services started. Application available at http://localhost:8080"

# Stop services with docker-compose
docker-down:
	@echo "Stopping services..."
	docker-compose down -v

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

# Run go mod tidy
tidy:
	@echo "Tidying go.mod..."
	go mod tidy
