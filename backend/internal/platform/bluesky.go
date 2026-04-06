package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type BlueskyAdapter struct {
	pdsURL string
}

func NewBlueskyAdapter(pdsURL string) *BlueskyAdapter {
	if pdsURL == "" {
		pdsURL = "https://bsky.social"
	}
	return &BlueskyAdapter{pdsURL: pdsURL}
}

func (b *BlueskyAdapter) GenerateAuthURL(state string) (string, map[string]string) {
	return "", nil
}

func (b *BlueskyAdapter) CreateSession(ctx context.Context, handle, appPassword string) (did string, accessToken string, refreshToken string, err error) {
	payload := map[string]string{
		"identifier": handle,
		"password":   appPassword,
	}

	body, err := jsonMarshal(payload)
	if err != nil {
		return "", "", "", err
	}

	respBody, err := DoRequest(ctx, "POST", b.pdsURL+"/xrpc/com.atproto.server.createSession", bytes.NewReader(body), map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil {
		return "", "", "", fmt.Errorf("bluesky create session: %w", err)
	}

	var session struct {
		Did        string `json:"did"`
		Handle     string `json:"handle"`
		AccessJwt  string `json:"accessJwt"`
		RefreshJwt string `json:"refreshJwt"`
	}
	if err := json.Unmarshal(respBody, &session); err != nil {
		return "", "", "", fmt.Errorf("decoding bluesky session: %w", err)
	}

	return session.Did, session.AccessJwt, session.RefreshJwt, nil
}

func (b *BlueskyAdapter) ExchangeCode(ctx context.Context, code string, extra map[string]string) (*TokenResult, error) {
	return nil, fmt.Errorf("bluesky uses app passwords, not OAuth")
}

func (b *BlueskyAdapter) RefreshToken(ctx context.Context, refreshToken string) (*TokenResult, error) {
	respBody, err := DoRequest(ctx, "POST", b.pdsURL+"/xrpc/com.atproto.server.refreshSession", nil, map[string]string{
		"Authorization": "Bearer " + refreshToken,
	})
	if err != nil {
		return nil, fmt.Errorf("bluesky refresh: %w", err)
	}

	var session struct {
		AccessJwt  string `json:"accessJwt"`
		RefreshJwt string `json:"refreshJwt"`
	}
	if err := json.Unmarshal(respBody, &session); err != nil {
		return nil, fmt.Errorf("decoding bluesky refresh: %w", err)
	}

	return &TokenResult{
		AccessToken:  session.AccessJwt,
		RefreshToken: session.RefreshJwt,
	}, nil
}

func (b *BlueskyAdapter) GetProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	respBody, err := DoRequest(ctx, "GET", b.pdsURL+"/xrpc/com.atproto.server.getSession", nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if err != nil {
		return nil, err
	}

	var session struct {
		Did    string `json:"did"`
		Handle string `json:"handle"`
	}
	if err := json.Unmarshal(respBody, &session); err != nil {
		return nil, fmt.Errorf("decoding bluesky session: %w", err)
	}

	return &UserProfile{
		ID:       session.Did,
		Username: session.Handle,
	}, nil
}

func (b *BlueskyAdapter) UploadMedia(ctx context.Context, accessToken, accountID, mimeType string, reader io.Reader) (string, error) {
	respBody, err := DoRequest(ctx, "POST", b.pdsURL+"/xrpc/com.atproto.repo.uploadBlob", reader, map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Content-Type":  mimeType,
	})
	if err != nil {
		return "", fmt.Errorf("bluesky upload blob: %w", err)
	}

	var result struct {
		Blob struct {
			Type string `json:"$type"`
			Ref  struct {
				Link string `json:"$link"`
			} `json:"ref"`
			MimeType string `json:"mimeType"`
			Size     int    `json:"size"`
		} `json:"blob"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decoding bluesky blob: %w", err)
	}

	blobJSON, err := json.Marshal(result.Blob)
	if err != nil {
		return "", fmt.Errorf("encoding bluesky blob: %w", err)
	}

	return string(blobJSON), nil
}

func (b *BlueskyAdapter) Publish(ctx context.Context, accessToken, accountID string, req *PublishRequest) (string, error) {
	record := map[string]interface{}{
		"$type":     "app.bsky.feed.post",
		"text":      req.Content,
		"createdAt": time.Now().UTC().Format(time.RFC3339Nano),
	}

	if len(req.PlatformMediaIDs) > 0 {
		images := make([]map[string]interface{}, 0, len(req.PlatformMediaIDs))
		for _, blobJSON := range req.PlatformMediaIDs {
			var blob map[string]interface{}
			if err := json.Unmarshal([]byte(blobJSON), &blob); err != nil {
				return "", fmt.Errorf("decoding bluesky blob: %w", err)
			}
			images = append(images, map[string]interface{}{
				"alt":   "",
				"image": blob,
			})
		}
		if len(images) > 0 {
			record["embed"] = map[string]interface{}{
				"$type":  "app.bsky.embed.images",
				"images": images,
			}
		}
	}

	if req.ReplyToID != "" {
		var parentRef map[string]interface{}
		if err := json.Unmarshal([]byte(req.ReplyToID), &parentRef); err != nil {
			return "", fmt.Errorf("decoding bluesky reply parent: %w", err)
		}

		rootRef := parentRef
		if rootRef["_root"] != nil {
			if rootMap, ok := rootRef["_root"].(map[string]interface{}); ok {
				rootRef = rootMap
			}
		}

		delete(parentRef, "_root")

		record["reply"] = map[string]interface{}{
			"root":   rootRef,
			"parent": parentRef,
		}
	}

	payload := map[string]interface{}{
		"repo":       accountID,
		"collection": "app.bsky.feed.post",
		"record":     record,
	}

	respBody, err := DoJSON(ctx, "POST", b.pdsURL+"/xrpc/com.atproto.repo.createRecord", payload, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if err != nil {
		return "", fmt.Errorf("posting to bluesky: %w", err)
	}

	var result struct {
		URI string `json:"uri"`
		CID string `json:"cid"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decoding bluesky post: %w", err)
	}

	externalID, _ := json.Marshal(map[string]interface{}{
		"uri":   result.URI,
		"cid":   result.CID,
		"_root": getParentRoot(req.ReplyToID),
	})
	return string(externalID), nil
}

func getParentRoot(replyToID string) interface{} {
	if replyToID == "" {
		return nil
	}
	var parent map[string]interface{}
	if err := json.Unmarshal([]byte(replyToID), &parent); err != nil {
		return nil
	}
	if parent["_root"] != nil {
		return parent["_root"]
	}
	return parent
}
