package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/openpost/backend/internal/api/middleware"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/queue"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/uptrace/bun"
)

type WorkspaceHandler struct {
	db   *bun.DB
	auth *auth.Service
}

func NewWorkspaceHandler(db *bun.DB, authService *auth.Service) *WorkspaceHandler {
	return &WorkspaceHandler{db: db, auth: authService}
}

type CreateWorkspaceInput struct {
	Body struct {
		Name string `json:"name" minLength:"1" maxLength:"100" doc:"Workspace name"`
	}
}

type CreateWorkspaceOutput struct {
	Body struct {
		WorkspaceID        string `json:"id"`
		WorkspaceName      string `json:"name"`
		WorkspaceCreatedAt string `json:"created_at"`
	}
}

type ListWorkspacesOutput struct {
	Body []struct {
		WorkspaceID        string `json:"id"`
		WorkspaceName      string `json:"name"`
		WorkspaceCreatedAt string `json:"created_at"`
	}
}

func (h *WorkspaceHandler) CreateWorkspace(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-workspace",
		Method:        http.MethodPost,
		Path:          "/workspaces",
		Summary:       "Create a new workspace",
		Tags:          []string{"Workspaces"},
		DefaultStatus: http.StatusOK,
		Middlewares:   huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
	}, func(ctx context.Context, input *CreateWorkspaceInput) (*CreateWorkspaceOutput, error) {
		userID := middleware.GetUserID(ctx)

		workspace := &models.Workspace{
			ID:        uuid.New().String(),
			Name:      input.Body.Name,
			CreatedAt: time.Now().UTC(),
		}

		member := &models.WorkspaceMember{
			WorkspaceID: workspace.ID,
			UserID:      userID,
			Role:        "admin",
		}

		err := h.db.RunInTx(ctx, &sql.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
			if _, err := tx.NewInsert().Model(workspace).Exec(txCtx); err != nil {
				return err
			}
			if _, err := tx.NewInsert().Model(member).Exec(txCtx); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create workspace")
		}

		resp := &CreateWorkspaceOutput{}
		resp.Body.WorkspaceID = workspace.ID
		resp.Body.WorkspaceName = workspace.Name
		resp.Body.WorkspaceCreatedAt = workspace.CreatedAt.Format(time.RFC3339)
		return resp, nil
	})
}

func (h *WorkspaceHandler) ListWorkspaces(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-workspaces",
		Method:      http.MethodGet,
		Path:        "/workspaces",
		Summary:     "List workspaces for the current user",
		Tags:        []string{"Workspaces"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
	}, func(ctx context.Context, _ *struct{}) (*ListWorkspacesOutput, error) {
		userID := middleware.GetUserID(ctx)

		var workspaces []models.Workspace
		err := h.db.NewSelect().
			Model(&workspaces).
			Join("JOIN workspace_members AS wm ON wm.workspace_id = workspace.id").
			Where("wm.user_id = ?", userID).
			Scan(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to fetch workspaces")
		}

		resp := &ListWorkspacesOutput{Body: []struct {
			WorkspaceID        string `json:"id"`
			WorkspaceName      string `json:"name"`
			WorkspaceCreatedAt string `json:"created_at"`
		}{}}
		for _, ws := range workspaces {
			resp.Body = append(resp.Body, struct {
				WorkspaceID        string `json:"id"`
				WorkspaceName      string `json:"name"`
				WorkspaceCreatedAt string `json:"created_at"`
			}{
				WorkspaceID:        ws.ID,
				WorkspaceName:      ws.Name,
				WorkspaceCreatedAt: ws.CreatedAt.Format(time.RFC3339),
			})
		}
		return resp, nil
	})
}

type GetWorkspaceSettingsInput struct {
	PathID string `path:"id" doc:"Workspace ID"`
}

type GetWorkspaceSettingsOutput struct {
	Body struct {
		Timezone            string `json:"timezone"`
		WeekStart           int    `json:"week_start"`
		MediaCleanupDays    int    `json:"media_cleanup_days"`
		RandomDelayMinutes  int    `json:"random_delay_minutes"`
		DraftGapMinutes     int    `json:"draft_gap_minutes"`
		SlotStartHour       int    `json:"slot_start_hour"`
		SlotEndHour         int    `json:"slot_end_hour"`
		SlotIntervalMinutes int    `json:"slot_interval_minutes"`
	}
}

type UpdateWorkspaceSettingsInput struct {
	PathID string `path:"id" doc:"Workspace ID"`
	Body   struct {
		Timezone            *string `json:"timezone,omitempty"`
		WeekStart           *int    `json:"week_start,omitempty"`
		MediaCleanupDays    *int    `json:"media_cleanup_days,omitempty"`
		RandomDelayMinutes  *int    `json:"random_delay_minutes,omitempty"`
		DraftGapMinutes     *int    `json:"draft_gap_minutes,omitempty"`
		SlotStartHour       *int    `json:"slot_start_hour,omitempty"`
		SlotEndHour         *int    `json:"slot_end_hour,omitempty"`
		SlotIntervalMinutes *int    `json:"slot_interval_minutes,omitempty"`
	}
}

type UpdateWorkspaceSettingsOutput struct {
	Body struct {
		Timezone            string `json:"timezone"`
		WeekStart           int    `json:"week_start"`
		MediaCleanupDays    int    `json:"media_cleanup_days"`
		RandomDelayMinutes  int    `json:"random_delay_minutes"`
		DraftGapMinutes     int    `json:"draft_gap_minutes"`
		SlotStartHour       int    `json:"slot_start_hour"`
		SlotEndHour         int    `json:"slot_end_hour"`
		SlotIntervalMinutes int    `json:"slot_interval_minutes"`
	}
}

func (h *WorkspaceHandler) GetWorkspaceSettings(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-workspace-settings",
		Method:      http.MethodGet,
		Path:        "/workspaces/{id}/settings",
		Summary:     "Get workspace settings",
		Tags:        []string{"Workspaces"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{403, 404},
	}, func(ctx context.Context, input *GetWorkspaceSettingsInput) (*GetWorkspaceSettingsOutput, error) {
		userID := middleware.GetUserID(ctx)

		var memberCount int
		memberCount, err := h.db.NewSelect().Model((*models.WorkspaceMember)(nil)).
			Where("workspace_id = ? AND user_id = ?", input.PathID, userID).
			Count(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to validate workspace access")
		}
		if memberCount == 0 {
			return nil, huma.Error403Forbidden("you do not have access to this workspace")
		}

		var workspace models.Workspace
		err = h.db.NewSelect().Model(&workspace).Where("id = ?", input.PathID).Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("workspace not found")
			}
			return nil, huma.Error500InternalServerError("failed to fetch workspace")
		}

		return &GetWorkspaceSettingsOutput{Body: struct {
			Timezone            string `json:"timezone"`
			WeekStart           int    `json:"week_start"`
			MediaCleanupDays    int    `json:"media_cleanup_days"`
			RandomDelayMinutes  int    `json:"random_delay_minutes"`
			DraftGapMinutes     int    `json:"draft_gap_minutes"`
			SlotStartHour       int    `json:"slot_start_hour"`
			SlotEndHour         int    `json:"slot_end_hour"`
			SlotIntervalMinutes int    `json:"slot_interval_minutes"`
		}{
			Timezone:            workspace.Timezone,
			WeekStart:           workspace.WeekStart,
			MediaCleanupDays:    workspace.MediaCleanupDays,
			RandomDelayMinutes:  workspace.RandomDelayMinutes,
			DraftGapMinutes:     workspace.DraftGapMinutes,
			SlotStartHour:       workspace.SlotStartHour,
			SlotEndHour:         workspace.SlotEndHour,
			SlotIntervalMinutes: workspace.SlotIntervalMinutes,
		}}, nil
	})
}

//nolint:gocyclo
func (h *WorkspaceHandler) UpdateWorkspaceSettings(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "update-workspace-settings",
		Method:      http.MethodPatch,
		Path:        "/workspaces/{id}/settings",
		Summary:     "Update workspace settings",
		Tags:        []string{"Workspaces"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 403, 404},
	}, func(ctx context.Context, input *UpdateWorkspaceSettingsInput) (*UpdateWorkspaceSettingsOutput, error) {
		userID := middleware.GetUserID(ctx)

		var memberCount int
		memberCount, err := h.db.NewSelect().Model((*models.WorkspaceMember)(nil)).
			Where("workspace_id = ? AND user_id = ?", input.PathID, userID).
			Count(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to validate workspace access")
		}
		if memberCount == 0 {
			return nil, huma.Error403Forbidden("you do not have access to this workspace")
		}

		var workspace models.Workspace
		err = h.db.NewSelect().Model(&workspace).Where("id = ?", input.PathID).Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("workspace not found")
			}
			return nil, huma.Error500InternalServerError("failed to fetch workspace")
		}

		if input.Body.Timezone != nil {
			workspace.Timezone = *input.Body.Timezone
		}
		if input.Body.WeekStart != nil {
			if *input.Body.WeekStart < 0 || *input.Body.WeekStart > 1 {
				return nil, huma.Error400BadRequest("week_start must be 0 (Sunday) or 1 (Monday)")
			}
			workspace.WeekStart = *input.Body.WeekStart
		}
		if input.Body.MediaCleanupDays != nil {
			if *input.Body.MediaCleanupDays < 0 || *input.Body.MediaCleanupDays > 365 {
				return nil, huma.Error400BadRequest("media_cleanup_days must be between 0 and 365")
			}
			workspace.MediaCleanupDays = *input.Body.MediaCleanupDays
		}
		if input.Body.RandomDelayMinutes != nil {
			if *input.Body.RandomDelayMinutes < 0 || *input.Body.RandomDelayMinutes > 60 {
				return nil, huma.Error400BadRequest("random_delay_minutes must be between 0 and 60")
			}
			workspace.RandomDelayMinutes = *input.Body.RandomDelayMinutes
		}
		if input.Body.DraftGapMinutes != nil {
			if *input.Body.DraftGapMinutes < 0 || *input.Body.DraftGapMinutes > 24*60 {
				return nil, huma.Error400BadRequest("draft_gap_minutes must be between 0 and 1440")
			}
			workspace.DraftGapMinutes = *input.Body.DraftGapMinutes
		}
		if input.Body.SlotStartHour != nil {
			if *input.Body.SlotStartHour < 0 || *input.Body.SlotStartHour > 23 {
				return nil, huma.Error400BadRequest("slot_start_hour must be between 0 and 23")
			}
			workspace.SlotStartHour = *input.Body.SlotStartHour
		}
		if input.Body.SlotEndHour != nil {
			if *input.Body.SlotEndHour < 0 || *input.Body.SlotEndHour > 23 {
				return nil, huma.Error400BadRequest("slot_end_hour must be between 0 and 23")
			}
			workspace.SlotEndHour = *input.Body.SlotEndHour
		}
		if input.Body.SlotIntervalMinutes != nil {
			if *input.Body.SlotIntervalMinutes < 1 || *input.Body.SlotIntervalMinutes > 180 {
				return nil, huma.Error400BadRequest("slot_interval_minutes must be between 1 and 180")
			}
			workspace.SlotIntervalMinutes = *input.Body.SlotIntervalMinutes
		}

		_, err = h.db.NewUpdate().Model(&workspace).
			Column("timezone", "week_start", "media_cleanup_days", "random_delay_minutes", "draft_gap_minutes", "slot_start_hour", "slot_end_hour", "slot_interval_minutes").
			Where("id = ?", input.PathID).
			Exec(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to update workspace")
		}

		if input.Body.MediaCleanupDays != nil {
			_ = queue.ScheduleMediaCleanup(h.db, input.PathID, workspace.MediaCleanupDays) //nolint:errcheck
		}

		return &UpdateWorkspaceSettingsOutput{Body: struct {
			Timezone            string `json:"timezone"`
			WeekStart           int    `json:"week_start"`
			MediaCleanupDays    int    `json:"media_cleanup_days"`
			RandomDelayMinutes  int    `json:"random_delay_minutes"`
			DraftGapMinutes     int    `json:"draft_gap_minutes"`
			SlotStartHour       int    `json:"slot_start_hour"`
			SlotEndHour         int    `json:"slot_end_hour"`
			SlotIntervalMinutes int    `json:"slot_interval_minutes"`
		}{
			Timezone:            workspace.Timezone,
			WeekStart:           workspace.WeekStart,
			MediaCleanupDays:    workspace.MediaCleanupDays,
			RandomDelayMinutes:  workspace.RandomDelayMinutes,
			DraftGapMinutes:     workspace.DraftGapMinutes,
			SlotStartHour:       workspace.SlotStartHour,
			SlotEndHour:         workspace.SlotEndHour,
			SlotIntervalMinutes: workspace.SlotIntervalMinutes,
		}}, nil
	})
}
