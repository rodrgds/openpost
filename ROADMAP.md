# OpenPost Roadmap

> Status: April 2026 — Comprehensive feature analysis and prioritized next steps

---

## Current State Summary

### What Exists

| Area | Status | Notes |
|------|--------|-------|
| **Auth** | Done | JWT-based register/login/logout |
| **Workspaces** | Done | Create/list workspaces, workspace membership |
| **Social Accounts** | Done | Connect X (OAuth1), Mastodon (OAuth2), Bluesky (app password), LinkedIn (OAuth2), Threads (OAuth2) |
| **Post Creation** | Done | Single post + thread creation, media attachments, account selection |
| **Scheduling** | Done | Calendar date picker + time slot selection, schedule overview API |
| **Publishing** | Done | Background worker polls `jobs` table, publishes via platform adapters |
| **Media Upload** | Done | Local filesystem blob storage, multipart upload endpoint |
| **Dashboard** | Basic | Flat table of all posts across workspaces |
| **Accounts Page** | Basic | List/connect/disconnect accounts per workspace |
| **i18n** | Scaffolded | Paraglide set up but only `hello_world` message defined |
| **Mobile** | Scaffolded | Capacitor config exists, Android build shell |
| **Compose Modal** | Done | Full compose-post with calendar, account selection, thread mode |

### Key Gaps (Not Yet Built)

- No post editing, deleting, or re-scheduling
- No media management page (browse, delete, usage tracking)
- No drafting system (all posts require a schedule or "Post Now")
- No social media sets (posts go to all selected accounts with no per-platform customization)
- No randomized posting times
- No per-platform content customization
- No settings page (timezone, week start, posting schedule)
- No prompt library
- No Directus integration
- No API key management
- No AI writing assistance
- No post detail dedicated page (shallow routing)

---

## Prioritized Roadmap

### Phase 1 — Core UX (Highest Priority)

These are the features that make the app usable day-to-day for an individual poster.

---

#### 1.1 Dedicated Post Page (Shallow Routing)

**Priority:** Critical  
**Effort:** Medium  
**Depends on:** None

**What:** Replace the compose-modal-only workflow with a proper dedicated post page at `/posts/new` (and `/posts/{id}` for editing). Use SvelteKit shallow routing so the calendar/sidebar context is preserved when opening from the dashboard, but the post editor is also accessible as a standalone minimal page.

**Why:** The compose modal is cramped inside the sidebar. A dedicated page gives room for per-platform customization, media management, drafting, and sets — all of which need more screen real estate.

**Implementation:**
- `src/routes/posts/new/+page.svelte` — Minimal compose interface: text area, basic publish controls, no calendar
- `src/routes/posts/{id}/+page.svelte` — Edit existing post/draft
- Keep `compose-modal.svelte` for quick-creation from the calendar (shallow route to `/posts/new` using `pushState`)
- Editor should detect if opened via shallow route (show minimal chrome) or directly (show full page layout)

**Backend changes:**  
- `PATCH /posts/{id}` — Update post content, schedule, destinations
- `DELETE /posts/{id}` — Delete a post (only if draft/scheduled, not published)
- `GET /posts/{id}` — Get single post with destinations and media

---

#### 1.2 Drafting System

**Priority:** Critical  
**Effort:** Medium  
**Depends on:** 1.1 (post page)

**What:** Allow creating posts without a scheduled date. Drafts should be first-class citizens visible on the dashboard with a "Draft" status.

**Why:** Users repeatedly emphasized this. Drafts are essential for the writing flow — capture ideas now, schedule later.

**Implementation:**

**Backend changes:**
- `Post.Status` field already supports `"draft"` — this is partially done
- `CreatePostInput` already makes `scheduled_at` optional (omitting it sets status to `"draft"`) — **this works**
- Add `PATCH /posts/{id}` to allow:
  - Converting a draft to scheduled (set `scheduled_at`, change status to `"scheduled"`, create `Job`)
  - Editing draft content
  - Adding/removing destinations and media
- Add `GET /posts?status=draft` filtering
- Create post endpoint should allow explicit `status: "draft"` even with a schedule (for "save as draft but remember the target date")

**Frontend changes:**
- Add "Save as Draft" button on compose page alongside "Schedule" and "Post Now"
- Dashboard should show drafts in a separate section or with a prominent "Drafts" filter
- Draft list view: quick actions to edit, schedule, or delete
- Post detail page shows draft status clearly

---

#### 1.3 Home/Dashboard Redesign

**Priority:** Critical  
**Effort:** Medium  
**Depends on:** 1.2 (drafts visible on dashboard)

**What:** Redesign the home screen from a raw table to a purposeful dashboard.

**Why:** Current dashboard is a flat post table with no visual hierarchy. It doesn't surface drafts, upcoming posts, or quick actions effectively.

**Design goals:**
- **Top section:** Quick-compose CTA ("What's on your mind?") that opens the dedicated post page
- **Upcoming section:** Next 5-7 scheduled posts with time, platform icons, and status
- **Drafts section:** Draft count with "View all drafts" link
- **Calendar:** Keep the existing sidebar calendar (it already works well)
- **Activity feed:** Recent publish results (success/failed)

**Implementation:**
- New `GET /posts?status=scheduled&limit=7&sort=scheduled_at` endpoint for upcoming
- New `GET /posts?status=draft&limit=10` endpoint for drafts
- Redesign `+page.svelte` with card-based layout instead of table
- Add dashboard API that combines counts (drafts, scheduled today, recently published) in one call

---

#### 1.4 Post Edit/Delete/Re-schedule

**Priority:** High  
**Effort:** Medium  
**Depends on:** 1.1

**What:** Ability to edit, delete, and change the schedule of existing posts.

**Why:** Currently posts are create-only. Users need to manage their content lifecycle.

**Implementation:**

**Backend:**
- `PATCH /posts/{id}` — Update content, scheduled_at, destinations, media. If re-scheduling, delete old job and insert new one. If removing schedule, cancel job and set status to draft.
- `DELETE /posts/{id}` — Delete only if status is `draft` or `scheduled`. Cancel any associated job.
- `GET /posts/{id}` — Return full post with destinations and media

**Frontend:**
- Post detail/edit page at `/posts/{id}`
- Quick actions on dashboard cards: edit, delete, re-schedule
- Confirmation dialog for destructive actions

---

### Phase 2 — Platform Customization & Social Media Sets

These are the features that differentiate OpenPost from basic schedulers.

---

#### 2.1 Per-Platform Content Customization

**Priority:** High  
**Effort:** Large  
**Depends on:** 1.1 (post page redesign)

**What:** When composing a post targeted to multiple platforms, allow users to unsync from the default content and customize per platform. Example: a LinkedIn version might be longer, a Twitter version shorter, and Instagram might have different media prioritization.

**Why:** This was explicitly called out as a key priority. Posts shouldn't be identical across platforms.

**Data model changes:**

```go
// NEW: PostVariant — per-platform content override
type PostVariant struct {
    bun.BaseModel `bun:"table:post_variants"`
    
    ID             string `bun:",pk" json:"id"`
    PostID         string `bun:",notnull" json:"post_id"`
    SocialAccountID string `bun:",notnull" json:"social_account_id"`
    Content        string `bun:",notnull" json:"content"`         // Override content (empty = use parent)
    MediaIDs       string `bun:",nullzero" json:"media_ids"`       // JSON array of media IDs override
    IsUnsynced     bool   `bun:",default:false" json:"is_unsynced"` // Whether this variant has diverged from parent
    CreatedAt      time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
    UpdatedAt      time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
}
```

**Implementation:**
- `post_variants` table in `database.go`
- When creating a post with destinations, also create default `PostVariant` entries (one per destination) with `is_unsynced = false`
- On the compose/edit page, show the "default" content at the top, then expandable per-platform sections
- When `is_unsynced = false`, the variant mirrors the parent post's content
- When the user edits a specific variant, mark `is_unsynced = true` and persist custom content
- Publisher reads `PostVariant.Content` if `is_unsynced`, otherwise falls back to `Post.Content`
- Per-variant media management (reorder, add/remove per platform)

**API:**
- `PUT /posts/{id}/variants` — Upsert variants for a post
- `GET /posts/{id}?include=variants` — Include variants in post response

**Frontend:**
- Compose page: "Customize per platform" section showing each destination
- Unsync button per platform (creates an editable copy)
- Character counter per platform (Twitter: 280, LinkedIn: 3000, etc.)
- Per-platform media preview and reorder

---

#### 2.2 Social Media Sets

**Priority:** High  
**Effort:** Medium  
**Depends on:** None (can be built independently)

**What:** Within a workspace, define named sets of accounts (e.g., "Tech Twitter" = Bluesky + Twitter, "Professional" = LinkedIn + Threads). Posts default to a set, but users can override.

**Why:** Users with many accounts don't want to select accounts every time. Sets provide sensible defaults and reduce cognitive load.

**Data model changes:**

```go
type SocialMediaSet struct {
    bun.BaseModel `bun:"table:social_media_sets"`

    ID          string `bun:",pk" json:"id"`
    WorkspaceID string `bun:",notnull" json:"workspace_id"`
    Name        string `bun:",notnull" json:"name"`
    IsDefault   bool   `bun:",default:false" json:"is_default"` // One default set per workspace
    CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

type SocialMediaSetAccount struct {
    bun.BaseModel `bun:"table:social_media_set_accounts"`

    SetID           string `bun:",pk" json:"set_id"`
    SocialAccountID string `bun:",pk" json:"social_account_id"`
    IsMain          bool   `bun:",default:false" json:"is_main"` // "Main" platform in this set
}
```

**Implementation:**
- CRUD API for sets: `POST /sets`, `GET /sets`, `PATCH /sets/{id}`, `DELETE /sets/{id}`
- Add/remove accounts from sets: `POST /sets/{id}/accounts`, `DELETE /sets/{id}/accounts/{account_id}`
- Mark a "main" platform per set (used as the primary view in per-platform customization)
- Compose page defaults to the workspace's default set
- Users can switch sets or manually override account selection

**Frontend:**
- Sets management in accounts page or new settings section
- Set selector on compose page (dropdown replacing "select all/clear all")
- Visual indicator of which set is active

---

#### 2.3 Accounts Page Redesign

**Priority:** High  
**Effort:** Medium  
**Depends on:** 2.2 (sets)

**What:** Redesign the accounts page to support social media sets, better visual hierarchy, and account health status.

**Why:** Current page is a flat list with no organization. It needs to support sets and provide clearer account management.

**Design:**
- Group accounts by set (with drag-to-assign)
- Show account health (last successful publish, token expiry, error state)
- Inline "set as main" toggle per account within a set
- Better visual design for connect/disconnect actions
- Platform icons per account in the set view

---

### Phase 3 — Media Management & Cleanup

---

#### 3.1 Media Library Page

**Priority:** Medium  
**Effort:** Medium  
**Depends on:** None

**What:** A dedicated `/media` page showing all uploaded media for a workspace with filtering, deletion, and usage status.

**Why:** Users need to manage their media assets — see what's uploaded, what's used, and clean up unused files.

**Backend changes:**
- `GET /media?workspace_id={id}` — List all media for a workspace with usage info
- `GET /media/{id}/usage` — Return which posts reference this media
- `DELETE /media/{id}` — Delete media (only if not attached to any post)
- Add `is_favorite` field to `MediaAttachment` model

**Data model changes:**

```go
type MediaAttachment struct {
    // ... existing fields ...
    IsFavorite bool      `bun:",default:false" json:"is_favorite"`
    CreatedAt  time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}
```

**Frontend:**
- New route: `/media/+page.svelte`
- Grid view of media thumbnails with:
  - Usage indicator: "Used in 2 posts" / "Unused" / "Attached to scheduled post"
  - Favorite toggle (heart/star)
  - Delete button (disabled if attached to scheduled/published posts with tooltip explaining why)
  - Filter: All / Used / Unused / Favorites
  - Sort: Newest / Oldest / Size
- Upload new media directly from the library page

---

#### 3.2 Automated Media Cleanup

**Priority:** Low  
**Effort:** Small  
**Depends on:** 3.1 (media page, `is_favorite` field)

**What:** Workspace setting to automatically delete unused, non-favorited media after a configurable time period (default: 7 days).

**Implementation:**
- Add `media_cleanup_days` field to `Workspace` model (0 = disabled, default: 0)
- Add a new job type `"media_cleanup"` that runs periodically (configurable interval)
- Background worker picks up the job, queries `MediaAttachment` where:
  - No `PostMedia` records reference it (unused)
  - `is_favorite = false`
  - `created_at < NOW() - INTERVAL cleanup_days DAYS`
- Deletes the media file from storage and the database record
- Add workspace settings UI for this toggle

---

### Phase 4 — Scheduling Intelligence

---

#### 4.1 Randomized Posting Times

**Priority:** Medium  
**Effort:** Small  
**Depends on:** None

**What:** When scheduling a post, optionally add a random delay of ±N minutes (configurable, e.g., ±5 to ±15 minutes) to make posts feel more natural.

**Implementation:**
- Add `random_delay_minutes` field to `Post` model (or to `Workspace`/`SocialMediaSet` as a default)
- When creating a scheduled post, if `random_delay_minutes > 0`, set `Job.RunAt` to `ScheduledAt + random(-N, +N) minutes`
- Store both the user-intended time and the actual scheduled time for display purposes
- UI: Toggle on compose page "Add random delay" with configurable range

**Data model changes:**

```go
type Post struct {
    // ... existing fields ...
    RandomDelayMinutes int       `bun:",default:0" json:"random_delay_minutes"`
    ActualRunAt        time.Time `json:"actual_run_at"` // Set by worker, differs from ScheduledAt if randomized
}
```

---

#### 4.2 Posting Schedule (Time Slots)

**Priority:** Medium  
**Effort:** Medium  
**Depends on:** 4.1 (or can be built independently)

**What:** Define preferred time slots per workspace or set (e.g., "9:00, 12:00, 18:00 on weekdays"). When composing, suggest these slots. Allow "schedule in next available slot" one-click.

**Implementation:**
- Add `PostingSchedule` model with day-of-week + time slots
- Settings page: weekly calendar with toggleable time slots
- Compose page: "Suggest time" button that picks the next available slot
- Can be combined with randomized delay for natural timing

**Data model:**

```go
type PostingSchedule struct {
    bun.BaseModel `bun:"table:posting_schedules"`

    ID          string `bun:",pk" json:"id"`
    WorkspaceID string `bun:",notnull" json:"workspace_id"`
    SetID       string `json:"set_id"` // Optional: per-set schedules
    DayOfWeek   int    `bun:",notnull" json:"day_of_week"` // 0=Sunday, 6=Saturday
    Hour        int    `bun:",notnull" json:"hour"`        // 0-23
    Minute      int    `bun:",notnull" json:"minute"`       // 0-59
}
```

---

### Phase 5 — Settings & User Preferences

---

#### 5.1 Settings Page

**Priority:** Medium  
**Effort:** Medium  
**Depends on:** Multiple

**What:** A `/settings` page with user and workspace preferences.

**Sections:**
- **Profile** — Email, password change
- **Workspace Settings** — Default timezone, week start (Mon/Sun), media cleanup
- **Posting Schedule** — Time slots management
- **Connected Accounts** — Manage sets (link to accounts page)
- **Danger Zone** — Delete workspace, leave workspace

**Backend:**
- `GET /settings` — Return user preferences + workspace settings
- `PATCH /settings` — Update preferences
- Add `timezone` and `week_start` fields to `Workspace` model

**Data model changes:**

```go
type Workspace struct {
    // ... existing fields ...
    Timezone           string `bun:",default:'UTC'" json:"timezone"`
    WeekStart          int    `bun:",default:1" json:"week_start"` // 0=Sunday, 1=Monday
    MediaCleanupDays   int    `bun:",default:0" json:"media_cleanup_days"` // 0 = disabled
}
```

---

### Phase 6 — Content Features

---

#### 6.1 Prompt Library

**Priority:** Medium  
**Effort:** Medium  
**Depends on:** 1.1 (compose page)

**What:** A `/prompts` page with a curated library of writing prompts, plus integration into the compose page for quick inspiration.

**Implementation:**

**Backend:**
- Seed prompts stored as JSON or in the database (start with a static file, move to DB later)
- `GET /prompts?category={cat}` — List prompts with optional category filter
- `POST /prompts` — Create custom prompt (user-specific)
- Categories: "Tools & Workflow", "Reflection", "Announcement", "Engagement", "Tutorial", etc.

**Data model:**

```go
type Prompt struct {
    bun.BaseModel `bun:"table:prompts"`

    ID         string `bun:",pk" json:"id"`
    WorkspaceID string `json:"workspace_id"` // null = global prompt
    UserID     string `json:"user_id"`        // null = workspace/global prompt
    Text       string `bun:",notnull" json:"text"`
    Category   string `bun:",notnull" json:"category"`
    IsBuiltIn  bool   `bun:",default:false" json:"is_built_in"`
    CreatedAt  time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}
```

**Frontend:**
- `/prompts` — Browse prompts by category, search, create custom
- Compose page: "Need inspiration?" button that opens a prompt picker
- Dashboard: Weekly prompt suggestion card

---

#### 6.2 AI Writing Assistance (OpenRouter)

**Priority:** Low (future)  
**Effort:** Large  
**Depends on:** 1.1, writing style config

**What:** Optional AI integration for writing assistance — suggest rewrites, expand ideas, adjust tone, generate drafts from prompts.

**Implementation:**
- Workspace-level OpenRouter API key configuration (encrypted at rest)
- User-defined "writing style" profile (tone, voice, length preferences, examples)
- `POST /ai/suggest` — Takes content + context + style profile, returns suggestions
- `POST /ai/generate` — Takes a prompt, generates a draft post
- Compose page: "AI Assist" button that shows inline suggestions
- All AI features are opt-in and require user-provided API key

**Data model:**

```go
type WritingStyle struct {
    bun.BaseModel `bun:"table:writing_styles"`

    ID          string `bun:",pk" json:"id"`
    WorkspaceID string `bun:",notnull" json:"workspace_id"`
    Name        string `bun:",notnull" json:"name"`
    Description string `json:"description"`
    Tone        string `json:"tone"`         // e.g., "casual", "professional", "witty"
    SamplePosts string `json:"sample_posts"` // JSON array of example posts
    IsDefault   bool   `bun:",default:false" json:"is_default"`
    CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}
```

---

### Phase 7 — Integrations

---

#### 7.1 Directus Integration

**Priority:** Low  
**Effort:** Large  
**Depends on:** Settings page for configuration

**What:** Allow users to configure a Directus CMS connection to automatically push published posts and media as items.

**Implementation:**
- Workspace settings: Directus server URL, API token, collection name
- Template mapping: define how OpenPost fields map to Directus fields
- Background sync: after successful publish, push post data to Directus via REST API
- Media upload: send media to Directus as files

**Data model:**

```go
type DirectusConfig struct {
    bun.BaseModel `bun:"table:directus_configs"`

    ID          string `bun:",pk" json:"id"`
    WorkspaceID string `bun:",notnull" json:"workspace_id"`
    ServerURL   string `bun:",notnull" json:"server_url"`
    APIToken    string `bun:",notnull" json:"-"` // encrypted
    Collection  string `bun:",notnull" json:"collection"` // target Directus collection
    FieldMap    string `json:"field_map"` // JSON mapping: {"content": "body", "media": "image", ...}
    AutoSync    bool   `bun:",default:false" json:"auto_sync"`
}
```

---

#### 7.2 API Key Management

**Priority:** Low  
**Effort:** Medium  
**Depends on:** Settings page

**What:** Generate API keys for programmatic access to OpenPost (create posts, upload media, check status).

**Implementation:**
- `POST /api-keys` — Generate a new API key (prefix + secret, hash stored in DB)
- `GET /api-keys` — List keys (show last 4 chars only)
- `DELETE /api-keys/{id}` — Revoke a key
- New auth middleware for API key auth (in addition to JWT)
- Rate limiting per key

**Data model:**

```go
type APIKey struct {
    bun.BaseModel `bun:"table:api_keys"`

    ID          string `bun:",pk" json:"id"`
    WorkspaceID string `bun:",notnull" json:"workspace_id"`
    Name        string `bun:",notnull" json:"name"`
    KeyPrefix   string `bun:",notnull" json:"key_prefix"` // First 8 chars for identification
    KeyHash     string `bun:",notnull" json:"-"` // SHA-256 hash of full key
    LastUsedAt  time.Time `json:"last_used_at"`
    CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
    ExpiresAt   time.Time `json:"expires_at"`
}
```

---

#### 7.3 MCP Server

**Priority:** Very Low (nice to have)  
**Effort:** Large  
**Depends on:** API key system (7.2)

**What:** Model Context Protocol server allowing AI agents to interact with OpenPost.

**Implementation:**
- New Go package `internal/mcp` implementing the MCP protocol
- Expose tools: `create_post`, `list_posts`, `get_post`, `upload_media`, `list_accounts`
- SSE or stdio transport
- Authentication via API keys
- Scope keys to specific workspaces and permissions

---

### Phase 8 — Polish & Mobile

---

#### 8.1 App Logo & Icon

**Priority:** Medium  
**Effort:** Small  
**Depends on:** None

**What:** Update the app logo and fix the missing Android app icon.

**Implementation:**
- Design/update SVG logo component (`Logo.svelte`)
- Generate all required Capacitor icon sizes from the source
- Update `favicon.svg`, Apple touch icon, PWA manifest icons
- Test on Android emulator to verify icon appears

---

#### 8.2 Codebase Cleanup

**Priority:** Medium  
**Effort:** Ongoing  
**Depends on:** None

**What:** General code quality improvements throughout the codebase.

**Areas to address:**
- Remove unused i18n scaffold or properly implement it
- Consistent error handling patterns in frontend (toast notifications vs inline errors)
- Add proper loading states and skeleton screens
- Type-safe API client regeneration workflow
- Backend: Consistent response shapes, proper HTTP status codes
- Frontend: Extract shared state management (workspaces, accounts) into proper stores
- Add Vitest tests for frontend components
- Add Go tests for handlers and services
- Fix any `any` type casts in frontend (e.g., in compose-post, accounts pages)

---

## Database Migration Strategy

Since `CreateSchema` uses `.IfNotExists()`, adding new tables is straightforward. For adding columns to existing tables, a migration system needs to be introduced:

1. Create a `migrations` table to track applied migrations
2. Write migration functions that execute `ALTER TABLE` statements
3. Run migrations on startup after `CreateSchema`
4. Each new feature (sets, variants, schedules, etc.) should include its migration

**Suggested migration structure:**

```go
type Migration struct {
    bun.BaseModel `bun:"table:migrations"`
    ID            string    `bun:",pk" json:"id"`
    AppliedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"applied_at"`
}

// In database.go, after CreateSchema:
func RunMigrations(db *bun.DB) error {
    // Apply each migration in order, tracking in migrations table
}
```

---

## Feature Dependency Graph

```
Phase 1 (Core UX)
├── 1.1 Post Page (shallow routing) ←── foundation for everything
├── 1.2 Drafting System ←── builds on 1.1
├── 1.3 Dashboard Redesign ←── needs 1.2 for drafts section
└── 1.4 Post Edit/Delete ←── builds on 1.1

Phase 2 (Platform Power)
├── 2.1 Per-Platform Customization ←── needs 1.1 for compose page redesign
├── 2.2 Social Media Sets ←── independent
└── 2.3 Accounts Redesign ←── needs 2.2 for sets UI

Phase 3 (Media)
├── 3.1 Media Library ←── independent
└── 3.2 Auto Cleanup ←── needs 3.1

Phase 4 (Smart Scheduling)
├── 4.1 Randomized Times ←── independent
└── 4.2 Time Slot Schedules ←── can use 2.2 sets

Phase 5 (Settings)
└── 5.1 Settings Page ←── aggregator for workspace prefs

Phase 6 (Content)
├── 6.1 Prompt Library ←── needs 1.1
└── 6.2 AI Assistance ←── future, needs config infrastructure

Phase 7 (Integrations)
├── 7.1 Directus ←── needs settings page
├── 7.2 API Keys ←── needs settings page
└── 7.3 MCP Server ←── needs 7.2

Phase 8 (Polish)
├── 8.1 App Icon ←── independent
└── 8.2 Code Cleanup ←── ongoing
```

---

## Suggested Implementation Order

Within the phases, here's the recommended order considering dependencies and impact:

| # | Feature | Phase | Sprint Estimate |
|---|---------|-------|-----------------|
| 1 | Post edit/delete/re-schedule (PATCH/DELETE APIs) | 1.4 | 3 days |
| 2 | Drafting system (backend already supports, needs UI) | 1.2 | 2 days |
| 3 | Dedicated post page + shallow routing | 1.1 | 3 days |
| 4 | Dashboard redesign | 1.3 | 3 days |
| 5 | Social media sets (data model + API) | 2.2 | 3 days |
| 6 | Per-platform content customization | 2.1 | 5 days |
| 7 | Accounts page redesign | 2.3 | 3 days |
| 8 | Media library page | 3.1 | 3 days |
| 9 | Randomized posting times | 4.1 | 1 day |
| 10 | App logo & icon update | 8.1 | 1 day |
| 11 | Prompt library | 6.1 | 3 days |
| 12 | Settings page (timezone, week start, cleanup) | 5.1 | 3 days |
| 13 | Media auto-cleanup | 3.2 | 2 days |
| 14 | Time slot schedules | 4.2 | 3 days |
| 15 | API key management | 7.2 | 3 days |
| 16 | Directus integration | 7.1 | 5 days |
| 17 | AI writing assistance | 6.2 | 5+ days |
| 18 | MCP server | 7.3 | 5+ days |
| 19 | Codebase cleanup | 8.2 | ongoing |

**Recommended first sprint (2 weeks):** Items 1-4 — Core UX improvements that make the app usable for real daily scheduling.

**Recommended second sprint (2 weeks):** Items 5-7 — Platform customization power features.

**Recommended third sprint (1-2 weeks):** Items 8-11 — Media management and content features.

---

## Low-Priority / Future Features

These were mentioned but should be deprioritized:

| Feature | Notes |
|---------|-------|
| **Auto-retweet / auto-plug** | Needs per-platform scheduling rules, potentially a separate job type |
| **Thread finishers** | Pre-defined endings for threads, related to prompt/templating system |
| **Timezone auto-detection** | Nice-to-have, start with manual selection in settings |
| **AI writing style (OpenRouter)** | Powerful but complex; start with prompt library first |
| **MCP server** | Requires API key system first; low user demand |

---

## Technical Architecture Notes

### Current Pain Points
1. **No migration system** — Only `IfNotExists` for table creation; no column additions
2. **Frontend state scattered** — Workspace/account state fetched in every component instead of shared stores
3. **No error boundary** — Frontend errors crash components silently
4. **Type safety gaps** — Several `as any` casts in API client usage
5. **No pagination** — `list-posts` returns max 50 with no pagination controls

### Recommended Technical Improvements
1. Add a proper database migration system (see Migration Strategy above)
2. Create Svelte stores for `workspaces`, `accounts`, `posts` with cache invalidation
3. Add toast/notification system for success/error feedback
4. Regenerate `types.d.ts` from OpenAPI spec in the build pipeline
5. Add pagination to all list endpoints (cursor-based for posts, offset for media)
6. Add workspace-scoped middleware to reduce boilerplate in handlers
7. Consider adding soft-delete (e.g., `deleted_at` column) for posts and media instead of hard deletes