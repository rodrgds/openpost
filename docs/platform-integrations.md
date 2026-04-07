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

All platform integrations follow a consistent pattern using the `PlatformAdapter` interface:

1. **OAuth Handler** (`internal/api/handlers/oauth.go`) - Handles OAuth flows for all platforms via map lookups
2. **Platform Adapters** (`internal/platform/`) - Implements `PlatformAdapter` interface (x.go, mastodon.go, bluesky.go, linkedin.go, threads.go)
3. **Token Manager** (`internal/services/tokenmanager/manager.go`) - Manages token refresh lifecycle
4. **Publisher** (`internal/services/publisher/publisher.go`) - Handles posting to platforms
5. **Shared HTTP Helpers** (`internal/platform/http.go`) - DoRequest, DoJSON, DoMultipart, DoFormURLEncoded

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

# Bluesky — no environment variables needed, users connect with handle + app password directly

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

1. Create a new file in `internal/platform/` (e.g., `newplatform.go`)
2. Implement the `PlatformAdapter` interface from `internal/platform/adapter.go`:
   - `GenerateAuthURL(state string) (authURL string, extra map[string]string)`
   - `ExchangeCode(ctx context.Context, code string, extra map[string]string) (*TokenResult, error)`
   - `RefreshToken(ctx context.Context, refreshToken string) (*TokenResult, error)`
   - `GetProfile(ctx context.Context, accessToken string) (*UserProfile, error)`
   - `UploadMedia(ctx context.Context, accessToken, accountID, mimeType string, reader io.Reader) (string, error)`
   - `Publish(ctx context.Context, accessToken, accountID string, req *PublishRequest) (string, error)`
3. Register the adapter in `main.go`'s providers map (e.g., `providers["newplatform"] = adapter`)
4. The provider key gets passed to `tokenManager.SetProvider()` and `publishSvc.SetProvider()` automatically
5. Add the platform's icon to the frontend's `compose-post.svelte` component
6. No switch statements needed — everything uses map lookups

## Troubleshooting

### Common Issues

1. **Token Refresh Failures**: Check that the refresh token hasn't expired. Some platforms (Threads, LinkedIn) require refresh within specific windows.

2. **OAuth State Mismatch**: Ensure the state parameter is being stored and retrieved correctly. For Twitter, the PKCE verifier must be stored server-side.

3. **Posting Failures**: Verify the access token has the correct scopes. Each platform requires specific permissions.

4. **Rate Limits**: Each platform has different rate limits. Check platform-specific docs for details.
