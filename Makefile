BINARY_NAME=brute-force-service

CMD_PATH=cmd/main.go

BUILD_FLAGS=

# Сборка
docker-build:
	@echo "Building Docker image if necessary..."
	@if [ ! `docker images -q brute-force-service` ]; then \
		docker build -t brute-force-service .; \
	else \
		echo "Docker image already exists. Skipping build."; \
	fi
	@echo "Docker build complete."

# Остановка контейнеров
docker-stop:
	@echo "Stopping containers..."
	-docker stop redis-container || true
	-docker stop brute-force-service || true

# Удаление контейнеров
docker-remove:
	@echo "Removing containers..."
	-docker rm redis-container || true
	-docker rm brute-force-service || true

# Удаление контейнеров и образов
docker-remove-images: docker-remove
	@echo "Removing Docker images..."
	@if docker images | grep -q brute-force-service; then docker rmi -f brute-force-service; else echo "No brute-force-service image found"; fi
	@if docker images | grep -q redis; then docker rmi -f redis; else echo "No redis image found"; fi

# Запуск контейнеров в фоновом режиме
docker-run: docker-build
	@echo "Starting Redis..."
	@if [ `docker ps -a -q -f name=redis-container` ]; then \
		echo "Redis container already exists. Starting..."; \
		docker start redis-container; \
	else \
		docker run -d --name redis-container -p 6379:6379 redis; \
	fi
	@echo "Starting brute-force-service..."
	@if [ `docker ps -a -q -f name=brute-force-service` ]; then \
		echo "Brute-force-service container already exists. Starting..."; \
		docker start brute-force-service; \
	else \
		docker run -d --name brute-force-service --link redis-container:redis -e REDIS_HOST=redis -e REDIS_PORT=6379 -p 50051:50051 brute-force-service; \
	fi

# Остановка и удаление всех контейнеров
docker-clean: docker-stop docker-remove

# Юнит-тесты
test:
	@echo "Running unit tests..."
	@go test -race -count=1 -v ./internal/grpc/


# Интеграционне тесты (нужее поднятый докер)
integration-test: docker-run
	@echo "Running integration tests..."
	@go test -tags=integration -v ./integration


# Полное тестирование
full-test: test integration-test
	@echo "All tests passed!"

# Запуска линтинеров
lint:
	@echo "Running linter..."
	@golangci-lint run
	@echo "Linting complete!"

# Очистка сборки
clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@echo "Clean complete!"
