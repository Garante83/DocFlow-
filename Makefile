.PHONY: docker-build docker-run docker-down docker-clean docker-test release release-linux release-clean

# Release build (single binary: frontend embedded + backend)
release:
	$(MAKE) -C backend release
	@echo "Release binary: backend/docflow"

# Cross-compile release binary for Linux (amd64)
release-linux:
	$(MAKE) -C backend frontend-build
	cd backend && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o docflow-linux-amd64 ./cmd/server
	@echo "Release binary: backend/docflow-linux-amd64"

# Remove release binaries
release-clean:
	rm -f backend/docflow backend/docflow-linux-amd64

# Docker targets (run from project root)

# Build Docker image
docker-build:
	docker build -t dokumentenscanner:latest .
	@echo "Docker image built: dokumentenscanner:latest"

# Build and run with docker-compose
docker-run:
	docker-compose up -d --build
	@echo "Dokumentenscanner running on http://localhost:8082"

# Stop containers
docker-down:
	docker-compose down
	@echo "Containers stopped"

# Stop containers and remove volumes
docker-clean:
	docker-compose down -v
	@echo "Containers and volumes removed"

# Run tests in container
docker-test:
	docker-compose run --rm dokumentenscanner go test ./backend/... -v
	@echo "Tests completed"

# Helper targets

# Show container logs
docker-logs:
	docker-compose logs -f dokumentenscanner

# Rebuild and restart
docker-rebuild:
	docker-compose up -d --build --force-recreate
	@echo "Dokumentenscanner rebuilt and restarted"
