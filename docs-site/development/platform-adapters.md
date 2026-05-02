# Platform Adapters

Provider integrations live under `backend/internal/platform/`.

## Current adapters

- `x.go`
- `mastodon.go`
- `bluesky.go`
- `linkedin.go`
- `threads.go`

## Adding a new platform

- [ ] Create `internal/platform/newplatform.go`
- [ ] Implement the platform adapter interface
- [ ] Register the provider in backend startup
- [ ] Add env vars to `.env.example`
- [ ] Add the frontend connect flow
- [ ] Add the platform icon
- [ ] Add provider docs
- [ ] Add tests or a manual test checklist
