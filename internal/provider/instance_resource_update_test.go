// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

// strList builds a types.List of strings from a slice.
func strList(t *testing.T, vals ...string) types.List {
	t.Helper()
	elems := make([]interface{}, len(vals))
	for i, v := range vals {
		elems[i] = types.StringValue(v)
	}
	l, diags := types.ListValueFrom(context.Background(), types.StringType, elems)
	if diags.HasError() {
		t.Fatalf("strList: %v", diags)
	}
	return l
}

// runUpdate runs applyUpdate against the mock and fails on error.
func runUpdate(t *testing.T, api InstancesAPI, state, plan InstanceResourceModel) {
	t.Helper()
	r := &instanceResource{api: api}
	if err := r.applyUpdate(context.Background(), &state, &plan); err != nil {
		t.Fatalf("applyUpdate error: %v", err)
	}
}

// --- Power state ---

func TestUpdate_Start(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"}}
	state := InstanceResourceModel{ID: types.StringValue("i1"), PowerState: types.StringValue("stopped")}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), PowerState: types.StringValue("running")}
	runUpdate(t, mock, state, plan)
	if mock.startCalls != 1 || mock.stopCalls != 0 {
		t.Fatalf("want start=1 stop=0, got start=%d stop=%d", mock.startCalls, mock.stopCalls)
	}
	if mock.startID != "i1" {
		t.Fatalf("startID = %q, want i1", mock.startID)
	}
}

func TestUpdate_Stop(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "SHUTOFF"}}
	state := InstanceResourceModel{ID: types.StringValue("i1"), PowerState: types.StringValue("running")}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), PowerState: types.StringValue("stopped")}
	runUpdate(t, mock, state, plan)
	if mock.stopCalls != 1 || mock.startCalls != 0 {
		t.Fatalf("want stop=1 start=0, got stop=%d start=%d", mock.stopCalls, mock.startCalls)
	}
}

func TestUpdate_PowerState_SameValue(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"}}
	state := InstanceResourceModel{ID: types.StringValue("i1"), PowerState: types.StringValue("running")}
	plan := state
	runUpdate(t, mock, state, plan)
	if mock.startCalls+mock.stopCalls != 0 {
		t.Fatalf("power unchanged: want 0 ops, got start=%d stop=%d", mock.startCalls, mock.stopCalls)
	}
}

func TestUpdate_PowerState_Invalid(t *testing.T) {
	mock := &mockInstancesAPI{}
	state := InstanceResourceModel{ID: types.StringValue("i1"), PowerState: types.StringValue("running")}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), PowerState: types.StringValue("suspended")}
	r := &instanceResource{api: mock}
	err := r.applyUpdate(context.Background(), &state, &plan)
	if err == nil {
		t.Fatal("expected error for invalid power_state, got nil")
	}
}

// --- Reboot ---

func TestUpdate_Reboot_TrueInPlanFalseInState(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"}}
	state := InstanceResourceModel{ID: types.StringValue("i1"), Reboot: types.BoolValue(false)}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), Reboot: types.BoolValue(true)}
	runUpdate(t, mock, state, plan)
	if mock.rebootCalls != 1 {
		t.Fatalf("want reboot=1, got %d", mock.rebootCalls)
	}
}

func TestUpdate_Reboot_Idempotent(t *testing.T) {
	// state.reboot == plan.reboot == true → should NOT reboot again.
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"}}
	state := InstanceResourceModel{ID: types.StringValue("i1"), Reboot: types.BoolValue(true)}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), Reboot: types.BoolValue(true)}
	runUpdate(t, mock, state, plan)
	if mock.rebootCalls != 0 {
		t.Fatalf("reboot on re-apply: want 0, got %d", mock.rebootCalls)
	}
}

func TestUpdate_Reboot_FalseSkipped(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"}}
	state := InstanceResourceModel{ID: types.StringValue("i1"), Reboot: types.BoolValue(false)}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), Reboot: types.BoolValue(false)}
	runUpdate(t, mock, state, plan)
	if mock.rebootCalls != 0 {
		t.Fatalf("reboot=false should not call, got %d", mock.rebootCalls)
	}
}

// --- Password ---

func TestUpdate_Password_Changed(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"}}
	state := InstanceResourceModel{ID: types.StringValue("i1"), AdminPassword: types.StringValue("old")}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), AdminPassword: types.StringValue("new")}
	runUpdate(t, mock, state, plan)
	if mock.passwordCalls != 1 {
		t.Fatalf("want pwd=1, got %d", mock.passwordCalls)
	}
	if mock.passwordValue != "new" {
		t.Fatalf("pwd sent = %q, want new", mock.passwordValue)
	}
}

func TestUpdate_Password_Unchanged(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"}}
	state := InstanceResourceModel{ID: types.StringValue("i1"), AdminPassword: types.StringValue("same")}
	plan := state
	runUpdate(t, mock, state, plan)
	if mock.passwordCalls != 0 {
		t.Fatalf("unchanged pwd: want 0, got %d", mock.passwordCalls)
	}
}

func TestUpdate_Password_EmptySkipped(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"}}
	state := InstanceResourceModel{ID: types.StringValue("i1")}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), AdminPassword: types.StringValue("")}
	runUpdate(t, mock, state, plan)
	if mock.passwordCalls != 0 {
		t.Fatalf("empty pwd: want 0, got %d", mock.passwordCalls)
	}
}

// --- Rename ---

func TestUpdate_Rename(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"}}
	state := InstanceResourceModel{ID: types.StringValue("i1"), Name: types.StringValue("old")}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), Name: types.StringValue("new")}
	runUpdate(t, mock, state, plan)
	if mock.renameCalls != 1 || mock.renameName != "new" {
		t.Fatalf("rename: calls=%d name=%q, want calls=1 name=new", mock.renameCalls, mock.renameName)
	}
}

// --- Reverse DNS ---

func TestUpdate_ReverseDNS(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE", AccessIPv4: "1.2.3.4"}}
	state := InstanceResourceModel{
		ID:         types.StringValue("i1"),
		ReverseDNS: types.StringValue("old.example.com"),
		AccessIPv4: types.StringValue("1.2.3.4"),
	}
	plan := InstanceResourceModel{
		ID:         types.StringValue("i1"),
		ReverseDNS: types.StringValue("new.example.com"),
		AccessIPv4: types.StringValue("1.2.3.4"),
	}
	runUpdate(t, mock, state, plan)
	if mock.reverseCalls != 1 || mock.reverseDNS != "new.example.com" || mock.reverseIP != "1.2.3.4" {
		t.Fatalf("reverse: calls=%d dns=%q ip=%q, want calls=1 dns=new ip=1.2.3.4",
			mock.reverseCalls, mock.reverseDNS, mock.reverseIP)
	}
}

// --- Security group diff ---

func TestUpdate_SG_AddNew(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"},
		currentSGs:  nil,
	}
	state := InstanceResourceModel{ID: types.StringValue("i1")}
	plan := InstanceResourceModel{
		ID:               types.StringValue("i1"),
		SecurityGroupIDs: strList(t, "sg-1"),
	}
	runUpdate(t, mock, state, plan)
	if mock.addSGCalls != 1 || mock.addSGID != "sg-1" || mock.removeSGCalls != 0 {
		t.Fatalf("add: addCalls=%d addID=%q removeCalls=%d, want addCalls=1 addID=sg-1 removeCalls=0",
			mock.addSGCalls, mock.addSGID, mock.removeSGCalls)
	}
}

func TestUpdate_SG_RemoveOrphan(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"},
		currentSGs:  []client.SecurityGroup{{ID: "sg-1"}},
	}
	state := InstanceResourceModel{ID: types.StringValue("i1")}
	plan := InstanceResourceModel{
		ID:               types.StringValue("i1"),
		SecurityGroupIDs: strList(t),
	}
	runUpdate(t, mock, state, plan)
	if mock.removeSGCalls != 1 || mock.removeSGID != "sg-1" || mock.addSGCalls != 0 {
		t.Fatalf("remove: removeCalls=%d removeID=%q addCalls=%d, want removeCalls=1 removeID=sg-1 addCalls=0",
			mock.removeSGCalls, mock.removeSGID, mock.addSGCalls)
	}
}

func TestUpdate_SG_NoOpWhenUnchanged(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"},
		currentSGs:  []client.SecurityGroup{{ID: "sg-1"}},
	}
	sgList := strList(t, "sg-1")
	state := InstanceResourceModel{ID: types.StringValue("i1"), SecurityGroupIDs: sgList}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), SecurityGroupIDs: sgList}
	runUpdate(t, mock, state, plan)
	if mock.listSGCalls != 0 {
		t.Fatalf("unchanged SGs: ListSG should be 0, got %d", mock.listSGCalls)
	}
}

func TestUpdate_SG_Swap(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"},
		currentSGs:  []client.SecurityGroup{{ID: "sg-1"}},
	}
	state := InstanceResourceModel{ID: types.StringValue("i1")}
	plan := InstanceResourceModel{
		ID:               types.StringValue("i1"),
		SecurityGroupIDs: strList(t, "sg-2"),
	}
	runUpdate(t, mock, state, plan)
	if mock.addSGCalls != 1 || mock.addSGID != "sg-2" {
		t.Fatalf("swap add: calls=%d id=%q, want 1 sg-2", mock.addSGCalls, mock.addSGID)
	}
	if mock.removeSGCalls != 1 || mock.removeSGID != "sg-1" {
		t.Fatalf("swap remove: calls=%d id=%q, want 1 sg-1", mock.removeSGCalls, mock.removeSGID)
	}
}

func TestUpdate_SG_AddMultiple(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"},
		currentSGs:  nil,
	}
	state := InstanceResourceModel{ID: types.StringValue("i1")}
	plan := InstanceResourceModel{
		ID:               types.StringValue("i1"),
		SecurityGroupIDs: strList(t, "sg-1", "sg-2", "sg-3"),
	}
	runUpdate(t, mock, state, plan)
	if mock.addSGCalls != 3 {
		t.Fatalf("add many: want addCalls=3, got %d", mock.addSGCalls)
	}
}

// --- Error paths ---

func TestUpdate_StartError(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance: &client.Instance{ID: "i1"},
		startErr:    errSentinel("boom"),
	}
	state := InstanceResourceModel{ID: types.StringValue("i1"), PowerState: types.StringValue("stopped")}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), PowerState: types.StringValue("running")}
	r := &instanceResource{api: mock}
	if err := r.applyUpdate(context.Background(), &state, &plan); err == nil {
		t.Fatal("expected start error, got nil")
	}
}

func TestUpdate_RenameError(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"},
		renameErr:   errSentinel("denied"),
	}
	state := InstanceResourceModel{ID: types.StringValue("i1"), Name: types.StringValue("old")}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), Name: types.StringValue("new")}
	r := &instanceResource{api: mock}
	if err := r.applyUpdate(context.Background(), &state, &plan); err == nil {
		t.Fatal("expected rename error, got nil")
	}
}

// --- Combined scenarios ---

func TestUpdate_Concurrent_StopRenamePassword(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "SHUTOFF"}}
	state := InstanceResourceModel{
		ID:         types.StringValue("i1"),
		Name:       types.StringValue("old"),
		PowerState: types.StringValue("running"),
	}
	plan := InstanceResourceModel{
		ID:            types.StringValue("i1"),
		Name:          types.StringValue("new"),
		PowerState:    types.StringValue("stopped"),
		AdminPassword: types.StringValue("secret"),
	}
	runUpdate(t, mock, state, plan)
	if mock.stopCalls != 1 || mock.renameCalls != 1 || mock.passwordCalls != 1 {
		t.Fatalf("concurrent: want stop=1 rename=1 pwd=1, got stop=%d rename=%d pwd=%d",
			mock.stopCalls, mock.renameCalls, mock.passwordCalls)
	}
}

func TestUpdate_NoChanges_AllOpsSkipped(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"}}
	state := InstanceResourceModel{
		ID:         types.StringValue("i1"),
		Name:       types.StringValue("same"),
		PowerState: types.StringValue("running"),
	}
	plan := state
	runUpdate(t, mock, state, plan)
	total := mock.startCalls + mock.stopCalls + mock.rebootCalls + mock.renameCalls +
		mock.passwordCalls + mock.reverseCalls + mock.addSGCalls + mock.removeSGCalls
	if total != 0 {
		t.Fatalf("no-change: want 0 ops, got start=%d stop=%d reboot=%d rename=%d pwd=%d reverse=%d addSG=%d removeSG=%d",
			mock.startCalls, mock.stopCalls, mock.rebootCalls, mock.renameCalls,
			mock.passwordCalls, mock.reverseCalls, mock.addSGCalls, mock.removeSGCalls)
	}
}

// errSentinel is a simple error for mocking API failures.
type errSentinel string

func (e errSentinel) Error() string { return string(e) }
