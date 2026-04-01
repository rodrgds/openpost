package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Workspace struct {
	bun.BaseModel `bun:"table:workspaces"`

	ID        string    `bun:",pk" json:"id"`
	Name      string    `bun:",notnull" json:"name"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
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

type Post struct {
	bun.BaseModel `bun:"table:posts"`

	ID          string `bun:",pk" json:"id"`
	WorkspaceID string `bun:",notnull" json:"workspace_id"`
	CreatedByID string `bun:"created_by,notnull" json:"created_by"`
	Content     string `bun:",notnull" json:"content"`

	ParentPostID   string `json:"parent_post_id"`
	ThreadSequence int    `bun:",default:0" json:"thread_sequence"`

	Status      string    `bun:",notnull" json:"status"` // 'draft', 'scheduled', 'publishing', 'published', 'failed'
	ScheduledAt time.Time `json:"scheduled_at"`
	PublishedAt time.Time `json:"published_at"`
	CreatedAt   time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"created_at"`
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

	ID               string `bun:",pk" json:"id"`
	WorkspaceID      string `bun:",notnull" json:"workspace_id"`
	FilePath         string `bun:",notnull" json:"file_path"`
	StorageType      string `bun:",default:'local'" json:"storage_type"` // 'local', 's3'
	MimeType         string `json:"mime_type"`
	ProcessingStatus string `bun:",default:'ready'" json:"processing_status"` // 'processing', 'ready', 'failed'
	Size             int64  `json:"size"`
	AltText          string `json:"alt_text"`
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
