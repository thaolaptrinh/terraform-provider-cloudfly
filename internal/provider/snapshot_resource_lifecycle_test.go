// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

func snapshotFixture(id, status, name string) *client.Snapshot {
	return &client.Snapshot{
		ID: id, Name: name, Status: status,
		Size: 2048, SizeInGB: client.FlexString("2"),
		Type: "snapshot", OSDistro: "ubuntu",
		CreatedAt: "2026-07-01T00:00:00Z", InstanceUUID: "i1",
		Description: "test desc",
	}
}

func TestCreateSnapshot_Success(t *testing.T) {
	mock := &mockSnapshotAPI{
		listResult: []client.Snapshot{
			*snapshotFixture("snap-x", "available", "[SNAPSHOT] my-snap - 1"),
		},
	}
	m := &SnapshotResourceModel{
		InstanceID:  types.StringValue("i1"),
		Name:        types.StringValue("my-snap"),
		Description: types.StringValue("test desc"),
	}

	err := createSnapshot(context.Background(), mock, m)
	if err != nil {
		t.Fatalf("createSnapshot error: %v", err)
	}
	if mock.createCalls != 1 {
		t.Errorf("expected 1 CreateSnapshot call, got %d", mock.createCalls)
	}
	if mock.listCalls < 1 {
		t.Errorf("expected at least 1 ListSnapshots call, got %d", mock.listCalls)
	}
	if m.ID.ValueString() != "snap-x" {
		t.Errorf("id = %q, want snap-x", m.ID.ValueString())
	}
	if m.Name.ValueString() != "my-snap" {
		t.Errorf("name = %q, want my-snap", m.Name.ValueString())
	}
	if m.Status.ValueString() != "available" {
		t.Errorf("status = %q, want available", m.Status.ValueString())
	}
	if m.InstanceID.ValueString() != "i1" {
		t.Errorf("instance_id = %q, want i1", m.InstanceID.ValueString())
	}
}

func TestCreateSnapshot_Error(t *testing.T) {
	mock := &mockSnapshotAPI{createErr: errSentinel("create failed")}
	m := &SnapshotResourceModel{
		InstanceID: types.StringValue("i1"),
		Name:       types.StringValue("my-snap"),
	}

	err := createSnapshot(context.Background(), mock, m)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReadSnapshot_Success(t *testing.T) {
	mock := &mockSnapshotAPI{
		getResult: snapshotFixture("snap-x", "available", "my-snap"),
	}
	m := &SnapshotResourceModel{
		ID:         types.StringValue("snap-x"),
		InstanceID: types.StringValue("i1"),
		Name:       types.StringValue("my-snap"),
	}

	err := readSnapshot(context.Background(), mock, m)
	if err != nil {
		t.Fatalf("readSnapshot error: %v", err)
	}
	if mock.getCalls != 1 {
		t.Errorf("expected 1 GetSnapshot call, got %d", mock.getCalls)
	}
	if mock.getCallID != "snap-x" {
		t.Errorf("getCallID = %q, want snap-x", mock.getCallID)
	}
	if m.Status.ValueString() != "available" {
		t.Errorf("status = %q, want available", m.Status.ValueString())
	}
}

func TestReadSnapshot_Error(t *testing.T) {
	mock := &mockSnapshotAPI{getErr: errSentinel("get failed")}
	m := &SnapshotResourceModel{
		ID:         types.StringValue("snap-x"),
		InstanceID: types.StringValue("i1"),
	}

	err := readSnapshot(context.Background(), mock, m)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
