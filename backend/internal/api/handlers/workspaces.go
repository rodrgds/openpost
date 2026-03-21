package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/openpost/backend/internal/models"
	"github.com/uptrace/bun"
)

type WorkspaceHandler struct {
	db *bun.DB
}

func NewWorkspaceHandler(db *bun.DB) *WorkspaceHandler {
	return &WorkspaceHandler{db: db}
}

type CreateWorkspaceReq struct {
	Name string `json:"name"`
}

func (h *WorkspaceHandler) CreateWorkspace(c echo.Context) error {
	req := new(CreateWorkspaceReq)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Assume Auth middleware sets user_id. MVP defaults to "test-user-1" if missing.
	userID, ok := c.Get("user_id").(string)
	if !ok {
		userID = "test-user-1"
	}

	workspace := &models.Workspace{
		ID:        uuid.New().String(),
		Name:      req.Name,
		CreatedAt: time.Now().UTC(),
	}

	member := &models.WorkspaceMember{
		WorkspaceID: workspace.ID,
		UserID:      userID,
		Role:        "admin", // Creator is always admin
	}

	err := h.db.RunInTx(c.Request().Context(), &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(workspace).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.NewInsert().Model(member).Exec(ctx); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, workspace)
}

func (h *WorkspaceHandler) ListWorkspaces(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok {
		userID = "test-user-1"
	}

	var workspaces []models.Workspace
	err := h.db.NewSelect().
		Model(&workspaces).
		Join("JOIN workspace_members AS wm ON wm.workspace_id = workspace.id").
		Where("wm.user_id = ?", userID).
		Scan(c.Request().Context())

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch workspaces"})
	}

	return c.JSON(http.StatusOK, workspaces)
}
