// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type PriceRequest struct {
	FlavorType string `json:"flavor_type"`
	RAM        int    `json:"ram"`
	Disk       int    `json:"disk"`
	VCPUs      int    `json:"vcpus"`
	Region     string `json:"region"`
	ImageName  string `json:"image_name"`
}

type Price struct {
	PricePerMonth float64 `json:"price_per_month"`
	PricePerHour  float64 `json:"price_per_hour"`
}

func (c *Client) GetPrice(ctx context.Context, req PriceRequest) (*Price, error) {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/get_price", req, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out Price
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode price: %w", err)
	}
	return &out, nil
}
