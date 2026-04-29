package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/platform"
	"github.com/openpost/backend/internal/services/tokenmanager"
	"github.com/uptrace/bun"
)

type Service struct {
	db                           *bun.DB
	tm                           *tokenmanager.TokenManager
	providers                    map[string]platform.Adapter
	disableLinkedInThreadReplies bool
	publicMediaURL               string
}

func NewService(db *bun.DB, tm *tokenmanager.TokenManager) *Service {
	return &Service{
		db:        db,
		tm:        tm,
		providers: make(map[string]platform.Adapter),
	}
}

func (s *Service) SetDisableLinkedInThreadReplies(disable bool) {
	s.disableLinkedInThreadReplies = disable
}

func (s *Service) SetPublicMediaURL(url string) {
	s.publicMediaURL = url
}

func (s *Service) SetProvider(platformName string, adapter platform.Adapter) {
	s.providers[platformName] = adapter
}

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

	var threadPosts []*models.Post
	if post.ThreadSequence == 0 {
		// Try to fetch the full thread in a single recursive CTE query to avoid N+1 DB queries.
		var fetched []models.Post
		cte := `WITH RECURSIVE thread AS (
            SELECT * FROM posts WHERE id = ?
            UNION ALL
            SELECT p.* FROM posts p JOIN thread t ON p.parent_post_id = t.id
        ) SELECT * FROM thread ORDER BY thread_sequence ASC`

		if err := s.db.NewRaw(cte, post.ID).Scan(ctx, &fetched); err == nil && len(fetched) > 0 {
			threadPosts = make([]*models.Post, 0, len(fetched))
			for i := range fetched {
				// copy to avoid referencing loop variable
				p := fetched[i]
				threadPosts = append(threadPosts, &p)
			}
			if len(threadPosts) > 1 {
				log.Printf("[Publisher] Thread detected: %d posts starting from %s", len(threadPosts), post.ID)
			}
		} else {
			// Fallback to iterative fetch if CTE fails for any reason
			threadPosts = append(threadPosts, post)
			currentParentID := post.ID

			for {
				var child models.Post
				err := s.db.NewSelect().Model(&child).
					Where("parent_post_id = ?", currentParentID).
					Order("thread_sequence ASC").
					Limit(1).
					Scan(ctx)

				if err != nil {
					break
				}
				threadPosts = append(threadPosts, &child)
				currentParentID = child.ID
			}

			if len(threadPosts) > 1 {
				log.Printf("[Publisher] Thread detected: %d posts starting from %s", len(threadPosts), post.ID)
			}
		}
	}

	if len(threadPosts) > 1 {
		return s.publishThread(ctx, threadPosts)
	}

	return s.publishSinglePost(ctx, post)
}

func (s *Service) publishSinglePost(ctx context.Context, post *models.Post) error {
	var dests []models.PostDestination
	if err := s.db.NewSelect().Model(&dests).
		Where("post_id = ? AND status IN ('pending', 'failed')", post.ID).
		Scan(ctx); err != nil {
		return err
	}

	log.Printf("[Publisher] Found %d destinations for post %s", len(dests), post.ID)

	if len(dests) == 0 {
		s.finalizePost(ctx, post)
		return nil
	}

	var firstError error
	for _, dest := range dests {
		log.Printf("[Publisher] Publishing to destination %s (account: %s)", dest.ID, dest.SocialAccountID)
		if err := s.publishToDestination(ctx, post, &dest); err != nil {
			firstError = err
			log.Printf("[Publisher] Failed to publish to %s: %v", dest.ID, err)
			if _, dbErr := s.db.NewUpdate().Model(&dest).
				Set("status = ?", "failed").
				Set("error_message = ?", err.Error()).
				Where("id = ?", dest.ID).
				Exec(ctx); dbErr != nil {
				log.Printf("[Publisher] Failed to update destination %s status: %v", dest.ID, dbErr)
			}
		} else {
			log.Printf("[Publisher] Successfully published to destination %s", dest.ID)
			if _, dbErr := s.db.NewUpdate().Model(&dest).
				Set("status = ?", "success").
				Where("id = ?", dest.ID).
				Exec(ctx); dbErr != nil {
				log.Printf("[Publisher] Failed to update destination %s status: %v", dest.ID, dbErr)
			}
		}
	}

	s.finalizePost(ctx, post)
	return firstError
}

func (s *Service) publishThread(ctx context.Context, posts []*models.Post) error {
	log.Printf("[Publisher] Publishing thread with %d posts", len(posts))

	successfulAccounts := make(map[string]bool)

	for i, post := range posts {
		log.Printf("[Publisher] Publishing thread post %d/%d: %s", i+1, len(posts), post.ID)

		var dests []models.PostDestination
		if err := s.db.NewSelect().Model(&dests).
			Where("post_id = ? AND status IN ('pending', 'failed')", post.ID).
			Scan(ctx); err != nil {
			log.Printf("[Publisher] Failed to fetch destinations for post %s: %v", post.ID, err)
			s.finalizePost(ctx, post)
			continue
		}

		if i > 0 {
			var filteredDests []models.PostDestination
			for _, dest := range dests {
				if successfulAccounts[dest.SocialAccountID] {
					filteredDests = append(filteredDests, dest)
				} else {
					if _, dbErr := s.db.NewUpdate().Model(&dest).
						Set("status = ?", "failed").
						Set("error_message = ?", "previous post in thread failed for this account").
						Where("id = ?", dest.ID).
						Exec(ctx); dbErr != nil {
						log.Printf("[Publisher] Failed to update destination %s status: %v", dest.ID, dbErr)
					}
				}
			}
			dests = filteredDests
		}

		var successfulInThisPost []string
		for _, dest := range dests {
			if err := s.publishToDestination(ctx, post, &dest); err != nil {
				log.Printf("[Publisher] Thread post %s failed at destination %s: %v", post.ID, dest.ID, err)
				if _, dbErr := s.db.NewUpdate().Model(&dest).
					Set("status = ?", "failed").
					Set("error_message = ?", err.Error()).
					Where("id = ?", dest.ID).
					Exec(ctx); dbErr != nil {
					log.Printf("[Publisher] Failed to update destination %s status: %v", dest.ID, dbErr)
				}
			} else {
				if _, dbErr := s.db.NewUpdate().Model(&dest).
					Set("status = ?", "success").
					Where("id = ?", dest.ID).
					Exec(ctx); dbErr != nil {
					log.Printf("[Publisher] Failed to update destination %s status: %v", dest.ID, dbErr)
				}
				successfulInThisPost = append(successfulInThisPost, dest.SocialAccountID)
			}
		}

		successfulAccounts = make(map[string]bool)
		for _, accountID := range successfulInThisPost {
			successfulAccounts[accountID] = true
		}

		s.finalizePost(ctx, post)
	}

	return nil
}

func (s *Service) finalizePost(ctx context.Context, post *models.Post) {
	var totalDests int
	totalDests, _ = s.db.NewSelect().Model((*models.PostDestination)(nil)).
		Where("post_id = ?", post.ID).
		Count(ctx)

	if totalDests == 0 {
		if _, err := s.db.NewUpdate().Model(post).Set("status = ?", "published").Where("id = ?", post.ID).Exec(ctx); err != nil {
			log.Printf("[Publisher] Failed to update post %s status: %v", post.ID, err)
		}
		return
	}

	var failedCount int
	failedCount, _ = s.db.NewSelect().Model((*models.PostDestination)(nil)).
		Where("post_id = ? AND status = 'failed'", post.ID).
		Count(ctx)

	if failedCount > 0 {
		if _, err := s.db.NewUpdate().Model(post).Set("status = ?", "failed").Where("id = ?", post.ID).Exec(ctx); err != nil {
			log.Printf("[Publisher] Failed to update post %s status: %v", post.ID, err)
		}
	} else {
		if _, err := s.db.NewUpdate().Model(post).
			Set("status = ?", "published").
			Set("published_at = CURRENT_TIMESTAMP").
			Where("id = ?", post.ID).
			Exec(ctx); err != nil {
			log.Printf("[Publisher] Failed to update post %s status: %v", post.ID, err)
		}
	}
}

//nolint:gocyclo
func (s *Service) publishToDestination(ctx context.Context, post *models.Post, dest *models.PostDestination) error {
	account := new(models.SocialAccount)
	if err := s.db.NewSelect().Model(account).Where("id = ?", dest.SocialAccountID).Scan(ctx); err != nil {
		return fmt.Errorf("account not found: %v", err)
	}

	providerKey := account.Platform
	if account.Platform == "mastodon" {
		providerKey = "mastodon:" + account.InstanceURL
	}

	provider, ok := s.providers[providerKey]
	if !ok {
		return fmt.Errorf("unsupported platform: %s (instance: %s)", account.Platform, account.InstanceURL)
	}

	token, err := s.tm.GetValidAccessToken(ctx, account.ID)
	if err != nil {
		return fmt.Errorf("auth error: %v", err)
	}

	var mediaAttachments []models.MediaAttachment
	if err := s.db.NewSelect().
		TableExpr("post_media AS pm").
		ColumnExpr("ma.*").
		Join("JOIN media_attachments AS ma ON ma.id = pm.media_id").
		Where("pm.post_id = ?", post.ID).
		Order("pm.display_order ASC").
		Scan(ctx, &mediaAttachments); err != nil {
		return fmt.Errorf("fetching media: %v", err)
	}

	var platformMediaIDs []string
	var mediaAltTexts []string
	for _, media := range mediaAttachments {
		mediaID, err := s.uploadMediaToPlatform(ctx, account, provider, token, media)
		if err != nil {
			log.Printf("[Publisher] Failed to upload media %s to %s: %v", media.ID, account.Platform, err)
			return fmt.Errorf("media upload failed for %s: %w", media.ID, err)
		}
		platformMediaIDs = append(platformMediaIDs, mediaID)
		mediaAltTexts = append(mediaAltTexts, media.AltText)
	}

	replyToID := ""
	if post.ThreadSequence > 0 && post.ParentPostID != "" {
		if s.disableLinkedInThreadReplies && account.Platform == "linkedin" {
			return fmt.Errorf("linkedin thread replies are disabled by OPENPOST_DISABLE_LINKEDIN_THREAD_REPLIES")
		}
		replyToID, _ = s.getPreviousPostExternalID(ctx, post.ID, dest.SocialAccountID)
	}

	req := &platform.PublishRequest{
		Content:          post.Content,
		PlatformMediaIDs: platformMediaIDs,
		MediaAltTexts:    mediaAltTexts,
		ReplyToID:        replyToID,
	}

	externalID, err := provider.Publish(ctx, token, account.AccountID, req)
	if err != nil {
		if account.Platform == "bluesky" && isExpiredTokenError(err) {
			log.Printf("[Publisher] Bluesky token expired for account %s, forcing refresh and retry", account.ID)
			refreshedToken, refreshErr := s.tm.ForceRefreshAccessToken(ctx, account.ID)
			if refreshErr != nil {
				return fmt.Errorf("bluesky token refresh failed after expiry: %w", refreshErr)
			}
			externalID, err = provider.Publish(ctx, refreshedToken, account.AccountID, req)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if externalID != "" {
		if _, dbErr := s.db.NewUpdate().Model(dest).
			Set("external_id = ?", externalID).
			Where("id = ?", dest.ID).
			Exec(ctx); dbErr != nil {
			log.Printf("[Publisher] Failed to update external_id for destination %s: %v", dest.ID, dbErr)
		}
	}

	return nil
}

func isExpiredTokenError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "expiredtoken") ||
		strings.Contains(msg, "token has expired") ||
		(strings.Contains(msg, "expired") && strings.Contains(msg, "token"))
}

func (s *Service) uploadMediaToPlatform(ctx context.Context, account *models.SocialAccount, provider platform.Adapter, token string, media models.MediaAttachment) (string, error) {
	if account.Platform == "threads" {
		return s.getPublicMediaURL(media), nil
	}

	data, err := os.ReadFile(media.FilePath)
	if err != nil {
		return "", fmt.Errorf("reading media file %s: %w", media.FilePath, err)
	}

	return provider.UploadMedia(ctx, token, account.AccountID, media.MimeType, bytes.NewReader(data))
}

func (s *Service) getPublicMediaURL(media models.MediaAttachment) string {
	if s.publicMediaURL != "" {
		return s.publicMediaURL + "/" + media.ID
	}
	return "/media/" + media.ID
}

func (s *Service) getPreviousPostExternalID(ctx context.Context, currentPostID, socialAccountID string) (string, error) {
	var parentPost models.Post
	if err := s.db.NewSelect().Model(&parentPost).
		Where("id = (SELECT parent_post_id FROM posts WHERE id = ?)", currentPostID).
		Scan(ctx); err != nil {
		return "", fmt.Errorf("finding parent post: %w", err)
	}

	var parentDest models.PostDestination
	if err := s.db.NewSelect().Model(&parentDest).
		Where("post_id = ? AND social_account_id = ?", parentPost.ID, socialAccountID).
		Scan(ctx); err != nil {
		return "", fmt.Errorf("finding parent destination: %w", err)
	}

	return parentDest.ExternalID, nil
}
