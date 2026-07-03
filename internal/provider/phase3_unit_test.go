// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

// --- snapshot Create polling (waitForSnapshot) ---

// mockSnapshotAPI implements SnapshotAPI for unit tests.
type mockSnapshotAPI struct {
	createErr     error
	listResult    []client.Snapshot
	listErr       error
	createCalls   int
	listCalls     int
	listCallNames []string
}

func (m *mockSnapshotAPI) CreateSnapshot(context.Context, string, client.SnapshotCreate) error {
	m.createCalls++
	return m.createErr
}
func (m *mockSnapshotAPI) ListSnapshots(_ context.Context, instID string) ([]client.Snapshot, error) {
	m.listCalls++
	m.listCallNames = append(m.listCallNames, instID)
	return m.listResult, m.listErr
}
func (m *mockSnapshotAPI) GetSnapshot(_ context.Context, _, _ string) (*client.Snapshot, error) {
	return nil, nil
}

func TestWaitForSnapshot_Immediate(t *testing.T) {
	mock := &mockSnapshotAPI{
		listResult: []client.Snapshot{{ID: "s1", Name: "[SNAPSHOT] foo - 1234"}},
	}
	got, err := waitForSnapshot(context.Background(), mock, "i1", "foo", time.Second, time.Millisecond)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.ID != "s1" {
		t.Fatalf("id=%s, want s1", got.ID)
	}
	if mock.listCalls != 1 {
		t.Fatalf("listCalls=%d, want 1", mock.listCalls)
	}
}

func TestWaitForSnapshot_SubstringMatch(t *testing.T) {
	// Real API wraps name with "[SNAPSHOT] <name> - <ts>"; we match by Contains.
	mock := &mockSnapshotAPI{
		listResult: []client.Snapshot{
			{ID: "other", Name: "[SNAPSHOT] other - 1"},
			{ID: "wanted", Name: "[SNAPSHOT] foo - 1234"},
		},
	}
	got, err := waitForSnapshot(context.Background(), mock, "i1", "foo", time.Second, time.Millisecond)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.ID != "wanted" {
		t.Fatalf("id=%s, want wanted", got.ID)
	}
}

func TestWaitForSnapshot_Timeout(t *testing.T) {
	mock := &mockSnapshotAPI{listResult: nil}
	_, err := waitForSnapshot(context.Background(), mock, "i1", "ghost", 50*time.Millisecond, 5*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if mock.listCalls < 1 {
		t.Fatalf("expected at least 1 list call, got %d", mock.listCalls)
	}
}

func TestWaitForSnapshot_ListError(t *testing.T) {
	mock := &mockSnapshotAPI{listErr: errors.New("net error")}
	_, err := waitForSnapshot(context.Background(), mock, "i1", "foo", 100*time.Millisecond, time.Millisecond)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestWaitForSnapshot_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mock := &mockSnapshotAPI{listResult: nil}
	_, err := waitForSnapshot(ctx, mock, "i1", "foo", time.Second, time.Millisecond)
	if err == nil {
		t.Fatal("expected ctx cancellation error, got nil")
	}
}

// --- backup schedules mapper (schedulesToList) ---

func TestSchedulesToList_Empty(t *testing.T) {
	l, diags := schedulesToList(nil)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !l.IsNull() && len(l.Elements()) != 0 {
		t.Fatalf("expected empty list, got %v", l)
	}
}

func TestSchedulesToList_One(t *testing.T) {
	in := []client.BackupSchedule{{
		ID: 1, Instance: "i1", Rotation: 7, RunAt: "03:00",
		BackupName: "daily", BackupType: "auto",
	}}
	l, diags := schedulesToList(in)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	elems := l.Elements()
	if len(elems) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elems))
	}
}

func TestSchedulesToList_Many(t *testing.T) {
	in := []client.BackupSchedule{
		{ID: 1, BackupName: "a"},
		{ID: 2, BackupName: "b"},
		{ID: 3, BackupName: "c"},
	}
	l, diags := schedulesToList(in)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if len(l.Elements()) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(l.Elements()))
	}
}

// --- snapshotToModel name not overwritten (already tested, but verify) ---

func TestSnapshotToModel_NameNotSet(t *testing.T) {
	m := &SnapshotResourceModel{Name: types.StringValue("user-name")}
	snapshotToModel(&client.Snapshot{ID: "s1", Name: "[SNAPSHOT] user-name - 1"}, m)
	if m.Name.ValueString() != "user-name" {
		t.Fatalf("snapshotToModel should not overwrite Name, got %q", m.Name.ValueString())
	}
	if m.ID.ValueString() != "s1" {
		t.Fatalf("id=%s, want s1", m.ID.ValueString())
	}
}
