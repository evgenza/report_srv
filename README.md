# report_srv

This project demonstrates an initial Go project structure following the ideas of Clean Architecture and DDD.

The service is intended to generate reports based on Word or XLSX templates filled with the results of SQL queries. It can connect to different databases by specifying the SQL driver and DSN in the configuration.

```
cmd/                - application entry points
internal/
  config/           - configuration entities
  domain/           - domain models
  infrastructure/   - frameworks and external integrations
  interface/        - delivery mechanisms (e.g. HTTP handlers)
  usecase/          - application business logic
```

Configuration allows specifying the database driver and DSN so the service can
work with various SQL engines (e.g. PostgreSQL, MySQL, SQLite).

The current implementation contains only stubs and placeholders.
