package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/openpost/backend/internal/api/middleware"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/uptrace/bun"
)

type PostHandler struct {
	db   *bun.DB
	auth *auth.Service
}

func NewPostHandler(db *bun.DB, authService *auth.Service) *PostHandler {
	return &PostHandler{db: db, auth: authService}
}

type CreatePostInput struct {
	Body struct {
		WorkspaceID      string     `json:"workspace_id" doc:"Target workspace ID"`
		Content          string     `json:"content" minLength:"1" doc:"Post content"`
		ScheduledAt      *time.Time `json:"scheduled_at,omitempty" doc:"Schedule time (ISO 8601). Omit for draft."`
		SocialAccountIDs []string   `json:"social_account_ids" doc:"Social account IDs to publish to"`
		MediaIDs         []string   `json:"media_ids,omitempty" doc:"Media attachment IDs to include"`
	}
}

type CreatePostOutput struct {
	Body *PostResponse
}

type PostDestinationResponse struct {
	SocialAccountID string `json:"social_account_id" doc:"Social account ID"`
	Platform        string `json:"platform" doc:"Platform name"`
	Status          string `json:"status" doc:"Destination status"`
}

type PostResponse struct {
	ID           string                    `json:"id" doc:"Post ID"`
	WorkspaceID  string                    `json:"workspace_id" doc:"Workspace ID"`
	CreatedByID  string                    `json:"created_by" doc:"Creator user ID"`
	Content      string                    `json:"content" doc:"Post content"`
	Status       string                    `json:"status" doc:"Post status (draft, scheduled, publishing, published, failed)"`
	ScheduledAt  string                    `json:"scheduled_at" doc:"Scheduled time (ISO 8601)"`
	CreatedAt    string                    `json:"created_at" doc:"Creation time (ISO 8601)"`
	Destinations []PostDestinationResponse `json:"destinations,omitempty" doc:"Post destinations"`
}

type ListPostsInput struct {
	WorkspaceID string `query:"workspace_id" doc:"Filter by workspace ID"`
	Date        string `query:"date" doc:"Filter by date (YYYY-MM-DD)"`
	Status      string `query:"status" doc:"Filter by status (draft, scheduled, published, failed)"`
	Limit       int    `query:"limit" doc:"Limit number of results (default 50, max 200)"`
}

type ListPostsOutput struct {
	Body []PostResponse
}

type ScheduleDayPlatform struct {
	Platform string `json:"platform" doc:"Platform name"`
	Count    int    `json:"count" doc:"Count for this platform on this day"`
}

type ScheduleDayWorkspace struct {
	WorkspaceID string `json:"workspace_id" doc:"Workspace ID"`
	Count       int    `json:"count" doc:"Count for this workspace on this day"`
}

type ScheduleDay struct {
	Date       string                 `json:"date" doc:"Date in YYYY-MM-DD format"`
	Count      int                    `json:"count" doc:"Number of scheduled posts"`
	Platforms  []ScheduleDayPlatform  `json:"platforms" doc:"Per-platform breakdown"`
	Workspaces []ScheduleDayWorkspace `json:"workspaces" doc:"Per-workspace breakdown"`
}

type ScheduleOverviewInput struct {
	WorkspaceID string `query:"workspace_id" doc:"Filter by workspace ID"`
	Platform    string `query:"platform" doc:"Filter by platform"`
	Month       string `query:"month" doc:"Month in YYYY-MM format (defaults to current month)"`
}

type ScheduleOverviewOutput struct {
	Body struct {
		Year                int             `json:"year" doc:"Year of the overview"`
		Month               int             `json:"month" doc:"Month of the overview (1-12)"`
		SelectedWorkspaceID string          `json:"selected_workspace_id" doc:"Currently selected workspace"`
		SelectedPlatform    string          `json:"selected_platform" doc:"Currently selected platform filter"`
		Workspaces          []WorkspaceResp `json:"workspaces" doc:"Available workspaces"`
		Platforms           []string        `json:"platforms" doc:"Available platforms"`
		Days                []ScheduleDay   `json:"days" doc:"Daily schedule data"`
	}
}

type WorkspaceResp struct {
	WorkspaceID        string `json:"id" doc:"Workspace ID"`
	WorkspaceName      string `json:"name" doc:"Workspace name"`
	WorkspaceCreatedAt string `json:"created_at" doc:"Creation time (ISO 8601)"`
}

func (h *PostHandler) CreatePost(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-post",
		Method:      http.MethodPost,
		Path:        "/posts",
		Summary:     "Create a new post",
		Tags:        []string{"Posts"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400},
	}, func(ctx context.Context, input *CreatePostInput) (*CreatePostOutput, error) {
		userID := middleware.GetUserID(ctx)

		status := "draft"
		if input.Body.ScheduledAt != nil {
			status = "scheduled"
		}

		post := &models.Post{
			ID:          uuid.New().String(),
			WorkspaceID: input.Body.WorkspaceID,
			CreatedByID: userID,
			Content:     input.Body.Content,
			Status:      status,
			CreatedAt:   time.Now().UTC(),
		}
		if input.Body.ScheduledAt != nil {
			post.ScheduledAt = *input.Body.ScheduledAt
		}

		destinations := make([]models.PostDestination, 0, len(input.Body.SocialAccountIDs))
		for _, accID := range input.Body.SocialAccountIDs {
			destinations = append(destinations, models.PostDestination{
				ID:              uuid.New().String(),
				PostID:          post.ID,
				SocialAccountID: accID,
				Status:          "pending",
			})
		}

		postMedia := make([]models.PostMedia, 0, len(input.Body.MediaIDs))
		for i, mediaID := range input.Body.MediaIDs {
			postMedia = append(postMedia, models.PostMedia{
				PostID:       post.ID,
				MediaID:      mediaID,
				DisplayOrder: i,
			})
		}

		err := h.db.RunInTx(ctx, &sql.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
			if _, err := tx.NewInsert().Model(post).Exec(txCtx); err != nil {
				return err
			}
			if len(destinations) > 0 {
				if _, err := tx.NewInsert().Model(&destinations).Exec(txCtx); err != nil {
					return err
				}
			}
			if len(postMedia) > 0 {
				if _, err := tx.NewInsert().Model(&postMedia).Exec(txCtx); err != nil {
					return err
				}
			}
			if post.Status == "scheduled" {
				payload, err := json.Marshal(map[string]string{"post_id": post.ID})
				if err != nil {
					return fmt.Errorf("failed to marshal job payload: %w", err)
				}
				job := &models.Job{
					ID:      uuid.New().String(),
					Type:    "publish_post",
					Payload: string(payload),
					RunAt:   post.ScheduledAt,
				}
				if _, err := tx.NewInsert().Model(job).Exec(txCtx); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create post")
		}

		resp := &CreatePostOutput{}
		resp.Body = &PostResponse{
			ID:          post.ID,
			WorkspaceID: post.WorkspaceID,
			CreatedByID: post.CreatedByID,
			Content:     post.Content,
			Status:      post.Status,
			ScheduledAt: post.ScheduledAt.Format(time.RFC3339),
			CreatedAt:   post.CreatedAt.Format(time.RFC3339),
		}
		return resp, nil
	})
}

func (h *PostHandler) ListPosts(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-posts",
		Method:      http.MethodGet,
		Path:        "/posts",
		Summary:     "List posts for a workspace",
		Tags:        []string{"Posts"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
	}, func(ctx context.Context, input *ListPostsInput) (*ListPostsOutput, error) {
		var posts []models.Post

		var workspaceIDs []string
		if input.WorkspaceID != "" {
			workspaceIDs = []string{input.WorkspaceID}
		} else {
			var workspaceMembers []models.WorkspaceMember
			userID := middleware.GetUserID(ctx)
			err := h.db.NewSelect().
				Model(&workspaceMembers).
				Where("user_id = ?", userID).
				Scan(ctx)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error500InternalServerError("failed to fetch workspaces")
			}
			for _, wm := range workspaceMembers {
				workspaceIDs = append(workspaceIDs, wm.WorkspaceID)
			}
		}

		if len(workspaceIDs) == 0 {
			return &ListPostsOutput{Body: []PostResponse{}}, nil
		}

		query := h.db.NewSelect().
			Model(&posts).
			Where("workspace_id IN (?)", bun.In(workspaceIDs))

		if input.Status != "" {
			query = query.Where("status = ?", input.Status)
		}

		if input.Date != "" {
			parsed, err := time.Parse("2006-01-02", input.Date)
			if err != nil {
				return nil, huma.Error400BadRequest("date must be in YYYY-MM-DD format")
			}
			dayStart := parsed.UTC()
			dayEnd := dayStart.AddDate(0, 0, 1)
			query = query.Where("scheduled_at >= ? AND scheduled_at < ?", dayStart, dayEnd)
		}

		limit := input.Limit
		if limit <= 0 || limit > 200 {
			limit = 50
		}

		err := query.Order("COALESCE(scheduled_at, created_at) DESC").Limit(limit).Scan(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list posts")
		}

		postIDs := make([]string, len(posts))
		for i, p := range posts {
			postIDs[i] = p.ID
		}

		var destinations []struct {
			PostID          string `bun:"post_id"`
			SocialAccountID string `bun:"social_account_id"`
			Platform        string `bun:"platform"`
			Status          string `bun:"status"`
		}
		if len(postIDs) > 0 {
			err = h.db.NewSelect().
				TableExpr("post_destinations AS pd").
				ColumnExpr("pd.post_id, pd.social_account_id, sa.platform, pd.status").
				Join("JOIN social_accounts AS sa ON sa.id = pd.social_account_id").
				Where("pd.post_id IN (?)", bun.In(postIDs)).
				Scan(ctx, &destinations)
			if err != nil {
				return nil, huma.Error500InternalServerError("failed to fetch destinations")
			}
		}

		destByPost := make(map[string][]PostDestinationResponse)
		for _, d := range destinations {
			destByPost[d.PostID] = append(destByPost[d.PostID], PostDestinationResponse{
				SocialAccountID: d.SocialAccountID,
				Platform:        d.Platform,
				Status:          d.Status,
			})
		}

		result := make([]PostResponse, len(posts))
		for i, p := range posts {
			result[i] = PostResponse{
				ID:           p.ID,
				WorkspaceID:  p.WorkspaceID,
				CreatedByID:  p.CreatedByID,
				Content:      p.Content,
				Status:       p.Status,
				ScheduledAt:  p.ScheduledAt.Format(time.RFC3339),
				CreatedAt:    p.CreatedAt.Format(time.RFC3339),
				Destinations: destByPost[p.ID],
			}
		}
		return &ListPostsOutput{Body: result}, nil
	})
}

func (h *PostHandler) GetScheduleOverview(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-schedule-overview",
		Method:      http.MethodGet,
		Path:        "/posts/schedule-overview",
		Summary:     "Get monthly schedule overview",
		Tags:        []string{"Posts"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 403},
	}, func(ctx context.Context, input *ScheduleOverviewInput) (*ScheduleOverviewOutput, error) {
		userID := middleware.GetUserID(ctx)

		var monthStart time.Time
		if input.Month == "" {
			now := time.Now().UTC()
			monthStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		} else {
			parsed, err := time.Parse("2006-01", input.Month)
			if err != nil {
				return nil, huma.Error400BadRequest("month must be in YYYY-MM format")
			}
			monthStart = parsed.UTC()
		}
		monthEnd := monthStart.AddDate(0, 1, 0)

		var workspaces []models.Workspace
		err := h.db.NewSelect().
			Model(&workspaces).
			Join("JOIN workspace_members AS wm ON wm.workspace_id = workspace.id").
			Where("wm.user_id = ?", userID).
			Order("workspace.created_at DESC").
			Scan(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to fetch workspaces")
		}

		selectedWorkspaceID := input.WorkspaceID
		if selectedWorkspaceID == "" && len(workspaces) > 0 {
			selectedWorkspaceID = workspaces[0].ID
		}

		if selectedWorkspaceID != "" {
			isMember := false
			for _, ws := range workspaces {
				if ws.ID == selectedWorkspaceID {
					isMember = true
					break
				}
			}
			if !isMember {
				return nil, huma.Error403Forbidden("workspace not accessible")
			}
		}

		selectedPlatform := input.Platform

		var platformRows []struct {
			Platform string `bun:"platform"`
		}
		platformQuery := h.db.NewSelect().
			TableExpr("social_accounts AS sa").
			ColumnExpr("DISTINCT sa.platform AS platform").
			Join("JOIN workspace_members AS wm ON wm.workspace_id = sa.workspace_id").
			Where("wm.user_id = ?", userID).
			Where("sa.is_active = ?", true)
		if selectedWorkspaceID != "" {
			platformQuery = platformQuery.Where("sa.workspace_id = ?", selectedWorkspaceID)
		}
		if err = platformQuery.Scan(ctx, &platformRows); err != nil {
			return nil, huma.Error500InternalServerError("failed to fetch platforms")
		}

		platforms := make([]string, 0, len(platformRows))
		for _, row := range platformRows {
			if row.Platform != "" {
				platforms = append(platforms, row.Platform)
			}
		}
		sort.Strings(platforms)

		if selectedPlatform != "" {
			hasSelectedPlatform := false
			for _, p := range platforms {
				if p == selectedPlatform {
					hasSelectedPlatform = true
					break
				}
			}
			if !hasSelectedPlatform {
				return nil, huma.Error400BadRequest("invalid platform filter")
			}
		}

		// Query daily counts
		var dayRows []struct {
			Date  string `bun:"date"`
			Count int    `bun:"count"`
		}

		dayQuery := `
			SELECT DATE(p.scheduled_at) AS date, COUNT(DISTINCT p.id) AS count
			FROM posts AS p
			JOIN workspace_members AS wm ON wm.workspace_id = p.workspace_id
		`
		dayArgs := []interface{}{userID, monthStart, monthEnd}

		if selectedPlatform != "" {
			dayQuery += `
				JOIN post_destinations AS pd ON pd.post_id = p.id
				JOIN social_accounts AS sa ON sa.id = pd.social_account_id
			`
		}

		dayQuery += `
			WHERE wm.user_id = ?
				AND p.scheduled_at >= ?
				AND p.scheduled_at < ?
				AND p.status IN ('scheduled', 'publishing', 'published')
		`

		if selectedWorkspaceID != "" {
			dayQuery += ` AND p.workspace_id = ?`
			dayArgs = append(dayArgs, selectedWorkspaceID)
		}
		if selectedPlatform != "" {
			dayQuery += ` AND sa.platform = ?`
			dayArgs = append(dayArgs, selectedPlatform)
		}

		dayQuery += ` GROUP BY DATE(p.scheduled_at) ORDER BY DATE(p.scheduled_at)`

		if err = h.db.NewRaw(dayQuery, dayArgs...).Scan(ctx, &dayRows); err != nil {
			return nil, huma.Error500InternalServerError("failed to fetch schedule days")
		}

		days := make([]ScheduleDay, 0, len(dayRows))
		dayIndexByDate := make(map[string]int, len(dayRows))
		for _, row := range dayRows {
			dayIndexByDate[row.Date] = len(days)
			days = append(days, ScheduleDay{
				Date:       row.Date,
				Count:      row.Count,
				Platforms:  []ScheduleDayPlatform{},
				Workspaces: []ScheduleDayWorkspace{},
			})
		}

		// Combined query: fetch per-platform and per-workspace counts in a single call (UNION ALL)
		var combinedRows []struct {
			Date        string `bun:"date"`
			Platform    string `bun:"platform"`
			WorkspaceID string `bun:"workspace_id"`
			Count       int    `bun:"count"`
		}

		combinedQuery := ``
		combinedArgs := make([]interface{}, 0)

		// Platform counts part (only includes posts that have destinations/platforms)
		platformPart := `
            SELECT DATE(p.scheduled_at) AS date, sa.platform AS platform, NULL AS workspace_id, COUNT(DISTINCT p.id) AS count
            FROM posts AS p
            JOIN workspace_members AS wm ON wm.workspace_id = p.workspace_id
            JOIN post_destinations AS pd ON pd.post_id = p.id
            JOIN social_accounts AS sa ON sa.id = pd.social_account_id
            WHERE wm.user_id = ?
                AND p.scheduled_at >= ?
                AND p.scheduled_at < ?
                AND p.status IN ('scheduled', 'publishing', 'published')
        `
		platformArgs := []interface{}{userID, monthStart, monthEnd}
		if selectedWorkspaceID != "" {
			platformPart += ` AND p.workspace_id = ?`
			platformArgs = append(platformArgs, selectedWorkspaceID)
		}
		if selectedPlatform != "" {
			platformPart += ` AND sa.platform = ?`
			platformArgs = append(platformArgs, selectedPlatform)
		}
		platformPart += ` GROUP BY DATE(p.scheduled_at), sa.platform`

		// Workspace counts part
		workspacePart := `
            SELECT DATE(p.scheduled_at) AS date, NULL AS platform, p.workspace_id AS workspace_id, COUNT(DISTINCT p.id) AS count
            FROM posts AS p
            JOIN workspace_members AS wm ON wm.workspace_id = p.workspace_id
            WHERE wm.user_id = ?
                AND p.scheduled_at >= ?
                AND p.scheduled_at < ?
                AND p.status IN ('scheduled', 'publishing', 'published')
        `
		workspaceArgs := []interface{}{userID, monthStart, monthEnd}
		if selectedWorkspaceID != "" {
			workspacePart += ` AND p.workspace_id = ?`
			workspaceArgs = append(workspaceArgs, selectedWorkspaceID)
		}
		workspacePart += ` GROUP BY DATE(p.scheduled_at), p.workspace_id`

		combinedQuery = platformPart + ` UNION ALL ` + workspacePart + ` ORDER BY date`
		combinedArgs = append(combinedArgs, platformArgs...)
		combinedArgs = append(combinedArgs, workspaceArgs...)

		if err = h.db.NewRaw(combinedQuery, combinedArgs...).Scan(ctx, &combinedRows); err != nil {
			return nil, huma.Error500InternalServerError("failed to fetch schedule details")
		}

		for _, row := range combinedRows {
			idx, ok := dayIndexByDate[row.Date]
			if !ok {
				continue
			}
			if row.Platform != "" {
				days[idx].Platforms = append(days[idx].Platforms, ScheduleDayPlatform{
					Platform: row.Platform,
					Count:    row.Count,
				})
			}
			if row.WorkspaceID != "" {
				days[idx].Workspaces = append(days[idx].Workspaces, ScheduleDayWorkspace{
					WorkspaceID: row.WorkspaceID,
					Count:       row.Count,
				})
			}
		}

		resp := &ScheduleOverviewOutput{}
		resp.Body.Year = monthStart.Year()
		resp.Body.Month = int(monthStart.Month())
		resp.Body.SelectedWorkspaceID = selectedWorkspaceID
		resp.Body.SelectedPlatform = selectedPlatform
		resp.Body.Workspaces = make([]WorkspaceResp, len(workspaces))
		for i, ws := range workspaces {
			resp.Body.Workspaces[i] = WorkspaceResp{
				WorkspaceID:        ws.ID,
				WorkspaceName:      ws.Name,
				WorkspaceCreatedAt: ws.CreatedAt.Format(time.RFC3339),
			}
		}
		resp.Body.Platforms = platforms
		resp.Body.Days = days
		return resp, nil
	})
}

type ThreadPostInput struct {
	Content  string   `json:"content" minLength:"1" doc:"Post content"`
	MediaIDs []string `json:"media_ids,omitempty" doc:"Media attachment IDs"`
}

type CreateThreadInput struct {
	Body struct {
		WorkspaceID      string            `json:"workspace_id" doc:"Target workspace ID"`
		ScheduledAt      *time.Time        `json:"scheduled_at,omitempty" doc:"Schedule time (ISO 8601). Omit for draft."`
		SocialAccountIDs []string          `json:"social_account_ids" doc:"Social account IDs to publish to"`
		Posts            []ThreadPostInput `json:"posts" minItems:"2" doc:"Thread posts in order"`
	}
}

type CreateThreadOutput struct {
	Body struct {
		PostIDs []string `json:"post_ids" doc:"Created post IDs in order"`
	}
}

func (h *PostHandler) CreateThread(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-thread",
		Method:      http.MethodPost,
		Path:        "/posts/thread",
		Summary:     "Create a thread of posts",
		Tags:        []string{"Posts"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400},
	}, func(ctx context.Context, input *CreateThreadInput) (*CreateThreadOutput, error) {
		userID := middleware.GetUserID(ctx)

		if len(input.Body.Posts) < 2 {
			return nil, huma.Error400BadRequest("a thread must have at least 2 posts")
		}

		status := "draft"
		if input.Body.ScheduledAt != nil {
			status = "scheduled"
		}

		posts := make([]*models.Post, 0, len(input.Body.Posts))
		var allDestinations []models.PostDestination
		var allPostMedia []models.PostMedia

		for i, threadPost := range input.Body.Posts {
			post := &models.Post{
				ID:             uuid.New().String(),
				WorkspaceID:    input.Body.WorkspaceID,
				CreatedByID:    userID,
				Content:        threadPost.Content,
				Status:         status,
				ThreadSequence: i,
				CreatedAt:      time.Now().UTC(),
			}
			if input.Body.ScheduledAt != nil {
				post.ScheduledAt = *input.Body.ScheduledAt
			}
			if i > 0 {
				post.ParentPostID = posts[i-1].ID
			}
			posts = append(posts, post)

			for _, accID := range input.Body.SocialAccountIDs {
				allDestinations = append(allDestinations, models.PostDestination{
					ID:              uuid.New().String(),
					PostID:          post.ID,
					SocialAccountID: accID,
					Status:          "pending",
				})
			}

			for j, mediaID := range threadPost.MediaIDs {
				allPostMedia = append(allPostMedia, models.PostMedia{
					PostID:       post.ID,
					MediaID:      mediaID,
					DisplayOrder: j,
				})
			}
		}

		err := h.db.RunInTx(ctx, &sql.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
			for _, post := range posts {
				if _, err := tx.NewInsert().Model(post).Exec(txCtx); err != nil {
					return err
				}
			}
			if len(allDestinations) > 0 {
				if _, err := tx.NewInsert().Model(&allDestinations).Exec(txCtx); err != nil {
					return err
				}
			}
			if len(allPostMedia) > 0 {
				if _, err := tx.NewInsert().Model(&allPostMedia).Exec(txCtx); err != nil {
					return err
				}
			}
			if status == "scheduled" {
				payload, _ := json.Marshal(map[string]string{"post_id": posts[0].ID})
				job := &models.Job{
					ID:      uuid.New().String(),
					Type:    "publish_post",
					Payload: string(payload),
					RunAt:   *input.Body.ScheduledAt,
				}
				if _, err := tx.NewInsert().Model(job).Exec(txCtx); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create thread")
		}

		postIDs := make([]string, len(posts))
		for i, post := range posts {
			postIDs[i] = post.ID
		}

		resp := &CreateThreadOutput{}
		resp.Body.PostIDs = postIDs
		return resp, nil
	})
}

type GetPostInput struct {
	PathID string `path:"id" doc:"Post ID"`
}

type GetPostOutput struct {
	Body *PostDetailResponse
}

type PostMediaResponse struct {
	MediaID      string `json:"media_id" doc:"Media ID"`
	DisplayOrder int    `json:"display_order" doc:"Display order"`
	FilePath     string `json:"file_path" doc:"File path for media URL"`
	MimeType     string `json:"mime_type" doc:"Media MIME type"`
}

type PostDetailResponse struct {
	ID           string                    `json:"id" doc:"Post ID"`
	WorkspaceID  string                    `json:"workspace_id" doc:"Workspace ID"`
	CreatedByID  string                    `json:"created_by" doc:"Creator user ID"`
	Content      string                    `json:"content" doc:"Post content"`
	Status       string                    `json:"status" doc:"Post status"`
	ScheduledAt  string                    `json:"scheduled_at" doc:"Scheduled time (ISO 8601)"`
	CreatedAt    string                    `json:"created_at" doc:"Creation time (ISO 8601)"`
	Media        []PostMediaResponse       `json:"media,omitempty" doc:"Attached media"`
	Destinations []PostDestinationResponse `json:"destinations,omitempty" doc:"Post destinations"`
}

func (h *PostHandler) GetPost(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-post",
		Method:      http.MethodGet,
		Path:        "/posts/{id}",
		Summary:     "Get a single post",
		Tags:        []string{"Posts"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{404},
	}, func(ctx context.Context, input *GetPostInput) (*GetPostOutput, error) {
		userID := middleware.GetUserID(ctx)

		var post models.Post
		err := h.db.NewSelect().
			Model(&post).
			Where("id = ?", input.PathID).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("post not found")
			}
			return nil, huma.Error500InternalServerError("failed to fetch post")
		}

		if err := h.checkWorkspaceAccess(ctx, post.WorkspaceID, userID); err != nil {
			return nil, err
		}

		var destinations []struct {
			PostID          string `bun:"post_id"`
			SocialAccountID string `bun:"social_account_id"`
			Platform        string `bun:"platform"`
			Status          string `bun:"status"`
		}
		err = h.db.NewSelect().
			TableExpr("post_destinations AS pd").
			ColumnExpr("pd.post_id, pd.social_account_id, sa.platform, pd.status").
			Join("JOIN social_accounts AS sa ON sa.id = pd.social_account_id").
			Where("pd.post_id = ?", input.PathID).
			Scan(ctx, &destinations)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to fetch destinations")
		}

		var mediaAttachments []struct {
			MediaID      string `bun:"media_id"`
			DisplayOrder int    `bun:"display_order"`
			FilePath     string `bun:"file_path"`
			MimeType     string `bun:"mime_type"`
		}
		err = h.db.NewSelect().
			TableExpr("post_media AS pm").
			ColumnExpr("pm.media_id, pm.display_order, ma.file_path, ma.mime_type").
			Join("JOIN media_attachments AS ma ON ma.id = pm.media_id").
			Where("pm.post_id = ?", input.PathID).
			Order("pm.display_order ASC").
			Scan(ctx, &mediaAttachments)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error500InternalServerError("failed to fetch media")
		}

		destResp := make([]PostDestinationResponse, len(destinations))
		for i, d := range destinations {
			destResp[i] = PostDestinationResponse{
				SocialAccountID: d.SocialAccountID,
				Platform:        d.Platform,
				Status:          d.Status,
			}
		}

		mediaResp := make([]PostMediaResponse, len(mediaAttachments))
		for i, m := range mediaAttachments {
			mediaResp[i] = PostMediaResponse{
				MediaID:      m.MediaID,
				DisplayOrder: m.DisplayOrder,
				FilePath:     m.FilePath,
				MimeType:     m.MimeType,
			}
		}

		return &GetPostOutput{Body: &PostDetailResponse{
			ID:           post.ID,
			WorkspaceID:  post.WorkspaceID,
			CreatedByID:  post.CreatedByID,
			Content:      post.Content,
			Status:       post.Status,
			ScheduledAt:  post.ScheduledAt.Format(time.RFC3339),
			CreatedAt:    post.CreatedAt.Format(time.RFC3339),
			Media:        mediaResp,
			Destinations: destResp,
		}}, nil
	})
}

type UpdatePostInput struct {
	PathID string `path:"id" doc:"Post ID"`
	Body   struct {
		Content          *string  `json:"content,omitempty" doc:"Post content"`
		ScheduledAt      *string  `json:"scheduled_at,omitempty" doc:"Schedule time (ISO 8601). Set to empty string to unschedule (make draft)."`
		SocialAccountIDs []string `json:"social_account_ids,omitempty" doc:"Social account IDs to publish to (replace all)"`
		MediaIDs         []string `json:"media_ids,omitempty" doc:"Media attachment IDs to include (replace all)"`
	}
}

type UpdatePostOutput struct {
	Body *PostDetailResponse
}

func (h *PostHandler) UpdatePost(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "update-post",
		Method:      http.MethodPatch,
		Path:        "/posts/{id}",
		Summary:     "Update a post",
		Tags:        []string{"Posts"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 403, 404},
	}, func(ctx context.Context, input *UpdatePostInput) (*UpdatePostOutput, error) {
		userID := middleware.GetUserID(ctx)

		var post models.Post
		err := h.db.NewSelect().
			Model(&post).
			Where("id = ?", input.PathID).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("post not found")
			}
			return nil, huma.Error500InternalServerError("failed to fetch post")
		}

		if err := h.checkWorkspaceAccess(ctx, post.WorkspaceID, userID); err != nil {
			return nil, err
		}

		if post.Status == "published" {
			return nil, huma.Error400BadRequest("cannot edit a published post")
		}

		err = h.db.RunInTx(ctx, &sql.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
			if input.Body.Content != nil {
				post.Content = *input.Body.Content
			}

			if input.Body.ScheduledAt != nil {
				if *input.Body.ScheduledAt == "" {
					post.Status = "draft"
					post.ScheduledAt = time.Time{}
					if _, err := tx.NewUpdate().Model(&post).Column("content", "status", "scheduled_at").Where("id = ?", post.ID).Exec(txCtx); err != nil {
						return fmt.Errorf("failed to unschedule post: %w", err)
					}
					if _, err := tx.NewDelete().Model(&models.Job{}).Where("payload LIKE ?", "%"+post.ID+"%").Exec(txCtx); err != nil {
						return fmt.Errorf("failed to cancel job: %w", err)
					}
				} else {
					parsed, parseErr := time.Parse(time.RFC3339, *input.Body.ScheduledAt)
					if parseErr != nil {
						return fmt.Errorf("invalid scheduled_at format: %w", parseErr)
					}
					oldScheduledAt := post.ScheduledAt
					post.ScheduledAt = parsed
					post.Status = "scheduled"
					if _, err := tx.NewUpdate().Model(&post).Column("content", "status", "scheduled_at").Where("id = ?", post.ID).Exec(txCtx); err != nil {
						return fmt.Errorf("failed to update post: %w", err)
					}
					if !oldScheduledAt.IsZero() {
						if _, err := tx.NewDelete().Model(&models.Job{}).Where("payload LIKE ?", "%"+post.ID+"%").Exec(txCtx); err != nil {
							return fmt.Errorf("failed to cancel old job: %w", err)
						}
					}
					payload, _ := json.Marshal(map[string]string{"post_id": post.ID})
					job := &models.Job{
						ID:      uuid.New().String(),
						Type:    "publish_post",
						Payload: string(payload),
						RunAt:   post.ScheduledAt,
					}
					if _, err := tx.NewInsert().Model(job).Exec(txCtx); err != nil {
						return fmt.Errorf("failed to create job: %w", err)
					}
				}
				return nil
			}

			if input.Body.Content != nil {
				if _, err := tx.NewUpdate().Model(&post).Column("content").Where("id = ?", post.ID).Exec(txCtx); err != nil {
					return fmt.Errorf("failed to update content: %w", err)
				}
			}

			if input.Body.SocialAccountIDs != nil {
				if _, err := tx.NewDelete().Model(&models.PostDestination{}).Where("post_id = ?", post.ID).Exec(txCtx); err != nil {
					return fmt.Errorf("failed to remove old destinations: %w", err)
				}
				for _, accID := range input.Body.SocialAccountIDs {
					dest := models.PostDestination{
						ID:              uuid.New().String(),
						PostID:          post.ID,
						SocialAccountID: accID,
						Status:          "pending",
					}
					if _, err := tx.NewInsert().Model(&dest).Exec(txCtx); err != nil {
						return fmt.Errorf("failed to add destination: %w", err)
					}
				}
			}

			if input.Body.MediaIDs != nil {
				if _, err := tx.NewDelete().Model(&models.PostMedia{}).Where("post_id = ?", post.ID).Exec(txCtx); err != nil {
					return fmt.Errorf("failed to remove old media: %w", err)
				}
				for i, mediaID := range input.Body.MediaIDs {
					pm := models.PostMedia{
						PostID:       post.ID,
						MediaID:      mediaID,
						DisplayOrder: i,
					}
					if _, err := tx.NewInsert().Model(&pm).Exec(txCtx); err != nil {
						return fmt.Errorf("failed to add media: %w", err)
					}
				}
			}

			return nil
		})
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}

		var respPost models.Post
		if err := h.db.NewSelect().Model(&respPost).Where("id = ?", post.ID).Scan(ctx); err != nil {
			return nil, huma.Error500InternalServerError("failed to refetch post")
		}

		var destinations []struct {
			PostID          string `bun:"post_id"`
			SocialAccountID string `bun:"social_account_id"`
			Platform        string `bun:"platform"`
			Status          string `bun:"status"`
		}
		h.db.NewSelect().
			TableExpr("post_destinations AS pd").
			ColumnExpr("pd.post_id, pd.social_account_id, sa.platform, pd.status").
			Join("JOIN social_accounts AS sa ON sa.id = pd.social_account_id").
			Where("pd.post_id = ?", post.ID).
			Scan(ctx, &destinations)

		var mediaAttachments []struct {
			MediaID      string `bun:"media_id"`
			DisplayOrder int    `bun:"display_order"`
			FilePath     string `bun:"file_path"`
			MimeType     string `bun:"mime_type"`
		}
		h.db.NewSelect().
			TableExpr("post_media AS pm").
			ColumnExpr("pm.media_id, pm.display_order, ma.file_path, ma.mime_type").
			Join("JOIN media_attachments AS ma ON ma.id = pm.media_id").
			Where("pm.post_id = ?", post.ID).
			Order("pm.display_order ASC").
			Scan(ctx, &mediaAttachments)

		destResp := make([]PostDestinationResponse, len(destinations))
		for i, d := range destinations {
			destResp[i] = PostDestinationResponse{
				SocialAccountID: d.SocialAccountID,
				Platform:        d.Platform,
				Status:          d.Status,
			}
		}

		mediaResp := make([]PostMediaResponse, len(mediaAttachments))
		for i, m := range mediaAttachments {
			mediaResp[i] = PostMediaResponse{
				MediaID:      m.MediaID,
				DisplayOrder: m.DisplayOrder,
				FilePath:     m.FilePath,
				MimeType:     m.MimeType,
			}
		}

		return &UpdatePostOutput{Body: &PostDetailResponse{
			ID:           respPost.ID,
			WorkspaceID:  respPost.WorkspaceID,
			CreatedByID:  respPost.CreatedByID,
			Content:      respPost.Content,
			Status:       respPost.Status,
			ScheduledAt:  respPost.ScheduledAt.Format(time.RFC3339),
			CreatedAt:    respPost.CreatedAt.Format(time.RFC3339),
			Media:        mediaResp,
			Destinations: destResp,
		}}, nil
	})
}

type DeletePostInput struct {
	PathID string `path:"id" doc:"Post ID"`
}

type DeletePostOutput struct {
	Body struct {
		Message string `json:"message" doc:"Success message"`
	}
}

func (h *PostHandler) DeletePost(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-post",
		Method:      http.MethodDelete,
		Path:        "/posts/{id}",
		Summary:     "Delete a post",
		Tags:        []string{"Posts"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 403, 404},
	}, func(ctx context.Context, input *DeletePostInput) (*DeletePostOutput, error) {
		userID := middleware.GetUserID(ctx)

		var post models.Post
		err := h.db.NewSelect().
			Model(&post).
			Where("id = ?", input.PathID).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("post not found")
			}
			return nil, huma.Error500InternalServerError("failed to fetch post")
		}

		if err := h.checkWorkspaceAccess(ctx, post.WorkspaceID, userID); err != nil {
			return nil, err
		}

		if post.Status == "published" || post.Status == "publishing" {
			return nil, huma.Error400BadRequest("cannot delete a post that is published or being published")
		}

		err = h.db.RunInTx(ctx, &sql.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
			if _, err := tx.NewDelete().Model(&models.PostMedia{}).Where("post_id = ?", post.ID).Exec(txCtx); err != nil {
				return fmt.Errorf("failed to delete post media: %w", err)
			}
			if _, err := tx.NewDelete().Model(&models.PostDestination{}).Where("post_id = ?", post.ID).Exec(txCtx); err != nil {
				return fmt.Errorf("failed to delete destinations: %w", err)
			}
			if _, err := tx.NewDelete().Model(&models.Job{}).Where("payload LIKE ?", "%"+post.ID+"%").Exec(txCtx); err != nil {
				return fmt.Errorf("failed to delete jobs: %w", err)
			}
			if _, err := tx.NewDelete().Model(&post).Where("id = ?", post.ID).Exec(txCtx); err != nil {
				return fmt.Errorf("failed to delete post: %w", err)
			}
			return nil
		})
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}

		return &DeletePostOutput{Body: struct {
			Message string `json:"message" doc:"Success message"`
		}{Message: "post deleted successfully"}}, nil
	})
}

func (h *PostHandler) checkWorkspaceAccess(ctx context.Context, workspaceID, userID string) error {
	var members []models.WorkspaceMember
	err := h.db.NewSelect().
		Model(&members).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Scan(ctx)
	if err != nil {
		return huma.Error500InternalServerError("failed to check workspace access")
	}
	if len(members) == 0 {
		return huma.Error403Forbidden("workspace not accessible")
	}
	return nil
}
