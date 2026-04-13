package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"path/filepath"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/openpost/backend/internal/api/middleware"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/openpost/backend/internal/services/mediastore"
	"github.com/uptrace/bun"
)

type MediaHandler struct {
	db      *bun.DB
	storage mediastore.BlobStorage
	auth    *auth.Service
}

func NewMediaHandler(db *bun.DB, storage mediastore.BlobStorage, authService *auth.Service) *MediaHandler {
	return &MediaHandler{db: db, storage: storage, auth: authService}
}

type MediaUsageItem struct {
	PostID    string `json:"post_id" doc:"Post ID"`
	Content   string `json:"content" doc:"Post content (truncated)"`
	Status    string `json:"status" doc:"Post status"`
	Scheduled string `json:"scheduled_at" doc:"Scheduled time"`
}

type MediaListItem struct {
	ID               string `json:"id" doc:"Media ID"`
	WorkspaceID      string `json:"workspace_id" doc:"Workspace ID"`
	MimeType         string `json:"mime_type" doc:"MIME type"`
	Size             int64  `json:"size" doc:"File size in bytes"`
	AltText          string `json:"alt_text" doc:"Alt text"`
	IsFavorite       bool   `json:"is_favorite" doc:"Whether media is favorited"`
	CreatedAt        string `json:"created_at" doc:"Creation time"`
	URL              string `json:"url" doc:"URL to access the media"`
	UsageCount       int    `json:"usage_count" doc:"Number of posts using this media"`
	ProcessingStatus string `json:"processing_status" doc:"Processing status"`
}

type ListMediaInput struct {
	WorkspaceID string `query:"workspace_id" doc:"Filter by workspace ID (required)"`
	Filter      string `query:"filter" doc:"Filter: all, used, unused, favorites"`
	Sort        string `query:"sort" doc:"Sort: newest, oldest, size"`
	Limit       int    `query:"limit" doc:"Limit (default 50, max 200)"`
}

type ListMediaOutput struct {
	Body struct {
		Media []MediaListItem `json:"media" doc:"Media attachments"`
		Total int             `json:"total" doc:"Total count matching filter"`
	}
}

type GetMediaUsageInput struct {
	PathID string `path:"id" doc:"Media ID"`
}

type GetMediaUsageOutput struct {
	Body struct {
		Usage []MediaUsageItem `json:"usage" doc:"Posts using this media"`
		Count int              `json:"count" doc:"Number of posts using this media"`
	}
}

type DeleteMediaInput struct {
	PathID string `path:"id" doc:"Media ID"`
}

type DeleteMediaOutput struct {
	Body struct {
		Message string `json:"message" doc:"Success message"`
	}
}

type UpdateMediaFavoriteInput struct {
	PathID string `path:"id" doc:"Media ID"`
}

type UpdateMediaFavoriteOutput struct {
	Body struct {
		IsFavorite bool `json:"is_favorite" doc:"Updated favorite status"`
	}
}

func (h *MediaHandler) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-media",
		Method:      http.MethodGet,
		Path:        "/media",
		Summary:     "List media attachments for a workspace",
		Tags:        []string{"Media"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 403},
	}, func(ctx context.Context, input *ListMediaInput) (*ListMediaOutput, error) {
		userID := middleware.GetUserID(ctx)

		if input.WorkspaceID == "" {
			return nil, huma.Error400BadRequest("workspace_id is required")
		}

		var memberCount int
		memberCount, err := h.db.NewSelect().Model((*models.WorkspaceMember)(nil)).
			Where("workspace_id = ? AND user_id = ?", input.WorkspaceID, userID).
			Count(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to validate workspace access")
		}
		if memberCount == 0 {
			return nil, huma.Error403Forbidden("you do not have access to this workspace")
		}

		limit := input.Limit
		if limit <= 0 || limit > 200 {
			limit = 50
		}

		query := h.db.NewSelect().Model(&models.MediaAttachment{}).
			Where("workspace_id = ?", input.WorkspaceID)

		switch input.Filter {
		case "favorites":
			query = query.Where("is_favorite = ?", true)
		case "used":
			query = query.Where("id IN (SELECT media_id FROM post_media)")
		case "unused":
			query = query.Where("id NOT IN (SELECT media_id FROM post_media)")
		}

		var total int
		total, err = query.Count(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to count media")
		}

		switch input.Sort {
		case "oldest":
			query = query.Order("created_at ASC")
		case "size":
			query = query.Order("size DESC")
		default:
			query = query.Order("created_at DESC")
		}

		var media []models.MediaAttachment
		err = query.Limit(limit).Scan(ctx, &media)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to fetch media")
		}

		result := make([]MediaListItem, len(media))
		for i, m := range media {
			var usageCount int
			h.db.NewSelect().Model(&models.PostMedia{}).
				Where("media_id = ?", m.ID).
				Count(ctx)

			result[i] = MediaListItem{
				ID:               m.ID,
				WorkspaceID:      m.WorkspaceID,
				MimeType:         m.MimeType,
				Size:             m.Size,
				AltText:          m.AltText,
				IsFavorite:       m.IsFavorite,
				CreatedAt:        m.CreatedAt.Format(time.RFC3339),
				URL:              "/media/" + m.ID,
				UsageCount:       usageCount,
				ProcessingStatus: m.ProcessingStatus,
			}
		}

		return &ListMediaOutput{Body: struct {
			Media []MediaListItem `json:"media" doc:"Media attachments"`
			Total int             `json:"total" doc:"Total count matching filter"`
		}{Media: result, Total: total}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-media-usage",
		Method:      http.MethodGet,
		Path:        "/media/{id}/usage",
		Summary:     "Get posts that use a media attachment",
		Tags:        []string{"Media"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{403, 404},
	}, func(ctx context.Context, input *GetMediaUsageInput) (*GetMediaUsageOutput, error) {
		userID := middleware.GetUserID(ctx)

		var media models.MediaAttachment
		err := h.db.NewSelect().Model(&media).Where("id = ?", input.PathID).Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("media not found")
			}
			return nil, huma.Error500InternalServerError("failed to fetch media")
		}

		var memberCount int
		memberCount, err = h.db.NewSelect().Model((*models.WorkspaceMember)(nil)).
			Where("workspace_id = ? AND user_id = ?", media.WorkspaceID, userID).
			Count(ctx)
		if err != nil || memberCount == 0 {
			return nil, huma.Error403Forbidden("you do not have access to this workspace")
		}

		var postMedia []models.PostMedia
		err = h.db.NewSelect().Model(&postMedia).
			Where("media_id = ?", input.PathID).
			Scan(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to fetch usage")
		}

		usage := make([]MediaUsageItem, 0, len(postMedia))
		for _, pm := range postMedia {
			var post models.Post
			if err := h.db.NewSelect().Model(&post).Where("id = ?", pm.PostID).Scan(ctx); err == nil {
				content := post.Content
				if len(content) > 100 {
					content = content[:100] + "..."
				}
				scheduled := ""
				if !post.ScheduledAt.IsZero() {
					scheduled = post.ScheduledAt.Format(time.RFC3339)
				}
				usage = append(usage, MediaUsageItem{
					PostID:    post.ID,
					Content:   content,
					Status:    post.Status,
					Scheduled: scheduled,
				})
			}
		}

		return &GetMediaUsageOutput{Body: struct {
			Usage []MediaUsageItem `json:"usage" doc:"Posts using this media"`
			Count int              `json:"count" doc:"Number of posts using this media"`
		}{Usage: usage, Count: len(usage)}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "delete-media",
		Method:      http.MethodDelete,
		Path:        "/media/{id}",
		Summary:     "Delete a media attachment (only if not used in any post)",
		Tags:        []string{"Media"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{403, 404},
	}, func(ctx context.Context, input *DeleteMediaInput) (*DeleteMediaOutput, error) {
		userID := middleware.GetUserID(ctx)

		var media models.MediaAttachment
		err := h.db.NewSelect().Model(&media).Where("id = ?", input.PathID).Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("media not found")
			}
			return nil, huma.Error500InternalServerError("failed to fetch media")
		}

		var memberCount int
		memberCount, err = h.db.NewSelect().Model((*models.WorkspaceMember)(nil)).
			Where("workspace_id = ? AND user_id = ?", media.WorkspaceID, userID).
			Count(ctx)
		if err != nil || memberCount == 0 {
			return nil, huma.Error403Forbidden("you do not have access to this workspace")
		}

		var usageCount int
		usageCount, err = h.db.NewSelect().Model(&models.PostMedia{}).
			Where("media_id = ?", input.PathID).
			Count(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to check usage")
		}
		if usageCount > 0 {
			return nil, huma.Error400BadRequest("cannot delete media that is attached to posts")
		}

		if err := h.storage.Delete(filepath.Base(media.FilePath)); err != nil {
			return nil, huma.Error500InternalServerError("failed to delete media file")
		}

		_, err = h.db.NewDelete().Model(&media).Where("id = ?", input.PathID).Exec(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to delete media record")
		}

		return &DeleteMediaOutput{Body: struct {
			Message string `json:"message" doc:"Success message"`
		}{Message: "media deleted successfully"}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-media-favorite",
		Method:      http.MethodPatch,
		Path:        "/media/{id}/favorite",
		Summary:     "Toggle favorite status of a media attachment",
		Tags:        []string{"Media"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{403, 404},
	}, func(ctx context.Context, input *UpdateMediaFavoriteInput) (*UpdateMediaFavoriteOutput, error) {
		userID := middleware.GetUserID(ctx)

		var media models.MediaAttachment
		err := h.db.NewSelect().Model(&media).Where("id = ?", input.PathID).Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("media not found")
			}
			return nil, huma.Error500InternalServerError("failed to fetch media")
		}

		var memberCount int
		memberCount, err = h.db.NewSelect().Model((*models.WorkspaceMember)(nil)).
			Where("workspace_id = ? AND user_id = ?", media.WorkspaceID, userID).
			Count(ctx)
		if err != nil || memberCount == 0 {
			return nil, huma.Error403Forbidden("you do not have access to this workspace")
		}

		media.IsFavorite = !media.IsFavorite
		_, err = h.db.NewUpdate().Model(&media).Column("is_favorite").Where("id = ?", input.PathID).Exec(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to update favorite status")
		}

		return &UpdateMediaFavoriteOutput{Body: struct {
			IsFavorite bool `json:"is_favorite" doc:"Updated favorite status"`
		}{IsFavorite: media.IsFavorite}}, nil
	})
}

func (h *MediaHandler) RegisterLegacyRoutes(e *echo.Echo) {
	e.POST("/api/v1/media/upload", h.uploadMedia, middleware.JWTMiddleware(h.auth))
	e.GET("/media/:id", h.serveMedia)
}

func (h *MediaHandler) uploadMedia(c echo.Context) error {
	userID := c.Get(string(middleware.UserIDKey)).(string)

	workspaceID := c.FormValue("workspace_id")
	if workspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "workspace_id is required"})
	}

	var memberCount int
	memberCount, err := h.db.NewSelect().Model((*models.WorkspaceMember)(nil)).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Count(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to validate workspace access"})
	}
	if memberCount == 0 {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "you do not have access to this workspace"})
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file is required"})
	}

	if fileHeader.Size > 50*1024*1024 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file size exceeds 50MB limit"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to open file"})
	}
	defer file.Close()

	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil || n == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to read file header"})
	}
	mimeType := http.DetectContentType(header[:n])
	if _, err := file.Seek(0, 0); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to seek file"})
	}

	mediaID := uuid.New().String()
	ext := filepath.Ext(fileHeader.Filename)
	filename := mediaID + ext

	savedPath, err := h.storage.Save(filename, file)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save media"})
	}

	altText := c.FormValue("alt_text")

	media := &models.MediaAttachment{
		ID:               mediaID,
		WorkspaceID:      workspaceID,
		FilePath:         savedPath,
		StorageType:      "local",
		MimeType:         mimeType,
		ProcessingStatus: "ready",
		Size:             fileHeader.Size,
		AltText:          altText,
	}

	if _, err := h.db.NewInsert().Model(media).Exec(c.Request().Context()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save media record"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":        mediaID,
		"mime_type": mimeType,
		"url":       "/media/" + mediaID,
		"size":      fileHeader.Size,
	})
}

func (h *MediaHandler) serveMedia(c echo.Context) error {
	mediaID := c.Param("id")

	media := new(models.MediaAttachment)
	if err := h.db.NewSelect().Model(media).Where("id = ?", mediaID).Scan(c.Request().Context()); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "media not found"})
	}

	file, err := h.storage.Open(filepath.Base(media.FilePath))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "media file not found"})
	}
	defer file.Close()

	c.Response().Header().Set("Content-Type", media.MimeType)
	c.Response().Header().Set("Cache-Control", "public, max-age=86400")

	return c.Stream(http.StatusOK, media.MimeType, file)
}
