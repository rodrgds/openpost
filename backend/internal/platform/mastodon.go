package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
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

func (m *MastodonAdapter) ExchangeCode(ctx context.Context, code string, _ map[string]string) (*TokenResult, error) {
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

func (m *MastodonAdapter) RefreshCapability() RefreshCapability {
	return RefreshCapability{
		Supported:        false,
		CredentialSource: RefreshCredentialNone,
	}
}

func (m *MastodonAdapter) RefreshToken(_ context.Context, _ RefreshTokenInput) (*TokenResult, error) {
	return nil, fmt.Errorf("mastodon tokens do not expire")
}

func (m *MastodonAdapter) GetProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	type mastodonProfile struct {
		ID   string `json:"id"`
		Acct string `json:"acct"`
	}

	profile, err := DoBearerJSON[mastodonProfile](ctx, "GET", m.instanceURL+"/api/v1/accounts/verify_credentials", accessToken, nil, "mastodon profile")
	if err != nil {
		return nil, err
	}

	return &UserProfile{
		ID:       profile.ID,
		Username: profile.Acct,
	}, nil
}

func (m *MastodonAdapter) UploadMedia(ctx context.Context, accessToken, _ string, mimeType string, reader io.Reader) (string, error) {
	ext := ".bin"
	if exts, err := mime.ExtensionsByType(mimeType); err == nil && len(exts) > 0 {
		ext = exts[0]
	}

	respBody, err := DoMultipart(
		ctx,
		m.instanceURL+"/api/v2/media",
		"file",
		reader,
		"upload"+ext,
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
	if unmarshalErr := json.Unmarshal(respBody, &mediaResp); unmarshalErr != nil {
		return "", fmt.Errorf("decoding mastodon media: %w", unmarshalErr)
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

func (m *MastodonAdapter) Publish(ctx context.Context, accessToken, _ string, req *PublishRequest) (string, error) {
	// Update alt text for each uploaded media before attaching to the status
	for i, mediaID := range req.PlatformMediaIDs {
		altText := ""
		if i < len(req.MediaAltTexts) {
			altText = req.MediaAltTexts[i]
		}
		if altText != "" {
			_, err := DoFormURLEncoded(ctx, "PUT", m.instanceURL+"/api/v1/media/"+mediaID, map[string]string{
				"description": altText,
			}, map[string]string{
				"Authorization": "Bearer " + accessToken,
			})
			if err != nil {
				return "", fmt.Errorf("updating mastodon media alt text: %w", err)
			}
		}
	}

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
	if unmarshalErr := json.Unmarshal(respBody, &statusResp); unmarshalErr != nil {
		return "", fmt.Errorf("decoding mastodon post: %w", unmarshalErr)
	}

	return statusResp.ID, nil
}
