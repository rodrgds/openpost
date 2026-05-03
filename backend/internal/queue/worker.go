package queue

import (
	"context"
	"encoding/json"
	"log"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/mediastore"
	"github.com/openpost/backend/internal/services/publisher"
	"github.com/openpost/backend/internal/services/tokenmanager"
	"github.com/uptrace/bun"
)

// BackgroundWorker polls the SQLite database for pending jobs.
type BackgroundWorker struct {
	db        *bun.DB
	workerID  string
	interval  time.Duration
	publisher *publisher.Service
	tokens    *tokenmanager.TokenManager
	storage   mediastore.BlobStorage
	done      chan struct{}
}

func NewWorker(db *bun.DB, id string, interval time.Duration, pub *publisher.Service, tokens *tokenmanager.TokenManager, storage mediastore.BlobStorage) *BackgroundWorker {
	return &BackgroundWorker{
		db:        db,
		workerID:  id,
		interval:  interval,
		publisher: pub,
		tokens:    tokens,
		storage:   storage,
		done:      make(chan struct{}),
	}
}

func (w *BackgroundWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Printf("Worker %s started polling every %v\n", w.workerID, w.interval)
	w.processDueJobs(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %s shutting down\n", w.workerID)
			close(w.done)
			return
		case <-ticker.C:
			w.processDueJobs(ctx)
		}
	}
}

// Stop signals the worker to stop and waits for it to finish.
func (w *BackgroundWorker) Stop() {
	<-w.done
}

func (w *BackgroundWorker) processDueJobs(ctx context.Context) {
	for {
		if !w.processNextJobIfAvailable(ctx) {
			return
		}
	}
}

func (w *BackgroundWorker) processNextJobIfAvailable(ctx context.Context) bool {
	job := new(models.Job)

	err := w.db.NewRaw(`
		UPDATE jobs
		SET status = 'processing', locked_at = CURRENT_TIMESTAMP, locked_by = ?
		WHERE id = (
			SELECT id FROM jobs 
			WHERE status = 'pending' AND run_at <= CURRENT_TIMESTAMP
			ORDER BY run_at ASC 
			LIMIT 1
		)
		RETURNING *
	`, w.workerID).Scan(ctx, job)

	if err != nil {
		if err.Error() != "sql: no rows in result set" {
			log.Printf("[Worker %s] database error polling for jobs: %v\n", w.workerID, err)
		}
		return false
	}

	w.handleLockedJob(ctx, job)
	return true
}

func (w *BackgroundWorker) handleLockedJob(ctx context.Context, job *models.Job) {
	log.Printf("[Worker %s] processing job: %s (Type: %s)\n", w.workerID, job.ID, job.Type)

	processErr := w.executeJob(ctx, job)

	if processErr != nil {
		log.Printf("[Worker %s] job %s failed: %v\n", w.workerID, job.ID, processErr)
		job.Attempts++
		if job.Attempts >= job.MaxAttempts {
			job.Status = "failed"
		} else {
			job.Status = "pending"
			backoff := time.Duration(1<<(job.Attempts-1)) * time.Minute
			job.RunAt = time.Now().Add(backoff)
		}
		job.LastError = processErr.Error()

		if _, dbErr := w.db.NewUpdate().Model(job).
			Column("status", "attempts", "last_error", "run_at").
			Where("id = ?", job.ID).
			Exec(ctx); dbErr != nil {
			log.Printf("[Worker %s] failed to update job %s status: %v\n", w.workerID, job.ID, dbErr)
		}
		return
	}

	if _, dbErr := w.db.NewUpdate().Model(job).
		Set("status = ?", "completed").
		Where("id = ?", job.ID).
		Exec(ctx); dbErr != nil {
		log.Printf("[Worker %s] failed to mark job %s as completed: %v\n", w.workerID, job.ID, dbErr)
	}

	log.Printf("[Worker %s] job %s completed successfully\n", w.workerID, job.ID)
}

func (w *BackgroundWorker) executeJob(ctx context.Context, job *models.Job) error {
	// Job handlers will be injected or called from here based on Type
	switch job.Type {
	case "publish_post":
		return w.publisher.HandlePublishJob(ctx, job.Payload)
	case "refresh_token":
		return w.handleRefreshTokenJob(ctx, job.Payload)
	case "media_cleanup":
		return w.handleMediaCleanup(ctx, job.Payload)
	default:
		return nil
	}
}

func (w *BackgroundWorker) handleRefreshTokenJob(ctx context.Context, payload string) error {
	if w.tokens == nil {
		return nil
	}

	accountID, err := tokenmanager.ParseRefreshJobPayload(payload)
	if err != nil {
		return err
	}

	_, err = w.tokens.ForceRefreshAccessToken(ctx, accountID)
	return err
}

func (w *BackgroundWorker) handleMediaCleanup(ctx context.Context, payload string) error {
	var cleanupJob struct {
		WorkspaceID string `json:"workspace_id"`
		Days        int    `json:"days"`
	}
	if err := json.Unmarshal([]byte(payload), &cleanupJob); err != nil {
		return err
	}

	if cleanupJob.Days <= 0 {
		return nil
	}

	cutoff := time.Now().Add(-time.Duration(cleanupJob.Days) * 24 * time.Hour)

	var media []models.MediaAttachment
	err := w.db.NewSelect().Model(&media).
		Where("workspace_id = ?", cleanupJob.WorkspaceID).
		Where("is_favorite = ?", false).
		Where("created_at < ?", cutoff).
		Where("id NOT IN (SELECT media_id FROM post_media)").
		Scan(ctx)
	if err != nil {
		return err
	}

	for _, m := range media {
		if err := w.storage.Delete(filepath.Base(m.FilePath)); err != nil {
			log.Printf("Failed to delete media file %s: %v", m.ID, err)
		}

		var thumbs struct {
			SM string `json:"sm,omitempty"`
			MD string `json:"md,omitempty"`
		}
		if m.ThumbnailsJSON != "" {
			if err := json.Unmarshal([]byte(m.ThumbnailsJSON), &thumbs); err == nil {
				if thumbs.SM != "" {
					if err := w.storage.Delete(thumbs.SM); err != nil {
						log.Printf("Failed to delete thumbnail %s: %v", thumbs.SM, err)
					}
				}
				if thumbs.MD != "" {
					if err := w.storage.Delete(thumbs.MD); err != nil {
						log.Printf("Failed to delete thumbnail %s: %v", thumbs.MD, err)
					}
				}
			}
		}

		if _, err := w.db.NewDelete().Model(&m).Where("id = ?", m.ID).Exec(ctx); err != nil {
			log.Printf("Failed to delete media record %s: %v", m.ID, err)
		}
		log.Printf("Cleaned up media %s for workspace %s", m.ID, cleanupJob.WorkspaceID)
	}

	var workspace models.Workspace
	if err := w.db.NewSelect().Model(&workspace).Where("id = ?", cleanupJob.WorkspaceID).Scan(ctx); err == nil && workspace.MediaCleanupDays > 0 {
		_ = w.scheduleMediaCleanup(ctx, cleanupJob.WorkspaceID, workspace.MediaCleanupDays)
	}

	return nil
}

func (w *BackgroundWorker) scheduleMediaCleanup(ctx context.Context, workspaceID string, days int) error {
	if days <= 0 {
		return nil
	}

	payload, err := json.Marshal(map[string]interface{}{
		"workspace_id": workspaceID,
		"days":         days,
	})
	if err != nil {
		return err
	}

	job := &models.Job{
		ID:      uuid.New().String(),
		Type:    "media_cleanup",
		Payload: string(payload),
		Status:  "pending",
		RunAt:   time.Now().Add(24 * time.Hour),
	}

	_, err = w.db.NewInsert().Model(job).Exec(ctx)
	if err != nil {
		log.Printf("Failed to schedule media cleanup for workspace %s: %v", workspaceID, err)
	}
	return err
}

func (w *BackgroundWorker) CancelMediaCleanup(ctx context.Context, workspaceID string) error {
	_, err := w.db.NewDelete().Model(&models.Job{}).
		Where("type = 'media_cleanup' AND payload LIKE ?", "%"+workspaceID+"%").
		Exec(ctx)
	return err
}

func ScheduleMediaCleanup(db *bun.DB, workspaceID string, days int) error {
	if days <= 0 {
		_, err := db.NewDelete().Model(&models.Job{}).
			Where("type = 'media_cleanup' AND payload LIKE ?", "%"+workspaceID+"%").
			Exec(context.Background())
		return err
	}

	payload, err := json.Marshal(map[string]interface{}{
		"workspace_id": workspaceID,
		"days":         days,
	})
	if err != nil {
		return err
	}

	var existing models.Job
	err = db.NewSelect().Model(&existing).
		Where("type = 'media_cleanup' AND payload LIKE ?", "%"+workspaceID+"%").
		Scan(context.Background())
	if err == nil {
		return nil
	}

	job := &models.Job{
		ID:      uuid.New().String(),
		Type:    "media_cleanup",
		Payload: string(payload),
		Status:  "pending",
		RunAt:   time.Now().Add(24 * time.Hour),
	}

	_, err = db.NewInsert().Model(job).Exec(context.Background())
	if err != nil {
		log.Printf("Failed to schedule media cleanup for workspace %s: %v", workspaceID, err)
	}
	return err
}
