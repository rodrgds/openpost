# Backend

The backend uses Echo for HTTP handling, Huma for OpenAPI generation, SQLite for persistence, and Bun ORM for database access.

## Layering

- Handlers
- Services
- Database/models

## Expectations

- Keep platform logic inside `internal/platform/`
- Prefer Bun ORM over raw SQL for normal queries
- Use dependency injection patterns from `main.go`
