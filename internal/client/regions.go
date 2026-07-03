package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Region struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type regionsResponse struct {
	Count   int      `json:"count"`
	Results []Region `json:"results"`
}

func (c *Client) ListRegions(ctx context.Context) ([]Region, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/cloud-regions", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out regionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode regions: %w", err)
	}
	return out.Results, nil
}
