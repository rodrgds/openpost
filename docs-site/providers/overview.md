# Provider Overview

OAuth and provider app setup are the most common source of deployment friction. Use this section when you are enabling networks one by one.

| Provider | Auth method | Server setup | Notes |
|---|---|---|---|
| X | OAuth 1.0a | Client ID + secret | Requires an X developer app with OAuth 1.0a user auth enabled. |
| Mastodon | OAuth 2.0 per instance | `MASTODON_SERVERS` JSON | One app per instance. |
| Bluesky | App password | None | Users connect with handle + app password. |
| LinkedIn | OAuth 2.0 | Client ID + secret | Replies may need extra approval. |
| Threads | Meta OAuth | Client ID + secret + redirect URI | Public media URL required. |

Start with one provider, confirm the callback works, then expand.
