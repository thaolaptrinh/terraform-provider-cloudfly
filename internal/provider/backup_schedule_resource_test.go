// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

type mockBackupScheduleAPI struct {
	createErr  error
	listResult []client.BackupSchedule
	listErr    error
	deleteErr  error
}

func (m *mockBackupScheduleAPI) CreateBackupSchedule(context.Context, string, client.BackupScheduleCreate) error {
	return m.createErr
}
func (m *mockBackupScheduleAPI) ListBackupSchedules(_ context.Context, _ string) ([]client.BackupSchedule, error) {
	return m.listResult, m.listErr
}
func (m *mockBackupScheduleAPI) GetBackupSchedule(_ context.Context, _, _ string) (*client.BackupSchedule, error) {
	return nil, nil
}
func (m *mockBackupScheduleAPI) DeleteBackupSchedule(_ context.Context, _ int64) error {
	return m.deleteErr
}

func TestBackupScheduleToModel(t *testing.T) {
	bs := &client.BackupSchedule{
		ID:         42,
		Instance:   "i1",
		Rotation:   7,
		RunAt:      "2026-07-04 00:00:00",
		BackupName: "my-backup",
		BackupType: "weekly",
	}
	var m BackupScheduleResourceModel
	backupScheduleToModel(bs, &m)

	if m.ID.ValueString() != "42" {
		t.Fatalf("id=%q, want 42", m.ID.ValueString())
	}
	if m.Rotation.ValueInt64() != 7 {
		t.Fatalf("rotation=%d, want 7", m.Rotation.ValueInt64())
	}
	if m.RunAt.ValueString() != "2026-07-04 00:00:00" {
		t.Fatalf("run_at=%q", m.RunAt.ValueString())
	}
	if m.InstanceID.ValueString() != "i1" {
		t.Fatalf("instance_id=%q, want i1", m.InstanceID.ValueString())
	}
	if m.Name.ValueString() != "my-backup" {
		t.Fatalf("name=%q, want my-backup", m.Name.ValueString())
	}
	if m.BackupType.ValueString() != "weekly" {
		t.Fatalf("backup_type=%q, want weekly", m.BackupType.ValueString())
	}
}

func TestWaitForBackupSchedule_Immediate(t *testing.T) {
	mock := &mockBackupScheduleAPI{
		listResult: []client.BackupSchedule{{
			ID: 1, Instance: "i1", BackupType: "weekly", BackupName: "test-backup",
		}},
	}
	got, err := waitForBackupSchedule(context.Background(), mock, "i1", "weekly", "test-backup", time.Second, time.Millisecond)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.ID != 1 {
		t.Fatalf("id=%d, want 1", got.ID)
	}
}

func TestWaitForBackupSchedule_Timeout(t *testing.T) {
	mock := &mockBackupScheduleAPI{listResult: nil}
	_, err := waitForBackupSchedule(context.Background(), mock, "i1", "weekly", "", 50*time.Millisecond, 5*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestWaitForBackupSchedule_ListError(t *testing.T) {
	mock := &mockBackupScheduleAPI{listErr: errors.New("net error")}
	_, err := waitForBackupSchedule(context.Background(), mock, "i1", "weekly", "", 100*time.Millisecond, time.Millisecond)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWaitForBackupSchedule_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mock := &mockBackupScheduleAPI{listResult: nil}
	_, err := waitForBackupSchedule(ctx, mock, "i1", "weekly", "", time.Second, time.Millisecond)
	if err == nil {
		t.Fatal("expected ctx cancellation error, got nil")
	}
}

func TestParseScheduleID(t *testing.T) {
	id, err := parseScheduleID("42")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if id != 42 {
		t.Fatalf("id=%d, want 42", id)
	}
}

func TestParseScheduleID_Invalid(t *testing.T) {
	_, err := parseScheduleID("abc")
	if err == nil {
		t.Fatal("expected error for invalid id")
	}
}

func TestBackupScheduleToModel_ID(t *testing.T) {
	m := &BackupScheduleResourceModel{}
	backupScheduleToModel(&client.BackupSchedule{ID: 42, Instance: "i1", BackupType: "weekly"}, m)
	if m.ID.ValueString() != "42" {
		t.Fatalf("id=%q, want 42", m.ID.ValueString())
	}
}
