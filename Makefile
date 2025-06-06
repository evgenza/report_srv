# Report Service Makefile

.PHONY: help build run test clean docker docker-up docker-down lint fmt vet mod-tidy

# Переменные
BINARY_NAME=report-service
MAIN_PATH=./cmd/server
BUILD_DIR=./build

# По умолчанию показываем help
help: ## Показать это сообщение
	@echo "Report Service - Makefile команды:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Сборка
build: ## Собрать приложение
	@echo "Сборка $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

build-linux: ## Собрать для Linux
	@echo "Сборка $(BINARY_NAME) для Linux..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux $(MAIN_PATH)

build-windows: ## Собрать для Windows
	@echo "Сборка $(BINARY_NAME) для Windows..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME).exe $(MAIN_PATH)

# Запуск
run: ## Запустить приложение в режиме разработки
	@echo "Запуск $(BINARY_NAME) в режиме разработки..."
	@APP_SERVER_DEBUG=true APP_LOGGING_LEVEL=debug go run $(MAIN_PATH)

run-prod: build ## Запустить собранное приложение
	@echo "Запуск $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

# Тестирование
test: ## Запустить тесты
	@echo "Запуск тестов..."
	@go test -v ./...

test-race: ## Запустить тесты с проверкой гонок
	@echo "Запуск тестов с race detector..."
	@go test -race -v ./...

test-coverage: ## Запустить тесты с покрытием
	@echo "Запуск тестов с покрытием..."
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Отчет о покрытии сохранен в coverage.html"

benchmark: ## Запустить бенчмарки
	@echo "Запуск бенчмарков..."
	@go test -bench=. -benchmem ./...

# Качество кода
lint: ## Запустить линтер
	@echo "Запуск линтера..."
	@golangci-lint run

fmt: ## Форматировать код
	@echo "Форматирование кода..."
	@go fmt ./...

vet: ## Запустить go vet
	@echo "Запуск go vet..."
	@go vet ./...

# Зависимости
mod-tidy: ## Обновить зависимости
	@echo "Обновление зависимостей..."
	@go mod tidy

mod-download: ## Скачать зависимости
	@echo "Скачивание зависимостей..."
	@go mod download

# Docker
docker: ## Собрать Docker образ
	@echo "Сборка Docker образа..."
	@docker build -t report-service:latest .

docker-run: ## Запустить в Docker контейнере
	@echo "Запуск в Docker..."
	@docker run --rm -p 8080:8080 --name report-service report-service:latest

docker-up: ## Запустить все сервисы с Docker Compose
	@echo "Запуск всех сервисов..."
	@docker-compose up -d

docker-down: ## Остановить все сервисы
	@echo "Остановка всех сервисов..."
	@docker-compose down

docker-logs: ## Посмотреть логи приложения
	@docker-compose logs -f app

docker-ps: ## Показать статус контейнеров
	@docker-compose ps

# Разработка
dev-setup: ## Настроить среду разработки
	@echo "Настройка среды разработки..."
	@go mod download
	@docker-compose up -d postgres localstack redis
	@echo "Среда разработки готова!"

dev-reset: ## Сбросить данные разработки
	@echo "Сброс данных разработки..."
	@docker-compose down -v
	@docker-compose up -d postgres localstack redis

# Очистка
clean: ## Очистить собранные файлы
	@echo "Очистка..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -f $(BINARY_NAME) $(BINARY_NAME).exe

clean-docker: ## Очистить Docker ресурсы
	@echo "Очистка Docker..."
	@docker-compose down -v --remove-orphans
	@docker system prune -f

# База данных
db-reset: ## Пересоздать базу данных
	@echo "Пересоздание базы данных..."
	@docker-compose down postgres
	@docker volume rm report_srv_postgres_data || true
	@docker-compose up -d postgres

# Проверки
check: lint vet test ## Выполнить все проверки

# Установка инструментов разработки
install-tools: ## Установить инструменты разработки
	@echo "Установка инструментов разработки..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Инструменты установлены!"

# Информация о проекте
info: ## Показать информацию о проекте
	@echo "=== Report Service ==="
	@echo "Go version: $(shell go version)"
	@echo "Module: $(shell go list -m)"
	@echo "Build dir: $(BUILD_DIR)"
	@echo "Binary: $(BINARY_NAME)" 