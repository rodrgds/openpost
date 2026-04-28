package handlers

import (
	"context"
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
	Limit  int    `query:"limit" doc:"Number of jobs to return (default 50, max 200)"`
	Status string `query:"status" doc:"Filter by status (pending, processing, completed, failed)"`
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
		limit := input.Limit
		if limit <= 0 || limit > 200 {
			limit = 50
		}

		var jobs []models.Job
		query := h.db.NewSelect().Model(&jobs).Order("run_at DESC").Limit(limit)

		if input.Status != "" {
			query = query.Where("status = ?", input.Status)
		}

		if err := query.Scan(ctx); err != nil {
			return nil, huma.Error500InternalServerError("failed to fetch jobs")
		}

		resp := make([]JobResponse, len(jobs))
		for i, j := range jobs {
			resp[i] = JobResponse{
				ID:          j.ID,
				Type:        j.Type,
				Status:      j.Status,
				Payload:     j.Payload,
				RunAt:       j.RunAt.Format(time.RFC3339),
				Attempts:    j.Attempts,
				MaxAttempts: j.MaxAttempts,
				LastError:   j.LastError,
			}
			if !j.LockedAt.IsZero() {
				resp[i].LockedAt = j.LockedAt.Format(time.RFC3339)
			}
		}

		return &ListJobsOutput{Body: resp}, nil
	})
}
