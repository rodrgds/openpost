package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/openpost/backend/internal/api/middleware"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/uptrace/bun"
)

type JobResponse struct {
	ID          string `json:"id" doc:"Job ID"`
	Type        string `json:"type" doc:"Job type"`
	Status      string `json:"status" doc:"Job status"`
	Payload     string `json:"payload,omitempty" doc:"Job payload"`
	RunAt       string `json:"run_at" doc:"Scheduled run time"`
	Attempts    int    `json:"attempts" doc:"Number of attempts"`
	MaxAttempts int    `json:"max_attempts" doc:"Maximum attempts"`
	LastError   string `json:"last_error,omitempty" doc:"Last error message"`
	LockedAt    string `json:"locked_at,omitempty" doc:"When job was locked"`
	CreatedAt   string `json:"created_at" doc:"Creation time"`
}

type ListJobsInput struct {
	Limit       int    `query:"limit" doc:"Number of jobs to return (default 50, max 200)"`
	Status      string `query:"status" doc:"Filter by status (pending, processing, completed, failed)"`
	WorkspaceID string `query:"workspace_id" doc:"Filter by workspace ID"`
}

type ListJobsOutput struct {
	Body []JobResponse
}

type JobHandler struct {
	db   *bun.DB
	auth *auth.Service
}

func NewJobHandler(db *bun.DB, authService *auth.Service) *JobHandler {
	return &JobHandler{db: db, auth: authService}
}

func (h *JobHandler) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-jobs",
		Method:      http.MethodGet,
		Path:        "/jobs",
		Summary:     "List recent background jobs",
		Tags:        []string{"Jobs"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
	}, func(ctx context.Context, input *ListJobsInput) (*ListJobsOutput, error) {
		userID := middleware.GetUserID(ctx)
		isAdmin, err := h.isInstanceAdmin(ctx, userID)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to load user")
		}

		limit := input.Limit
		if limit <= 0 || limit > 200 {
			limit = 50
		}

		allowedWorkspaces, err := h.allowedWorkspaces(ctx, userID, isAdmin, input.WorkspaceID)
		if err != nil {
			var humaErr huma.StatusError
			if errors.As(err, &humaErr) {
				return nil, humaErr
			}
			return nil, huma.Error500InternalServerError("failed to resolve workspace scope")
		}

		var jobs []models.Job
		query := h.db.NewSelect().
			Model(&jobs).
			ModelTableExpr("jobs AS job").
			ColumnExpr("job.*").
			Join("LEFT JOIN posts AS p ON p.id = json_extract(job.payload, '$.post_id')").
			Join("LEFT JOIN social_accounts AS sa ON sa.id = json_extract(job.payload, '$.account_id')").
			Order("job.run_at DESC").
			Limit(limit)

		if input.Status != "" {
			query = query.Where("job.status = ?", input.Status)
		}
		if input.WorkspaceID != "" {
			query = query.Where("COALESCE(p.workspace_id, sa.workspace_id) = ?", input.WorkspaceID)
		} else if !isAdmin {
			workspaceIDs := make([]string, 0, len(allowedWorkspaces))
			for workspaceID := range allowedWorkspaces {
				workspaceIDs = append(workspaceIDs, workspaceID)
			}
			if len(workspaceIDs) == 0 {
				return &ListJobsOutput{Body: []JobResponse{}}, nil
			}
			query = query.Where("COALESCE(p.workspace_id, sa.workspace_id) IN (?)", bun.List(workspaceIDs))
		}

		if err := query.Scan(ctx); err != nil {
			return nil, huma.Error500InternalServerError("failed to fetch jobs")
		}

		resp := make([]JobResponse, 0, len(jobs))
		for _, j := range jobs {
			item := JobResponse{
				ID:          j.ID,
				Type:        j.Type,
				Status:      j.Status,
				RunAt:       j.RunAt.Format(time.RFC3339),
				Attempts:    j.Attempts,
				MaxAttempts: j.MaxAttempts,
				LastError:   j.LastError,
			}
			if !j.LockedAt.IsZero() {
				item.LockedAt = j.LockedAt.Format(time.RFC3339)
			}
			if isAdmin {
				item.Payload = j.Payload
			}
			resp = append(resp, item)
		}

		return &ListJobsOutput{Body: resp}, nil
	})
}

func (h *JobHandler) isInstanceAdmin(ctx context.Context, userID string) (bool, error) {
	var user models.User
	if err := h.db.NewSelect().
		Model(&user).
		Where("id = ?", userID).
		Scan(ctx); err != nil {
		return false, err
	}
	return user.IsAdmin, nil
}

func (h *JobHandler) allowedWorkspaces(ctx context.Context, userID string, isAdmin bool, requestedWorkspaceID string) (map[string]bool, error) {
	if requestedWorkspaceID != "" {
		if isAdmin {
			return map[string]bool{requestedWorkspaceID: true}, nil
		}

		count, err := h.db.NewSelect().
			Model((*models.WorkspaceMember)(nil)).
			Where("workspace_id = ? AND user_id = ?", requestedWorkspaceID, userID).
			Count(ctx)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, huma.Error403Forbidden("workspace not accessible")
		}
		return map[string]bool{requestedWorkspaceID: true}, nil
	}

	if isAdmin {
		return nil, nil
	}

	var members []models.WorkspaceMember
	if err := h.db.NewSelect().
		Model(&members).
		Where("user_id = ?", userID).
		Scan(ctx); err != nil {
		return nil, err
	}

	allowed := make(map[string]bool, len(members))
	for _, member := range members {
		allowed[member.WorkspaceID] = true
	}
	return allowed, nil
}
