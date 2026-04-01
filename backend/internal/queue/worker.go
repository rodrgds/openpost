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
		// Check if it's just no jobs available vs a real error
		if err.Error() != "sql: no rows in result set" {
			log.Printf("[Worker %s] database error polling for jobs: %v\n", w.workerID, err)
		}
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
			// Exponential backoff: 1 min, 2 min, 4 min, 8 min...
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

	// Mark as completed
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
		// TODO: Call token manager
		return nil
	default:
		return nil
	}
}
