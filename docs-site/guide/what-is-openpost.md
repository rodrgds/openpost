# What Is OpenPost?

OpenPost is a lightweight, self-hosted social media scheduler. It lets you connect social accounts, compose posts, attach media, schedule publishing, and manage threads from your own server.

It is built for operators who want a practical publishing tool without handing content, tokens, and schedules to a hosted SaaS. OpenPost keeps the stack intentionally small: Go, SvelteKit, SQLite, local media storage, and a single deployable binary or container.

## Who it is for

- Homelab and self-hosted users
- Small teams that want control over credentials and data
- Developers who want to extend a focused open source scheduler

## What it supports

- X
- Mastodon
- Bluesky
- LinkedIn
- Threads

## What it deliberately avoids

- Redis or external queue requirements for simple deployments
- Postgres as a mandatory dependency
- Hosted-account lock-in
- Splitting the app into multiple services before it is necessary
