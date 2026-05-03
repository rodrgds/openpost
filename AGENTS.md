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
- **Framework:** Echo (`github.com/labstack/echo/v4`). Huma for OpenAPI spec generation.
- **Database:** SQLite (configured with PRAGMA WAL, busy_timeout).
- **ORM:** Bun (`github.com/uptrace/bun`).
- **Background Jobs:** Custom SQLite-backed polling worker using the `jobs` table (no external Redis dependency).
- **Media Storage:** Local filesystem via `BlobStorage` interface (configurable via `OPENPOST_MEDIA_PATH`).

**Deployment:**
- **Strategy:** Single Go binary. SvelteKit's static output is embedded directly into the Go executable using `go:embed`.

## 2. Platform Adapter Architecture

All social platform integrations follow a unified `PlatformAdapter` interface defined in `internal/platform/adapter.go`. Each platform implements this interface in its own file within `internal/platform/`:

| File | Platform | Auth Method |
|------|----------|-------------|
| `x.go` | Twitter/X | OAuth 2.0 PKCE |
| `mastodon.go` | Mastodon | OAuth 2.0 (per-instance) |
| `bluesky.go` | Bluesky | App Passwords |
| `linkedin.go` | LinkedIn | OAuth 2.0 |
| `threads.go` | Threads | Meta OAuth 2.0 |

Adapters are registered in `main.go` via a `map[string]platform.PlatformAdapter` and passed to the token manager, publisher, and OAuth handler. **No switch statements** — everything uses map lookups.

Shared HTTP helpers are in `internal/platform/http.go`:
- `DoRequest` — generic HTTP request with error handling
- `DoJSON` — JSON marshaled request
- `DoMultipart` — multipart file upload
- `DoFormURLEncoded` — form-encoded request

## 3. Agent Guidelines & Coding Mandates

When an AI agent is invoked to assist with this repository, it MUST adhere to the following rules:

### A. Commit & Branch Conventions
- **Always use Conventional Commits** (e.g., `feat:`, `fix:`, `chore:`, `refactor:`, `docs:`). Follow https://www.conventionalcommits.org/
- **Always use Conventional Branches** (e.g., `feature/add-login`, `fix/header-alignment`, `hotfix/emergency-patch`)
- **Always update the Changelog** for any major features, bug fixes, or breaking changes. Use the `## [Unreleased]` section to document changes since the last release.
- **Go Backend:** Use Echo for HTTP handlers and Huma for OpenAPI endpoints. Follow the dependency injection pattern in `main.go`. Maintain separation of concerns: Handlers -> Services -> Database.
- **Platform Adapters:** Implement `PlatformAdapter` interface. Never put platform logic outside the `internal/platform/` package. Use shared HTTP helpers from `http.go`.
- **SvelteKit Frontend:** Always use standard Svelte 5 runes (`$state`, `$derived`, `$effect`, `$props`, `$bindable`). Use `+page.svelte`/`+page.ts` structures. Use the openapi-fetch typed client against `/api/v1` routes.
- **ORM Patterns:** Always use `github.com/uptrace/bun` for database operations. Do not write raw SQL strings unless doing complex SQLite pragmas or advanced queue polling.

### B. State Management & Single Binary Constraints
- **Filesystem Constraints:** OpenPost is meant to be highly portable. Local file storage (e.g., SQLite DB file, local media uploads) should be configurable via environment variables (e.g., `OPENPOST_DATABASE_PATH`, `OPENPOST_MEDIA_PATH`).
- **Asset Embedding:** Do not modify the SvelteKit build pipeline in a way that breaks `adapter-static`. The backend relies on a static `build/` directory to embed into the binary.

### C. Security & Credentials
- Tokens for social accounts (Access Tokens, Refresh Tokens) MUST ALWAYS be encrypted at rest using the `TokenEncryptor` service (AES-256-GCM).
- Do NOT hardcode cryptographic secrets in the codebase. Always load from environment variables (e.g., `OPENPOST_ENCRYPTION_KEY`, `OPENPOST_JWT_SECRET`).

### D. Workflow for Feature Implementation
1. **Model First:** If a feature requires data, update the `models.go` and `database.go` schema creation first.
2. **Backend Logic:** Implement the Service and the Echo API Handler.
3. **Frontend Implementation:** Write Svelte components and SvelteKit routes to interact with the new endpoint.
4. **Queue (if applicable):** If the action is async (e.g., publishing a post), insert a payload into the `jobs` table instead of blocking the HTTP request.

## 4. Prompts & Agent Commands (For Quick Context)

*If you are an agent, read these context hints before performing actions:*

- **"Add a new social platform"**: Create a new file in `internal/platform/` implementing `PlatformAdapter`. Register it in `main.go` under the provider map. Add platform icon to frontend's `compose-post.svelte`. Update the accounts page (`/accounts`) with connect UI.
- **"Modify database schema"**: Update `internal/models/models.go` struct fields with appropriate bun tags. Since we rely on `.IfNotExists()` in `database.go` currently, provide migration steps or table alter scripts if the table already exists.
- **"Create a background job"**: Do not use `goroutine` blindly for tasks that must survive server restarts. Insert a row into the `models.Job` table so the `BackgroundWorker` can pick it up.
- **"Handle media uploads"**: Use the `BlobStorage` interface for file storage. The publisher fetches media from disk via `os.ReadFile()` and passes to `adapter.UploadMedia()`. For Threads, media must be served at a publicly accessible URL.
- **"Implement threading"**: Use `Post.ParentPostID` and `Post.ThreadSequence`. The publisher detects thread chains and publishes sequentially. Each adapter's `Publish` method handles `ReplyToID` platform-specifically.

## 5. Media & Threading Per Platform

| Platform | Media Upload | Threading |
|----------|-------------|-----------|
| X/Twitter | `POST /2/media/upload` (chunked for video) | `reply.in_reply_to_tweet_id` |
| Mastodon | `POST /api/v2/media` (async poll for large files) | `in_reply_to_id` |
| Bluesky | `com.atproto.repo.uploadBlob` (raw binary) | `reply: {root, parent}` with uri+cid JSON |
| LinkedIn | Vector Assets API (register→PUT→URN) | Comments API (`/socialActions/{urn}/comments`) |
| Threads | Public URL in `image_url`/`video_url` | `reply_to_id` |

## 6. Provider Key Convention

Provider keys in the `providers` map follow specific formats:

| Platform | Provider Key Format | Example |
|----------|---------------------|---------|
| X/Twitter | `"x"` | `"x"` |
| Mastodon | `"mastodon:" + server.Name` | `"mastodon:Personal"` |
| Bluesky | `"bluesky"` | `"bluesky"` |
| LinkedIn | `"linkedin"` | `"linkedin"` |
| Threads | `"threads"` | `"threads"` |

**Important:** For Mastodon, the `instanceURL` stored in `SocialAccount.InstanceURL` must match exactly with the key used to register the adapter. The adapter is registered with `"mastodon:" + server.InstanceURL` (the full URL from config, e.g., `https://masto.pt`). When looking up the provider, use `"mastodon:" + account.InstanceURL` without modification.
