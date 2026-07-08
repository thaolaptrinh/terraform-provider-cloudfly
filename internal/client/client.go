// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	DefaultBaseURL = "https://api.cloudfly.vn/backend/api"
	defaultTimeout = 30 * time.Second
	retryMax       = 4
)

type Config struct {
	APIToken   string
	BaseURL    string
	HTTPClient *http.Client // optional override (tests/advanced)
}

type Client struct {
	BaseURL    string
	APIToken   string
	HTTPClient *http.Client
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIToken) == "" {
		return nil, fmt.Errorf("api_token is required")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	rc := retryablehttp.NewClient()
	rc.HTTPClient = httpClient
	rc.RetryMax = retryMax
	rc.RetryWaitMin = 200 * time.Millisecond
	rc.RetryWaitMax = 5 * time.Second
	rc.Logger = nil

	std := rc.StandardClient()
	std.Timeout = defaultTimeout

	return &Client{
		BaseURL:    baseURL,
		APIToken:   cfg.APIToken,
		HTTPClient: std,
	}, nil
}

func (c *Client) Do(ctx context.Context, method, path string, body interface{}, out interface{}) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(buf)
	}

	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.APIToken)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	return resp, nil
}
