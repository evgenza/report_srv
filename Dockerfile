# этап сборки
FROM golang:1.20-alpine AS build
WORKDIR /src
COPY go.mod .
RUN go mod download
COPY . .
RUN go build -o report ./cmd/report_srv

# этап запуска
FROM alpine
WORKDIR /app
COPY --from=build /src/report .
CMD ["./report"]
