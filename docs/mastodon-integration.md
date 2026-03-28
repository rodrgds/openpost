# Mastodon Integration

## Overview

Mastodon uses standard OAuth 2.0 for authentication. Each Mastodon instance requires separate app registration and configuration.

## OAuth Flow

1. **Authorization Request**: Redirect user to instance OAuth page
2. **User Authorization**: User approves access
3. **Callback/Exchange**: Exchange authorization code for access token
4. **Profile Fetch**: Retrieve user profile (optional)

## Required Scopes

| Scope | Description |
|-------|-------------|
| `read` | Read account data |
| `write` | Write statuses, etc. |

## Token Management

- **Access Token TTL**: No expiration (valid until revoked)
- **Refresh Token**: Not required for Mastodon
- Tokens remain valid until the user revokes access or the app is removed

## Setup

### 1. Create Mastodon App

For each Mastodon instance you want to support:

1. Go to `{instance_url}/settings/applications`
2. Create a new application
3. Set redirect URI: `http://localhost:8080/api/v1/accounts/mastodon/callback` (or `urn:ietf:wg:oauth:2.0:oob`) for local use
4. Note the Client ID and Client Secret

### 2. Configure Environment

Mastodon uses a JSON array configuration for multiple instances:

```bash
MASTODON_REDIRECT_URI=http://localhost:8080/api/v1/accounts/mastodon/callback
MASTODON_SERVERS=[\
  {"name":"mastodon.social","client_id":"your_client_id","client_secret":"your_client_secret","instance_url":"https://mastodon.social"},\
  {"name":"fosstodon","client_id":"your_client_id","client_secret":"your_client_secret","instance_url":"https://fosstodon.org"}\
]
```

## API Endpoints

### Authorization URL
```
GET {instance_url}/oauth/authorize
  ?client_id={client_id}
  &redirect_uri={redirect_uri}
  &response_type=code
  &scope=read+write
  &state={workspace_id}
```

### Token Exchange
```
POST {instance_url}/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
&code={auth_code}
&redirect_uri={redirect_uri}
&client_id={client_id}
&client_secret={client_secret}
```

### Post Status
```
POST {instance_url}/api/v1/statuses
Authorization: Bearer {access_token}
Content-Type: application/x-www-form-urlencoded

status=Hello%20World!&visibility=public
```

## Instance-Specific Notes

### Popular Instances

| Instance | Notes |
|----------|-------|
| mastodon.social | Largest instance, general purpose |
| fosstodon.org | Free/open-source software |
| hachyderm.io | Tech community |
| mas.to | General purpose |

### Federation Considerations

- Each instance has its own OAuth credentials
- Users can authenticate with any instance
- Posts are federated across instances
- Rate limits vary by instance

##Rate Limits

Mastodon instances typically have the following limits:

| Endpoint | Default Limit |
|----------|---------------|
| POST /api/v1/statuses | 300 per 30 min |
| GET /api/v1/accounts/:id | 300 per 30 min |

## Troubleshooting

### Common Errors

1. **"invalid_grant"**: Authorization code expired or callback URL mismatch
2. **"access_denied"**: User denied authorization
3. **"invalid_client"**: Incorrect client ID/secret for this instance
4. **401 Unauthorized**: Token revoked or invalid

### Instance-Specific Issues

Some instances have additional restrictions:

- **fosstodon.org**: Requires FLOSS-related content
- **Some instances**: Block certain external media

## Example Configuration

```json
[
  {
    "name": "mastodon.social",
    "client_id": "your_client_id_here",
    "client_secret": "your_client_secret_here",
    "instance_url": "https://mastodon.social"
  }
]
```