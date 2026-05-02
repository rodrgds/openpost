# Architecture

## Frontend

- SvelteKit
- TailwindCSS
- Paraglide
- Vitest
- Bun

## Backend

- Go
- Echo
- Huma
- SQLite
- Bun ORM

## Background jobs

Publishing and other durable work flows through a SQLite-backed jobs table.

## Media

Media is stored locally via the `BlobStorage` abstraction.

## Deployment

The built frontend is embedded into the Go binary for single-binary deployment.
