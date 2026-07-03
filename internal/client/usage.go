// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type UsageItem map[string]interface{}

type UsageSummaryResponse struct {
	CSVPath string `json:"csv_path"`
}

func (c *Client) GetUsageHistory(ctx context.Context, instanceID string) ([]UsageItem, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/instances/"+instanceID+"/get_usage_history", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out []UsageItem
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode usage history: %w", err)
	}
	return out, nil
}

func (c *Client) GetUsageSummary(ctx context.Context) (*UsageSummaryResponse, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/instances/csv-summary", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out UsageSummaryResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode usage summary: %w", err)
	}
	return &out, nil
}
