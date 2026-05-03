# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

## [1.0.1] - 2026-05-03

### Fixed
- Docker release builds now copy the repo `scripts/` directory so the frontend asset-sync step works in GitHub Actions and container releases complete successfully.

## [1.0.0] - 2026-05-03

### Added
- Account-level MFA with QR-based TOTP enrollment, passkey registration, and step-up login verification, plus settings UI for managing both methods.
- VitePress documentation site scaffold under `docs-site/`, including landing page, sidebar/navigation config, OpenPost-themed styling, and first-pass operator/contributor docs.
- Shared asset sync pipeline that copies canonical repo assets into frontend and docs public directories.
- GitHub Pages workflow for building and deploying the docs site.
- Token refresh job scheduling plus backend tests covering queued refresh execution and provider-specific refresh credentials.
- Dedicated account-connection success callback page for returning OAuth users to `/accounts`.
- Workspace migration scaffold for configurable draft gap minutes.
- Workspace setting for `draft_gap_minutes`, used by suggested queue times when a day's configured schedule slots are already occupied.

### Changed
- Settings now include account-security controls, while login can require a second factor when TOTP or passkeys are enabled.
- Optimized GitHub Actions CI by priming a shared Nix store cache before lint/test jobs, caching Go/lint/Bun dependencies, skipping unaffected backend/frontend jobs, and moving Go race tests off pull request runs.
- README reduced to a shorter front door that points detailed setup and operations content at the docs site.
- Docs site base-path handling now defaults to `/` for custom-domain hosting, with `OPENPOST_DOCS_BASE` available as an explicit override for repository-path deployments like `/openpost/`.
- README docs links now point at the custom docs domain `https://op.rgo.pt`.
- Docs now include a Nix module deployment page backed by a build-time sync of the production module from `rodrgds/nix-config`.
- Token refresh handling now declares platform capabilities explicitly, retries publish attempts on any supported expired account, and routes OAuth success redirects through the new callback screen.
- Workspace settings no longer auto-overwrite shared timezone and week-start values from the first browser locale that opens a workspace.
- Posting schedule settings now use a local-time weekly grid with per-day toggles and row-based time management instead of a flat UTC slot list.
- Suggested posting times now consider already scheduled posts and fall back to the configured minimum draft gap when a day has no unused schedule slots left.
- Weekly posting schedules now preserve the configured workspace-local time across DST changes instead of drifting by the current UTC offset.

### Fixed
- Mastodon accounts now persist their configured `instance_url` as the canonical provider key, avoiding publish/token-refresh mismatches after OAuth connection.
- The default Mastodon callback URI now matches the documented backend callback endpoint on `localhost:8080`.
- Mastodon server listings now avoid duplicate entries when adapters are registered with both UI labels and canonical instance-url keys.

## [0.4.4] - 2026-04-19

Changes since `v0.4.3`.

### Added
- X OAuth request store handler for temporary request-state persistence.
- Frontend OpenAPI snapshot and generated API TypeScript declarations tracked in-repo for CI consistency.
- Placeholder file in embedded web public directory to keep `go:embed` stable in clean checkouts.

### Changed
- X OAuth handler and platform integration flow refinements.
- Backend model and database updates supporting the latest auth/request-state behavior.
- Frontend pre-commit/devenv validation flow now runs deterministic generation/check steps for i18n and OpenAPI types.
- Frontend dashboard and media routes fixed strict TypeScript nullability errors found in CI.
- Frontend ignore/format rules adjusted to avoid generated-file drift during hooks.

## [0.4.3] - 2026-04-19

Changes since `v0.4.2`.

### Added
- Prompt management backend API (`/prompts`, `/prompts/random`, `/prompts/categories`, create/delete custom prompts).
- Built-in prompt catalog seeding and prompt category support.
- Posting schedule backend API (`/posting-schedules` list/create/update/delete).
- Prompt browsing UI at `/prompts` with category filtering, random prompt selection, and custom prompt creation.
- Compose flow integration for using prompts directly in new posts.
- Settings UI support for posting schedule slot management.

### Changed
- Post handler logic expanded for improved post management and scheduling workflows.
- Media handler behavior updated for media lifecycle and cleanup flow alignment.
- Authentication middleware updated for request auth handling refinements.
- Database/model layer updated with new scheduling and prompt entities.
- Queue worker updated to process scheduling-related jobs.
- Frontend layout refactors for improved page consistency (`PageContainer`, `EmptyState`, sidebar and dashboard updates).
- Favicon assets refreshed.

### Project And Docs
- Frontend page layout refactor and onboarding/UI refinements.
- Added AI agent skill definitions and repo agent guideline updates.
- Added/expanded roadmap and planning documentation updates.

### Commit Summary Since v0.4.2
- `681e3ab` refactor(frontend): unify page layouts with PageContainer and EmptyState components
- `bde9cc1` docs(agents): add conventional commits and branches requirement
- `a6f60ee` feat(frontend): add onboarding page and UI refinements
- `a53ef22` feat(agents): add AI agent skill definitions
- `7289963` feat: implement Phase 3 - Media Management & Cleanup
- `87a1901` feat: implement Phase 2 - Platform Customization & Social Media Sets
- `80c302c` feat: enhance post management features
- `cb8a110` feat: add comprehensive roadmap for OpenPost features and priorities
