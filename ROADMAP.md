# OpenPost Roadmap

> Status: May 2026 — Comprehensive feature analysis and prioritized next steps

OpenPost is a lightweight, self-hosted social media scheduler. This roadmap outlines the current state and future directions for the project.

---

## Current State Summary

### ✅ What is Done

| Area | Status | Notes |
|------|--------|-------|
| **Auth** | Done | JWT-based register/login/logout |
| **Workspaces** | Done | Multi-tenant workspaces with timezone and cleanup settings |
| **Social Accounts** | Done | X, Mastodon, Bluesky, LinkedIn, Threads |
| **Post Creation** | Done | Unified composer for single posts and threads |
| **Platform Variants**| Done | Per-platform content overrides (Content variants) |
| **Scheduling** | Done | Calendar + Time slots + Randomized posting times (±N min) |
| **Publishing** | Done | Sequential thread publishing with parent tracking |
| **Media Library** | Done | Workspace-scoped media management with cleanup |
| **i18n** | Done | English, Spanish, and Portuguese support via Paraglide |
| **Mobile** | Done | Capacitor Android app support |

---

## 🚀 Future Roadmap

### Phase 1 — Platform Power & Customization

#### 1.1 Per-Platform Media Overrides
**Priority:** High
Allow selecting different media for different platforms within the same post.
- **Backend:** Update `publisher.go` to respect `PostVariant.MediaIDs` if present.
- **Frontend:** Add media selection to the "Customize per platform" view in `compose-simple.svelte`.

#### 1.2 Enhanced Thread Management
**Priority:** Medium  
Improve editing of existing threads.
- **Backend:** Update `PATCH /posts/{id}` to handle atomic thread updates (updating content of all posts in a chain).
- **Frontend:** Better visualization of long threads in the composer.

---

### Phase 2 — Integrations & Extensibility

#### 2.1 API Key Management
**Priority:** High
Generate scoped API keys for programmatic access.
- **Backend:** `api_keys` table + middleware for `X-API-Key` auth.
- **Frontend:** API Key management UI in Settings.

#### 2.2 Directus Integration
**Priority:** Medium
Two-way sync with Directus CMS.
- **Backend:** Sync published posts and media to a Directus collection.
- **Frontend:** Integration settings for Directus URL and tokens.

#### 2.3 MCP Server
**Priority:** Medium
Model Context Protocol server to allow AI agents (Claude, Cursor, etc.) to schedule posts.
- Implement official Go MCP SDK.
- Expose tools for listing accounts, scheduling posts, and uploading media.

---

### Phase 3 — Content Intelligence

#### 3.1 AI Writing Assistance (Genkit)
**Priority:** Medium
Integrated AI for rewrites, tone adjustment, and brainstorming.
- Use Genkit (Firebase/Google) for typed flows.
- Support Gemini and OpenAI providers.
- Structured output for post suggestions.

#### 3.2 Analytics & Engagement
**Priority:** Low
Track post performance (likes, reposts, clicks) across platforms.
- Background worker to poll for engagement metrics.
- Dashboard charts for growth and reach.

---

### Phase 4 — Security & Operations

#### 4.1 2FA (TOTP)
**Priority:** Medium
Add Two-Factor Authentication for user accounts.

#### 4.2 Active Session Management
**Priority:** Low
View and revoke active login sessions.

---

## Technical Debt & Polish

- **Test Coverage:** Increase backend test coverage to >80% for critical paths.
- **Error Handling:** More robust error recovery in the background worker.
- **Performance:** Add pagination to all list endpoints.
- **UI/UX:** Continuous polish of the Svelte 5 composer and dashboard.
