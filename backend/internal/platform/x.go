package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dghubble/oauth1"
)

type XAdapter struct {
	consumerKey    string
	consumerSecret string
	redirectURI    string
	requestStore   XRequestStore
	requestMeta    sync.Map
	cleanupDone    chan struct{}
}

type XRequestStore interface {
	Save(requestToken, requestSecret, workspaceID string, createdAt time.Time) error
	Consume(requestToken string, maxAge time.Duration) (XRequestMeta, bool, error)
}

type XRequestMeta struct {
	Secret      string
	WorkspaceID string
	CreatedAt   time.Time
}

func NewXAdapter(clientID, clientSecret, redirectURI string) *XAdapter {
	x := &XAdapter{
		consumerKey:    clientID,
		consumerSecret: clientSecret,
		redirectURI:    redirectURI,
		cleanupDone:    make(chan struct{}),
	}
	go x.cleanupLoop()
	return x
}

func (x *XAdapter) SetRequestStore(store XRequestStore) {
	x.requestStore = store
}

func (x *XAdapter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-x.cleanupDone:
			return
		case <-ticker.C:
			x.purgeOldEntries()
		}
	}
}

func (x *XAdapter) purgeOldEntries() {
	const maxAge = 10 * time.Minute
	now := time.Now()

	x.requestMeta.Range(func(key, value any) bool {
		meta, ok := value.(XRequestMeta)
		if !ok {
			return true
		}
		if now.Sub(meta.CreatedAt) > maxAge {
			x.requestMeta.Delete(key)
		}
		return true
	})
}

func (x *XAdapter) GenerateAuthURL(state string) (string, map[string]string) {
	authURL, err := x.GenerateAuthURLWithError(state)
	if err != nil {
		return "", nil
	}
	return authURL, nil
}

func (x *XAdapter) GenerateAuthURLWithError(workspaceID string) (string, error) {
	callback := x.redirectURI

	config := oauth1.Config{
		ConsumerKey:    x.consumerKey,
		ConsumerSecret: x.consumerSecret,
		CallbackURL:    callback,
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: "https://api.twitter.com/oauth/request_token",
			AuthorizeURL:    "https://api.twitter.com/oauth/authorize",
			AccessTokenURL:  "https://api.twitter.com/oauth/access_token",
		},
	}

	requestToken, requestSecret, err := config.RequestToken()
	if err != nil {
		return "", fmt.Errorf("x oauth1 request token failed: %w", err)
	}
	meta := XRequestMeta{Secret: requestSecret, WorkspaceID: workspaceID, CreatedAt: time.Now().UTC()}
	if x.requestStore != nil {
		if saveErr := x.requestStore.Save(requestToken, meta.Secret, meta.WorkspaceID, meta.CreatedAt); saveErr != nil {
			return "", fmt.Errorf("x oauth1 request token persist failed: %w", saveErr)
		}
	} else {
		x.requestMeta.Store(requestToken, meta)
	}

	authURL, err := config.AuthorizationURL(requestToken)
	if err != nil {
		return "", fmt.Errorf("x oauth1 authorization url failed: %w", err)
	}

	return authURL.String(), nil
}

func (x *XAdapter) GetWorkspaceIDForRequestToken(requestToken string) (string, bool) {
	if x.requestStore != nil {
		meta, ok, err := x.requestStore.Consume(requestToken, 10*time.Minute)
		if err != nil || !ok {
			return "", false
		}
		// Re-store for subsequent token exchange call in same request path.
		x.requestMeta.Store(requestToken, meta)
		return meta.WorkspaceID, true
	}

	metaRaw, ok := x.requestMeta.Load(requestToken)
	if !ok {
		return "", false
	}
	meta := metaRaw.(XRequestMeta)
	return meta.WorkspaceID, true
}

func (x *XAdapter) ExchangeCode(_ context.Context, _ string, extra map[string]string) (*TokenResult, error) {
	oauthToken := extra["oauth_token"]
	oauthVerifier := extra["oauth_verifier"]
	if oauthToken == "" || oauthVerifier == "" {
		return nil, fmt.Errorf("missing oauth_token or oauth_verifier for X token exchange")
	}

	var (
		meta XRequestMeta
		ok   bool
	)

	if x.requestStore != nil {
		consumed, found, err := x.requestStore.Consume(oauthToken, 10*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("x oauth1 request token lookup failed: %w", err)
		}
		if !found {
			return nil, fmt.Errorf("missing request token secret for oauth_token")
		}
		meta = consumed
		ok = true
	} else {
		metaRaw, found := x.requestMeta.Load(oauthToken)
		if !found {
			return nil, fmt.Errorf("missing request token secret for oauth_token")
		}
		meta = metaRaw.(XRequestMeta)
		x.requestMeta.Delete(oauthToken)
		ok = true
	}

	if !ok {
		return nil, fmt.Errorf("missing request token secret for oauth_token")
	}
	requestSecret := meta.Secret

	config := oauth1.Config{
		ConsumerKey:    x.consumerKey,
		ConsumerSecret: x.consumerSecret,
		Endpoint: oauth1.Endpoint{
			RequestTokenURL: "https://api.twitter.com/oauth/request_token",
			AuthorizeURL:    "https://api.twitter.com/oauth/authorize",
			AccessTokenURL:  "https://api.twitter.com/oauth/access_token",
		},
	}

	accessToken, accessSecret, err := config.AccessToken(oauthToken, requestSecret, oauthVerifier)
	if err != nil {
		return nil, fmt.Errorf("x oauth1 access token exchange failed: %w", err)
	}

	combined := accessToken + "|" + accessSecret
	resultExtra := map[string]string{}
	if meta.WorkspaceID != "" {
		resultExtra["_workspace_id"] = meta.WorkspaceID
	}

	return &TokenResult{
		AccessToken: combined,
		TokenType:   "OAuth1",
		Extra:       resultExtra,
	}, nil
}

func (x *XAdapter) RefreshToken(_ context.Context, _ string) (*TokenResult, error) {
	return nil, fmt.Errorf("x oauth1 tokens do not support refresh")
}

func (x *XAdapter) GetProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	respBody, err := x.doSignedRequest(ctx, accessToken, "GET", "https://api.twitter.com/2/users/me?user.fields=id,name,username", nil, nil)
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

func (x *XAdapter) UploadMedia(ctx context.Context, accessToken, _ string, mimeType string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("reading media: %w", err)
	}

	totalBytes := len(data)
	isVideo := strings.Contains(mimeType, "video")
	isGIF := strings.Contains(mimeType, "gif")

	mediaCategory := "tweet_image"
	if isVideo {
		mediaCategory = "tweet_video"
	} else if isGIF {
		mediaCategory = "tweet_gif"
	}

	if totalBytes <= 5*1024*1024 && !isVideo && !isGIF {
		return x.uploadMediaSimple(ctx, accessToken, data, mediaCategory)
	}

	return x.uploadMediaChunked(ctx, accessToken, mimeType, mediaCategory, data, totalBytes)
}

func (x *XAdapter) uploadMediaSimple(ctx context.Context, accessToken string, data []byte, mediaCategory string) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("media_category", mediaCategory); err != nil {
		return "", fmt.Errorf("writing media_category: %w", err)
	}
	part, err := writer.CreateFormFile("media", "upload.bin")
	if err != nil {
		return "", fmt.Errorf("creating media form file: %w", err)
	}
	if _, writeErr := part.Write(data); writeErr != nil {
		return "", fmt.Errorf("writing media content: %w", writeErr)
	}
	if closeErr := writer.Close(); closeErr != nil {
		return "", fmt.Errorf("closing multipart writer: %w", closeErr)
	}

	respBody, err := x.doSignedRequest(ctx, accessToken, "POST", "https://upload.twitter.com/1.1/media/upload.json", &body, map[string]string{
		"Content-Type": writer.FormDataContentType(),
	})
	if err != nil {
		return "", err
	}

	var result struct {
		MediaIDString string `json:"media_id_string"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decoding X media response: %w", err)
	}
	if result.MediaIDString == "" {
		return "", fmt.Errorf("missing media_id_string in X response")
	}
	return result.MediaIDString, nil
}

func (x *XAdapter) uploadMediaChunked(ctx context.Context, accessToken, mimeType, mediaCategory string, data []byte, totalBytes int) (string, error) {
	initValues := url.Values{}
	initValues.Set("command", "INIT")
	initValues.Set("total_bytes", strconv.Itoa(totalBytes))
	initValues.Set("media_type", mimeType)
	initValues.Set("media_category", mediaCategory)

	respBody, err := x.doSignedRequest(ctx, accessToken, "POST", "https://upload.twitter.com/1.1/media/upload.json", strings.NewReader(initValues.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	})
	if err != nil {
		return "", fmt.Errorf("x INIT failed: %w", err)
	}

	var initResp struct {
		MediaIDString  string                `json:"media_id_string"`
		ProcessingInfo *xMediaProcessingInfo `json:"processing_info"`
	}
	if unmarshalErr := json.Unmarshal(respBody, &initResp); unmarshalErr != nil {
		return "", fmt.Errorf("decoding X INIT: %w", unmarshalErr)
	}
	if initResp.MediaIDString == "" {
		return "", fmt.Errorf("missing media_id_string in X INIT")
	}
	mediaID := initResp.MediaIDString

	chunkSize := 5 * 1024 * 1024
	segmentIndex := 0
	for offset := 0; offset < totalBytes; offset += chunkSize {
		end := offset + chunkSize
		if end > totalBytes {
			end = totalBytes
		}

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("command", "APPEND")
		_ = writer.WriteField("media_id", mediaID)
		_ = writer.WriteField("segment_index", strconv.Itoa(segmentIndex))
		part, createErr := writer.CreateFormFile("media", "chunk.bin")
		if createErr != nil {
			return "", fmt.Errorf("x APPEND create form file: %w", createErr)
		}
		if _, writeErr := part.Write(data[offset:end]); writeErr != nil {
			return "", fmt.Errorf("x APPEND write segment %d: %w", segmentIndex, writeErr)
		}
		if closeErr := writer.Close(); closeErr != nil {
			return "", fmt.Errorf("x APPEND close writer: %w", closeErr)
		}

		_, err = x.doSignedRequest(ctx, accessToken, "POST", "https://upload.twitter.com/1.1/media/upload.json", &body, map[string]string{
			"Content-Type": writer.FormDataContentType(),
		})
		if err != nil {
			return "", fmt.Errorf("x APPEND segment %d: %w", segmentIndex, err)
		}
		segmentIndex++
	}

	finalizeValues := url.Values{}
	finalizeValues.Set("command", "FINALIZE")
	finalizeValues.Set("media_id", mediaID)

	respBody, err = x.doSignedRequest(ctx, accessToken, "POST", "https://upload.twitter.com/1.1/media/upload.json", strings.NewReader(finalizeValues.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	})
	if err != nil {
		return "", fmt.Errorf("x FINALIZE: %w", err)
	}

	var finalizeResp struct {
		ProcessingInfo *xMediaProcessingInfo `json:"processing_info"`
	}
	if err := json.Unmarshal(respBody, &finalizeResp); err != nil {
		return "", fmt.Errorf("decoding X FINALIZE: %w", err)
	}

	if finalizeResp.ProcessingInfo != nil {
		if err := x.waitForMediaProcessing(ctx, accessToken, mediaID, finalizeResp.ProcessingInfo); err != nil {
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

		statusURL := "https://upload.twitter.com/1.1/media/upload.json?command=STATUS&media_id=" + url.QueryEscape(mediaID)
		respBody, err := x.doSignedRequest(ctx, accessToken, "GET", statusURL, nil, nil)
		if err != nil {
			return fmt.Errorf("x STATUS check: %w", err)
		}

		var statusResp struct {
			ProcessingInfo *xMediaProcessingInfo `json:"processing_info"`
		}
		if err := json.Unmarshal(respBody, &statusResp); err != nil {
			return fmt.Errorf("decoding X STATUS: %w", err)
		}

		if statusResp.ProcessingInfo == nil {
			return nil
		}
		*info = *statusResp.ProcessingInfo

		if info.State == "failed" {
			return fmt.Errorf("x media processing failed")
		}
	}

	if info.State == "succeeded" {
		return nil
	}
	return fmt.Errorf("x media processing unexpected state: %s", info.State)
}

func (x *XAdapter) Publish(ctx context.Context, accessToken, _ string, req *PublishRequest) (string, error) {
	// Set alt text for each media before posting
	for i, mediaID := range req.PlatformMediaIDs {
		altText := ""
		if i < len(req.MediaAltTexts) {
			altText = req.MediaAltTexts[i]
		}
		if altText != "" {
			metaPayload := map[string]interface{}{
				"media_id": mediaID,
				"alt_text": map[string]string{
					"text": altText,
				},
			}
			metaBody, err := jsonMarshal(metaPayload)
			if err != nil {
				return "", fmt.Errorf("marshaling X media metadata: %w", err)
			}
			_, err = x.doSignedRequest(ctx, accessToken, "POST", "https://upload.twitter.com/1.1/media/metadata/create.json", bytes.NewReader(metaBody), map[string]string{
				"Content-Type": "application/json",
			})
			if err != nil {
				return "", fmt.Errorf("setting X media alt text: %w", err)
			}
		}
	}

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

	body, err := jsonMarshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling X tweet payload: %w", err)
	}

	respBody, err := x.doSignedRequest(ctx, accessToken, "POST", "https://api.twitter.com/2/tweets", bytes.NewReader(body), map[string]string{
		"Content-Type": "application/json",
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

func (x *XAdapter) doSignedRequest(ctx context.Context, combinedAccessToken, method, requestURL string, body io.Reader, headers map[string]string) ([]byte, error) {
	accessToken, accessSecret, err := splitXCombinedToken(combinedAccessToken)
	if err != nil {
		return nil, err
	}

	config := oauth1.NewConfig(x.consumerKey, x.consumerSecret)
	token := oauth1.NewToken(accessToken, accessSecret)
	client := config.Client(ctx, token)

	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s returned %d: %s", method, requestURL, resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func splitXCombinedToken(combined string) (string, string, error) {
	parts := strings.SplitN(combined, "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("x account requires OAuth 1.0a reconnect for media support")
	}
	return parts[0], parts[1], nil
}
