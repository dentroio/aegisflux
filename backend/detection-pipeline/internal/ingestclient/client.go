package ingestclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// VisibilityEvent mirrors ingest HTTP query rows (subset).
type VisibilityEvent struct {
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
	DeviceID  string          `json:"device_id"`
}

type queryResponse struct {
	OK     bool              `json:"ok"`
	Events []VisibilityEvent `json:"events"`
}

// Client fetches lab telemetry from the ingest visibility API.
type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:9090"
	}
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) QueryEvents(ctx context.Context, deviceID, tenantID, eventType string, limit int) ([]VisibilityEvent, error) {
	u, err := url.Parse(c.baseURL + "/v1/visibility/events")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if deviceID != "" {
		q.Set("device_id", deviceID)
	}
	if tenantID != "" {
		q.Set("tenant_id", tenantID)
	}
	if eventType != "" {
		q.Set("event_type", eventType)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ingest %s: %s", res.Status, truncate(body, 512))
	}
	var parsed queryResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode ingest: %w", err)
	}
	if !parsed.OK {
		return nil, fmt.Errorf("ingest response not ok")
	}
	return parsed.Events, nil
}

// PostVisibilityEvents posts a JSON array or JSONL body to ingest visibility HTTP API.
func (c *Client) PostVisibilityEvents(ctx context.Context, body []byte) error {
	u := c.baseURL + "/v1/visibility/events"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	rb, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("ingest post %s: %s", res.Status, truncate(rb, 512))
	}
	return nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
