package gcclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Options struct {
	EndpointOverride string
	TokenOverride    string
	Timeout          time.Duration
}

type Client struct {
	baseURL *url.URL
	token   string
	hc      *http.Client
}

type WhitelistMutateRequest struct {
	Member      string `json:"member"`
	WhitelistID string `json:"whitelistId,omitempty"`
}

type WhitelistMutateResponse struct {
	Member     string          `json:"member"`
	Already    bool            `json:"already,omitempty"`
	NotPresent bool            `json:"notPresent,omitempty"`
	Tx         json.RawMessage `json:"tx,omitempty"`
}

func NewFromEnv(opt Options) (*Client, error) {
	endpoint := strings.TrimSpace(opt.EndpointOverride)
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("GC_ENDPOINT"))
	}
	if endpoint == "" {
		return nil, fmt.Errorf("GC_ENDPOINT is not set (or pass --gc-endpoint)")
	}

	// Handle missing scheme
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid GC_ENDPOINT: %w", err)
	}
	u.Path = strings.TrimRight(u.Path, "/")

	token := strings.TrimSpace(opt.TokenOverride)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("GC_API_TOKEN"))
	}

	to := opt.Timeout
	if to == 0 {
		to = 15 * time.Second
	}

	return &Client{
		baseURL: u,
		token:   token,
		hc:      &http.Client{Timeout: to},
	}, nil
}

func (c *Client) WhitelistAdd(ctx context.Context, req WhitelistMutateRequest) (*WhitelistMutateResponse, error) {
	return c.post(ctx, "/v1/whitelist/add", req)
}

func (c *Client) WhitelistRemove(ctx context.Context, req WhitelistMutateRequest) (*WhitelistMutateResponse, error) {
	return c.post(ctx, "/v1/whitelist/remove", req)
}

func (c *Client) post(ctx context.Context, path string, body any) (*WhitelistMutateResponse, error) {
	full := c.baseURL.ResolveReference(&url.URL{Path: c.baseURL.Path + path})

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, full.String(), bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	r.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		r.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.hc.Do(r)
	if err != nil {
		return nil, fmt.Errorf("gc api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if strings.TrimSpace(e.Error) == "" {
			return nil, fmt.Errorf("gc api error: status=%d", resp.StatusCode)
		}
		return nil, fmt.Errorf("gc api error: %s", e.Error)
	}

	var out WhitelistMutateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}
