package handlers

import (
	"io"
	"net/http"
	"path/filepath"

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

func (h *MediaHandler) RegisterRoutes(e *echo.Echo) {
	e.POST("/api/v1/media/upload", h.uploadMedia, middleware.JWTMiddleware(h.auth))
	e.GET("/media/:id", h.serveMedia)
}

func (h *MediaHandler) uploadMedia(c echo.Context) error {
	workspaceID := c.FormValue("workspace_id")
	if workspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "workspace_id is required"})
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file is required"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to open file"})
	}
	defer file.Close()

	header := make([]byte, 512)
	n, _ := file.Read(header)
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

	media := &models.MediaAttachment{
		ID:               mediaID,
		WorkspaceID:      workspaceID,
		FilePath:         savedPath,
		StorageType:      "local",
		MimeType:         mimeType,
		ProcessingStatus: "ready",
		Size:             fileHeader.Size,
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

	if _, err := io.Copy(c.Response().Writer, file); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to stream media"})
	}

	return nil
}
