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
- `GET /media?workspace_id={id}` — List all media for a workspace with usage info (pagination support)
- `GET /media/{id}/usage` — Return which posts reference this media
- `DELETE /media/{id}` — Delete media (only if not attached to any post)
- `PATCH /media/{id}` — Update alt text, favorite status
- `POST /media/batch-delete` — Delete multiple unused media at once
- `POST /media/upload` — Accept batch upload (multiple files)

**Data model changes:**

```go
type MediaAttachment struct {
    bun.BaseModel `bun:"table:media_attachments"`

    ID               string    `bun:",pk" json:"id"`
    WorkspaceID      string    `bun:",notnull" json:"workspace_id"`
    FilePath         string    `bun:",notnull" json:"file_path"`
    StorageType      string    `bun:",default:'local'" json:"storage_type"` // 'local', 's3'

    // Original file info
    MimeType         string    `json:"mime_type"`
    Size             int64     `json:"size"`
    OriginalFilename string    `json:"original_filename"`

    // Image-specific metadata
    Width            int       `json:"width"`
    Height           int       `json:"height"`

    // Thumbnail paths (JSON: {"sm": "sm_xxx.jpg", "md": "md_xxx.jpg"})
    ThumbnailsJSON   string    `bun:"thumbnails" json:"thumbnails"`

    // Deduplication
    FileHash         string    `bun:",unique" json:"-"` // SHA-256 for detecting duplicates

    // User-facing metadata
    AltText          string    `json:"alt_text"`
    IsFavorite       bool      `bun:",default:false" json:"is_favorite"`

    // Usage tracking (denormalized for quick filtering)
    UsageCount       int       `bun:",default:0" json:"usage_count"`
    LastUsedAt       time.Time `json:"last_used_at"`

    CreatedAt        time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

// MediaUsage tracks which posts use which media (replaces simple PostMedia join for richer queries)
type MediaUsage struct {
    bun.BaseModel `bun:"table:media_usage"`

    MediaID    string    `bun:",pk" json:"media_id"`
    PostID     string    `bun:",pk" json:"post_id"`
    VariantID  string    `json:"variant_id"` // If used in a platform variant
    UsedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"used_at"`
}
```

**Thumbnail generation:**
- On upload, generate small (150px) and medium (400px) thumbnails using `github.com/disintegration/imaging`
- Store thumbnails alongside originals with size prefix (`sm_`, `md_`)
- Return thumbnail URLs in API responses for grid views
- Use original file for compose preview and publishing

**Deduplication:**
- Compute SHA-256 hash on upload before storing
- If hash matches existing media in same workspace, return existing media ID instead
- Prevents storage bloat from repeated uploads

**Batch operations:**
- Batch delete: `POST /media/batch-delete` with array of IDs, onlyDeletes unused ones
- Batch upload: multipart form with multiple files, returns array of media objects

**Frontend:**
- New route: `/media/+page.svelte`
- Grid view of media thumbnails (use the `sm` thumbnail for grid, `md` for preview)
- Usage indicator: "Used in 2 posts" / "Unused" / "Attached to scheduled post"
- Favorite toggle (heart/star)
- Delete button (disabled if attached to scheduled/published posts with tooltip explaining why)
- Batch selection with toolbar actions (delete selected, favorite selected)
- Filter: All / Used / Unused / Favorites
- Sort: Newest / Oldest / Size
- Upload new media directly from the library page (drag-and-drop zone)

**Libraries:**

| Library | Purpose |
|---------|---------|
| `github.com/disintegration/imaging` | Thumbnail generation, image resizing |
| `crypto/sha256` (stdlib) | File deduplication hashing |

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
- Also deletes any orphaned thumbnail files
- Add workspace settings UI for this toggle

**Important consideration:** The cleanup job should also remove thumbnail files (`sm_*`, `md_*` prefixes) when deleting media, not just the original.

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

**Edge case:** If the randomized time pushes the post to the next day, the calendar should still show the user-intended time, not the actual run time.

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

    // Store times in UTC for consistency, convert on read using workspace timezone
    UTCHour    int    `bun:",notnull" json:"utc_hour"`    // 0-23 UTC
    UTCMinute  int    `bun:",notnull" json:"utc_minute"`  // 0-59 UTC
    DayOfWeek  int    `bun:",notnull" json:"day_of_week"` // 0=Sunday, 6=Saturday (in UTC)

    // Display/helpers
    Label      string `json:"label"`     // e.g., "Morning", "Lunch", "Evening"
    IsActive   bool   `bun:",default:true" json:"is_active"`

    CreatedAt  time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}
```

**Important:** Times must be stored in UTC and converted to/from the workspace's timezone on read. This prevents bugs when users change their timezone or when daylight saving time shifts occur. The `Workspace.Timezone` field (from Phase 5) is used for all display conversions.

**"Next available slot" logic:**
1. Get current time in workspace timezone
2. Look up active schedule entries for the current day-of-week
3. Find the first slot where `UTCHour:UTCMinute` is after `now`
4. If no slots remain today, check the next day, wrapping around the week
5. Apply `RandomDelayMinutes` if configured

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
- **Security** — Two-factor authentication (TOTP), active sessions, login history
- **Workspace Settings** — Default timezone, week start (Mon/Sun), media cleanup
- **Posting Schedule** — Time slots management
- **Connected Accounts** — Manage sets (link to accounts page)
- **Integrations** — Directus config, AI config (keys for later phases)
- **Danger Zone** — Delete workspace, leave workspace, export data

**Backend:**
- `GET /settings` — Return user preferences + workspace settings
- `PATCH /settings` — Update preferences
- `POST /settings/2fa/enable` — Enable TOTP 2FA (returns QR code + backup codes)
- `POST /settings/2fa/verify` — Verify 2FA setup
- `POST /settings/2fa/disable` — Disable 2FA
- `GET /settings/sessions` — List active sessions
- `DELETE /settings/sessions/{id}` — Revoke a session
- Add `timezone` and `week_start` fields to `Workspace` model

**Data model changes:**

```go
type Workspace struct {
    // ... existing fields ...
    Timezone           string `bun:",default:'UTC'" json:"timezone"`
    WeekStart          int    `bun:",default:1" json:"week_start"` // 0=Sunday, 1=Monday
    MediaCleanupDays   int    `bun:",default:0" json:"media_cleanup_days"` // 0 = disabled
}

// NEW: Two-factor authentication
type User2FA struct {
    bun.BaseModel `bun:"table:user_2fa"`

    ID           string    `bun:",pk" json:"id"`
    UserID       string    `bun:",unique,notnull" json:"user_id"`
    Secret       string    `bun:",notnull" json:"-"` // TOTP secret (encrypted)
    BackupCodes  string    `bun:",notnull" json:"-"` // JSON array of hashed backup codes
    Enabled      bool      `bun:",default:false" json:"enabled"`
    VerifiedAt   time.Time `json:"verified_at"`
    CreatedAt    time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

// NEW: Session tracking
type UserSession struct {
    bun.BaseModel `bun:"table:user_sessions"`

    ID           string    `bun:",pk" json:"id"`
    UserID       string    `bun:",notnull" json:"user_id"`
    TokenHash    string    `bun:",notnull" json:"-"` // Hashed refresh token
    DeviceInfo   string    `json:"device_info"`     // User-Agent parsed
    IPAddress    string    `json:"ip_address"`
    LastActiveAt time.Time `json:"last_active_at"`
    ExpiresAt    time.Time `bun:",notnull" json:"expires_at"`
    CreatedAt    time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}
```

**2FA Library:**

| Library | Purpose |
|---------|---------|
| `github.com/pquerna/otp` | TOTP/HOTP implementation for 2FA |

---

### Phase 6 — Content Features

---

#### 6.1 Prompt Library & Content Templates

**Priority:** Medium  
**Effort:** Medium  
**Depends on:** 1.1 (compose page)

**What:** A `/prompts` page with a curated library of writing prompts, plus content templates with variable substitution for reuse. Also, integration into the compose page for quick inspiration.

**Implementation:**

**Backend:**
- Seed prompts stored as JSON or in the database (start with a static file, move to DB later)
- `GET /prompts?category={cat}` — List prompts with optional category filter
- `POST /prompts` — Create custom prompt (user-specific)
- `GET /templates` — List content templates
- `POST /templates` — Create template with variable placeholders
- `POST /templates/{id}/render` — Render a template with given variables
- Categories: "Tools & Workflow", "Reflection", "Announcement", "Engagement", "Tutorial", etc.

**Data models:**

```go
type Prompt struct {
    bun.BaseModel `bun:"table:prompts"`

    ID          string    `bun:",pk" json:"id"`
    WorkspaceID string    `json:"workspace_id"` // null = global prompt
    UserID      string    `json:"user_id"`      // null = workspace/global prompt
    Text        string    `bun:",notnull" json:"text"`
    Category    string    `bun:",notnull" json:"category"`
    IsBuiltIn   bool      `bun:",default:false" json:"is_built_in"`
    CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

// NEW: Content templates with variable substitution
type ContentTemplate struct {
    bun.BaseModel `bun:"table:content_templates"`

    ID           string `bun:",pk" json:"id"`
    WorkspaceID  string `bun:",notnull" json:"workspace_id"`
    UserID       string `bun:",notnull" json:"user_id"`
    Name         string `bun:",notnull" json:"name"`
    Category     string `json:"category"` // "thread", "announcement", "cta", "engagement"
    Content      string `bun:",notnull" json:"content"` // Template text with {{variable}} placeholders
    VariablesJSON string `bun:"variables" json:"variables"` // ["product_name", "link", "cta_text"]
    IsShared     bool   `bun:",default:false" json:"is_shared"` // Share with workspace
    UsageCount   int    `bun:",default:0" json:"usage_count"`
    CreatedAt    time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}
```

**Frontend:**
- `/prompts` — Browse prompts by category, search, create custom
- `/templates` — Browse and manage content templates
- Compose page: "Need inspiration?" button that opens a prompt picker
- Compose page: "Use template" button that opens template picker with variable filling
- Dashboard: Weekly prompt suggestion card

---

#### 6.2 AI Writing Assistance (Genkit)

**Priority:** Low (future)  
**Effort:** Medium  
**Depends on:** 1.1, writing style config, settings page (for API key)

**What:** Optional AI integration for writing assistance — suggest rewrites, expand ideas, adjust tone, generate drafts from prompts. Uses [Genkit](https://genkit.dev/) (by Firebase/Google) for a type-safe, observable AI framework with structured output, streaming, Dotprompt templates, and multi-provider support.

**Why Genkit over a raw API SDK:** Genkit provides more than just a unified API — it gives us **typed flows** (type-safe in/out with Go generics), **structured output** (generate Go structs directly from AI), **built-in observability** (tracing, usage tracking per flow), **Dotprompt** (prompt template files with versioning), a **local Developer UI** for testing, and **tool calling** (letting AI interact with OpenPost data). This means less boilerplate, built-in budget tracking via flow tracing, and a much better developer experience.

**Libraries:**

| Library | Purpose |
|---------|---------|
| `github.com/firebase/genkit/go` | Genkit core — flows, structured output, streaming, observability |
| `github.com/firebase/genkit/go/plugins/googlegenai` | Google AI / Gemini provider (generous free tier, no credit card needed) |
| `github.com/firebase/genkit/go/plugins/openai` | OpenAI provider |
| `github.com/firebase/genkit/go/plugins/anthropic` | Anthropic provider (can be added when available) |
| Dotprompt (built-in) | Prompt template management with `.prompt` files |

**Key Genkit advantages for OpenPost:**
- **Flows** wrap AI calls in typed, observable functions — perfect for exposing as API endpoints
- **`GenerateData[T]`** with Go generics produces typed structs directly (PostSuggestion, ToneAdjustment, etc.)
- **Streaming flows** work over SSE for real-time compose-page suggestions
- **Dotprompt** files let us version and iterate on prompts without code changes
- **Developer UI** (`genkit start`) lets us test AI flows locally before deploying
- **Built-in tracing** gives us token counts, latency, and cost per flow invocation — no manual usage tracking needed
- **Tool calling** lets AI flows call OpenPost APIs (e.g., "list my drafts" → AI calls the posts tool)

**Implementation:**

**Backend — Genkit flows:**

```go
// internal/ai/flows.go
package ai

import (
    "context"
    "fmt"

    "github.com/firebase/genkit/go/ai"
    "github.com/firebase/genkit/go/genkit"
    "github.com/firebase/genkit/go/plugins/googlegenai"
)

// Typed input/output schemas using Go structs with jsonschema tags
type SuggestInput struct {
    Content       string `json:"content" jsonschema:"description=Current post content to improve"`
    StyleID       string `json:"styleId,omitempty" jsonschema:"description=Writing style ID to apply"`
    PlatformHint  string `json:"platformHint,omitempty" jsonschema:"description=Target platform for tone adjustment"`
}

type SuggestOutput struct {
    Suggestions []Suggestion `json:"suggestions"`
}

type Suggestion struct {
    Type       string `json:"type"`        // "rewrite", "expand", "shorten", "tone_adjust"
    Content    string `json:"content"`     // The suggested text
    Reasoning  string `json:"reasoning"`   // Why this suggestion was made
    Confidence float64 `json:"confidence"` // 0.0-1.0
}

type GenerateInput struct {
    Prompt       string `json:"prompt" jsonschema:"description=Topic or idea to generate a post about"`
    StyleID      string `json:"styleId,omitempty" jsonschema:"description=Writing style ID"`
    PlatformHint string `json:"platformHint,omitempty" jsonschema:"description=Target platform"`
}

type GenerateOutput struct {
    Content string `json:"content"`
    Title   string `json:"title"`
}

// Initialize Genkit and register flows
func Setup(g *genkit.Genkit) {
    // Register flows — these become observable, typed API endpoints
    genkit.DefineFlow(g, "suggest", func(ctx context.Context, input *SuggestInput) (*SuggestOutput, error) {
        systemPrompt := buildSystemPrompt(input.StyleID, input.PlatformHint)
        
        result, err := genkit.GenerateData[SuggestOutput](ctx, g,
            ai.WithPrompt(input.Content),
            ai.WithSystemPrompt(systemPrompt),
        )
        if err != nil {
            return nil, fmt.Errorf("AI suggest failed: %w", err)
        }
        return result, nil
    })

    genkit.DefineFlow(g, "generate", func(ctx context.Context, input *GenerateInput) (*GenerateOutput, error) {
        systemPrompt := buildSystemPrompt(input.StyleID, input.PlatformHint)
        
        result, err := genkit.GenerateData[GenerateOutput](ctx, g,
            ai.WithPrompt(input.Prompt),
            ai.WithSystemPrompt(systemPrompt),
        )
        if err != nil {
            return nil, fmt.Errorf("AI generate failed: %w", err)
        }
        return result, nil
    })
}

// In main.go:
func main() {
    ctx := context.Background()
    
    g := genkit.Init(ctx,
        genkit.WithPlugins(&googlegenai.GoogleAI{}),
        genkit.WithDefaultModel("googleai/gemini-2.5-flash"),
    )
    
    ai.Setup(g)
    
    // Expose flows as HTTP endpoints
    mux.HandleFunc("POST /api/v1/ai/suggest", genkit.Handler(suggestFlow))
    mux.HandleFunc("POST /api/v1/ai/generate", genkit.Handler(generateFlow))
    
    // ... rest of server setup
}
```

**Dotprompt templates** (stored as `.prompt` files for versioning):

```yaml
# prompts/suggest.prompt
---
model: googleai/gemini-2.5-flash
input:
  schema:
    content: string
    platformHint: string
output:
  schema:
    suggestions:
      type: array
      items:
        type: object
        properties:
          type: string
          content: string
          reasoning: string
---

You are a social media writing assistant. The user has written the following post content.
Suggest improvements based on the platform hint: {{platformHint}}.

Rules:
- Keep the tone authentic, not robotic
- Suggest concise alternatives
- Consider platform-specific constraints

User's content:
{{content}}
```

**Streaming for compose page:**
- Genkit streaming flows send partial results via SSE
- Frontend uses `fetch` with `ReadableStream` to display suggestions incrementally
- Flow tracing automatically captures token counts and latency for budgeting

**API endpoints:**
- `POST /ai/suggest` — Takes content + context + style, returns typed `SuggestOutput`
- `POST /ai/generate` — Takes a prompt, returns typed `GenerateOutput`
- `POST /ai/stream` — SSE streaming endpoint (Genkit streaming flow)
- `GET /ai/usage` — Return current month's usage (aggregated from Genkit flow traces)
- All AI features are opt-in and require workspace-level API key configuration

**Budget tracking and rate limiting:**
- Genkit's built-in flow tracing captures token counts per invocation
- Aggregate traces into `ai_usage` table for per-workspace budgeting
- Per-workspace monthly spend limits (configurable)
- Fallback model support: configure a secondary model in Genkit init
- Content safety: basic input sanitization, optional output filtering

**Data models:**

```go
type AIConfig struct {
    bun.BaseModel `bun:"table:ai_configs"`

    ID              string `bun:",pk" json:"id"`
    WorkspaceID     string `bun:",notnull,unique" json:"workspace_id"`
    APIKeyEnc       []byte `bun:",notnull" json:"-"` // Encrypted provider API key

    // Provider configuration — Genkit supports multiple providers via plugins
    Provider        string `bun:",default:'googleai'" json:"provider"` // "googleai", "openai", "anthropic"
    DefaultModel    string `bun:",default:'googleai/gemini-2.5-flash'" json:"default_model"`
    FallbackModel   string `bun:",default:'googleai/gemini-2.0-flash'" json:"fallback_model"`
    MaxTokens       int    `bun:",default:1000" json:"max_tokens"`
    Temperature     float64 `bun:",default:0.7" json:"temperature"`

    // Budgeting
    MonthlyBudgetCents int `bun:",default:0" json:"monthly_budget_cents"` // 0 = unlimited
    CurrentSpendCents int `bun:",default:0" json:"current_spend_cents"`

    // Feature toggles
    EnableRewrite   bool `bun:",default:true" json:"enable_rewrite"`
    EnableGenerate  bool `bun:",default:true" json:"enable_generate"`
    EnableExpand    bool `bun:",default:true" json:"enable_expand"`

    CreatedAt       time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

type WritingStyle struct {
    bun.BaseModel `bun:"table:writing_styles"`

    ID           string `bun:",pk" json:"id"`
    WorkspaceID  string `bun:",notnull" json:"workspace_id"`
    Name         string `bun:",notnull" json:"name"`
    Description  string `json:"description"`

    // Style parameters
    Tone        string `json:"tone"`         // "casual", "professional", "witty"
    Formality   int    `json:"formality"`    // 1-5 scale
    Length      string `json:"length"`       // "short", "medium", "long"

    // Few-shot examples for the AI (JSON array of {input, output} pairs)
    ExamplesJSON string `bun:"examples" json:"examples"`

    // System prompt override (full control for advanced users)
    SystemPrompt string `json:"system_prompt"`

    IsDefault    bool      `bun:",default:false" json:"is_default"`
    CreatedAt    time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

// Usage tracking — populated from Genkit flow traces
type AIUsage struct {
    bun.BaseModel `bun:"table:ai_usage"`

    ID          string    `bun:",pk" json:"id"`
    WorkspaceID string    `bun:",notnull" json:"workspace_id"`
    UserID      string    `bun:",notnull" json:"user_id"`
    FlowName    string    `bun:",notnull" json:"flow_name"` // "suggest", "generate", etc.
    RequestType string    `json:"request_type"` // "rewrite", "generate", "expand"
    Provider    string    `json:"provider"`
    Model       string    `json:"model"`
    TokensIn    int       `json:"tokens_in"`
    TokensOut   int       `json:"tokens_out"`
    CostCents   int       `json:"cost_cents"` // Cost in cents for tracking
    DurationMs  int       `json:"duration_ms"`
    CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}
```

**Frontend:**
- Settings > AI: Configure provider, API key, budget, default model
- Compose page: "AI Assist" button that shows inline streaming suggestions
- Writing style selector dropdown in compose page
- Usage dashboard in settings showing monthly spend (data from Genkit traces)

**Developer workflow:**
- `genkit start -- go run .` launches the app with the Developer UI at `localhost:4000`
- Test all AI flows interactively with typed inputs/outputs
- Inspect traces for token usage, latency, and cost
- Iterate on `.prompt` files without code changes

---

### Phase 7 — Integrations

---

#### 7.1 Directus Integration

**Priority:** Low  
**Effort:** Large  
**Depends on:** Settings page for configuration

**What:** Allow users to configure a Directus CMS connection to automatically push published posts and media as items.

**Libraries:**

| Library | Purpose |
|---------|---------|
| `github.com/altipla-consulting/directus-go/v2` | Directus 11 Go SDK for REST API interaction |

**Implementation:**

**Connection:**
- Workspace settings: Directus server URL, API token, collection name
- Verify connectivity on save (`GET /server/info` health check)
- Store API token encrypted at rest using existing `TokenEncryptor`

**Field mapping:**
- Visual field mapper: drag-and-drop UI mapping OpenPost fields → Directus fields
- Auto-detect Directus collection fields via `GET /collections/{collection}` API
- Pre-populate sensible defaults: `content` → `body`, `status` → `publish_status`, `media` → `featured_image`
- Support custom field mapping via JSON config stored in `DirectusConfig.FieldMapJSON`

**Sync behavior:**
- **One-way (default):** OpenPost → Directus. After successful publish, push post data and media
- **Two-way (optional):** Directus changes sync back via webhook (`DirectusConfig.WebhookSecret` for verification)
- **Directus-master (optional):** OpenPost creates draft, Directus manages published state
- Store `PostStatusMapJSON` to map OpenPost statuses (draft/scheduled/published) to Directus statuses

**Media handling:**
- Upload media to Directus as files before creating the collection item
- Reference uploaded file IDs in the item payload
- Configure target folder in Directus (`DirectusFolderID`)

**Conflict resolution:**
- If a post is edited in both systems, use `LastSyncAt` + `UpdatedAt` timestamps to detect conflicts
- Default strategy: "last write wins" with a sync log for manual review
- `DirectusSyncLog` table tracks every sync attempt with status and error details

**Background sync:**
- After successful publish, insert a `Job` of type `"directus_sync"` with the post ID as payload
- Background worker picks up the job, pushes post data to Directus via REST API
- On failure, retry with exponential backoff (up to `MaxAttempts`)

**Data model:**

```go
type DirectusConfig struct {
    bun.BaseModel `bun:"table:directus_configs"`

    ID          string `bun:",pk" json:"id"`
    WorkspaceID string `bun:",notnull,unique" json:"workspace_id"`

    // Connection
    ServerURL     string `bun:",notnull" json:"server_url"`
    APITokenEnc   []byte `bun:",notnull" json:"-"` // Encrypted
    WebhookSecret string `json:"-"` // For verifying incoming Directus webhooks

    // Collection mapping
    Collection     string `bun:",notnull" json:"collection"`
    FieldMapJSON   string `bun:"field_map" json:"field_map"`
    // Example: {"content": "body", "media": "featured_image", "status": "publish_status", "scheduled_at": "publish_date"}

    // Status mapping (OpenPost status → Directus status)
    PostStatusMapJSON string `bun:"post_status_map" json:"post_status_map"`
    // Example: {"draft": "draft", "scheduled": "review", "published": "published"}

    // Sync settings
    SyncDirection string `bun:",default:'oneway'" json:"sync_direction"` // "oneway", "twoway", "directus_master"
    AutoSync      bool   `bun:",default:false" json:"auto_sync"`
    SyncMedia     bool   `bun:",default:true" json:"sync_media"`
    DirectusFolderID string `json:"directus_folder_id"` // Target folder for uploads

    // Diagnostics
    LastSyncAt    time.Time `json:"last_sync_at"`
    LastSyncError string    `json:"last_sync_error"`

    CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

type DirectusSyncLog struct {
    bun.BaseModel `bun:"table:directus_sync_logs"`

    ID          string    `bun:",pk" json:"id"`
    ConfigID    string    `bun:",notnull" json:"config_id"`
    PostID      string    `json:"post_id"`
    Direction   string    `bun:",notnull" json:"direction"` // "to_directus", "from_directus"
    Status      string    `bun:",notnull" json:"status"`    // "success", "failed"
    ErrorMessage string   `json:"error_message"`
    CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
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
- Rate limiting per key using Echo middleware (`golang.org/x/time/rate`)
- API key permissions (scopes): `read`, `write`, `admin`

**Data model:**

```go
type APIKey struct {
    bun.BaseModel `bun:"table:api_keys"`

    ID          string    `bun:",pk" json:"id"`
    WorkspaceID string    `bun:",notnull" json:"workspace_id"`
    Name        string    `bun:",notnull" json:"name"`
    KeyPrefix   string    `bun:",notnull" json:"key_prefix"` // First 8 chars for identification
    KeyHash     string    `bun:",notnull" json:"-"`          // SHA-256 hash of full key

    // Scopes: comma-separated list of permissions: "read", "write", "admin"
    Scopes      string    `bun:",default:'read'" json:"scopes"`

    LastUsedAt  time.Time `json:"last_used_at"`
    CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
    ExpiresAt   time.Time `json:"expires_at"`
}
```

**Key format:** `op_{8-char-prefix}_{24-char-secret}` (e.g., `op_abc12345_xYz9...`)
**Hashing:** SHA-256 of the full key, stored in `KeyHash`. On authentication, hash the provided key and compare.
**Scopes:**
- `read`: GET endpoints only
- `write`: POST/PATCH/DELETE endpoints
- `admin`: Full access including API key management

---

#### 7.3 MCP Server

**Priority:** Low (nice to have)  
**Effort:** Medium  
**Depends on:** API key system (7.2)

**What:** Model Context Protocol server allowing AI agents to interact with OpenPost. Uses the official Go MCP SDK for protocol compliance. Integrates with Genkit flows from Phase 6.2 for AI-powered tools.

**Why rethink: The original spec was too brief.** MCP is a well-defined protocol with official Go SDK support. The implementation needs proper tool schemas, transport handling, authentication, and error handling. This is synergistic with Phase 6.2 (Genkit AI) — the MCP server can expose Genkit flows as MCP tools, and Genkit's tool-calling can also connect to external MCP servers.

**Libraries:**

| Library | Purpose |
|---------|---------|
| `github.com/modelcontextprotocol/go-sdk/mcp` | Official MCP Go SDK — server, tools, resources, SSE/stdio/HTTP transports |
| `github.com/firebase/genkit/go` | Genkit — typed flows, structured output, streaming, Dotprompt, observability |
| `github.com/firebase/genkit/go/plugins/googlegenai` | Google AI / Gemini provider (free tier, no CC) |
| `github.com/firebase/genkit/go/plugins/openai` | OpenAI provider |

**Architecture:**

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────┐
│  AI Client      │────▶│  OpenPost MCP   │────▶│  OpenPost   │
│  (Claude, etc.) │     │  Server          │     │  REST API   │
└─────────────────┘     └──────┬───────────┘     └─────────────┘
                               │
                               ▼
                        ┌──────────────┐
                        │  API Keys    │
                        │  (Auth)      │
                        └──────────────┘
```

**Transports:**
- **Stdio:** For local development and CLI tools (Cursor, Claude Desktop). Server runs as subprocess.
- **Streamable HTTP:** For web-accessible deployments. Uses `mcp.NewStreamableHTTPHandler` from the official SDK. Mount at `/mcp` on the existing Echo server.
- Default: stdio for local, HTTP for deployed.

**Authentication:**
- API key-based: MCP requests include an API key in the connection metadata
- Keys are scoped to workspaces (key can only access its workspace's data)
- Rate limiting per key (inherited from 7.2's rate limiter)

**Tools exposed:**

| Tool | Description | Input Schema |
|------|-------------|-------------|
| `create_post` | Create a new post (draft or scheduled) | `workspace_id`, `content`, `status`, `scheduled_at?`, `account_ids?` |
| `list_posts` | List posts with optional filtering | `workspace_id`, `status?`, `limit?`, `offset?` |
| `get_post` | Get a specific post by ID | `post_id` |
| `update_post` | Update post content or schedule | `post_id`, `content?`, `scheduled_at?` |
| `delete_post` | Delete a draft or scheduled post | `post_id` |
| `schedule_post` | Schedule a draft post | `post_id`, `scheduled_at` |
| `list_accounts` | List connected social accounts | `workspace_id` |
| `upload_media` | Upload a media file | `workspace_id`, `file_path`, `alt_text?` |
| `list_prompts` | Get writing prompts | `category?` |
| `generate_draft` | Generate a draft using AI (requires AI config) | `workspace_id`, `prompt`, `style_id?` |

**Implementation sketch:**

```go
// internal/mcp/server.go
package mcp

import (
    "context"
    "net/http"

    mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type Server struct {
    server    *mcp.Server
    db        *bun.DB
    apiKeySvc *services.APIKeyService
    cfg       MCPServerConfig
}

type MCPServerConfig struct {
    Enabled bool
    Port    int // For HTTP transport, 0 = disabled
    RateLimitRequestsPerMinute int
    AllowedTools []string // empty = all
}

func New(db *bun.DB, apiKeySvc *services.APIKeyService, cfg MCPServerConfig) *Server {
    s := &Server{db: db, apiKeySvc: apiKeySvc, cfg: cfg}

    s.server = mcp.NewServer(
        &mcp.Implementation{
            Name:    "openpost-mcp",
            Version: "1.0.0",
        },
        &mcp.ServerOptions{},
    )

    s.registerTools()
    return s
}

func (s *Server) registerTools() {
    // create_post tool
    mcp.AddTool(s.server, &mcp.Tool{
        Name:        "create_post",
        Description: "Create a new post in OpenPost (draft or scheduled)",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "workspace_id": map[string]interface{}{
                    "type":        "string",
                    "description": "Workspace ID to create post in",
                },
                "content": map[string]interface{}{
                    "type":        "string",
                    "description": "Post content text",
                },
                "status": map[string]interface{}{
                    "type":        "string",
                    "enum":        []string{"draft", "scheduled"},
                    "description": "Post status",
                },
                "scheduled_at": map[string]interface{}{
                    "type":        "string",
                    "format":      "date-time",
                    "description": "ISO 8601 datetime for scheduled posts",
                },
                "account_ids": map[string]interface{}{
                    "type":        "array",
                    "items":       map[string]interface{}{"type": "string"},
                    "description": "Social account IDs to publish to",
                },
            },
            "required": []string{"workspace_id", "content"},
        },
    }, s.handleCreatePost)

    // ... register other tools similarly
}

// Transport handlers
func (s *Server) ServeStdio(ctx context.Context) error {
    return s.server.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) ServeHTTP(ctx context.Context, addr string) error {
    handler := mcp.NewStreamableHTTPHandler(
        func(r *http.Request) *mcp.Server {
            // Authenticate API key from request headers here
            return s.server
        },
        nil,
    )
    httpServer := &http.Server{Addr: addr, Handler: handler}
    return httpServer.ListenAndServe()
}
```

**Integration with Genkit (Phase 6.2 synergy):**
- If AI config is enabled for a workspace, the `generate_draft` MCP tool calls the Genkit `generate` flow
- Genkit's tool-calling capabilities can connect to external MCP servers for advanced AI features (web search, code execution, etc.)
- This means the MCP server can both serve tools AND Genkit flows can invoke external tools

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

**Backend:**
- Add Go tests for handlers and services (target: 80% coverage on critical paths)
- Use `github.com/stretchr/testify` (already in project) for assertions and mocks
- Use `go:generate` with `github.com/vektra/mockery/v2` for mock generation
- Consistent response shapes across all endpoints
- Proper HTTP status codes (currently inconsistent)
- Add workspace-scoped middleware to reduce boilerplate in handlers
- Consider soft-delete (`deleted_at` column) for posts and media instead of hard deletes
- Add pagination to all list endpoints (cursor-based for posts, offset for media)

**Frontend:**
- Remove unused i18n scaffold or properly implement it via Paraglide
- Svelte 5 migration debt:
  - Replace any remaining Svelte 4 reactive statements (`$:`) with `$derived`/`$effect`
  - Convert Svelte stores to `.svelte.ts` files using `$state` for shared state
  - Remove any `svelte/legacy` imports
- Consistent error handling patterns (toast notifications vs inline errors)
- Add proper loading states and skeleton screens
- Type-safe API client — regenerate `types.d.ts` from OpenAPI spec in the build pipeline
- Fix all `any` type casts in frontend (especially compose-post, accounts pages)
- Extract shared state management into proper Svelte 5 stores:

```typescript
// lib/stores/workspaces.svelte.ts
import type { components } from '$lib/api/types'

type Workspace = components['schemas']['Workspace']

class WorkspaceStore {
    workspaces = $state<Workspace[]>([])
    currentWorkspace = $state<Workspace | null>(null)
    loading = $state(false)
    error = $state<string | null>(null)

    async fetchWorkspaces() {
        this.loading = true
        this.error = null
        try {
            const { data, error } = await api.GET('/workspaces')
            if (error) throw new Error(error.message)
            this.workspaces = data
        } catch (e) {
            this.error = e.message
        } finally {
            this.loading = false
        }
    }

    setCurrent(id: string) {
        this.currentWorkspace = this.workspaces.find(w => w.id === id) ?? null
    }
}

export const workspaceStore = new WorkspaceStore()
```

**Testing:**

| Library | Purpose |
|---------|---------|
| `vitest` | Unit tests (already installed) |
| `@testing-library/svelte` | Component testing utilities |
| `@playwright/test` | E2E testing |
| `msw` | API mocking in tests |
| `github.com/vektra/mockery/v2` | Go mock generation |
| `github.com/stretchr/testify` | Go test assertions (already installed) |

---

## Database Migration Strategy

Since `CreateSchema` uses `.IfNotExists()`, adding new tables is straightforward. For adding columns to existing tables, a proper migration system is essential — **this must be implemented before any schema changes.**

SQLite `ALTER TABLE` has significant limitations (cannot drop columns, limited type changes). Use a migration tool that handles these constraints.

**Recommended library:** `github.com/pressly/goose/v3` — industry-standard, SQLite-friendly, supports SQL and Go migrations.

**Migration strategy:**

1. Add `goose` as a dependency
2. Create a `migrations/` directory with numbered SQL files
3. Create a `migrations` table to track applied migrations
4. Run migrations on startup after `CreateSchema` (for new installations)
5. For existing installations, migrations handle `ALTER TABLE` and data migration
6. Each new feature (sets, variants, schedules, etc.) gets its own migration file

**Migration files structure:**

```
backend/migrations/
├── 001_create_migrations_table.sql
├── 002_add_workspace_timezone.sql
├── 003_add_media_fields.sql
├── 004_add_post_variant_random_delay.sql
├── 005_add_posting_schedules.sql
├── 006_add_ai_config_writing_styles.sql
├── 007_add_api_keys.sql
├── 008_add_directus_configs.sql
└── 009_add_user_2fa_sessions.sql
```

**Example migration:**

```sql
-- 002_add_workspace_timezone.sql
-- +migrate Up
ALTER TABLE workspaces ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC';
ALTER TABLE workspaces ADD COLUMN week_start INTEGER NOT NULL DEFAULT 1;
ALTER TABLE workspaces ADD COLUMN media_cleanup_days INTEGER NOT NULL DEFAULT 0;

-- +migrate Down
-- SQLite doesn't support DROP COLUMN before 3.35.0
-- For downgrade, we'd need to recreate the table
-- This is acceptable for early-stage development
```

**Migration runner:**

```go
// internal/database/migrate.go
package database

import (
    "embed"
    "github.com/pressly/goose/v3"
    "github.com/uptrace/bun"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func RunMigrations(db *bun.DB) error {
    goose.SetBaseFS(embedMigrations)
    sqlDB := db.DB // Get underlying *sql.DB
    return goose.Up(sqlDB, "migrations")
}
```

---

## Feature Dependency Graph

```
Phase 0 (Prerequisites)
└── 0.1 Database Migration System ←── MUST be done before any schema changes

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
├── 3.1 Media Library ←── independent (needs 0.1 for schema changes)
└── 3.2 Auto Cleanup ←── needs 3.1

Phase 4 (Smart Scheduling)
├── 4.1 Randomized Times ←── independent
└── 4.2 Time Slot Schedules ←── needs 5.1 (timezone settings) for display

Phase 5 (Settings)
└── 5.1 Settings Page ←── aggregator for workspace prefs + security (2FA)

Phase 6 (Content)
├── 6.1 Prompt Library & Templates ←── needs 1.1
└── 6.2 AI Assistance (Genkit) ←── needs 5.1 (settings for API key config)

Phase 7 (Integrations)
├── 7.1 Directus ←── needs 5.1 (settings for config)
├── 7.2 API Keys ←── needs 5.1 (settings page)
└── 7.3 MCP Server ←── needs 7.2, uses official Go MCP SDK

Phase 8 (Polish)
├── 8.1 App Icon ←── independent
└── 8.2 Code Cleanup ←── ongoing
```

---

## Suggested Implementation Order

Within the phases, here's the recommended order considering dependencies and impact:

| # | Feature | Phase | Sprint Estimate | Notes |
|---|---------|-------|-----------------|-------|
| 0 | **Database migration system (goose)** | 0.1 | 2 days | **Must be first — blocks all schema changes** |
| 1 | Post edit/delete/re-schedule (PATCH/DELETE APIs) | 1.4 | 3 days | |
| 2 | Drafting system (backend already supports, needs UI) | 1.2 | 2 days | |
| 3 | Dedicated post page + shallow routing | 1.1 | 3 days | |
| 4 | Dashboard redesign | 1.3 | 3 days | |
| 5 | Social media sets (data model + API) | 2.2 | 3 days | |
| 6 | Per-platform content customization | 2.1 | 5 days | |
| 7 | Accounts page redesign | 2.3 | 3 days | |
| 8 | Media library page (incl. thumbnails, dedup) | 3.1 | 4 days | Enhanced from original |
| 9 | Randomized posting times | 4.1 | 1 day | |
| 10 | App logo & icon update | 8.1 | 1 day | |
| 11 | Prompt library & content templates | 6.1 | 3 days | Expanded from original |
| 12 | Settings page (timezone, week start, cleanup, 2FA) | 5.1 | 4 days | Expanded from original |
| 13 | Media auto-cleanup | 3.2 | 2 days | |
| 14 | Time slot schedules (timezone-aware) | 4.2 | 3 days | Must use UTC storage |
| 15 | API key management (with scopes) | 7.2 | 3 days | Enhanced from original |
| 16 | Directus integration (with conflict resolution) | 7.1 | 5 days | Enhanced from original |
| 17 | AI writing assistance (Genkit) | 6.2 | 5 days | Genkit flows, Dotprompt, streaming, budgeting |
| 18 | MCP server (official Go SDK) | 7.3 | 4 days | Uses official MCP Go SDK + Genkit flows |
| 19 | Codebase cleanup | 8.2 | ongoing | |

**Recommended first sprint (2 weeks):** Items 0-4 — Migration system + Core UX improvements that make the app usable for real daily scheduling.

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
| **Analytics dashboard** | Post performance, best posting times, platform comparison — consider for v2 |
| **Real-time notifications** | WebSocket-based live dashboard updates when posts are published |
| **Collaborative editing** | Multi-user real-time editing indicators |

---

## Technical Architecture Notes

### Current Pain Points
1. **No migration system** — Only `IfNotExists` for table creation; no column additions
2. **Frontend state scattered** — Workspace/account state fetched in every component instead of shared stores
3. **No error boundary** — Frontend errors crash components silently
4. **Type safety gaps** — Several `as any` casts in API client usage
5. **No pagination** — `list-posts` returns max 50 with no pagination controls

### Recommended Technical Improvements
1. Add a proper database migration system using `github.com/pressly/goose/v3`
2. Create Svelte 5 stores (`.svelte.ts` files with `$state`) for `workspaces`, `accounts`, `posts` with cache invalidation
3. Add toast/notification system for success/error feedback
4. Regenerate `types.d.ts` from OpenAPI spec in the build pipeline
5. Add pagination to all list endpoints (cursor-based for posts, offset for media)
6. Add workspace-scoped middleware to reduce boilerplate in handlers
7. Consider adding soft-delete (`deleted_at` column) for posts and media instead of hard deletes
8. Add Svelte error boundaries (`try/catch` in component boundaries) to prevent cascading failure

### Dependency Reference

**Go Backend:**

| Library | Purpose | Phase |
|---------|---------|-------|
| `github.com/labstack/echo/v4` | HTTP framework (installed) | Existing |
| `github.com/uptrace/bun` | ORM (installed) | Existing |
| `github.com/danielgtaylor/huma/v2` | OpenAPI spec generation (installed) | Existing |
| `github.com/pressly/goose/v3` | Database migrations | Phase 0 |
| `github.com/disintegration/imaging` | Image thumbnails, resizing | Phase 3 |
| `github.com/pquerna/otp` | TOTP 2FA implementation | Phase 5 |
| `github.com/firebase/genkit/go` | Genkit AI framework — flows, structured output, streaming, Dotprompt, observability | Phase 6 |
| `github.com/altipla-consulting/directus-go/v2` | Directus CMS integration | Phase 7 |
| `github.com/modelcontextprotocol/go-sdk/mcp` | Official MCP Go SDK | Phase 7 |
| `github.com/vektra/mockery/v2` | Mock generation for tests | Phase 8 |
| `golang.org/x/time/rate` | API key rate limiting | Phase 7 |

**Frontend:**

| Library | Purpose | Phase |
|---------|---------|-------|
| `svelte` ^5.51.0 | Framework (installed) | Existing |
| `bits-ui` ^2.16.3 | UI primitives (installed) | Existing |
| `openapi-fetch` ^0.17.0 | Typed API client (installed) | Existing |
| `@playwright/test` | E2E testing | Phase 8 |
| `msw` | API mocking in tests | Phase 8 |
| `@testing-library/svelte` | Component testing | Phase 8 |