# Threads Integration

## Overview

Threads uses Meta's Graph API with a two-step container model for publishing. It's publicly available but requires app review for production access.

## OAuth Flow

1. **Authorization Request**: Redirect user to Threads OAuth page
2. **User Authorization**: User approves app permissions  
3. **Callback**: Exchange authorization code for short-lived token
4. **Token Exchange**: Exchange short-lived token for long-lived token
5. **Profile Fetch**: Retrieve user profile

## Required Permissions

| Permission | Description |
|------------|-------------|
| `threads_basic` | Required - Basic profile access |
| `threads_content_publish` | Create and publish posts |

Both permissions should be enabled in your Meta App settings. They work in "Ready for testing" mode for your own account.

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
5. Add redirect URL (must be HTTPS - see Local Development below)
6. Note App ID and App Secret

### 2. Local Development with ngrok

**Important**: Meta requires HTTPS for OAuth redirect URIs. For localdevelopment, use ngrok:

1. Install ngrok: `brew install ngrok` (macOS) or download from [ngrok.com](https://ngrok.com)
2. Run: `ngrok http 8080`
3. Copy the HTTPS URL (e.g., `https://abc123.ngrok-free.app`)
4. In your Meta app settings, add: `https://abc123.ngrok-free.app/api/v1/accounts/threads/callback`
5. Set in your `.env`:
   ```bash
   THREADS_REDIRECT_URI=https://abc123.ngrok-free.app/api/v1/accounts/threads/callback
   ```

### 3. Configure Environment

```bash
THREADS_CLIENT_ID=your_app_id
THREADS_CLIENT_SECRET=your_app_secret
THREADS_REDIRECT_URI=https://your-ngrok-url/api/v1/accounts/threads/callback
```

### 4. Request Production Access

1. Submit app for review
2. Provide screencast of user flow
3. Explain data usage and privacy practices
4. Wait for approval (~2-4 weeks)

## API Endpoints

### Authorization URL
```
GET https://www.threads.com/oauth/authorize
  ?client_id={app_id}
  &redirect_uri={redirect_uri}
  &scope=threads_basic,threads_content_publish
  &response_type=code
```

Note: The library generates its own CSRF state parameter. We store a mapping from the generated state to the workspace_id for retrieval in the callback.

### Token Exchange (Authorization Code → Short-lived Token)
```
POST https://graph.threads.net/oauth/access_token
Content-Type: application/x-www-form-urlencoded

client_id={app_id}
&client_secret={app_secret}
&redirect_uri={redirect_uri}
&code={authorization_code}
&grant_type=authorization_code
```

### Token Exchange (Short-lived → Long-lived Token)
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
GET https://graph.threads.net/v1.0/{user_id}
  ?fields=id,username,name
  &access_token={token}
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
2. **"Permission denied"**: App doesn't have `threads_content_publish` permission
3. **"Application does not have permission for this action"**: Permission error (code 10) - occurs during publishing. Appears as error but post may have been created. Check your Threads account to verify.
4. **"Container not ready"**: Wait ~30 seconds between container creation and publishing

### Debug Tips

1. Exchange short-lived token immediately after auth
2. Refresh tokens every 50 days to prevent expiration
3. Use ngrok for local development (Meta requires HTTPS)
4. If you see duplicate posts, check that retries aren't enabled

### Implementation Notes

The integration uses direct HTTP calls to the Threads API:
- No external library dependency (removed threads-go due to retry issues)
- Two-step publish: create container, then publish
- OAuth state mapping stored in memory for workspace association
- Token refresh handled automatically by the token manager

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