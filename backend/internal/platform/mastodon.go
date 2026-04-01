package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"time"
)

type MastodonAdapter struct {
	clientID     string
	clientSecret string
	redirectURI  string
	instanceURL  string
}

func NewMastodonAdapter(clientID, clientSecret, redirectURI, instanceURL string) *MastodonAdapter {
	return &MastodonAdapter{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		instanceURL:  instanceURL,
	}
}

func (m *MastodonAdapter) InstanceURL() string {
	return m.instanceURL
}

func (m *MastodonAdapter) GenerateAuthURL(state string) (string, map[string]string) {
	params := url.Values{}
	params.Set("client_id", m.clientID)
	params.Set("redirect_uri", m.redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "read write")
	params.Set("state", state)

	return m.instanceURL + "/oauth/authorize?" + params.Encode(), nil
}

func (m *MastodonAdapter) ExchangeCode(ctx context.Context, code string, extra map[string]string) (*TokenResult, error) {
	values := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  m.redirectURI,
		"client_id":     m.clientID,
		"client_secret": m.clientSecret,
	}

	respBody, err := DoFormURLEncoded(ctx, "POST", m.instanceURL+"/oauth/token", values, nil)
	if err != nil {
		return nil, fmt.Errorf("mastodon token exchange: %w", err)
	}

	var tokenResp TokenResult
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("decoding mastodon token: %w", err)
	}

	return &tokenResp, nil
}

func (m *MastodonAdapter) RefreshToken(ctx context.Context, refreshToken string) (*TokenResult, error) {
	return nil, fmt.Errorf("mastodon tokens do not expire")
}

func (m *MastodonAdapter) GetProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	respBody, err := DoJSON(ctx, "GET", m.instanceURL+"/api/v1/accounts/verify_credentials", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if err != nil {
		return nil, err
	}

	var profile struct {
		ID   string `json:"id"`
		Acct string `json:"acct"`
	}
	if err := json.Unmarshal(respBody, &profile); err != nil {
		return nil, fmt.Errorf("decoding mastodon profile: %w", err)
	}

	return &UserProfile{
		ID:       profile.ID,
		Username: profile.Acct,
	}, nil
}

func (m *MastodonAdapter) UploadMedia(ctx context.Context, accessToken, accountID, mimeType string, reader io.Reader) (string, error) {
	respBody, err := DoMultipart(
		ctx,
		m.instanceURL+"/api/v2/media",
		"file",
		reader,
		"upload.bin",
		nil,
		map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
	)
	if err != nil {
		return "", fmt.Errorf("mastodon media upload: %w", err)
	}

	var mediaResp struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(respBody, &mediaResp); err != nil {
		return "", fmt.Errorf("decoding mastodon media: %w", err)
	}

	if mediaResp.URL == "" {
		mediaResp.ID, err = m.waitForMediaProcessing(ctx, accessToken, mediaResp.ID)
		if err != nil {
			return "", err
		}
	}

	return mediaResp.ID, nil
}

func (m *MastodonAdapter) waitForMediaProcessing(ctx context.Context, accessToken, mediaID string) (string, error) {
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)

		respBody, err := DoJSON(ctx, "GET", m.instanceURL+"/api/v1/media/"+mediaID, nil, map[string]string{
			"Authorization": "Bearer " + accessToken,
		})
		if err != nil {
			return "", fmt.Errorf("mastodon media status: %w", err)
		}

		var statusResp struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		}
		if err := json.Unmarshal(respBody, &statusResp); err != nil {
			return "", fmt.Errorf("decoding mastodon media status: %w", err)
		}

		if statusResp.URL != "" {
			return statusResp.ID, nil
		}
	}

	return "", fmt.Errorf("mastodon media processing timed out")
}

func (m *MastodonAdapter) Publish(ctx context.Context, accessToken, accountID string, req *PublishRequest) (string, error) {
	formValues := url.Values{}
	formValues.Set("status", req.Content)
	formValues.Set("visibility", "public")

	for _, mediaID := range req.PlatformMediaIDs {
		formValues.Add("media_ids[]", mediaID)
	}

	if req.ReplyToID != "" {
		formValues.Set("in_reply_to_id", req.ReplyToID)
	}

	respBody, err := DoFormURLEncodedValues(ctx, "POST", m.instanceURL+"/api/v1/statuses", formValues, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if err != nil {
		return "", fmt.Errorf("posting to mastodon: %w", err)
	}

	var statusResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &statusResp); err != nil {
		return "", fmt.Errorf("decoding mastodon post: %w", err)
	}

	return statusResp.ID, nil
}
