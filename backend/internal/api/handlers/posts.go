package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/openpost/backend/internal/models"
	"github.com/uptrace/bun"
)

type PostHandler struct {
	db *bun.DB
}

func NewPostHandler(db *bun.DB) *PostHandler {
	return &PostHandler{db: db}
}

type CreatePostReq struct {
	WorkspaceID      string     `json:"workspace_id"`
	Content          string     `json:"content"`
	ScheduledAt      *time.Time `json:"scheduled_at"`
	SocialAccountIDs []string   `json:"social_account_ids"`
}

func (h *PostHandler) CreatePost(c echo.Context) error {
	req := new(CreatePostReq)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	userID, ok := c.Get("user_id").(string)
	if !ok {
		userID = "test-user-1"
	}

	status := "draft"
	if req.ScheduledAt != nil {
		status = "scheduled"
	}

	post := &models.Post{
		ID:          uuid.New().String(),
		WorkspaceID: req.WorkspaceID,
		CreatedByID: userID,
		Content:     req.Content,
		Status:      status,
		CreatedAt:   time.Now(),
	}
	if req.ScheduledAt != nil {
		post.ScheduledAt = *req.ScheduledAt
	}

	destinations := make([]models.PostDestination, 0, len(req.SocialAccountIDs))
	for _, accID := range req.SocialAccountIDs {
		destinations = append(destinations, models.PostDestination{
			ID:              uuid.New().String(),
			PostID:          post.ID,
			SocialAccountID: accID,
			Status:          "pending",
		})
	}

	err := h.db.RunInTx(c.Request().Context(), &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(post).Exec(ctx); err != nil {
			return err
		}
		if len(destinations) > 0 {
			if _, err := tx.NewInsert().Model(&destinations).Exec(ctx); err != nil {
				return err
			}
		}

		// If scheduled, automatically insert a Job for the polling worker
		if post.Status == "scheduled" {
			job := &models.Job{
				ID:      uuid.New().String(),
				Type:    "publish_post",
				Payload: `{"post_id":"` + post.ID + `"}`,
				RunAt:   post.ScheduledAt,
			}
			if _, err := tx.NewInsert().Model(job).Exec(ctx); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, post)
}

func (h *PostHandler) ListPosts(c echo.Context) error {
	workspaceID := c.QueryParam("workspace_id")
	if workspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "workspace_id required"})
	}

	var posts []models.Post
	err := h.db.NewSelect().
		Model(&posts).
		Where("workspace_id = ?", workspaceID).
		Order("created_at DESC").
		Limit(50).
		Scan(c.Request().Context())

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list posts"})
	}

	return c.JSON(http.StatusOK, posts)
}

type ScheduleDay struct {
	Date      string                `json:"date"`
	Count     int                   `json:"count"`
	Platforms []ScheduleDayPlatform `json:"platforms"`
}

type ScheduleDayPlatform struct {
	Platform string `json:"platform"`
	Count    int    `json:"count"`
}

type ScheduleOverviewResponse struct {
	Year                int                `json:"year"`
	Month               int                `json:"month"`
	SelectedWorkspaceID string             `json:"selected_workspace_id"`
	SelectedPlatform    string             `json:"selected_platform"`
	Workspaces          []models.Workspace `json:"workspaces"`
	Platforms           []string           `json:"platforms"`
	Days                []ScheduleDay      `json:"days"`
}

func (h *PostHandler) GetScheduleOverview(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		userID = "test-user-1"
	}

	monthParam := c.QueryParam("month")
	var monthStart time.Time
	if monthParam == "" {
		now := time.Now().UTC()
		monthStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		parsed, err := time.Parse("2006-01", monthParam)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "month must be in YYYY-MM format"})
		}
		monthStart = parsed.UTC()
	}
	monthEnd := monthStart.AddDate(0, 1, 0)

	ctx := c.Request().Context()

	var workspaces []models.Workspace
	err := h.db.NewSelect().
		Model(&workspaces).
		Join("JOIN workspace_members AS wm ON wm.workspace_id = workspace.id").
		Where("wm.user_id = ?", userID).
		Order("workspace.created_at DESC").
		Scan(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch workspaces"})
	}

	selectedWorkspaceID := c.QueryParam("workspace_id")
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
			return c.JSON(http.StatusForbidden, map[string]string{"error": "workspace not accessible"})
		}
	}

	selectedPlatform := c.QueryParam("platform")

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
	err = platformQuery.Scan(ctx, &platformRows)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch platforms"})
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
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid platform filter"})
		}
	}

	var dayRows []struct {
		Date  string `bun:"date"`
		Count int    `bun:"count"`
	}

	query := `
		SELECT DATE(p.scheduled_at) AS date, COUNT(DISTINCT p.id) AS count
		FROM posts AS p
		JOIN workspace_members AS wm ON wm.workspace_id = p.workspace_id
	`
	args := []interface{}{userID, monthStart, monthEnd}

	if selectedPlatform != "" {
		query += `
			JOIN post_destinations AS pd ON pd.post_id = p.id
			JOIN social_accounts AS sa ON sa.id = pd.social_account_id
		`
	}

	query += `
		WHERE wm.user_id = ?
			AND p.scheduled_at >= ?
			AND p.scheduled_at < ?
			AND p.status IN ('scheduled', 'publishing', 'published')
	`

	if selectedWorkspaceID != "" {
		query += ` AND p.workspace_id = ?`
		args = append(args, selectedWorkspaceID)
	}

	if selectedPlatform != "" {
		query += ` AND sa.platform = ?`
		args = append(args, selectedPlatform)
	}

	query += ` GROUP BY DATE(p.scheduled_at) ORDER BY DATE(p.scheduled_at)`

	err = h.db.NewRaw(query, args...).Scan(ctx, &dayRows)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch schedule days"})
	}

	days := make([]ScheduleDay, 0, len(dayRows))
	dayIndexByDate := make(map[string]int, len(dayRows))
	for _, row := range dayRows {
		dayIndexByDate[row.Date] = len(days)
		days = append(days, ScheduleDay{Date: row.Date, Count: row.Count, Platforms: []ScheduleDayPlatform{}})
	}

	var platformDayRows []struct {
		Date     string `bun:"date"`
		Platform string `bun:"platform"`
		Count    int    `bun:"count"`
	}

	platformCountsQuery := `
		SELECT DATE(p.scheduled_at) AS date, sa.platform AS platform, COUNT(DISTINCT p.id) AS count
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
		platformCountsQuery += ` AND p.workspace_id = ?`
		platformArgs = append(platformArgs, selectedWorkspaceID)
	}

	if selectedPlatform != "" {
		platformCountsQuery += ` AND sa.platform = ?`
		platformArgs = append(platformArgs, selectedPlatform)
	}

	platformCountsQuery += ` GROUP BY DATE(p.scheduled_at), sa.platform ORDER BY DATE(p.scheduled_at), sa.platform`

	err = h.db.NewRaw(platformCountsQuery, platformArgs...).Scan(ctx, &platformDayRows)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch schedule platform days"})
	}

	for _, row := range platformDayRows {
		idx, ok := dayIndexByDate[row.Date]
		if !ok {
			continue
		}
		days[idx].Platforms = append(days[idx].Platforms, ScheduleDayPlatform{
			Platform: row.Platform,
			Count:    row.Count,
		})
	}

	return c.JSON(http.StatusOK, ScheduleOverviewResponse{
		Year:                monthStart.Year(),
		Month:               int(monthStart.Month()),
		SelectedWorkspaceID: selectedWorkspaceID,
		SelectedPlatform:    selectedPlatform,
		Workspaces:          workspaces,
		Platforms:           platforms,
		Days:                days,
	})
}
