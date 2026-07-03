// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type MetricsResponse struct {
	Data json.RawMessage
}

func (c *Client) GetMetrics(ctx context.Context, instanceID, metricType, startTime string) (*MetricsResponse, error) {
	path := fmt.Sprintf("/instances/%s/metrics?metrcic_type=%s&start_time=%s",
		instanceID, url.QueryEscape(metricType), url.QueryEscape(startTime))
	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode metrics: %w", err)
	}
	return &MetricsResponse{Data: raw}, nil
}
