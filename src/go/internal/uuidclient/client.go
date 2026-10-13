// Package uuidclient is the API client the provider uses to talk to its
// upstream service. Replace it with the client for your own API.
package uuidclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	// DefaultEndpoint is used when the provider is not given an endpoint.
	DefaultEndpoint = "https://www.uuidtools.com/api/generate/v1"

	// MaxCount is the largest batch the upstream API returns in one call.
	MaxCount = 100

	maxResponseBytes = 1 << 20
)

// Client generates UUIDs through an HTTP API.
type Client struct {
	endpoint   string
	httpClient *http.Client
	userAgent  string
}

// New returns a Client for the API rooted at endpoint.
func New(endpoint string, httpClient *http.Client, userAgent string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		endpoint:   strings.TrimRight(endpoint, "/"),
		httpClient: httpClient,
		userAgent:  userAgent,
	}
}

// Generate returns count new UUIDs.
func (c *Client) Generate(ctx context.Context, count int) ([]string, error) {
	if count < 1 || count > MaxCount {
		return nil, fmt.Errorf("count must be between 1 and %d, got %d", MaxCount, count)
	}

	url := fmt.Sprintf("%s/count/%d", c.endpoint, count)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s from %s: %s", resp.Status, url, strings.TrimSpace(string(body)))
	}

	var uuids []string
	if err := json.Unmarshal(body, &uuids); err != nil {
		return nil, fmt.Errorf("parsing response body: %w", err)
	}

	if len(uuids) != count {
		return nil, fmt.Errorf("requested %d UUIDs but received %d", count, len(uuids))
	}

	return uuids, nil
}
