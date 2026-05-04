<p align="center">
  <a href="https://github.com/rodrgds/openpost">
    <img alt="OpenPost Logo" src="./assets/brand/logo.svg" width="280"/>
  </a>
</p>

<p align="center">
  <a href="https://github.com/rodrgds/openpost/releases">
    <img src="https://img.shields.io/github/v/release/rodrgds/openpost?sort=semver&label=Release" alt="Latest Release">
  </a>
  <a href="https://github.com/rodrgds/openpost/pkgs/container/openpost">
    <img src="https://img.shields.io/github/v/release/rodrgds/openpost?sort=semver&label=Image&include_prereleases" alt="Container Image">
  </a>
  <a href="https://github.com/rodrgds/openpost/actions/workflows/ci.yml">
    <img src="https://img.shields.io/github/actions/workflow/status/rodrgds/openpost/ci.yml?label=CI" alt="CI Status">
  </a>
  <a href="LICENSE">
    <img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT">
  </a>
  <a href="SECURITY.md">
    <img src="https://img.shields.io/badge/Security-Security%20Policy-blue" alt="Security Policy">
  </a>
</p>

<div align="center">
  <strong>
    <h2>A lightweight, self-hosted social media scheduler</h2>
  </strong>
  Post to X, Mastodon, Bluesky, Threads, and LinkedIn from your own server.<br/>
  One binary or container. Your data stays on your machine.
</div>

<div align="center">
  <br/>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logos/x-white.svg">
    <img alt="X (Twitter)" src="./assets/logos/x.svg" width="24">
  </picture>
  &nbsp;
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logos/mastodon-white.svg">
    <img alt="Mastodon" src="./assets/logos/mastodon.svg" width="24">
  </picture>
  &nbsp;
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logos/bluesky-white.svg">
    <img alt="Bluesky" src="./assets/logos/bluesky.svg" width="24">
  </picture>
  &nbsp;
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logos/threads-white.svg">
    <img alt="Threads" src="./assets/logos/threads.svg" width="24">
  </picture>
  &nbsp;
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/logos/linkedin-white.svg">
    <img alt="LinkedIn" src="./assets/logos/linkedin.svg" width="24">
  </picture>
</div>

<p align="center">
  <br/>
  <a href="https://op.rgo.pt/"><strong>Documentation</strong></a>
  ·
  <a href="https://op.rgo.pt/guide/quickstart"><strong>Quickstart</strong></a>
  ·
  <a href="https://github.com/rodrgds/openpost/releases"><strong>Releases</strong></a>
</p>

<p align="center">
  <img alt="OpenPost main dashboard screenshot" src="./assets/screenshots/main-dark.png" width="960">
</p>

## Why OpenPost

- Self-hosted: your data stays on your server.
- Single binary or container: no Redis, no Postgres, no external queue.
- SQLite-backed scheduling: queued posts survive restarts.
- Multi-platform publishing: X, Mastodon, Bluesky, Threads, and LinkedIn.
- Encrypted tokens: OAuth tokens are encrypted at rest with AES-256-GCM.
- Thread support: publish multi-post threads in sequence.

## Quickstart

```bash
cp backend/.env.example .env
docker compose up -d
```

Set fresh values for `OPENPOST_JWT_SECRET` and `OPENPOST_ENCRYPTION_KEY` before using OpenPost outside local testing. The first account created on an instance becomes the instance admin automatically. For the full install path, reverse proxy setup, provider OAuth guides, and operations docs, use the docs site.

## Supported Platforms

- X
- Mastodon
- Bluesky
- Threads
- LinkedIn

## Documentation

- [Landing and docs site](https://op.rgo.pt/)
- [Quickstart](https://op.rgo.pt/guide/quickstart)
- [Installation](https://op.rgo.pt/installation/docker-compose)
- [Configuration](https://op.rgo.pt/configuration/environment-variables)
- [Providers](https://op.rgo.pt/providers/overview)
- [Operations](https://op.rgo.pt/operations/troubleshooting)
- [Development](https://op.rgo.pt/development/setup)

## Contributing

Use the development docs in the documentation site, the repo guidance in `AGENTS.md`, and the existing code patterns in `frontend/` and `backend/`.

## Security

Report security issues through [SECURITY.md](SECURITY.md).

## License

MIT
