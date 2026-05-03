package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultLinkedInVersionLagMonths = 1

func linkedInAPIVersion() string {
	if version := os.Getenv("LINKEDIN_API_VERSION"); version != "" {
		return version
	}

	// LinkedIn monthly versions are sometimes not active at the start of a month.
	// Default to previous month to avoid NONEXISTENT_VERSION failures.
	return time.Now().UTC().AddDate(0, -defaultLinkedInVersionLagMonths, 0).Format("200601")
}

type LinkedInAdapter struct {
	clientID             string
	clientSecret         string
	redirectURI          string
	disableThreadReplies bool
}

func NewLinkedInAdapter(clientID, clientSecret, redirectURI string, disableThreadReplies bool) *LinkedInAdapter {
	return &LinkedInAdapter{
		clientID:             clientID,
		clientSecret:         clientSecret,
		redirectURI:          redirectURI,
		disableThreadReplies: disableThreadReplies,
	}
}

func (l *LinkedInAdapter) GenerateAuthURL(state string) (string, map[string]string) {
	scope := "openid profile w_member_social w_member_social_feed"
	if l.disableThreadReplies {
		scope = "openid profile w_member_social"
	}

	params := map[string]string{
		"response_type": "code",
		"client_id":     l.clientID,
		"redirect_uri":  l.redirectURI,
		"scope":         scope,
		"state":         state,
	}
	return "https://www.linkedin.com/oauth/v2/authorization?" + encodeQueryString(params), nil
}

func (l *LinkedInAdapter) ExchangeCode(ctx context.Context, code string, _ map[string]string) (*TokenResult, error) {
	values := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  l.redirectURI,
		"client_id":     l.clientID,
		"client_secret": l.clientSecret,
	}

	respBody, err := DoFormURLEncoded(ctx, "POST", "https://www.linkedin.com/oauth/v2/accessToken", values, nil)
	if err != nil {
		return nil, fmt.Errorf("linkedin token exchange: %w", err)
	}

	var tokenResp struct {
		AccessToken           string `json:"access_token"`
		ExpiresIn             int    `json:"expires_in"`
		RefreshToken          string `json:"refresh_token"`
		RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("decoding linkedin token: %w", err)
	}

	return &TokenResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    "Bearer",
	}, nil
}

func (l *LinkedInAdapter) RefreshCapability() RefreshCapability {
	return RefreshCapability{
		Supported:        true,
		CredentialSource: RefreshCredentialRefreshToken,
	}
}

func (l *LinkedInAdapter) RefreshToken(ctx context.Context, input RefreshTokenInput) (*TokenResult, error) {
	if input.RefreshToken == "" {
		return nil, fmt.Errorf("linkedin refresh requires a refresh token")
	}

	values := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": input.RefreshToken,
		"client_id":     l.clientID,
		"client_secret": l.clientSecret,
	}

	respBody, err := DoFormURLEncoded(ctx, "POST", "https://www.linkedin.com/oauth/v2/accessToken", values, nil)
	if err != nil {
		return nil, fmt.Errorf("linkedin token refresh: %w", err)
	}

	var tokenResp struct {
		AccessToken           string `json:"access_token"`
		ExpiresIn             int    `json:"expires_in"`
		RefreshToken          string `json:"refresh_token"`
		RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("decoding linkedin refresh: %w", err)
	}

	return &TokenResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    "Bearer",
	}, nil
}

func (l *LinkedInAdapter) GetProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	respBody, err := DoJSON(ctx, "GET", "https://api.linkedin.com/v2/userinfo", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if err != nil {
		return nil, err
	}

	var profile struct {
		Sub       string `json:"sub"`
		Name      string `json:"name"`
		GivenName string `json:"given_name"`
	}
	if err := json.Unmarshal(respBody, &profile); err != nil {
		return nil, fmt.Errorf("decoding linkedin profile: %w", err)
	}

	return &UserProfile{
		ID:          profile.Sub,
		Username:    profile.GivenName,
		DisplayName: profile.Name,
	}, nil
}

func (l *LinkedInAdapter) UploadMedia(ctx context.Context, accessToken, personID, mimeType string, reader io.Reader) (string, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("reading media: %w", err)
	}

	isVideo := strings.Contains(mimeType, "video")

	if isVideo {
		return l.uploadVideo(ctx, accessToken, personID, mimeType, data)
	}
	return l.uploadImage(ctx, accessToken, personID, mimeType, data)
}

func (l *LinkedInAdapter) uploadImage(ctx context.Context, accessToken, personID, _ string, data []byte) (string, error) {
	apiVersion := linkedInAPIVersion()

	registerPayload := map[string]interface{}{
		"initializeUploadRequest": map[string]interface{}{
			"owner": "urn:li:person:" + personID,
		},
	}

	respBody, err := DoJSON(ctx, "POST", "https://api.linkedin.com/rest/images?action=initializeUpload", registerPayload, linkedinHeaders(accessToken, apiVersion))
	if err != nil {
		return "", fmt.Errorf("linkedin image register: %w", err)
	}

	return l.completeUpload(ctx, accessToken, respBody, data)
}

func (l *LinkedInAdapter) uploadVideo(ctx context.Context, accessToken, personID, _ string, data []byte) (string, error) {
	apiVersion := linkedInAPIVersion()

	registerPayload := map[string]interface{}{
		"initializeUploadRequest": map[string]interface{}{
			"owner":           "urn:li:person:" + personID,
			"uploadCaptions":  false,
			"uploadThumbnail": false,
		},
	}

	respBody, err := DoJSON(ctx, "POST", "https://api.linkedin.com/rest/videos?action=initializeUpload", registerPayload, linkedinHeaders(accessToken, apiVersion))
	if err != nil {
		return "", fmt.Errorf("linkedin video register: %w", err)
	}

	return l.completeUpload(ctx, accessToken, respBody, data)
}

func (l *LinkedInAdapter) completeUpload(ctx context.Context, accessToken string, registerResp []byte, data []byte) (string, error) {
	var registerResult struct {
		Value struct {
			Image              string `json:"image"`
			DigitalmediaAsset  string `json:"digitalmediaAsset"`
			UploadURL          string `json:"uploadUrl"`
			UploadInstructions struct {
				UploadURL       string `json:"uploadUrl"`
				UploadMechanism struct {
					MediaUploadHTTPRequest struct {
						Headers map[string]string `json:"headers"`
					} `json:"com.linkedin.digitalmedia.uploading.MediaUploadHttpRequest"`
				} `json:"uploadMechanism"`
			} `json:"uploadInstructions"`
		} `json:"value"`
	}
	if err := json.Unmarshal(registerResp, &registerResult); err != nil {
		return "", fmt.Errorf("decoding linkedin register: %w", err)
	}

	uploadURL := registerResult.Value.UploadURL
	if uploadURL == "" {
		uploadURL = registerResult.Value.UploadInstructions.UploadURL
	}
	if uploadURL == "" {
		return "", fmt.Errorf("no upload URL in linkedin response: %s", string(registerResp))
	}

	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Content-Type":  "application/octet-stream",
	}
	extraHeaders := registerResult.Value.UploadInstructions.UploadMechanism.MediaUploadHTTPRequest.Headers
	if auth, ok := extraHeaders["Authorization"]; ok {
		headers["Authorization"] = auth
	}

	_, err := DoRequest(ctx, "PUT", uploadURL, bytes.NewReader(data), headers)
	if err != nil {
		return "", fmt.Errorf("linkedin media PUT upload: %w", err)
	}

	assetURN := registerResult.Value.Image
	if assetURN == "" {
		assetURN = registerResult.Value.DigitalmediaAsset
	}

	if assetURN == "" {
		return "", fmt.Errorf("no asset URN in linkedin response")
	}

	return assetURN, nil
}

func (l *LinkedInAdapter) Publish(ctx context.Context, accessToken, personID string, req *PublishRequest) (string, error) {
	apiVersion := linkedInAPIVersion()
	authorURN := "urn:li:person:" + personID

	if req.ReplyToID != "" {
		return l.postComment(ctx, accessToken, authorURN, req.ReplyToID, req.Content)
	}

	return l.createPost(ctx, accessToken, authorURN, apiVersion, req)
}

func (l *LinkedInAdapter) createPost(ctx context.Context, accessToken, authorURN, apiVersion string, req *PublishRequest) (string, error) {
	payload := map[string]interface{}{
		"author":     authorURN,
		"commentary": req.Content,
		"visibility": "PUBLIC",
		"distribution": map[string]interface{}{
			"feedDistribution":               "MAIN_FEED",
			"targetEntities":                 []interface{}{},
			"thirdPartyDistributionChannels": []interface{}{},
		},
		"lifecycleState":            "PUBLISHED",
		"isReshareDisabledByAuthor": false,
	}

	if len(req.PlatformMediaIDs) > 0 {
		mediaItem := map[string]interface{}{
			"id": req.PlatformMediaIDs[0],
		}
		if len(req.MediaAltTexts) > 0 && req.MediaAltTexts[0] != "" {
			mediaItem["altText"] = req.MediaAltTexts[0]
		}
		payload["content"] = map[string]interface{}{
			"media": mediaItem,
		}
	}

	respHeaders, err := DoJSONWithHeaders(ctx, "POST", "https://api.linkedin.com/rest/posts", payload, linkedinHeaders(accessToken, apiVersion))
	if err != nil {
		return "", fmt.Errorf("posting to linkedin: %w", err)
	}

	postID := respHeaders.Get("x-restli-id")
	if postID == "" {
		return "", nil
	}

	return postID, nil
}

func (l *LinkedInAdapter) postComment(ctx context.Context, accessToken, actorURN, activityURN, content string) (string, error) {
	apiVersion := linkedInAPIVersion()
	encodedActivityURN := url.QueryEscape(activityURN)

	payload := map[string]interface{}{
		"actor":  actorURN,
		"object": activityURN,
		"message": map[string]interface{}{
			"text": content,
		},
	}

	respBody, err := DoJSON(ctx, "POST", "https://api.linkedin.com/rest/socialActions/"+encodedActivityURN+"/comments", payload, linkedinHeaders(accessToken, apiVersion))
	if err != nil {
		return "", fmt.Errorf("posting linkedin comment: %w", err)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decoding linkedin comment: %w", err)
	}

	return result.ID, nil
}

func linkedinHeaders(accessToken, apiVersion string) map[string]string {
	return map[string]string{
		"Authorization":             "Bearer " + accessToken,
		"Content-Type":              "application/json",
		"X-Restli-Protocol-Version": "2.0.0",
		"Linkedin-Version":          apiVersion,
	}
}

func encodeQueryString(params map[string]string) string {
	parts := make([]string, 0, len(params))
	for k, v := range params {
		parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
	}
	return strings.Join(parts, "&")
}

func DoJSONWithHeaders(ctx context.Context, method, url string, payload any, headers map[string]string) (http.Header, error) {
	var bodyReader io.Reader
	if payload != nil {
		data, err := jsonMarshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshaling JSON: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	if headers == nil {
		headers = make(map[string]string)
	}
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s returned %d: %s", method, url, resp.StatusCode, string(respBody))
	}

	return resp.Header, nil
}
