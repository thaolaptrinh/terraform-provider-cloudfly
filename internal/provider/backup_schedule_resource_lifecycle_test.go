// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

func bsFixture() *client.BackupSchedule {
	return &client.BackupSchedule{
		ID: 42, Instance: "i1", Rotation: 7,
		RunAt: "2026-07-04 00:00:00", BackupName: "my-backup", BackupType: "weekly",
	}
}

func TestCreateBackupSchedule_Success(t *testing.T) {
	mock := &mockBackupScheduleAPI{
		listResult: []client.BackupSchedule{*bsFixture()},
	}
	m := &BackupScheduleResourceModel{
		InstanceID: types.StringValue("i1"),
		BackupType: types.StringValue("weekly"),
	}

	err := createBackupSchedule(context.Background(), mock, m)
	if err != nil {
		t.Fatalf("createBackupSchedule error: %v", err)
	}
	if mock.createCalls != 1 {
		t.Errorf("expected 1 CreateBackupSchedule call, got %d", mock.createCalls)
	}
	if mock.listCalls < 1 {
		t.Errorf("expected at least 1 ListBackupSchedules call, got %d", mock.listCalls)
	}
	if m.ID.ValueString() != "42" {
		t.Errorf("id = %q, want 42", m.ID.ValueString())
	}
	if m.InstanceID.ValueString() != "i1" {
		t.Errorf("instance_id = %q, want i1", m.InstanceID.ValueString())
	}
	if m.BackupType.ValueString() != "weekly" {
		t.Errorf("backup_type = %q, want weekly", m.BackupType.ValueString())
	}
	if m.Rotation.ValueInt64() != 7 {
		t.Errorf("rotation = %d, want 7", m.Rotation.ValueInt64())
	}
	if m.Name.ValueString() != "my-backup" {
		t.Errorf("name = %q, want my-backup", m.Name.ValueString())
	}
	if m.RunAt.ValueString() != "2026-07-04 00:00:00" {
		t.Errorf("run_at = %q", m.RunAt.ValueString())
	}
}

func TestCreateBackupSchedule_DefaultType(t *testing.T) {
	mock := &mockBackupScheduleAPI{
		listResult: []client.BackupSchedule{*bsFixture()},
	}
	m := &BackupScheduleResourceModel{
		InstanceID: types.StringValue("i1"),
	}

	err := createBackupSchedule(context.Background(), mock, m)
	if err != nil {
		t.Fatalf("createBackupSchedule error: %v", err)
	}
	if mock.createReq.BackupType != "weekly" {
		t.Errorf("createReq.BackupType = %q, want weekly (default)", mock.createReq.BackupType)
	}
}

func TestCreateBackupSchedule_Error(t *testing.T) {
	mock := &mockBackupScheduleAPI{createErr: errSentinel("create failed")}
	m := &BackupScheduleResourceModel{
		InstanceID: types.StringValue("i1"),
		BackupType: types.StringValue("weekly"),
	}

	err := createBackupSchedule(context.Background(), mock, m)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReadBackupSchedule_Success(t *testing.T) {
	mock := &mockBackupScheduleAPI{getResult: bsFixture()}
	m := &BackupScheduleResourceModel{
		ID:         types.StringValue("42"),
		InstanceID: types.StringValue("i1"),
	}

	err := readBackupSchedule(context.Background(), mock, m)
	if err != nil {
		t.Fatalf("readBackupSchedule error: %v", err)
	}
	if mock.getCalls != 1 {
		t.Errorf("expected 1 GetBackupSchedule call, got %d", mock.getCalls)
	}
	if m.BackupType.ValueString() != "weekly" {
		t.Errorf("backup_type = %q, want weekly", m.BackupType.ValueString())
	}
}

func TestReadBackupSchedule_Error(t *testing.T) {
	mock := &mockBackupScheduleAPI{getErr: errSentinel("get failed")}
	m := &BackupScheduleResourceModel{
		ID:         types.StringValue("42"),
		InstanceID: types.StringValue("i1"),
	}

	err := readBackupSchedule(context.Background(), mock, m)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
