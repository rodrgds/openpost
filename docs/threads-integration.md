# Threads Integration

## Overview

Threads uses Meta's Graph API with a two-step container model for publishing. It's publicly available but requires app review for production access.

## OAuth Flow

1. **Authorization Request**: Redirect user to Threads OAuth page
2. **User Authorization**: User approvesapp permissions
3. **Callback**: Exchange authorization code for short-lived token
4. **Token Exchange**: Exchange short-lived token for long-lived token
5. **Profile Fetch**: Retrieve user profile

## Required Scopes

| Scope | Description |
|-------|-------------|
| `threads_basic` | Required - Basic profile access |
| `threads_content_publish` | Create and publish posts |

## Token Management

- **Short-lived Token TTL**: 1 hour
- **Long-lived Token TTL**: 60 days
- Refresh window: Tokens can be refreshed if at least 24 hours old
- Tokens must be refreshed within 60 days or they expire permanently

## Setup

### 1. Create Meta App

1. Go to [Meta for Developers](https://developers.facebook.com/)
2. Create a new app (Business type)
3. Add Threads API product
4. Configure OAuth settings
5. Add redirect URL: `http://localhost:8080/api/v1/accounts/threads/callback`
6. NoteApp ID and App Secret

### 2. Configure Environment

```bash
THREADS_CLIENT_ID=your_app_id
THREADS_CLIENT_SECRET=your_app_secret
```

### 3. Request Production Access

1. Submit app for review
2. Provide screencast of user flow
3. Explain data usage and privacy practices
4. Wait for approval (~2-4 weeks)

## API Endpoints

### Authorization URL
```
GET https://threads.net/oauth/authorize
  ?client_id={app_id}
  &redirect_uri={redirect_uri}
  &scope=threads_basic,threads_content_publish
  &response_type=code
  &state={workspace_id}
```

### Token Exchange (Short-lived to Long-lived)
```
GET https://graph.threads.net/access_token
  ?grant_type=th_exchange_token
  &client_secret={app_secret}
  &access_token={short_lived_token}
```

### Token Refresh
```
GET https://graph.threads.net/refresh_access_token
  ?grant_type=th_refresh_token
  &access_token={long_lived_token}
```

### Get Profile
```
GET https://graph.threads.net/v1.0/{user_id}?fields=id,username,name&access_token={token}
```

### Create Post (Two-StepProcess)

**Step 1: Create Media Container**
```
POST https://graph.threads.net/v1.0/{user_id}/threads
  ?media_type=TEXT
  &text=Hello%20World!
  &access_token={token}

Response: {"id": "container_id"}
```

**Step 2: Publish Container**
```
POST https://graph.threads.net/v1.0/{user_id}/threads_publish
  ?creation_id={container_id}
  &access_token={token}

Response: {"id": "post_id"}
```

## Media Types

### Text Post
```
media_type=TEXT
text=Your post content
```

### Image Post
```
media_type=IMAGE
image_url=https://example.com/image.jpg
text=Caption (optional)
```

### Video Post
```
media_type=VIDEO
video_url=https://example.com/video.mp4
text=Caption (optional)
```

Note: Media must be at publicly accessible URLs.

## Rate Limits

| Resource | Limit | Window |
|----------|-------|--------|
| Posts per profile | 250 | Day |
| Replies per profile | 1,000 | Day |
| API calls | Variable | Day |

Check rate limit status:
```
GET https://graph.threads.net/v1.0/{user_id}/threads_publishing_limit
  ?fields=quota_usage,config
  &access_token={token}
```

## Troubleshooting

### Common Errors

1. **"Invalid OAuth 2.0 Access Token"**: Token expired or revoked
2. **"Permission denied"**: App doesn't have `threads_content_publish` scope
3. **"Application does not have permission"**: Requires app review
4. **"Container not ready"**: Wait ~30 seconds between steps

### Debug Tips

1. Exchange short-lived token immediately after auth
2. Refresh tokens every 50 days to prevent expiration
3. Check container status before publishing:
   ```
   GET https://graph.threads.net/v1.0/{container_id}?fields=status
   ```
4. Use the `auto_publish_text` parameter for text-only posts (skips step 2)

## Development Mode

Without production approval:
- Can only post to your own Threads account
- Can add up to 5 test accounts
- All other functionality works normally

## API Versioning

Threads API uses versioned endpoints:
```
https://graph.threads.net/v1.0/{endpoint}
```

Current stable version: `v1.0`

## References

- [Threads API Documentation](https://developers.facebook.com/docs/threads)
- [Graph API Explorer](https://developers.facebook.com/tools/explorer/)
- [App Review Process](https://developers.facebook.com/docs/app-review)