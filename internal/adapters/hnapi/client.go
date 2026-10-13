// Package hnapi is the HTTP adapter for ports.HNClient against the public
// Hacker News Firebase API (https://github.com/HackerNews/API).
package hnapi

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

	"interview-process/internal/core/domain"
	"interview-process/internal/core/ports"
)

const DefaultBaseURL = "https://hacker-news.firebaseio.com/v0/"

// Client implements ports.HNClient over HTTP.
type Client struct {
	baseURL string
	http    *http.Client
	retries int
	backoff time.Duration
}

var _ ports.HNClient = (*Client)(nil)

type Option func(*Client)

func WithBaseURL(u string) Option          { return func(c *Client) { c.baseURL = u } }
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }
func WithRetries(n int) Option             { return func(c *Client) { c.retries = n } }
func WithBackoff(d time.Duration) Option   { return func(c *Client) { c.backoff = d } }

func New(opts ...Option) *Client {
	c := &Client{
		baseURL: DefaultBaseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
		retries: 3,
		backoff: 200 * time.Millisecond,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *Client) TopStoryIDs(ctx context.Context) ([]int, error) {
	var ids []int
	if _, err := c.get(ctx, "topstories.json", &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (c *Client) Item(ctx context.Context, id int) (*domain.Item, error) {
	var it domain.Item
	found, err := c.get(ctx, "item/"+strconv.Itoa(id)+".json", &it)
	if err != nil || !found {
		return nil, err
	}
	return &it, nil
}

func (c *Client) User(ctx context.Context, id string) (*domain.User, error) {
	var u domain.User
	found, err := c.get(ctx, "user/"+url.PathEscape(id)+".json", &u)
	if err != nil || !found {
		return nil, err
	}
	return &u, nil
}

// get fetches path into v. It reports found=false when the API returns a
// JSON null. Network errors, 429 and 5xx are retried with exponential backoff.
func (c *Client) get(ctx context.Context, path string, v any) (found bool, err error) {
	for attempt := 0; ; attempt++ {
		var body []byte
		var retryable bool
		body, retryable, err = c.do(ctx, c.baseURL+path)
		if err == nil {
			if bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
				return false, nil
			}
			if err := json.Unmarshal(body, v); err != nil {
				return false, fmt.Errorf("decode %s: %w", path, err)
			}
			return true, nil
		}
		if !retryable || attempt >= c.retries {
			return false, err
		}
		select {
		case <-time.After(c.backoff << attempt):
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
}

func (c *Client) do(ctx context.Context, u string) (body []byte, retryable bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, false, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, ctx.Err() == nil, err
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	if resp.StatusCode != http.StatusOK {
		retryable = resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		return nil, retryable, fmt.Errorf("GET %s: %s", u, resp.Status)
	}
	return body, false, nil
}
