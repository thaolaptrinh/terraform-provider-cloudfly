// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type BackupSchedule struct {
	ID         int64  `json:"id"`
	Instance   string `json:"instance"`
	Rotation   int64  `json:"rotation"`
	RunAt      string `json:"run_at"`
	BackupName string `json:"backup_name"`
	BackupType string `json:"backup_type"`
}

func (c *Client) GetBackupSchedules(ctx context.Context, instanceID string) ([]BackupSchedule, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/instances/"+instanceID+"/backup-server", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out []BackupSchedule
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode backup schedules: %w", err)
	}
	return out, nil
}
