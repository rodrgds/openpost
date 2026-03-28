# Bluesky Integration

## Overview

Bluesky uses the AT Protocol with **app passwords** for authentication. No OAuth setup or environment variables needed - just your handle and an app password.

## How It Works

1. **User enters credentials**: Handle + app password on the accounts page
2. **Create session**: POST to `com.atproto.server.createSession` with handle + password
3. **Receive tokens**: Access token (JWT) + refresh token (JWT)
4. **Post content**: Use access token to call `com.atproto.repo.createRecord`

## Setup (User Steps)

1. Go to [Bluesky Settings](https://bsky.app/settings/app-passwords)
2. Click "Add App Password"
3. Name it (e.g., "OpenPost")
4. Copy the generated password (format: `xxxx-xxxx-xxxx-xxxx`)
5. In OpenPost, click Connect on Bluesky, enter your handle and app password

## Token Management

- **Access Token**: JWT with ~2 hour expiry
- **Refresh Token**: JWT with longer validity
- Tokens refresh automatically via `com.atproto.server.refreshSession`

## API Endpoints Used

| Operation | XRPC Method |
|-----------|-------------|
| Create Session | `com.atproto.server.createSession` |
| Refresh Session | `com.atproto.server.refreshSession` |
| Create Record (Post) | `com.atproto.repo.createRecord` |

## Post Format

```json
{
  "repo": "did:plc:...",
  "collection": "app.bsky.feed.post",
  "record": {
    "$type": "app.bsky.feed.post",
    "text": "Hello World!",
    "createdAt": "2024-01-01T00:00:00.000Z"
  }
}
```

## Rate Limits

| Operation | Limit | Window |
|-----------|-------|--------|
| Create Record | 5000 | Hour |
| Create Session | 30 | 5 min |

## Why App Passwords?

App passwords are the standard way third-party Bluesky clients authenticate:
- No complex OAuth setup required
- Scoped permissions (can't delete account, change password, etc.)
- Can be revoked anytime from Bluesky settings
- Works with any Bluesky-compatible service (not just bsky.social)

## Troubleshooting

1. **"Invalid credentials"**: Double-check your handle and app password
2. **"Rate limited"**: Too many attempts, wait a few minutes
3. **"Account not found"**: Verify your handle is correct (e.g., `user.bsky.social`)
