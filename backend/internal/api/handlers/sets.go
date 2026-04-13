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
	"github.com/openpost/backend/internal/services/auth"
	"github.com/uptrace/bun"
)

type SetHandler struct {
	db   *bun.DB
	auth *auth.Service
}

func NewSetHandler(db *bun.DB, authService *auth.Service) *SetHandler {
	return &SetHandler{db: db, auth: authService}
}

type CreateSetInput struct {
	Body struct {
		WorkspaceID string   `json:"workspace_id" doc:"Target workspace ID"`
		Name        string   `json:"name" minLength:"1" doc:"Set name"`
		IsDefault   bool     `json:"is_default" doc:"Set as the default set for this workspace"`
		AccountIDs  []string `json:"account_ids" doc:"Initial account IDs to include in the set"`
	}
}

type CreateSetOutput struct {
	Body *SetResponse
}

type SetAccountResponse struct {
	SocialAccountID string `json:"social_account_id" doc:"Social account ID"`
	Platform        string `json:"platform" doc:"Platform name"`
	AccountUsername string `json:"account_username" doc:"Account username"`
	IsMain          bool   `json:"is_main" doc:"Whether this is the main platform in the set"`
}

type SetResponse struct {
	ID          string               `json:"id" doc:"Set ID"`
	WorkspaceID string               `json:"workspace_id" doc:"Workspace ID"`
	Name        string               `json:"name" doc:"Set name"`
	IsDefault   bool                 `json:"is_default" doc:"Is default set"`
	CreatedAt   string               `json:"created_at" doc:"Creation time"`
	Accounts    []SetAccountResponse `json:"accounts,omitempty" doc:"Accounts in this set"`
}

type ListSetsInput struct {
	WorkspaceID string `query:"workspace_id" doc:"Filter by workspace ID"`
}

type ListSetsOutput struct {
	Body []SetResponse
}

func (h *SetHandler) checkWorkspaceAccess(ctx context.Context, workspaceID, userID string) error {
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

func (h *SetHandler) CreateSet(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-set",
		Method:      http.MethodPost,
		Path:        "/sets",
		Summary:     "Create a social media set",
		Tags:        []string{"Sets"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 403},
	}, func(ctx context.Context, input *CreateSetInput) (*CreateSetOutput, error) {
		userID := middleware.GetUserID(ctx)

		if err := h.checkWorkspaceAccess(ctx, input.Body.WorkspaceID, userID); err != nil {
			return nil, err
		}

		set := &models.SocialMediaSet{
			ID:          uuid.New().String(),
			WorkspaceID: input.Body.WorkspaceID,
			Name:        input.Body.Name,
			IsDefault:   input.Body.IsDefault,
			CreatedAt:   time.Now().UTC(),
		}

		err := h.db.RunInTx(ctx, &sql.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
			if input.Body.IsDefault {
				if _, err := tx.NewUpdate().
					Model((*models.SocialMediaSet)(nil)).
					Set("is_default = ?", false).
					Where("workspace_id = ?", input.Body.WorkspaceID).
					Exec(txCtx); err != nil {
					return err
				}
			}

			if _, err := tx.NewInsert().Model(set).Exec(txCtx); err != nil {
				return err
			}

			for _, accID := range input.Body.AccountIDs {
				setAcc := models.SocialMediaSetAccount{
					SetID:           set.ID,
					SocialAccountID: accID,
					IsMain:          false,
				}
				if _, err := tx.NewInsert().Model(&setAcc).Exec(txCtx); err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to create set")
		}

		resp := &CreateSetOutput{}
		resp.Body = &SetResponse{
			ID:          set.ID,
			WorkspaceID: set.WorkspaceID,
			Name:        set.Name,
			IsDefault:   set.IsDefault,
			CreatedAt:   set.CreatedAt.Format(time.RFC3339),
			Accounts:    []SetAccountResponse{},
		}
		return resp, nil
	})
}

func (h *SetHandler) ListSets(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-sets",
		Method:      http.MethodGet,
		Path:        "/sets",
		Summary:     "List social media sets for a workspace",
		Tags:        []string{"Sets"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
	}, func(ctx context.Context, input *ListSetsInput) (*ListSetsOutput, error) {
		userID := middleware.GetUserID(ctx)

		if err := h.checkWorkspaceAccess(ctx, input.WorkspaceID, userID); err != nil {
			return nil, err
		}

		var sets []models.SocialMediaSet
		err := h.db.NewSelect().
			Model(&sets).
			Where("workspace_id = ?", input.WorkspaceID).
			Order("created_at DESC").
			Scan(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error500InternalServerError("failed to list sets")
		}

		if len(sets) == 0 {
			return &ListSetsOutput{Body: []SetResponse{}}, nil
		}

		setIDs := make([]string, len(sets))
		for i, s := range sets {
			setIDs[i] = s.ID
		}

		var setAccounts []struct {
			SetID           string `bun:"set_id"`
			SocialAccountID string `bun:"social_account_id"`
			Platform        string `bun:"platform"`
			AccountUsername string `bun:"account_username"`
			IsMain          bool   `bun:"is_main"`
		}
		err = h.db.NewSelect().
			TableExpr("social_media_set_accounts AS ssa").
			ColumnExpr("ssa.set_id, ssa.social_account_id, sa.platform, sa.account_username, ssa.is_main").
			Join("JOIN social_accounts AS sa ON sa.id = ssa.social_account_id").
			Where("ssa.set_id IN (?)", bun.In(setIDs)).
			Scan(ctx, &setAccounts)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error500InternalServerError("failed to fetch set accounts")
		}

		accountsBySet := make(map[string][]SetAccountResponse)
		for _, sa := range setAccounts {
			accountsBySet[sa.SetID] = append(accountsBySet[sa.SetID], SetAccountResponse{
				SocialAccountID: sa.SocialAccountID,
				Platform:        sa.Platform,
				AccountUsername: sa.AccountUsername,
				IsMain:          sa.IsMain,
			})
		}

		result := make([]SetResponse, len(sets))
		for i, s := range sets {
			result[i] = SetResponse{
				ID:          s.ID,
				WorkspaceID: s.WorkspaceID,
				Name:        s.Name,
				IsDefault:   s.IsDefault,
				CreatedAt:   s.CreatedAt.Format(time.RFC3339),
				Accounts:    accountsBySet[s.ID],
			}
		}

		return &ListSetsOutput{Body: result}, nil
	})
}

type GetSetInput struct {
	PathID string `path:"id" doc:"Set ID"`
}

type GetSetOutput struct {
	Body *SetResponse
}

func (h *SetHandler) GetSet(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-set",
		Method:      http.MethodGet,
		Path:        "/sets/{id}",
		Summary:     "Get a single social media set",
		Tags:        []string{"Sets"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{404},
	}, func(ctx context.Context, input *GetSetInput) (*GetSetOutput, error) {
		userID := middleware.GetUserID(ctx)

		var set models.SocialMediaSet
		err := h.db.NewSelect().
			Model(&set).
			Where("id = ?", input.PathID).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("set not found")
			}
			return nil, huma.Error500InternalServerError("failed to fetch set")
		}

		if err := h.checkWorkspaceAccess(ctx, set.WorkspaceID, userID); err != nil {
			return nil, err
		}

		var setAccounts []struct {
			SocialAccountID string `bun:"social_account_id"`
			Platform        string `bun:"platform"`
			AccountUsername string `bun:"account_username"`
			IsMain          bool   `bun:"is_main"`
		}
		err = h.db.NewSelect().
			TableExpr("social_media_set_accounts AS ssa").
			ColumnExpr("ssa.social_account_id, sa.platform, sa.account_username, ssa.is_main").
			Join("JOIN social_accounts AS sa ON sa.id = ssa.social_account_id").
			Where("ssa.set_id = ?", input.PathID).
			Scan(ctx, &setAccounts)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, huma.Error500InternalServerError("failed to fetch set accounts")
		}

		accounts := make([]SetAccountResponse, len(setAccounts))
		for i, sa := range setAccounts {
			accounts[i] = SetAccountResponse{
				SocialAccountID: sa.SocialAccountID,
				Platform:        sa.Platform,
				AccountUsername: sa.AccountUsername,
				IsMain:          sa.IsMain,
			}
		}

		return &GetSetOutput{Body: &SetResponse{
			ID:          set.ID,
			WorkspaceID: set.WorkspaceID,
			Name:        set.Name,
			IsDefault:   set.IsDefault,
			CreatedAt:   set.CreatedAt.Format(time.RFC3339),
			Accounts:    accounts,
		}}, nil
	})
}

type UpdateSetInput struct {
	PathID string `path:"id" doc:"Set ID"`
	Body   struct {
		Name      *string `json:"name,omitempty" doc:"Set name"`
		IsDefault *bool   `json:"is_default,omitempty" doc:"Set as default"`
	}
}

type UpdateSetOutput struct {
	Body *SetResponse
}

func (h *SetHandler) UpdateSet(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "update-set",
		Method:      http.MethodPatch,
		Path:        "/sets/{id}",
		Summary:     "Update a social media set",
		Tags:        []string{"Sets"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 403, 404},
	}, func(ctx context.Context, input *UpdateSetInput) (*UpdateSetOutput, error) {
		userID := middleware.GetUserID(ctx)

		var set models.SocialMediaSet
		err := h.db.NewSelect().
			Model(&set).
			Where("id = ?", input.PathID).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("set not found")
			}
			return nil, huma.Error500InternalServerError("failed to fetch set")
		}

		if err := h.checkWorkspaceAccess(ctx, set.WorkspaceID, userID); err != nil {
			return nil, err
		}

		err = h.db.RunInTx(ctx, &sql.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
			if input.Body.IsDefault != nil && *input.Body.IsDefault {
				if _, err := tx.NewUpdate().
					Model((*models.SocialMediaSet)(nil)).
					Set("is_default = ?", false).
					Where("workspace_id = ?", set.WorkspaceID).
					Exec(txCtx); err != nil {
					return err
				}
			}

			if input.Body.Name != nil {
				set.Name = *input.Body.Name
			}
			if input.Body.IsDefault != nil {
				set.IsDefault = *input.Body.IsDefault
			}

			if _, err := tx.NewUpdate().Model(&set).Where("id = ?", set.ID).Exec(txCtx); err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to update set")
		}

		var setAccounts []struct {
			SocialAccountID string `bun:"social_account_id"`
			Platform        string `bun:"platform"`
			AccountUsername string `bun:"account_username"`
			IsMain          bool   `bun:"is_main"`
		}
		h.db.NewSelect().
			TableExpr("social_media_set_accounts AS ssa").
			ColumnExpr("ssa.social_account_id, sa.platform, sa.account_username, ssa.is_main").
			Join("JOIN social_accounts AS sa ON sa.id = ssa.social_account_id").
			Where("ssa.set_id = ?", input.PathID).
			Scan(ctx, &setAccounts)

		accounts := make([]SetAccountResponse, len(setAccounts))
		for i, sa := range setAccounts {
			accounts[i] = SetAccountResponse{
				SocialAccountID: sa.SocialAccountID,
				Platform:        sa.Platform,
				AccountUsername: sa.AccountUsername,
				IsMain:          sa.IsMain,
			}
		}

		return &UpdateSetOutput{Body: &SetResponse{
			ID:          set.ID,
			WorkspaceID: set.WorkspaceID,
			Name:        set.Name,
			IsDefault:   set.IsDefault,
			CreatedAt:   set.CreatedAt.Format(time.RFC3339),
			Accounts:    accounts,
		}}, nil
	})
}

type DeleteSetInput struct {
	PathID string `path:"id" doc:"Set ID"`
}

type DeleteSetOutput struct {
	Body struct {
		Message string `json:"message" doc:"Success message"`
	}
}

func (h *SetHandler) DeleteSet(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-set",
		Method:      http.MethodDelete,
		Path:        "/sets/{id}",
		Summary:     "Delete a social media set",
		Tags:        []string{"Sets"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{403, 404},
	}, func(ctx context.Context, input *DeleteSetInput) (*DeleteSetOutput, error) {
		userID := middleware.GetUserID(ctx)

		var set models.SocialMediaSet
		err := h.db.NewSelect().
			Model(&set).
			Where("id = ?", input.PathID).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("set not found")
			}
			return nil, huma.Error500InternalServerError("failed to fetch set")
		}

		if err := h.checkWorkspaceAccess(ctx, set.WorkspaceID, userID); err != nil {
			return nil, err
		}

		err = h.db.RunInTx(ctx, &sql.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
			if _, err := tx.NewDelete().Model(&models.SocialMediaSetAccount{}).Where("set_id = ?", input.PathID).Exec(txCtx); err != nil {
				return err
			}
			if _, err := tx.NewDelete().Model(&set).Where("id = ?", input.PathID).Exec(txCtx); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to delete set")
		}

		return &DeleteSetOutput{Body: struct {
			Message string `json:"message" doc:"Success message"`
		}{Message: "set deleted successfully"}}, nil
	})
}

type AddSetAccountsInput struct {
	PathID string `path:"id" doc:"Set ID"`
	Body   struct {
		AccountIDs []string `json:"account_ids" doc:"Account IDs to add"`
		IsMain     *bool    `json:"is_main,omitempty" doc:"Mark as main platform"`
	}
}

type AddSetAccountsOutput struct {
	Body *SetResponse
}

func (h *SetHandler) AddSetAccounts(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "add-set-accounts",
		Method:      http.MethodPost,
		Path:        "/sets/{id}/accounts",
		Summary:     "Add accounts to a social media set",
		Tags:        []string{"Sets"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400, 403, 404},
	}, func(ctx context.Context, input *AddSetAccountsInput) (*AddSetAccountsOutput, error) {
		userID := middleware.GetUserID(ctx)

		var set models.SocialMediaSet
		err := h.db.NewSelect().
			Model(&set).
			Where("id = ?", input.PathID).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("set not found")
			}
			return nil, huma.Error500InternalServerError("failed to fetch set")
		}

		if err := h.checkWorkspaceAccess(ctx, set.WorkspaceID, userID); err != nil {
			return nil, err
		}

		isMain := false
		if input.Body.IsMain != nil {
			isMain = *input.Body.IsMain
		}

		err = h.db.RunInTx(ctx, &sql.TxOptions{}, func(txCtx context.Context, tx bun.Tx) error {
			for _, accID := range input.Body.AccountIDs {
				var existing models.SocialMediaSetAccount
				err := tx.NewSelect().
					Model(&existing).
					Where("set_id = ? AND social_account_id = ?", input.PathID, accID).
					Scan(txCtx)
				if err == nil {
					continue
				}
				if !errors.Is(err, sql.ErrNoRows) {
					return err
				}

				setAcc := models.SocialMediaSetAccount{
					SetID:           input.PathID,
					SocialAccountID: accID,
					IsMain:          isMain,
				}
				if _, err := tx.NewInsert().Model(&setAcc).Exec(txCtx); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to add accounts to set")
		}

		var setAccounts []struct {
			SocialAccountID string `bun:"social_account_id"`
			Platform        string `bun:"platform"`
			AccountUsername string `bun:"account_username"`
			IsMain          bool   `bun:"is_main"`
		}
		h.db.NewSelect().
			TableExpr("social_media_set_accounts AS ssa").
			ColumnExpr("ssa.social_account_id, sa.platform, sa.account_username, ssa.is_main").
			Join("JOIN social_accounts AS sa ON sa.id = ssa.social_account_id").
			Where("ssa.set_id = ?", input.PathID).
			Scan(ctx, &setAccounts)

		accounts := make([]SetAccountResponse, len(setAccounts))
		for i, sa := range setAccounts {
			accounts[i] = SetAccountResponse{
				SocialAccountID: sa.SocialAccountID,
				Platform:        sa.Platform,
				AccountUsername: sa.AccountUsername,
				IsMain:          sa.IsMain,
			}
		}

		return &AddSetAccountsOutput{Body: &SetResponse{
			ID:          set.ID,
			WorkspaceID: set.WorkspaceID,
			Name:        set.Name,
			IsDefault:   set.IsDefault,
			CreatedAt:   set.CreatedAt.Format(time.RFC3339),
			Accounts:    accounts,
		}}, nil
	})
}

type RemoveSetAccountInput struct {
	PathID    string `path:"id" doc:"Set ID"`
	PathAccID string `path:"account_id" doc:"Account ID to remove"`
}

type RemoveSetAccountOutput struct {
	Body *SetResponse
}

func (h *SetHandler) RemoveSetAccount(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "remove-set-account",
		Method:      http.MethodDelete,
		Path:        "/sets/{id}/accounts/{account_id}",
		Summary:     "Remove an account from a social media set",
		Tags:        []string{"Sets"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{403, 404},
	}, func(ctx context.Context, input *RemoveSetAccountInput) (*RemoveSetAccountOutput, error) {
		userID := middleware.GetUserID(ctx)

		var set models.SocialMediaSet
		err := h.db.NewSelect().
			Model(&set).
			Where("id = ?", input.PathID).
			Scan(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, huma.Error404NotFound("set not found")
			}
			return nil, huma.Error500InternalServerError("failed to fetch set")
		}

		if err := h.checkWorkspaceAccess(ctx, set.WorkspaceID, userID); err != nil {
			return nil, err
		}

		_, err = h.db.NewDelete().
			Model(&models.SocialMediaSetAccount{}).
			Where("set_id = ? AND social_account_id = ?", input.PathID, input.PathAccID).
			Exec(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to remove account from set")
		}

		var setAccounts []struct {
			SocialAccountID string `bun:"social_account_id"`
			Platform        string `bun:"platform"`
			AccountUsername string `bun:"account_username"`
			IsMain          bool   `bun:"is_main"`
		}
		h.db.NewSelect().
			TableExpr("social_media_set_accounts AS ssa").
			ColumnExpr("ssa.social_account_id, sa.platform, sa.account_username, ssa.is_main").
			Join("JOIN social_accounts AS sa ON sa.id = ssa.social_account_id").
			Where("ssa.set_id = ?", input.PathID).
			Scan(ctx, &setAccounts)

		accounts := make([]SetAccountResponse, len(setAccounts))
		for i, sa := range setAccounts {
			accounts[i] = SetAccountResponse{
				SocialAccountID: sa.SocialAccountID,
				Platform:        sa.Platform,
				AccountUsername: sa.AccountUsername,
				IsMain:          sa.IsMain,
			}
		}

		return &RemoveSetAccountOutput{Body: &SetResponse{
			ID:          set.ID,
			WorkspaceID: set.WorkspaceID,
			Name:        set.Name,
			IsDefault:   set.IsDefault,
			CreatedAt:   set.CreatedAt.Format(time.RFC3339),
			Accounts:    accounts,
		}}, nil
	})
}
