# Twitter/X Integration

## Overview

Twitter/X integration uses OAuth 2.0 with PKCE (Proof Key for Code Exchange) for secure authentication.

## OAuth Flow

1. **Authorization Request**: Generate auth URL with PKCE code challenge
2. **User Authorization**: User approves access on Twitter
3. **Callback**: Exchange authorization code for tokens
4. **Profile Fetch**: Retrieve user profile with access token

## Required Scopes

| Scope | Description |
|-------|-------------|
| `tweet.read` | Read tweets |
| `tweet.write` | Post tweets |
| `users.read` | Read user profile |
| `offline.access` | Refresh tokens |

## Token Management

- **Access Token TTL**: 2 hours
- **Refresh Token TTL**: No expiration (until revoked)
- Tokens are automatically refreshed within 5 minutes of expiration

## Setup

### 1. Create Twitter Developer App

1. Go to [Twitter Developer Portal](https://developer.twitter.com/en/portal/dashboard)
2. Create a new project and app
3. Enable OAuth 2.0 in app settings
4. Add callback URL: `http://localhost:8080/api/v1/accounts/x/callback`
5. Request scopes: `tweet.read`, `tweet.write`, `users.read`, `offline.access`

### 2. Configure Environment

```bash
TWITTER_CLIENT_ID=your_client_id
TWITTER_CLIENT_SECRET=your_client_secret
```

## API Endpoints

### Authorization URL
```
GET https://twitter.com/i/oauth2/authorize
```

### Token Exchange
```
POST https://api.twitter.com/2/oauth2/token
```

### Post Tweet
```
POST https://api.twitter.com/2/tweets
Authorization: Bearer {access_token}
Content-Type: application/json

{"text": "Hello World!"}
```

## Implementation Details

### Code Structure

- `internal/services/oauth/twitter.go` - OAuth implementation
- PKCE verifier stored in-memory with state as key
- Profile fetched via `/2/users/me` endpoint

### PKCE Flow

```go
codeVerifier := oauth2.GenerateVerifier()
codeChallenge := oauth2.S256ChallengeFromVerifier(codeVerifier)

// Store verifier with state
twitterOAuth.StoreVerifier(state, codeVerifier)

// On callback, retrieve verifier
verifier, ok := twitterOAuth.GetVerifier(state)
```

## Rate Limits

| Endpoint | Limit | Window |
|----------|-------|--------|
| POST /2/tweets | 200 | 15 min |
| GET /2/users/me | 187 | 15 min |

## Troubleshooting

### Common Errors

1. **"invalid_client"**: Client ID/secret mismatch
2. **"invalid_grant"**: Authorization code expired or already used
3. **"access_denied"**: User denied authorization
4. **403 Forbidden**: Missing required scopes

### Debug Mode

Enable verbose logging:
```bash
LOG_LEVEL=debug
```

Check token validity:
```bash
curl -H "Authorization: Bearer YOUR_TOKEN" https://api.twitter.com/2/users/me
```