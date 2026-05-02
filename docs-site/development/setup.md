# Development Setup

## Clone the repository

```bash
git clone https://github.com/rodrgds/openpost.git
cd openpost
```

## Frontend

```bash
cd frontend
bun install
bun run dev
```

Frontend dev server: `http://localhost:5173`

## Backend

```bash
cd ../backend
cp .env.example .env
go run ./cmd/openpost
```

Backend server: `http://localhost:8080`

## Docs site

From the repo root:

```bash
bun run sync:assets
cd docs-site
bun install
bun run docs:dev
```

Docs site: `http://localhost:4174`
