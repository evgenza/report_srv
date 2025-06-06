# Report Service

Современный микросервис для генерации и управления отчетами, написанный на Go с использованием чистой архитектуры.

## 🚀 Особенности

- **Современная архитектура**: Clean Architecture с dependency injection (uber/fx)
- **HTTP API**: Высокопроизводительный REST API на Echo framework
- **База данных**: PostgreSQL с GORM ORM и автомиграциями
- **Хранилище файлов**: Поддержка S3-совместимых хранилищ и локального файловой системы
- **Асинхронная генерация**: Фоновая генерация отчетов в Excel формате
- **Структурированное логирование**: logrus с JSON и текстовым форматами
- **Graceful shutdown**: Корректное завершение работы сервиса
- **Health checks**: Мониторинг состояния сервиса
- **Docker**: Готовые образы для контейнеризации
- **Конфигурация**: Гибкая настройка через файлы и переменные окружения

## 📋 Требования

- Go 1.23+
- PostgreSQL 12+
- S3-совместимое хранилище (AWS S3, MinIO, LocalStack)
- Docker & Docker Compose (для разработки)

## 🛠 Установка и запуск

### Быстрый старт с Docker Compose

```bash
# Клонируем репозиторий
git clone <repository-url>
cd report_srv

# Запускаем все сервисы
docker-compose up -d

# Проверяем состояние
curl http://localhost:8080/health
```

### Ручная установка

1. **Установка зависимостей:**
```bash
go mod download
```

2. **Настройка конфигурации:**
```bash
cp config.yaml.example config.yaml
# Редактируем config.yaml под ваши нужды
```

3. **Запуск базы данных:**
```bash
# PostgreSQL
docker run -d --name postgres \
  -e POSTGRES_DB=report_db \
  -e POSTGRES_USER=report_user \
  -e POSTGRES_PASSWORD=report_password \
  -p 5432:5432 postgres:15-alpine

# LocalStack для S3
docker run -d --name localstack \
  -e SERVICES=s3 \
  -p 4566:4566 localstack/localstack
```

4. **Запуск приложения:**
```bash
go run cmd/server/main.go
```

## ⚙️ Конфигурация

Сервис поддерживает конфигурацию через файл `config.yaml` и переменные окружения:

```yaml
server:
  address: ":8080"
  debug: false

database:
  driver: postgres
  dsn: postgres://user:pass@localhost:5432/reports?sslmode=disable

storage:
  type: s3  # или "local"
  s3:
    region: us-east-1
    bucket: report-srv-bucket
    endpoint: http://localhost:4566  # для LocalStack
    access_key: test
    secret_key: test

logging:
  level: info
  format: json
```

### Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `APP_SERVER_ADDRESS` | Адрес HTTP сервера | `:8080` |
| `APP_SERVER_DEBUG` | Режим отладки | `false` |
| `APP_DATABASE_DRIVER` | Драйвер БД (postgres/sqlite) | `postgres` |
| `APP_DATABASE_DSN` | Строка подключения к БД | - |
| `APP_STORAGE_TYPE` | Тип хранилища (s3/local) | `local` |
| `APP_STORAGE_S3_*` | Настройки S3 | - |
| `APP_LOGGING_LEVEL` | Уровень логирования | `info` |
| `APP_LOGGING_FORMAT` | Формат логов (json/text) | `text` |

## 📚 API Документация

### Endpoints

#### Health Check
```bash
GET /health
```
Возвращает состояние сервиса.

#### Reports

**Создание отчета:**
```bash
POST /api/v1/reports
Content-Type: application/json

{
  "title": "Отчет по продажам",
  "description": "Месячный отчет по продажам",
  "parameters": {
    "period": "2024-01",
    "department": "sales"
  },
  "created_by": "john.doe"
}
```

**Получение списка отчетов:**
```bash
GET /api/v1/reports
```

**Получение отчета по ID:**
```bash
GET /api/v1/reports/{id}
```

**Удаление отчета:**
```bash
DELETE /api/v1/reports/{id}
```

**Скачивание отчета:**
```bash
GET /api/v1/reports/{id}/download
```

### Примеры запросов

```bash
# Создание отчета
curl -X POST http://localhost:8080/api/v1/reports \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Report",
    "description": "Test description",
    "created_by": "test_user"
  }'

# Получение списка отчетов
curl http://localhost:8080/api/v1/reports

# Проверка здоровья
curl http://localhost:8080/health
```

## 🏗 Архитектура

Проект использует Clean Architecture принципы:

```
cmd/
├── server/           # Точка входа приложения
internal/
├── config/          # Конфигурация
├── models/          # Модели данных
├── database/        # Слой базы данных
├── storage/         # Слой хранилища файлов
├── service/         # Бизнес-логика
└── server/          # HTTP сервер
```

### Основные компоненты

- **Config**: Управление конфигурацией с помощью viper
- **Database**: GORM ORM с автомиграциями
- **Storage**: Абстракция над файловыми хранилищами (S3/Local)
- **Service**: Бизнес-логика генерации отчетов
- **Server**: HTTP API с middleware и роутингом
- **DI Container**: Dependency injection с uber/fx

## 🧪 Тестирование

```bash
# Запуск всех тестов
go test ./...

# Тесты с покрытием
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Линтер
golangci-lint run
```

## 📦 Деплой

### Docker

```bash
# Сборка образа
docker build -t report-service .

# Запуск контейнера
docker run -d --name report-service \
  -p 8080:8080 \
  -e APP_DATABASE_DSN="postgres://..." \
  report-service
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: report-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: report-service
  template:
    metadata:
      labels:
        app: report-service
    spec:
      containers:
      - name: report-service
        image: report-service:latest
        ports:
        - containerPort: 8080
        env:
        - name: APP_DATABASE_DSN
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: dsn
```

## 🔧 Разработка

### Настройка среды разработки

1. **Установка инструментов:**
```bash
# Линтер
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Инструменты для работы с БД
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

2. **Запуск в режиме разработки:**
```bash
# Запуск зависимостей
docker-compose up -d postgres localstack

# Запуск приложения
APP_SERVER_DEBUG=true go run cmd/server/main.go
```

### Добавление новых функций

1. Обновите модели в `internal/models/`
2. Добавьте бизнес-логику в `internal/service/`
3. Обновите HTTP handlers в `internal/server/`
4. Добавьте тесты
5. Обновите документацию

## 🤝 Участие в разработке

1. Fork репозитория
2. Создайте feature branch
3. Внесите изменения с тестами
4. Убедитесь что линтер проходит
5. Создайте Pull Request

## 📄 Лицензия

MIT License - см. файл LICENSE для деталей.
