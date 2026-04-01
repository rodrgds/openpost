package platform

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type XAdapter struct {
	config        *oauth2.Config
	verifierStore sync.Map
}

func NewXAdapter(clientID, clientSecret, redirectURI string) *XAdapter {
	return &XAdapter{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://twitter.com/i/oauth2/authorize",
				TokenURL: "https://api.twitter.com/2/oauth2/token",
			},
			Scopes: []string{"tweet.read", "tweet.write", "users.read", "offline.access"},
		},
	}
}

func (x *XAdapter) GenerateAuthURL(state string) (string, map[string]string) {
	codeVerifier := oauth2.GenerateVerifier()
	authURL := x.config.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(codeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	return authURL, map[string]string{
		"code_verifier": codeVerifier,
	}
}

func (x *XAdapter) ExchangeCode(ctx context.Context, code string, extra map[string]string) (*TokenResult, error) {
	codeVerifier := extra["code_verifier"]
	if codeVerifier == "" {
		return nil, fmt.Errorf("missing code_verifier for X token exchange")
	}

	values := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  x.config.RedirectURL,
		"code_verifier": codeVerifier,
		"client_id":     x.config.ClientID,
	}

	headers := map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Authorization": basicAuthHeader(x.config.ClientID, x.config.ClientSecret),
	}

	respBody, err := DoFormURLEncoded(ctx, "POST", x.config.Endpoint.TokenURL, values, headers)
	if err != nil {
		return nil, err
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("decoding X token response: %w", err)
	}

	return &TokenResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    tokenResp.TokenType,
	}, nil
}

func (x *XAdapter) RefreshToken(ctx context.Context, refreshToken string) (*TokenResult, error) {
	values := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     x.config.ClientID,
	}

	headers := map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Authorization": basicAuthHeader(x.config.ClientID, x.config.ClientSecret),
	}

	respBody, err := DoFormURLEncoded(ctx, "POST", x.config.Endpoint.TokenURL, values, headers)
	if err != nil {
		return nil, err
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("decoding X refresh response: %w", err)
	}

	return &TokenResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    tokenResp.TokenType,
	}, nil
}

func (x *XAdapter) GetProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	respBody, err := DoJSON(ctx, "GET", "https://api.twitter.com/2/users/me?user.fields=id,name,username", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if err != nil {
		return nil, err
	}

	var userResp struct {
		Data struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Username string `json:"username"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &userResp); err != nil {
		return nil, fmt.Errorf("decoding X profile: %w", err)
	}

	return &UserProfile{
		ID:          userResp.Data.ID,
		Username:    userResp.Data.Username,
		DisplayName: userResp.Data.Name,
	}, nil
}

func (x *XAdapter) UploadMedia(ctx context.Context, accessToken, accountID, mimeType string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("reading media: %w", err)
	}

	totalBytes := len(data)
	isVideo := strings.Contains(mimeType, "video")
	isGIF := strings.Contains(mimeType, "gif")

	if totalBytes <= 5*1024*1024 && !isVideo && !isGIF {
		return x.uploadMediaSimple(ctx, accessToken, data)
	}

	mediaCategory := "tweet_image"
	if isVideo {
		mediaCategory = "tweet_video"
	} else if isGIF {
		mediaCategory = "tweet_gif"
	}
	return x.uploadMediaChunked(ctx, accessToken, mimeType, mediaCategory, data, totalBytes)
}

func (x *XAdapter) uploadMediaSimple(ctx context.Context, accessToken string, data []byte) (string, error) {
	respBody, err := DoMultipart(
		ctx,
		"https://api.twitter.com/2/media/upload",
		"media",
		bytes.NewReader(data),
		"upload.bin",
		map[string]string{"media_category": "tweet_image"},
		map[string]string{"Authorization": "Bearer " + accessToken},
	)
	if err != nil {
		return "", err
	}

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decoding X media response: %w", err)
	}
	return result.Data.ID, nil
}

func (x *XAdapter) uploadMediaChunked(ctx context.Context, accessToken, mimeType, mediaCategory string, data []byte, totalBytes int) (string, error) {
	initURL := fmt.Sprintf(
		"https://api.twitter.com/2/media/upload?command=INIT&total_bytes=%d&media_type=%s&media_category=%s",
		totalBytes, mimeType, mediaCategory,
	)

	respBody, err := DoJSON(ctx, "POST", initURL, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if err != nil {
		return "", fmt.Errorf("X INIT failed: %w", err)
	}

	var initResp struct {
		Data struct {
			ID             string                `json:"id"`
			ProcessingInfo *xMediaProcessingInfo `json:"processing_info"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &initResp); err != nil {
		return "", fmt.Errorf("decoding X INIT: %w", err)
	}
	mediaID := initResp.Data.ID

	chunkSize := 5 * 1024 * 1024
	segmentIndex := 0
	for offset := 0; offset < totalBytes; offset += chunkSize {
		end := offset + chunkSize
		if end > totalBytes {
			end = totalBytes
		}

		appendURL := fmt.Sprintf(
			"https://api.twitter.com/2/media/upload?command=APPEND&media_id=%s&segment_index=%d",
			mediaID, segmentIndex,
		)

		_, err := DoMultipart(
			ctx,
			appendURL,
			"media",
			bytes.NewReader(data[offset:end]),
			"chunk.bin",
			nil,
			map[string]string{"Authorization": "Bearer " + accessToken},
		)
		if err != nil {
			return "", fmt.Errorf("X APPEND segment %d: %w", segmentIndex, err)
		}
		segmentIndex++
	}

	finalizeURL := fmt.Sprintf("https://api.twitter.com/2/media/upload?command=FINALIZE&media_id=%s", mediaID)
	respBody, err = DoJSON(ctx, "POST", finalizeURL, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if err != nil {
		return "", fmt.Errorf("X FINALIZE: %w", err)
	}

	var finalizeResp struct {
		Data struct {
			ProcessingInfo *xMediaProcessingInfo `json:"processing_info"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &finalizeResp); err != nil {
		return "", fmt.Errorf("decoding X FINALIZE: %w", err)
	}

	if finalizeResp.Data.ProcessingInfo != nil {
		if err := x.waitForMediaProcessing(ctx, accessToken, mediaID, finalizeResp.Data.ProcessingInfo); err != nil {
			return "", err
		}
	}

	return mediaID, nil
}

type xMediaProcessingInfo struct {
	State           string `json:"state"`
	CheckAfterSecs  int    `json:"check_after_secs"`
	ProgressPercent int    `json:"progress_percent"`
}

func (x *XAdapter) waitForMediaProcessing(ctx context.Context, accessToken, mediaID string, info *xMediaProcessingInfo) error {
	for info.State == "pending" || info.State == "in_progress" {
		if info.CheckAfterSecs > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(info.CheckAfterSecs) * time.Second):
			}
		}

		statusURL := fmt.Sprintf("https://api.twitter.com/2/media/upload?command=STATUS&media_id=%s", mediaID)
		respBody, err := DoJSON(ctx, "GET", statusURL, nil, map[string]string{
			"Authorization": "Bearer " + accessToken,
		})
		if err != nil {
			return fmt.Errorf("X STATUS check: %w", err)
		}

		var statusResp struct {
			Data struct {
				ProcessingInfo *xMediaProcessingInfo `json:"processing_info"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respBody, &statusResp); err != nil {
			return fmt.Errorf("decoding X STATUS: %w", err)
		}

		if statusResp.Data.ProcessingInfo == nil {
			return nil
		}
		*info = *statusResp.Data.ProcessingInfo

		if info.State == "failed" {
			return fmt.Errorf("X media processing failed")
		}
	}

	if info.State == "succeeded" {
		return nil
	}
	return fmt.Errorf("X media processing unexpected state: %s", info.State)
}

func (x *XAdapter) Publish(ctx context.Context, accessToken, accountID string, req *PublishRequest) (string, error) {
	payload := map[string]interface{}{
		"text": req.Content,
	}

	if len(req.PlatformMediaIDs) > 0 {
		payload["media"] = map[string]interface{}{
			"media_ids": req.PlatformMediaIDs,
		}
	}

	if req.ReplyToID != "" {
		payload["reply"] = map[string]interface{}{
			"in_reply_to_tweet_id": req.ReplyToID,
		}
	}

	respBody, err := DoJSON(ctx, "POST", "https://api.twitter.com/2/tweets", payload, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if err != nil {
		return "", fmt.Errorf("posting to X: %w", err)
	}

	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decoding X post response: %w", err)
	}

	return result.Data.ID, nil
}

func basicAuthHeader(username, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}

func (x *XAdapter) GetVerifier(state string) (string, bool) {
	if v, ok := x.verifierStore.Load(state); ok {
		x.verifierStore.Delete(state)
		return v.(string), true
	}
	return "", false
}

func (x *XAdapter) StoreVerifier(state, verifier string) {
	x.verifierStore.Store(state, verifier)
}
