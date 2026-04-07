# LinkedIn Integration

## Overview

LinkedIn uses OAuth 2.0 with the Posts API for publishing content. The API requires specific scopes and includes token refresh capabilities.

## OAuth Flow

1. **Authorization Request**: Redirect user to LinkedIn OAuth page
2. **User Authorization**: User approves with theirLinkedIn account
3. **Callback**: Exchange authorization code for access and refresh tokens
4. **Profile Fetch**: Retrieve user profile for account identification

## Required Scopes

| Scope | Description |
|-------|-------------|
| `openid` | OpenID Connect |
| `profile` | Basic profile access |
| `w_member_social` | Post on behalf of member |
| `w_member_social_feed` | Create comments/replies via Social Actions API (required for OpenPost thread replies) |

## Token Management

- **Access Token TTL**: 60 days
- **Refresh Token TTL**: 365 days (1 year)
- Refresh tokens can be used before expiration to get new tokens
- Both tokens refresh together with each refresh request

## Setup

### 1. Create LinkedIn App

1. Go to [LinkedIn Developer Portal](https://www.linkedin.com/developers/apps)
2. Create a new app
3. Request "Sign In with LinkedIn" and "Share on LinkedIn" products
4. Request access/approval for `w_member_social_feed` (Social Actions create permission)
5. Add redirect URL: `http://localhost:8080/api/v1/accounts/linkedin/callback`
6. Note the Client ID and Client Secret

### 2. Configure Environment

```bash
LINKEDIN_CLIENT_ID=your_client_id
LINKEDIN_CLIENT_SECRET=your_client_secret
# Optional: disable LinkedIn thread child replies if your app cannot get
# w_member_social_feed approval yet.
OPENPOST_DISABLE_LINKEDIN_THREAD_REPLIES=false
```

### 3. Request API Access

For production use, submit your app for review to get:
- `w_member_social` scope (posting capability)
- `w_member_social_feed` scope (comment/reply capability used for OpenPost thread replies)
- Access to the Posts API

## API Endpoints

### Authorization URL
```
GET https://www.linkedin.com/oauth/v2/authorization
  ?response_type=code
  &client_id={client_id}
  &redirect_uri={redirect_uri}
  &scope=openid%20profile%20w_member_social%20w_member_social_feed
  &state={workspace_id}
```

### Token Exchange
```
POST https://www.linkedin.com/oauth/v2/accessToken
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code={auth_code}
&redirect_uri={redirect_uri}
&client_id={client_id}
&client_secret={client_secret}
```

### Token Refresh
```
POST https://www.linkedin.com/oauth/v2/accessToken
Content-Type: application/x-www-form-urlencoded

grant_type=refresh_token
&refresh_token={refresh_token}
&client_id={client_id}
&client_secret={client_secret}
```

### Get Profile
```
GET https://api.linkedin.com/v2/userinfo
Authorization: Bearer {access_token}
```

### Create Post
```
POST https://api.linkedin.com/rest/posts
Authorization: Bearer {access_token}
Content-Type: application/json
X-Restli-Protocol-Version: 2.0.0
Linkedin-Version: {YYYYMM}  # dynamically computed (previous month)

{
  "author": "urn:li:person:{person_id}",
  "commentary": "Hello World!",
  "visibility": "PUBLIC",
  "distribution": {
    "feedDistribution": "MAIN_FEED",
    "targetEntities": [],
    "thirdPartyDistributionChannels": []
  },
  "lifecycleState": "PUBLISHED"
}
```

## API Versioning

LinkedIn requires a version header:

```http
Linkedin-Version: YYYYMM
```

The version is **dynamically computed** at runtime — defaults to the previous month (to avoid `NONEXISTENT_VERSION` errors when a new version isn't yet active). Override with the `LINKEDIN_API_VERSION` environment variable if needed.

Versions follow `YYYYMM` format and are supported for at least 1 year.

## RateLimits

| Resource | Limit | Window |
|----------|-------|--------|
| POST /posts | 100,000 | Day |
| GET /userinfo | 100,000 | Day |

Member-level limits may apply.

## Troubleshooting

### Common Errors

1. **"invalid_grant"**: Authorization code expired or already used
2. **"insufficient_scope"**: App doesn't have required permissions
3. **403 Forbidden**: Token lacks `w_member_social` or `w_member_social_feed` scope
4. **429 Too Many Requests**: Rate limit exceeded

### Debug Tips

1. Check token permissions in Developer Portal
2. Verify the `Linkedin-Version` header is current (or set `LINKEDIN_API_VERSION`)
3. Use `/v2/userinfo` to validate token
4. Test with Postman before implementing
5. If replies fail with `partnerApiSocialActions.CREATE`, your app is missing `w_member_social_feed` approval

### Code Structure

- `internal/platform/linkedin.go` — LinkedIn platform adapter
- Media upload uses Vector Assets API (register → PUT upload → asset URN)
- Threading uses Comments API (`/rest/socialActions/{urn}/comments`)
- `OPENPOST_DISABLE_LINKEDIN_THREAD_REPLIES` env var disables thread replies when app lacks `w_member_social_feed`

## Access Levels

### Development Mode
- Post only to ownprofile
- Limited to 5 test accounts
- No approval needed

### Production Mode
- Requires app review
- Approval time: ~2-4 weeks per permission
- Screencast demonstration required
