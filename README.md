# Report Service

A service for generating and managing reports using Go, Echo, PostgreSQL, and S3 storage.

## Features

- HTTP API using Echo framework
- PostgreSQL database with GORM ORM
- S3-compatible storage for report files
- Asynchronous report generation
- Debug mode for development
- Comprehensive test coverage

## Prerequisites

- Go 1.23 or later
- PostgreSQL 12 or later
- S3-compatible storage (e.g., AWS S3, MinIO, LocalStack)

## Configuration

The service is configured using a YAML file (`config.yaml`):

```yaml
server:
  address: ":8080"
  debug: true

database:
  driver: postgres
  dsn: postgres://user:pass@localhost:5432/dbname?sslmode=disable

storage:
  type: s3
  s3:
    region: us-east-1
    bucket: report-srv-bucket
    endpoint: http://localhost:4566  # LocalStack endpoint for local development
    access_key: test
    secret_key: test
  local:
    basepath: ./templates

logging:
  level: debug
  format: json
```

## API Endpoints

### Reports

- `POST /api/v1/reports` - Create a new report
- `GET /api/v1/reports` - List all reports
- `GET /api/v1/reports/:id` - Get a specific report
- `DELETE /api/v1/reports/:id` - Delete a report
- `GET /api/v1/reports/:id/download` - Download a report file

### Health Check

- `GET /health` - Check service health

## Development

### Local Setup

1. Start PostgreSQL:
```bash
docker-compose up -d postgres
```

2. Start LocalStack (for S3):
```bash
docker-compose up -d localstack
```

3. Run migrations:
```bash
go run cmd/migrate/main.go
```

4. Start the service:
```bash
go run cmd/server/main.go
```

### Running Tests

```bash
go test ./...
```

## Project Structure

```
.
├── cmd/
│   ├── server/     # Main application entry point
│   └── migrate/    # Database migration tool
├── internal/
│   ├── database/   # Database connection and migrations
│   ├── models/     # Data models
│   ├── server/     # HTTP server implementation
│   ├── service/    # Business logic
│   └── storage/    # Storage implementations
├── config.yaml     # Configuration file
└── docker-compose.yml
```

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request
