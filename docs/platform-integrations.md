# OpenPost Social Platform Integrations

This document provides an overview of all supported social media platforms and their integration details.

## Supported Platforms

| Platform | Status | Auth Method | Token Refresh | Posting API |
|----------|--------|------------|---------------|-------------|
| Twitter/X | ✅ Working | OAuth2 + PKCE | ✅ Yes | v2 API |
| Mastodon | ✅ Working | OAuth2 | N/A (tokens don't expire) | REST API |
| Bluesky | ✅ Implemented | App passwords | ✅ Yes | XRPC API |
| LinkedIn | ✅ Implemented | OAuth2 | ✅ Yes (60-day cycles) | Posts API |
| Threads | ✅ Implemented | Meta OAuth2 | ✅ Yes (60-day cycles) | Graph API |

## Architecture Overview

All platform integrations follow a consistent pattern:

1. **OAuth Handler** (`internal/api/handlers/oauth.go`) - Handles OAuth flows for all platforms
2. **OAuth Services** (`internal/services/oauth/`) - Platform-specific OAuth implementations
3. **Token Manager** (`internal/services/tokenmanager/`) - Manages token refresh lifecycle
4. **Publisher** (`internal/services/publisher/`) - Handles posting to platforms

## Token Security

All access tokens and refresh tokens are encrypted at rest using AES-256-GCM encryption via the `TokenEncryptor` service. Tokens are stored in the `social_accounts` table with encrypted columns:

- `access_token_encrypted` -Encrypted access token
- `refresh_token_encrypted` - Encrypted refresh token
- `token_expires_at` - Token expiration timestamp

## Environment Variables

Configure each platform in your `.env` file:

```bash
# Twitter/X
TWITTER_CLIENT_ID=your_client_id
TWITTER_CLIENT_SECRET=your_client_secret

# Mastodon (JSON array of servers)
MASTODON_REDIRECT_URI=http://localhost:8080/api/v1/accounts/mastodon/callback
MASTODON_SERVERS=[{"name":"mastodon.social","client_id":"id","client_secret":"secret","instance_url":"https://mastodon.social"}]

# Bluesky
BLUESKY_CLIENT_ID=your_client_id
BLUESKY_CLIENT_SECRET=your_client_secret

# LinkedIn
LINKEDIN_CLIENT_ID=your_client_id
LINKEDIN_CLIENT_SECRET=your_client_secret
# Requires LinkedIn approval for w_member_social_feed to support thread replies/comments

# Threads
THREADS_CLIENT_ID=your_app_id
THREADS_CLIENT_SECRET=your_app_secret
```

## Adding a New Platform

To add a new social platform:

1. Create a service file in `internal/services/oauth/{platform}.go`
2. Implement `GenerateAuthURL()`, `ExchangeCode()`, and `RefreshToken()` methods
3. Add platform to the switch statements in `oauth.go` handler
4. Add publishing method in `internal/services/publisher/publisher.go`
5. Add refresh logic in `internal/services/tokenmanager/manager.go`
6. Update models if platform-specific fields are needed

## Troubleshooting

### Common Issues

1. **Token Refresh Failures**: Check that the refresh token hasn't expired. Some platforms (Threads, LinkedIn) require refresh within specific windows.

2. **OAuth State Mismatch**: Ensure the state parameter is being stored and retrieved correctly. For Twitter, the PKCE verifier must be stored server-side.

3. **Posting Failures**: Verify the access token has the correct scopes. Each platform requires specific permissions.

4. **Rate Limits**: Each platform has different rate limits. Check platform-specific docs for details.
