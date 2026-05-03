---
layout: home

hero:
  name: OpenPost
  text: Self-hosted social media scheduling.
  tagline: Post to X, Mastodon, Bluesky, Threads, and LinkedIn from your own server. One binary or container. Your data stays on your machine.
  image:
    src: /assets/brand/logo-docs.svg
    alt: OpenPost logo
  actions:
    - theme: brand
      text: Get Started
      link: /guide/quickstart
    - theme: alt
      text: View on GitHub
      link: https://github.com/rodrgds/openpost

features:
  - icon: 🧡
    title: Self-hosted
    details: Run OpenPost on your own server with Docker Compose or a single binary.
  - icon: 🗃️
    title: SQLite by default
    details: No Postgres, Redis, or external queue required for a simple deployment.
  - icon: 🔐
    title: Encrypted tokens
    details: OAuth tokens are encrypted at rest with your own encryption key.
  - icon: 📅
    title: Scheduling built in
    details: Queue posts, use posting schedules, and keep work durable across restarts.
  - icon: 🧵
    title: Thread support
    details: Publish multi-post threads in sequence across supported providers.
  - icon: 🖼️
    title: Media library
    details: Store reusable media locally and attach it to scheduled posts.
---

## Install in a minute

```yaml
services:
  openpost:
    image: ghcr.io/rodrgds/openpost:latest
    container_name: openpost
    restart: unless-stopped
    env_file:
      - .env
    ports:
      - "8080:8080"
    volumes:
      - openpost_data:/data
    environment:
      - OPENPOST_PORT=8080
      - OPENPOST_DATABASE_PATH=/data/db/openpost.db
      - OPENPOST_MEDIA_PATH=/data/media

volumes:
  openpost_data:
```

::: tip
New to OpenPost? Start with the [Quickstart](/guide/quickstart) guide.
:::
