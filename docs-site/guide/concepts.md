# Concepts

Understanding OpenPost's model makes the OAuth and scheduling docs easier to follow.

## Workspace

A workspace groups accounts, media, prompts, and scheduling settings. Most content in OpenPost is workspace-scoped.

## Social account

A connected provider account, such as one X account or one Mastodon profile.

## Post

A single scheduled or published unit of content. A post can target one or multiple providers.

## Thread

A chain of posts published in sequence. OpenPost maps thread replies to each provider's API model.

## Variant

Account-specific content for a post when one message does not fit every connected destination equally well.

## Media

Files stored locally and attached to posts. Some providers, especially Threads, require the media to be publicly reachable through `OPENPOST_MEDIA_URL`.

## Job

Durable background work stored in SQLite. Publishing should go through the jobs table rather than transient goroutines.

## Provider

An adapter implementing one social platform's auth, publish, and media behavior.

## Callback URL

The URL a provider redirects back to after auth. These must match what you configure in the provider developer console.

## Public media URL

The externally reachable base URL for uploaded media. This matters most for Threads.
