# OpenPost AI Agents & Development Guidelines

This document serves as a guideline for autonomous AI agents (like Copilot, Cursor, Codeium, or CLI agents) and human developers contributing to **OpenPost**. It outlines the core tech stack, architectural rules, and specific instructions for AI behavior.

## 1. Core Architecture & Tech Stack

**Frontend:**
- **Framework:** SvelteKit (using `@sveltejs/adapter-static` for simple SPA deployment).
- **Styling:** TailwindCSS.
- **i18n:** Paraglide.
- **Testing:** Vitest.
- **Package Manager:** Bun (`bun.com`).

**Backend:**
- **Language:** Go (1.25+).
- **Framework:** Echo (`github.com/labstack/echo/v4`). *Note: Earlier plans mentioned Fiber, but we are standardizing on Echo for this repository.*
- **Database:** SQLite (configured with PRAGMA WAL, busy_timeout).
- **ORM:** Bun (`github.com/uptrace/bun`).
- **Background Jobs:** Custom SQLite-backed polling worker using the `jobs` table (no external Redis dependency).

**Deployment:**
- **Strategy:** Single Go binary. SvelteKit's static output is embedded directly into the Go executable using `go:embed`.

## 2. Agent Guidelines & Coding Mandates

When an AI agent is invoked to assist with this repository, it MUST adhere to the following rules:

### A. Idiomatic Code & Consistency
- **Go Backend:** Always use the Echo framework for HTTP handlers. Follow the established dependency injection pattern in `main.go`. Maintain separation of concerns: Handlers -> Services -> Database.
- **SvelteKit Frontend:** Always use standard Svelte 5 runes and `+page.svelte`/`+page.ts` structures. Use standard `fetch` against `/api/v1` routes.
- **ORM Patterns:** Always use `github.com/uptrace/bun` for database operations. Do not write raw SQL strings unless doing complex SQLite pragmas or advanced queue polling.

### B. State Management & Single Binary Constraints
- **Filesystem Constraints:** OpenPost is meant to be highly portable. Local file storage (e.g., SQLite DB file, local media uploads) should be configurable via environment variables (e.g., `OPENPOST_DB_PATH`, `OPENPOST_MEDIA_PATH`).
- **Asset Embedding:** Do not modify the SvelteKit build pipeline in a way that breaks `adapter-static`. The backend relies on a static `build/` directory to embed into the binary.

### C. Security & Credentials
- Tokens for social accounts (Access Tokens, Refresh Tokens) MUST ALWAYS be encrypted at rest using the `TokenEncryptor` service (AES-256-GCM).
- Do NOT hardcode cryptographic secrets in the codebase. Always load from environment variables (e.g., `ENCRYPTION_KEY`, `JWT_SECRET`).

### D. Workflow for Feature Implementation
1. **Model First:** If a feature requires data, update the `models.go` and `database.go` schema creation first.
2. **Backend Logic:** Implement the Service and the Echo API Handler.
3. **Frontend Implementation:** Write Svelte components and SvelteKit routes to interact with the new endpoint.
4. **Queue (if applicable):** If the action is async (e.g., publishing a post), insert a payload into the `jobs` table instead of blocking the HTTP request.

## 3. Prompts & Agent Commands (For Quick Context)

*If you are an agent, read these context hints before performing actions:*

- **"Add a new social platform"**: Requires adding OAuth logic in `internal/services/oauth`, updating `models.SocialAccount.Platform` check, adding the platform icon in SvelteKit, and ensuring the `Publisher` service knows how to talk to that API.
- **"Modify database schema"**: Update `internal/models/models.go` struct fields with appropriate bun tags. Since we rely on `.IfNotExists()` in `database.go` currently, provide migration steps or table alter scripts if the table already exists.
- **"Create a background job"**: Do not use `goroutine` blindly for tasks that must survive server restarts. Insert a row into the `models.Job` table so the `BackgroundWorker` can pick it up.