.PHONY: test all build clean

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
	rm -f server coverage.out coverage.html

# Tidy go.mod
tidy:
	go mod tidy

# Run all tests and build
all: test build
