// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type InstanceCreate struct {
	Name          string `json:"name,omitempty"`
	FlavorType    string `json:"flavor_type"`
	Region        string `json:"region"`
	ImageName     string `json:"image_name"`
	RAM           int    `json:"ram"`
	Disk          int    `json:"disk"`
	VCPUs         int    `json:"vcpus"`
	EnableIPv6    bool   `json:"enable_ipv6"`
	EnablePrivNet bool   `json:"enable_private_network"`
	AutoBackup    bool   `json:"auto_backup"`
	SSHKeyIDs     []int  `json:"ssh_key_ids,omitempty"`
}

type Instance struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Region      string `json:"region"`
	AccessIPv4  string `json:"accessIPv4"`
	Created     string `json:"created"`
}

type listInstancesResponse struct {
	Count   int        `json:"count"`
	Results []Instance `json:"results"`
}

// CreateInstance POSTs and then polls the list (filtered by name) until the
// instance appears, because the POST response contains only {detail} (no ID).
func (c *Client) CreateInstance(ctx context.Context, req InstanceCreate) (string, error) {
	resp, err := c.Do(ctx, http.MethodPost, "/instances", req, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return "", err
	}
	// POST returns only {detail}; poll list filtered by name until found.
	if req.Name == "" {
		return "", fmt.Errorf("create returned no id and no name was supplied to search for")
	}
	deadline := time.Now().Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		list, err := c.ListInstances(ctx, req.Name)
		if err != nil {
			return "", err
		}
		var match *Instance
		for i := range list {
			if list[i].DisplayName == req.Name {
				if match != nil {
					return "", fmt.Errorf("ambiguous name %q: multiple instances match", req.Name)
				}
				match = &list[i]
			}
		}
		if match != nil {
			return match.ID, nil
		}
		time.Sleep(10 * time.Second)
	}
	return "", fmt.Errorf("timed out waiting for instance %q to appear in list", req.Name)
}

func (c *Client) GetInstance(ctx context.Context, id string) (*Instance, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/instances/"+id, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out Instance
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode instance: %w", err)
	}
	return &out, nil
}

func (c *Client) DeleteInstance(ctx context.Context, id string) error {
	resp, err := c.Do(ctx, http.MethodDelete, "/instances/"+id, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) ListInstances(ctx context.Context, search string) ([]Instance, error) {
	path := "/instances"
	if search != "" {
		path += "?search=" + search
	}
	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out listInstancesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode instances list: %w", err)
	}
	return out.Results, nil
}

// WaitInstanceActive polls GetInstance until Status == "ACTIVE" (UPPERCASE per API).
func (c *Client) WaitInstanceActive(ctx context.Context, id string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		inst, err := c.GetInstance(ctx, id)
		if err == nil && inst.Status == "ACTIVE" {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("timed out waiting for instance %s to become ACTIVE", id)
}

// WaitInstanceDeleted polls GetInstance until HTTP 404.
func (c *Client) WaitInstanceDeleted(ctx context.Context, id string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		resp, err := c.Do(ctx, http.MethodGet, "/instances/"+id, nil, nil)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("timed out waiting for instance %s to be deleted", id)
}
