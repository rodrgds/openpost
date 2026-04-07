# Twitter/X Integration

## Overview

Twitter/X integration uses **OAuth 1.0a** (not OAuth 2.0) for authentication. This is the classic three-legged OAuth flow with request tokens, user authorization, and access tokens.

## OAuth Flow

1. **Request Token**: POST to `https://api.twitter.com/oauth/request_token` to get a request token and secret
2. **Authorization Request**: Redirect user to Twitter authorization page with request token
3. **User Authorization**: User approves access on Twitter
4. **Callback**: Exchange request token + verifier for access token and secret
5. **Profile Fetch**: Retrieve user profile with signed requests

## Required Scopes

OAuth 1.0a uses **App Permissions** in the Twitter Developer Portal rather than scopes. Ensure your app has:
- **Read** — Read tweets and user profiles
- **Write** — Post tweets
- **Offline access** is not applicable (OAuth 1.0a tokens do not expire)

## Token Management

- **Access Token**: Does not expire (OAuth 1.0a)
- **Refresh Token**: Not supported — OAuth 1.0a tokens are permanent until revoked
- The access token is stored as a combined string: `oauth_token|oauth_token_secret`
- No automatic refresh is performed; `RefreshToken()` returns an error

## Setup

### 1. Create Twitter Developer App

1. Go to [Twitter Developer Portal](https://developer.twitter.com/en/portal/dashboard)
2. Create a new project and app
3. Set App Permissions to **Read and Write**
4. Add callback URL: `http://localhost:8080/api/v1/accounts/x/callback`
5. Note the API Key (Consumer Key) and API Key Secret (Consumer Secret)

### 2. Configure Environment

```bash
TWITTER_CLIENT_ID=your_api_key_here
TWITTER_CLIENT_SECRET=your_api_key_secret_here
```

## API Endpoints

### Request Token
```
POST https://api.twitter.com/oauth/request_token
```

### Authorization URL
```
GET https://api.twitter.com/oauth/authorize?oauth_token={request_token}
```

### Access Token Exchange
```
POST https://api.twitter.com/oauth/access_token
```

### Post Tweet
```
POST https://api.twitter.com/2/tweets
Authorization: OAuth (signed request)
Content-Type: application/json

{"text": "Hello World!"}
```

### Get Profile
```
GET https://api.twitter.com/2/users/me?user.fields=id,name,username
Authorization: OAuth (signed request)
```

## Implementation Details

### Code Structure

- `internal/platform/x.go` — Twitter/X platform adapter
- Uses `github.com/dghubble/oauth1` for OAuth 1.0a signing
- Request token metadata (secret, workspace ID) stored in-memory via `sync.Map`
- Profile fetched via `/2/users/me` endpoint
- Media upload supports both simple (≤5MB images) and chunked (video/GIF) modes

### OAuth 1.0a Flow

```go
// Step 1: Get request token
config := oauth1.Config{ConsumerKey, ConsumerSecret, CallbackURL, Endpoint}
requestToken, requestSecret, _ := config.RequestToken()

// Step 2: Store secret with request token for callback lookup
x.requestMeta.Store(requestToken, xRequestMeta{Secret: requestSecret, WorkspaceID: workspaceID})

// Step 3: On callback, retrieve secret and exchange for access token
meta, _ := x.requestMeta.Load(oauthToken)
accessToken, accessSecret, _ := config.AccessToken(oauthToken, requestSecret, oauthVerifier)

// Step 4: Store as combined token
combined := accessToken + "|" + accessSecret
```

### Media Upload

- **Simple upload**: Images ≤5MB use `POST /1.1/media/upload.json` with multipart form
- **Chunked upload**: Videos and GIFs use the INIT → APPEND → FINALIZE flow
- Media IDs expire ~2 hours after upload, so upload happens at publish time, not creation time

## Rate Limits

| Endpoint | Limit | Window |
|----------|-------|--------|
| POST /2/tweets | 200 | 15 min |
| GET /2/users/me | 187 | 15 min |
| POST /1.1/media/upload.json | Varies by media type | 24 hours |

## Troubleshooting

### Common Errors

1. **"invalid_client"**: API Key/Secret mismatch
2. **"invalid_grant"**: Request token expired or already used
3. **"access_denied"**: User denied authorization
4. **403 Forbidden**: App lacks Write permission
5. **"x oauth1 tokens do not support refresh"**: Expected — OAuth 1.0a tokens don't expire

### Debug Mode

Enable verbose logging:
```bash
LOG_LEVEL=debug
```