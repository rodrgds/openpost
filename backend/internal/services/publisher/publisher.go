package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/tokenmanager"
	"github.com/uptrace/bun"
)

type Service struct {
	db *bun.DB
	tm *tokenmanager.TokenManager
}

func NewService(db *bun.DB, tm *tokenmanager.TokenManager) *Service {
	return &Service{db: db, tm: tm}
}

// HandlePublishJob is called by the queue worker
func (s *Service) HandlePublishJob(ctx context.Context, jobPayload string) error {
	var payload struct {
		PostID string `json:"post_id"`
	}
	if err := json.Unmarshal([]byte(jobPayload), &payload); err != nil {
		return err
	}

	log.Printf("[Publisher] Processing post %s", payload.PostID)

	post := new(models.Post)
	if err := s.db.NewSelect().Model(post).Where("id = ?", payload.PostID).Scan(ctx); err != nil {
		return err
	}

	var dests []models.PostDestination
	if err := s.db.NewSelect().Model(&dests).Where("post_id = ? AND status = 'pending'", post.ID).Scan(ctx); err != nil {
		return err
	}

	log.Printf("[Publisher] Found %d destinations for post %s", len(dests), post.ID)

	if len(dests) == 0 {
		log.Printf("[Publisher] No destinations for post %s - marking as published", post.ID)
		_, _ = s.db.NewUpdate().Model(post).Set("status = ?", "published").Where("id = ?", post.ID).Exec(ctx)
		return nil
	}

	var firstError error

	for _, dest := range dests {
		log.Printf("[Publisher] Publishing to destination %s (account: %s)", dest.ID, dest.SocialAccountID)
		if err := s.publishToDestination(ctx, post, &dest); err != nil {
			firstError = err
			log.Printf("[Publisher] Failed to publish to %s: %v", dest.ID, err)

			// Update destination status
			_, _ = s.db.NewUpdate().Model(&dest).
				Set("status = ?", "failed").
				Set("error_message = ?", err.Error()).
				Where("id = ?", dest.ID).
				Exec(ctx)
		} else {
			log.Printf("[Publisher] Successfully published to destination %s", dest.ID)
			// Success
			_, _ = s.db.NewUpdate().Model(&dest).
				Set("status = ?", "success").
				Where("id = ?", dest.ID).
				Exec(ctx)
		}
	}

	// Update Post overall status
	if firstError != nil {
		_, _ = s.db.NewUpdate().Model(post).Set("status = ?", "failed").Where("id = ?", post.ID).Exec(ctx)
	} else {
		_, _ = s.db.NewUpdate().Model(post).
			Set("status = ?", "published").
			Set("published_at = CURRENT_TIMESTAMP").
			Where("id = ?", post.ID).
			Exec(ctx)
	}

	return firstError
}

func (s *Service) publishToDestination(ctx context.Context, post *models.Post, dest *models.PostDestination) error {
	account := new(models.SocialAccount)
	if err := s.db.NewSelect().Model(account).Where("id = ?", dest.SocialAccountID).Scan(ctx); err != nil {
		return fmt.Errorf("account not found: %v", err)
	}

	log.Printf("[Publisher] Publishing to %s (platform: %s)", account.AccountID, account.Platform)

	token, err := s.tm.GetValidAccessToken(ctx, account.ID)
	if err != nil {
		return fmt.Errorf("auth error: %v", err)
	}

	switch account.Platform {
	case "x":
		return s.publishToX(ctx, token, post.Content)
	case "mastodon":
		if account.InstanceURL == "" {
			return fmt.Errorf("mastodon instance URL is missing")
		}
		return s.publishToMastodon(ctx, token, account.InstanceURL, post.Content)
	default:
		return fmt.Errorf("unsupported platform: %s", account.Platform)
	}
}

func (s *Service) publishToX(ctx context.Context, token, content string) error {
	payload := fmt.Sprintf(`{"text":"%s"}`, strings.ReplaceAll(content, `"`, `\"`))
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.twitter.com/2/tweets", strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("X API returned status: %d", resp.StatusCode)
	}
	return nil
}

func (s *Service) publishToMastodon(ctx context.Context, token, instanceURL, content string) error {
	data := fmt.Sprintf("status=%s", url.QueryEscape(content))
	req, _ := http.NewRequestWithContext(ctx, "POST", instanceURL+"/api/v1/statuses", strings.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mastodon API returned status: %d", resp.StatusCode)
	}
	return nil
}
