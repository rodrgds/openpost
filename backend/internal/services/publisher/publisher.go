package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/oauth"
	"github.com/openpost/backend/internal/services/tokenmanager"
	"github.com/uptrace/bun"
)

type Service struct {
	db       *bun.DB
	tm       *tokenmanager.TokenManager
	bluesky  *oauth.BlueskyOAuth
	linkedin *oauth.LinkedInOAuth
	threads  *oauth.ThreadsOAuth
}

func NewService(db *bun.DB, tm *tokenmanager.TokenManager) *Service {
	return &Service{
		db: db,
		tm: tm,
	}
}

// SetBlueskyOAuth sets the Bluesky OAuth provider
func (s *Service) SetBlueskyOAuth(bs *oauth.BlueskyOAuth) {
	s.bluesky = bs
}

// SetLinkedInOAuth sets the LinkedIn OAuth provider
func (s *Service) SetLinkedInOAuth(li *oauth.LinkedInOAuth) {
	s.linkedin = li
}

// SetThreadsOAuth sets the Threads OAuth provider
func (s *Service) SetThreadsOAuth(th *oauth.ThreadsOAuth) {
	s.threads = th
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
	if err := s.db.NewSelect().Model(&dests).Where("post_id = ? AND status IN ('pending', 'failed')", post.ID).Scan(ctx); err != nil {
		return err
	}

	log.Printf("[Publisher] Found %d destinations for post %s", len(dests), post.ID)

	if len(dests) == 0 {
		// Check if all destinations are already successful
		var totalDests int
		totalDests, _ = s.db.NewSelect().Model((*models.PostDestination)(nil)).
			Where("post_id = ?", post.ID).
			Count(ctx)

		if totalDests == 0 {
			log.Printf("[Publisher] No destinations for post %s - marking as published", post.ID)
			_, _ = s.db.NewUpdate().Model(post).Set("status = ?", "published").Where("id = ?", post.ID).Exec(ctx)
		} else {
			// All destinations are either success or failed - check for failures
			var failedCount int
			failedCount, _ = s.db.NewSelect().Model((*models.PostDestination)(nil)).
				Where("post_id = ? AND status = 'failed'", post.ID).
				Count(ctx)
			if failedCount > 0 {
				log.Printf("[Publisher] Post %s has %d failed destinations", post.ID, failedCount)
				_, _ = s.db.NewUpdate().Model(post).Set("status = ?", "failed").Where("id = ?", post.ID).Exec(ctx)
			} else {
				_, _ = s.db.NewUpdate().Model(post).
					Set("status = ?", "published").
					Set("published_at = CURRENT_TIMESTAMP").
					Where("id = ?", post.ID).
					Exec(ctx)
			}
		}
		return nil
	}

	var firstError error

	for _, dest := range dests {
		log.Printf("[Publisher] Publishing to destination %s (account: %s)", dest.ID, dest.SocialAccountID)
		if err := s.publishToDestination(ctx, post, &dest); err != nil {
			firstError = err
			log.Printf("[Publisher] Failed to publish to %s: %v", dest.ID, err)

			_, _ = s.db.NewUpdate().Model(&dest).
				Set("status = ?", "failed").
				Set("error_message = ?", err.Error()).
				Where("id = ?", dest.ID).
				Exec(ctx)
		} else {
			log.Printf("[Publisher] Successfully published to destination %s", dest.ID)
			_, _ = s.db.NewUpdate().Model(&dest).
				Set("status = ?", "success").
				Where("id = ?", dest.ID).
				Exec(ctx)
		}
	}

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
	case "bluesky":
		pdsURL := account.InstanceURL
		if pdsURL == "" {
			pdsURL = "https://bsky.social"
		}
		return s.publishToBluesky(ctx, token, pdsURL, account.AccountID, post.Content)
	case "linkedin":
		return s.publishToLinkedIn(ctx, token, account.AccountID, post.Content)
	case "threads":
		return s.publishToThreads(ctx, token, account.AccountID, post.Content)
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

func (s *Service) publishToBluesky(ctx context.Context, token, pdsURL, did, content string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	payload := map[string]interface{}{
		"repo":       did,
		"collection": "app.bsky.feed.post",
		"record": map[string]interface{}{
			"$type":     "app.bsky.feed.post",
			"text":      content,
			"createdAt": now,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", pdsURL+"/xrpc/com.atproto.repo.createRecord", bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("bluesky API returned status: %d", resp.StatusCode)
	}

	log.Printf("[Publisher] Successfully published to Bluesky")
	return nil
}

func (s *Service) publishToLinkedIn(ctx context.Context, token, personID, content string) error {
	payload := map[string]interface{}{
		"author":     fmt.Sprintf("urn:li:person:%s", personID),
		"commentary": content,
		"visibility": "PUBLIC",
		"distribution": map[string]interface{}{
			"feedDistribution":               "MAIN_FEED",
			"targetEntities":                 []interface{}{},
			"thirdPartyDistributionChannels": []interface{}{},
		},
		"lifecycleState":            "PUBLISHED",
		"isReshareDisabledByAuthor": false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.linkedin.com/rest/posts", bytes.NewReader(body))
	if err != nil {
		return err
	}

	// Use current month as API version (LinkedIn requires YYYYMM format)
	apiVersion := time.Now().UTC().Format("200601")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")
	req.Header.Set("Linkedin-Version", apiVersion)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("linkedin API returned status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[Publisher] Successfully published to LinkedIn")
	return nil
}

func (s *Service) publishToThreads(ctx context.Context, token, userID, content string) error {
	if s.threads == nil {
		return fmt.Errorf("threads oauth not configured")
	}

	_, err := s.threads.PublishTextPost(ctx, token, userID, content)
	if err != nil {
		return fmt.Errorf("failed to publish to threads: %w", err)
	}

	log.Printf("[Publisher] Successfully published to Threads")
	return nil
}
