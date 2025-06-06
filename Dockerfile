# Build stage
FROM golang:1.23-alpine AS builder

# Устанавливаем необходимые пакеты
RUN apk add --no-cache git ca-certificates tzdata

# Создаем пользователя для приложения
RUN adduser -D -g '' appuser

# Устанавливаем рабочую директорию
WORKDIR /build

# Копируем go mod файлы
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o app ./cmd/server

# Production stage
FROM alpine:latest

# Устанавливаем ca-certificates для HTTPS запросов
RUN apk --no-cache add ca-certificates tzdata

# Создаем необходимые директории
RUN mkdir -p /app/templates /app/logs

# Копируем пользователя из builder
COPY --from=builder /etc/passwd /etc/passwd

# Копируем собранное приложение
COPY --from=builder /build/app /app/

# Копируем конфигурацию
COPY config.yaml /app/

# Копируем шаблоны если есть
COPY templates/ /app/templates/

# Устанавливаем владельца файлов
RUN chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Устанавливаем рабочую директорию
WORKDIR /app

# Открываем порт
EXPOSE 8080

# Устанавливаем переменные окружения
ENV APP_SERVER_ADDRESS=:8080
ENV APP_SERVER_DEBUG=false
ENV APP_LOGGING_LEVEL=info
ENV APP_LOGGING_FORMAT=json

# Проверка здоровья
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Запускаем приложение
CMD ["./app"]
