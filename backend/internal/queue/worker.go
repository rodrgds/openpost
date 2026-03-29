package queue

import (
	"context"
	"log"
	"time"

	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/publisher"
	"github.com/uptrace/bun"
)

// BackgroundWorker polls the SQLite database for pending jobs
type BackgroundWorker struct {
	db        *bun.DB
	workerID  string
	interval  time.Duration
	publisher *publisher.Service
}

func NewWorker(db *bun.DB, id string, interval time.Duration, pub *publisher.Service) *BackgroundWorker {
	return &BackgroundWorker{
		db:        db,
		workerID:  id,
		interval:  interval,
		publisher: pub,
	}
}

func (w *BackgroundWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Printf("Worker %s started polling every %v\n", w.workerID, w.interval)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %s shutting down\n", w.workerID)
			return
		case <-ticker.C:
			w.processNextJob(ctx)
		}
	}
}

func (w *BackgroundWorker) processNextJob(ctx context.Context) {
	// Attempt to lock a pending job atomically
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
		// Normal condition when queue is empty, or error scanning.
		return
	}

	log.Printf("[Worker %s] processing job: %s (Type: %s)\n", w.workerID, job.ID, job.Type)

	processErr := w.executeJob(ctx, job)

	if processErr != nil {
		log.Printf("[Worker %s] job %s failed: %v\n", w.workerID, job.ID, processErr)
		job.Attempts++
		if job.Attempts >= job.MaxAttempts {
			job.Status = "failed"
		} else {
			job.Status = "pending"
			job.RunAt = time.Now().Add(5 * time.Minute) // Basic backoff
		}
		job.LastError = processErr.Error()

		_, _ = w.db.NewUpdate().Model(job).
			Column("status", "attempts", "last_error", "run_at").
			Where("id = ?", job.ID).
			Exec(ctx)
		return
	}

	// Mark as completed
	_, _ = w.db.NewUpdate().Model(job).
		Set("status = ?", "completed").
		Where("id = ?", job.ID).
		Exec(ctx)

	log.Printf("[Worker %s] job %s completed successfully\n", w.workerID, job.ID)
}

func (w *BackgroundWorker) executeJob(ctx context.Context, job *models.Job) error {
	// Job handlers will be injected or called from here based on Type
	switch job.Type {
	case "publish_post":
		return w.publisher.HandlePublishJob(ctx, job.Payload)
	case "refresh_token":
		// TODO: Call token manager
		return nil
	default:
		return nil
	}
}
