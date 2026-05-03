package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Workspace struct {
	bun.BaseModel `bun:"table:workspaces"`

	ID                  string    `bun:",pk" json:"id"`
	Name                string    `bun:",notnull" json:"name"`
	Timezone            string    `bun:",default:'UTC'" json:"timezone"`
	WeekStart           int       `bun:",default:1" json:"week_start"`             // 0=Sunday, 1=Monday
	MediaCleanupDays    int       `bun:",default:0" json:"media_cleanup_days"`     // 0 = disabled
	RandomDelayMinutes  int       `bun:",default:0" json:"random_delay_minutes"`   // ±N minutes natural posting
	DraftGapMinutes     int       `bun:",default:60" json:"draft_gap_minutes"`     // Minimum gap when spilling past configured schedule slots
	SlotStartHour       int       `bun:",default:5" json:"slot_start_hour"`        // 0-23
	SlotEndHour         int       `bun:",default:23" json:"slot_end_hour"`         // 0-23
	SlotIntervalMinutes int       `bun:",default:15" json:"slot_interval_minutes"` // 1-180
	CreatedAt           time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID           string    `bun:",pk" json:"id"`
	Email        string    `bun:",unique,notnull" json:"email"`
	PasswordHash string    `bun:",notnull" json:"-"`
	CreatedAt    time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

type WorkspaceMember struct {
	bun.BaseModel `bun:"table:workspace_members"`

	WorkspaceID string `bun:",pk" json:"workspace_id"`
	UserID      string `bun:",pk" json:"user_id"`
	Role        string `bun:",notnull" json:"role"` // 'admin', 'editor', 'viewer'
}

type SocialAccount struct {
	bun.BaseModel `bun:"table:social_accounts"`

	ID               string `bun:",pk" json:"id"`
	WorkspaceID      string `bun:",notnull" json:"workspace_id"`
	Platform         string `bun:",notnull" json:"platform"` // 'x', 'threads', 'linkedin', 'mastodon', 'bluesky'
	AccountID        string `bun:",notnull" json:"account_id"`
	AccountUsername  string `json:"account_username"`
	AccountAvatarURL string `json:"account_avatar_url"`
	InstanceURL      string `json:"instance_url"` // Used for Mastodon domains and Bluesky PDS

	AccessTokenEnc  []byte    `bun:"access_token_encrypted,notnull" json:"-"`
	RefreshTokenEnc []byte    `bun:"refresh_token_encrypted" json:"-"`
	TokenExpiresAt  time.Time `json:"token_expires_at"`

	IsActive     bool      `bun:",default:true" json:"is_active"`
	ErrorMessage string    `json:"error_message"`
	CreatedAt    time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

type XOAuthRequestToken struct {
	bun.BaseModel `bun:"table:x_oauth_request_tokens"`

	RequestToken  string    `bun:",pk" json:"request_token"`
	RequestSecret string    `bun:",notnull" json:"-"`
	WorkspaceID   string    `bun:",notnull" json:"workspace_id"`
	CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

type Post struct {
	bun.BaseModel `bun:"table:posts"`

	ID          string `bun:",pk" json:"id"`
	WorkspaceID string `bun:",notnull" json:"workspace_id"`
	CreatedByID string `bun:"created_by,notnull" json:"created_by"`
	Content     string `bun:",notnull" json:"content"`

	ParentPostID   string `json:"parent_post_id"`
	ThreadSequence int    `bun:",default:0" json:"thread_sequence"`

	Status             string    `bun:",notnull" json:"status"` // 'draft', 'scheduled', 'publishing', 'published', 'failed'
	ScheduledAt        time.Time `json:"scheduled_at"`
	PublishedAt        time.Time `json:"published_at"`
	RandomDelayMinutes int       `bun:",default:0" json:"random_delay_minutes"`
	ActualRunAt        time.Time `bun:",nullzero" json:"actual_run_at"` // Set by worker, differs from ScheduledAt if randomized
	CreatedAt          time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

type PostDestination struct {
	bun.BaseModel `bun:"table:post_destinations"`

	ID              string `bun:",pk" json:"id"`
	PostID          string `bun:",notnull" json:"post_id"`
	SocialAccountID string `bun:",notnull" json:"social_account_id"`
	ExternalID      string `json:"external_id"`
	Status          string `bun:",notnull" json:"status"` // 'pending', 'success', 'failed'
	ErrorMessage    string `json:"error_message"`
}

type MediaAttachment struct {
	bun.BaseModel `bun:"table:media_attachments"`

	ID               string    `bun:",pk" json:"id"`
	WorkspaceID      string    `bun:",notnull" json:"workspace_id"`
	FilePath         string    `bun:",notnull" json:"file_path"`
	StorageType      string    `bun:",default:'local'" json:"storage_type"` // 'local', 's3'
	MimeType         string    `json:"mime_type"`
	ProcessingStatus string    `bun:",default:'ready'" json:"processing_status"` // 'processing', 'ready', 'failed'
	Size             int64     `json:"size"`
	OriginalFilename string    `json:"original_filename"`
	Width            int       `json:"width"`
	Height           int       `json:"height"`
	ThumbnailsJSON   string    `bun:"thumbnails" json:"thumbnails"` // JSON: {"sm": "sm_xxx.jpg", "md": "md_xxx.jpg"}
	FileHash         string    `bun:",unique" json:"-"`             // SHA-256 for deduplication
	AltText          string    `json:"alt_text"`
	IsFavorite       bool      `bun:",default:false" json:"is_favorite"`
	CreatedAt        time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

type PostMedia struct {
	bun.BaseModel `bun:"table:post_media"`

	PostID       string `bun:",pk" json:"post_id"`
	MediaID      string `bun:",pk" json:"media_id"`
	DisplayOrder int    `json:"display_order"`
}

type Job struct {
	bun.BaseModel `bun:"table:jobs"`

	ID          string    `bun:",pk" json:"id"`
	Type        string    `bun:",notnull" json:"type"` // 'publish_post', 'refresh_token'
	Payload     string    `bun:",notnull" json:"payload"`
	Status      string    `bun:",default:'pending'" json:"status"` // 'pending', 'processing', 'completed', 'failed'
	RunAt       time.Time `bun:",notnull" json:"run_at"`
	Attempts    int       `bun:",default:0" json:"attempts"`
	MaxAttempts int       `bun:",default:3" json:"max_attempts"`
	LastError   string    `json:"last_error"`
	LockedAt    time.Time `json:"locked_at"`
	LockedBy    string    `json:"locked_by"`
}

type SocialMediaSet struct {
	bun.BaseModel `bun:"table:social_media_sets"`

	ID          string    `bun:",pk" json:"id"`
	WorkspaceID string    `bun:",notnull" json:"workspace_id"`
	Name        string    `bun:",notnull" json:"name"`
	IsDefault   bool      `bun:",default:false" json:"is_default"`
	CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

type SocialMediaSetAccount struct {
	bun.BaseModel `bun:"table:social_media_set_accounts"`

	SetID           string `bun:",pk" json:"set_id"`
	SocialAccountID string `bun:",pk" json:"social_account_id"`
	IsMain          bool   `bun:",default:false" json:"is_main"`
}

type PostVariant struct {
	bun.BaseModel `bun:"table:post_variants"`

	ID              string    `bun:",pk" json:"id"`
	PostID          string    `bun:",notnull" json:"post_id"`
	SocialAccountID string    `bun:",notnull" json:"social_account_id"`
	Content         string    `bun:",notnull" json:"content"`
	MediaIDs        string    `bun:",nullzero" json:"media_ids"` // JSON array of media IDs override
	IsUnsynced      bool      `bun:",default:false" json:"is_unsynced"`
	CreatedAt       time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt       time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updated_at"`
}

// PostingSchedule defines preferred time slots for posting per workspace.
type PostingSchedule struct {
	bun.BaseModel `bun:"table:posting_schedules"`

	ID          string `bun:",pk" json:"id"`
	WorkspaceID string `bun:",notnull" json:"workspace_id"`
	SetID       string `json:"set_id"` // Optional: per-set schedules

	// Store times in UTC for consistency, convert on read using workspace timezone
	UTCHour   int `bun:",notnull" json:"utc_hour"`    // 0-23 UTC
	UTCMinute int `bun:",notnull" json:"utc_minute"`  // 0-59 UTC
	DayOfWeek int `bun:",notnull" json:"day_of_week"` // 0=Sunday, 6=Saturday (in UTC)

	// Display/helpers
	Label    string `json:"label"` // e.g., "Morning", "Lunch", "Evening"
	IsActive bool   `bun:",default:true" json:"is_active"`

	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
}

// Prompt represents a writing prompt for content inspiration.
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
