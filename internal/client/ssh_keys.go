// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type SSHKey struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
	CreatedAt   string `json:"created_at"`
}

type sshKeysResponse struct {
	Count   int      `json:"count"`
	Results []SSHKey `json:"results"`
}

func (c *Client) ListSSHKeys(ctx context.Context) ([]SSHKey, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/instances/ssh-keys", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out sshKeysResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode ssh keys: %w", err)
	}
	return out.Results, nil
}
