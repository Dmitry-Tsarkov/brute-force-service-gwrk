SERVICE_NAME=brute-force-service

run:
	@echo "Starting containers with Docker Compose..."
	docker-compose up -d

down:
	@echo "Stopping containers..."
	docker-compose down

build:
	@echo "Building Docker images..."
	docker-compose build

test:
	@echo "Running unit tests..."
	go test -race -count=1 -v ./internal/grpc/

integration-test:
	@echo "Running integration tests..."
	go test -tags=integration -v ./integration

full-test: test integration-test
	@echo "All tests passed!"

lint:
	@echo "Running linter..."
	golangci-lint run
	@echo "Linting complete!"

clean:
	@echo "Cleaning up..."
	docker-compose down --volumes
	@echo "Clean complete!"

clean-images:
	@echo "Removing Docker images for ${SERVICE_NAME} and redis..."
	@if docker images | grep -q ${SERVICE_NAME}; then \
		docker rmi -f ${SERVICE_NAME}; \
	else \
		echo "No ${SERVICE_NAME} image found"; \
	fi
	@if docker images | grep -q redis; then \
		docker rmi -f redis; \
	else \
		echo "No redis image found"; \
	fi
	@echo "Images removed."
