package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type InstanceOption struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Prices      OptionPrices `json:"prices"`
	Region      OptionRef    `json:"region"`
	MemoryMB    int          `json:"memory_mb"`
	VCPUs       int          `json:"vcpus"`
	RootGB      int          `json:"root_gb"`
	FlavorGroup FlavorGroup  `json:"flavor_group"`
}

type OptionPrices struct {
	PricePerMonth float64 `json:"price_per_month"`
	PricePerHour  float64 `json:"price_per_hour"`
}

type OptionRef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type FlavorGroup struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	MaxIPAddon  int    `json:"max_ip_addon"`
}

type instanceOptionsResponse struct {
	Configs []InstanceOption `json:"configs"`
}

func (c *Client) GetInstanceOptions(ctx context.Context) ([]InstanceOption, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/instances/get_options", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out instanceOptionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode instance options: %w", err)
	}
	return out.Configs, nil
}
