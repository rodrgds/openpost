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
  - icon: 🌐
    title: One post, many networks
    details: Publish from one place to X, Mastodon, Bluesky, Threads, and LinkedIn.
  - icon: ✍️
    title: Provider-specific variants
    details: Start from one canonical post, then tailor copy where each network needs its own version.
  - icon: 🧵
    title: Thread composer
    details: Build multi-post threads and publish them in sequence instead of stitching replies together manually.
  - icon: 📅
    title: Scheduling that stays queued
    details: Plan posts ahead, use posting schedules, and keep jobs durable through restarts.
  - icon: 🖼️
    title: Reusable media library
    details: Upload once, reuse across drafts and scheduled posts, and keep media close to your content workflow.
  - icon: 🗂️
    title: Workspaces for separate brands
    details: Keep accounts, media, prompts, and schedules organized per workspace.
  - icon: 🔐
    title: Your server, your credentials
    details: Keep content, schedules, and connected account tokens under your own control.
  - icon: ⚡
    title: Fast to deploy
    details: Run OpenPost with Docker Compose or a single binary without turning setup into a platform project.
---

<p>
  <img
    src="/assets/screenshots/main-dark.png"
    alt="OpenPost main dashboard"
    style="width: 100%; max-width: 1200px; border-radius: 16px; border: 1px solid var(--vp-c-divider);"
  >
</p>

<div
  style="display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 16px; align-items: start;"
>
  <div>
    <img
      src="/assets/screenshots/settings-dark.png"
      alt="OpenPost settings page"
      style="width: 100%; border-radius: 16px; border: 1px solid var(--vp-c-divider);"
    >
  </div>
  <div style="display: grid; gap: 16px;">
    <img
      src="/assets/screenshots/accounts-dark.png"
      alt="OpenPost accounts page"
      style="width: 100%; border-radius: 16px; border: 1px solid var(--vp-c-divider);"
    >
    <img
      src="/assets/screenshots/media-dark.png"
      alt="OpenPost media page"
      style="width: 100%; border-radius: 16px; border: 1px solid var(--vp-c-divider);"
    >
  </div>
</div>

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
