// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type BackupSchedule struct {
	ID         int64  `json:"id"`
	Instance   string `json:"instance"`
	Rotation   int64  `json:"rotation"`
	RunAt      string `json:"run_at"`
	BackupName string `json:"backup_name"`
	BackupType string `json:"backup_type"`
}

type BackupScheduleCreate struct {
	Name       string `json:"name,omitempty"`
	BackupType string `json:"backup_type"`
}

func (c *Client) ListBackupSchedules(ctx context.Context, instanceID string) ([]BackupSchedule, error) {
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

func (c *Client) CreateBackupSchedule(ctx context.Context, instanceID string, req BackupScheduleCreate) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+instanceID+"/create_backup_schedule", req, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) GetBackupSchedule(ctx context.Context, instanceID, scheduleID string) (*BackupSchedule, error) {
	schedules, err := c.ListBackupSchedules(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	idInt, err := strconv.ParseInt(scheduleID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule id %q: %w", scheduleID, err)
	}
	for i := range schedules {
		if schedules[i].ID == idInt {
			return &schedules[i], nil
		}
	}
	return nil, fmt.Errorf("backup schedule %q not found on instance %q", scheduleID, instanceID)
}

func (c *Client) DeleteBackupSchedule(ctx context.Context, scheduleID int64) error {
	resp, err := c.Do(ctx, http.MethodDelete, "/instances/backup-servers/"+strconv.FormatInt(scheduleID, 10), nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}
