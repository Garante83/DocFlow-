.PHONY: test all build clean config-dev config-prod build-dev build-prod

# Run all tests
test:
	go test ./... -v

# Run tests with coverage
coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html

# Build the server
build:
	go build -o server ./cmd/server

# Run the server
run:
	go run ./cmd/server

# Clean build artifacts
clean:
	rm -f server server-dev server-prod coverage.out coverage.html

# Tidy go.mod
tidy:
	go mod tidy

# Run all tests and build
all: test build

# Config targets
# Copy dev config to config.yaml
config-dev:
	cp config/config.dev.yaml config/config.yaml
	@echo "Switched to development config"

# Copy prod config to config.yaml
config-prod:
	cp config/config.prod.yaml config/config.yaml
	@echo "Switched to production config"

# Build with dev config
build-dev:
	cp config/config.dev.yaml config/config.yaml
	go build -o server-dev ./cmd/server
	@echo "Built with development config"

# Build with prod config
build-prod:
	cp config/config.prod.yaml config/config.yaml
	go build -o server-prod ./cmd/server
	@echo "Built with production config"
