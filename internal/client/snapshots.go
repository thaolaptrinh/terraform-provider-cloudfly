// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Snapshot represents a CloudFly instance snapshot.
type Snapshot struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	Size         int64  `json:"size"`
	SizeInGB     string `json:"size_in_gb"`
	Type         string `json:"type"`
	OSDistro     string `json:"os_distro"`
	CreatedAt    string `json:"created_at"`
	InstanceUUID string `json:"instance_uuid"`
	Description  string `json:"description,omitempty"`
}

// SnapshotCreate is the request body for creating a snapshot.
type SnapshotCreate struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CreateSnapshot sends POST /instances/{instanceID}/create_snapshot.
func (c *Client) CreateSnapshot(ctx context.Context, instanceID string, req SnapshotCreate) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+instanceID+"/create_snapshot", req, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

// ListSnapshots sends GET /instances/{instanceID}/snapshots.
func (c *Client) ListSnapshots(ctx context.Context, instanceID string) ([]Snapshot, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/instances/"+instanceID+"/snapshots", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var snaps []Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snaps); err != nil {
		return nil, fmt.Errorf("decode snapshots: %w", err)
	}
	return snaps, nil
}

// GetSnapshot returns a single snapshot by ID, filtering the list.
func (c *Client) GetSnapshot(ctx context.Context, instanceID, snapshotID string) (*Snapshot, error) {
	snaps, err := c.ListSnapshots(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	for i := range snaps {
		if snaps[i].ID == snapshotID {
			return &snaps[i], nil
		}
	}
	return nil, fmt.Errorf("snapshot %q not found on instance %q", snapshotID, instanceID)
}
