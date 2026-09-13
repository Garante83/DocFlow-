.PHONY: test all build clean lint fmt config-dev config-prod build-dev build-prod tidy coverage run release frontend-build

# Run all tests
test:
	go test ./... -v

# Run tests with coverage
coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html

# Lint: vet + format check
lint:
	go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)
	@echo "Lint passed"

# Format code
fmt:
	gofmt -w .
	@echo "Code formatted"

# Build the server
build:
	go build -o server ./cmd/server

# Run the server
run:
	go run ./cmd/server

# Clean build artifacts
clean:
	rm -f server server-dev server-prod docflow coverage.out coverage.html

# Tidy go.mod
tidy:
	go mod tidy

# Lint + Test + Build
all: lint test build

# Config targets
config-dev:
	cp config/config.dev.yaml config/config.yaml
	@echo "Switched to development config"

config-prod:
	cp config/config.prod.yaml config/config.yaml
	@echo "Switched to production config"

build-dev:
	cp config/config.dev.yaml config/config.yaml
	go build -o server-dev ./cmd/server
	@echo "Built with development config"

build-prod:
	cp config/config.prod.yaml config/config.yaml
	go build -o server-prod ./cmd/server
	@echo "Built with production config"

# Build frontend and copy assets into the embed directory
frontend-build:
	cd ../frontend/dokumentenscanner && npm run build
	rm -rf cmd/server/assets
	cp ../frontend/dokumentenscanner/dist/index.html cmd/server/index.html
	cp ../frontend/dokumentenscanner/dist/favicon.* cmd/server/
	cp -r ../frontend/dokumentenscanner/dist/assets cmd/server/assets
	@echo "Frontend assets copied to cmd/server/"

# Single-binary release build (frontend + backend)
release: frontend-build
	go build -ldflags="-s -w" -o docflow ./cmd/server
	@echo "Release binary built: docflow"
