.PHONY: docker-build docker-run docker-down docker-clean docker-test

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
