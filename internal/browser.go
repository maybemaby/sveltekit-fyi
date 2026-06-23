package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type BrowserRenderer interface {
	Capture(ctx context.Context, url string) (io.Reader, error)
}

type CloudflareRenderer struct {
	endpoint string
	apiKey   string
}

type Viewport struct {
	Height int `json:"height"`
	Width  int `json:"width"`
}

type CloudflareScreenshotRequest struct {
	URL         string            `json:"url"`
	Viewport    Viewport          `json:"viewport"`
	GotoOptions map[string]string `json:"gotoOptions,omitempty"`
}

func NewCloudflareRenderer(accountId, apiKey string) (*CloudflareRenderer, error) {

	if accountId == "" || apiKey == "" {
		return nil, errors.New("missing environment variables CLOUDFLARE_ACCOUNT_ID or CLOUDFLARE_API_KEY")
	}

	return &CloudflareRenderer{
		endpoint: fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/browser-rendering/screenshot", accountId),
		apiKey:   apiKey,
	}, nil
}

func (r *CloudflareRenderer) Capture(ctx context.Context, url string) (io.Reader, error) {

	ctx, cancel := context.WithTimeoutCause(ctx, 10*time.Second, errors.New("screenshot request timed out for "+url))
	defer cancel()

	reqData := CloudflareScreenshotRequest{
		URL: url,
		Viewport: Viewport{
			Height: 800,
			Width:  1280,
		},
		GotoOptions: map[string]string{
			"waitUntil": "networkidle0",
		},
	}

	data := bytes.NewBuffer(nil)

	err := json.NewEncoder(data).Encode(reqData)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, data)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+r.apiKey)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code %d", res.StatusCode)
	}

	image, err := io.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	return bytes.NewReader(image), nil
}
