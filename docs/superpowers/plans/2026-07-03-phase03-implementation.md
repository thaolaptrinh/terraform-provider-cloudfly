# Phase 03 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement network interface management, backup schedule resource, and IPv6 post-create for the CloudFly Terraform provider.

**Architecture:** Client layer (`internal/client/`) gets new types and API methods. `instance_resource.go` gets `network_ids` attribute with reconcile logic (pattern: `security_group_ids`) and IPv6 update logic. New `backup_schedule_resource.go` follows `snapshot_resource.go` pattern (poll-for-ID post-create, RequiresReplace on all user attrs).

**Tech Stack:** Go, terraform-plugin-framework, existing `client.Client` HTTP wrapper

---

## Task 1: Client — Interface management types and methods

**Files:**
- Modify: `internal/client/instances.go`

> **Note:** All line numbers in this plan are approximate and reference the current file state. When executing, use content-based search to find the right insertion points.

- [ ] **Step 1: Add interface types**

Insert after `SecurityGroup` struct (search for `type SecurityGroup struct`):

```go
type InterfaceItem struct {
	InterfaceID string `json:"interface_id"`
	NetworkID   string `json:"network_id"`
	SubnetID    string `json:"subnet_id"`
	IPVersion   string `json:"ip_version"`
	IsDefault   bool   `json:"is_default"`
	Gateway     string `json:"gateway"`
	IPAddress   string `json:"ip_address"`
}

type InterfaceGroup struct {
	Data        []InterfaceItem `json:"data"`
	NetworkName string          `json:"network_name"`
	IsPublic    bool            `json:"is_public"`
	IPV6Range   []interface{}   `json:"ipv6_range"`
}

type attachInterfaceRequest struct {
	NetworkID string `json:"network_id"`
}

type detachInterfaceRequest struct {
	InterfaceID string `json:"interface_id"`
}
```

- [ ] **Step 2: Add client methods**

Insert at the end of `instances.go` (after `RebuildInstance`, before EOF):

```go
func (c *Client) ListInterfaces(ctx context.Context, id string) ([]InterfaceGroup, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/instances/"+id+"/interfaces", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out []InterfaceGroup
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode interfaces: %w", err)
	}
	return out, nil
}

func (c *Client) AttachInterface(ctx context.Context, id, networkID string) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+id+"/attach-interface", attachInterfaceRequest{NetworkID: networkID}, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) DetachInterface(ctx context.Context, id, interfaceID string) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+id+"/detach-interface", detachInterfaceRequest{InterfaceID: interfaceID}, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}
```

- [ ] **Step 3: Run tests to verify no regressions**

```bash
go build ./...
```

Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add internal/client/instances.go
git commit -m "feat(client): add interface management types and methods"
```

---

## Task 2: Client — EnableIPv6Range method

**Files:**
- Modify: `internal/client/instances.go`

- [ ] **Step 1: Add EnableIPv6Range method**

Insert after `RebuildInstance` at end of `instances.go`:

```go
func (c *Client) EnableIPv6Range(ctx context.Context, id string) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+id+"/enable-ipv6-range", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}
```

- [ ] **Step 2: Build check**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/client/instances.go
git commit -m "feat(client): add EnableIPv6Range method"
```

---

## Task 3: Client — Backup schedule CRUD methods + fix callers

**Files:**
- Modify: `internal/client/backups.go`
- Modify: `internal/provider/backup_schedules_data_source.go`

- [ ] **Step 1: Replace entire `backups.go` content**

```go
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
```

- [ ] **Step 2: Update BackupSchedulesAPI interface in backup_schedules_data_source.go**

In `internal/provider/backup_schedules_data_source.go`, change line 18:

```go
	GetBackupSchedules(ctx context.Context, instanceID string) ([]client.BackupSchedule, error)
```

to:

```go
	ListBackupSchedules(ctx context.Context, instanceID string) ([]client.BackupSchedule, error)
```

And change line 89:

```go
	schedules, err := d.api.GetBackupSchedules(ctx, config.InstanceID.ValueString())
```

to:

```go
	schedules, err := d.api.ListBackupSchedules(ctx, config.InstanceID.ValueString())
```

- [ ] **Step 3: Build check**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 4: Run existing unit tests**

```bash
go test ./internal/provider/ -run "TestSchedulesToList" -v
```

Expected: All TestSchedulesToList tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/client/backups.go internal/provider/backup_schedules_data_source.go
git commit -m "feat(client): add backup schedule CRUD, rename GetBackupSchedules -> ListBackupSchedules"
```

---

## Task 4: IPv6 — Remove RequiresReplace, add update logic

**Files:**
- Modify: `internal/provider/instance_resource.go`
- Modify: `internal/provider/instance_resource_mocks_test.go`

- [ ] **Step 1: Remove RequiresReplace from enable_ipv6 schema**

Search for `"enable_ipv6"` in `instance_resource.go`, change:

```go
"enable_ipv6": schema.BoolAttribute{Optional: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
```

to:

```go
"enable_ipv6": schema.BoolAttribute{Optional: true},
```

- [ ] **Step 2: Add EnableIPv6Range to InstancesAPI interface**

In `InstancesAPI` interface, add after `WaitInstanceStopped` line:

```go
	EnableIPv6Range(ctx context.Context, id string) error
```

- [ ] **Step 3: Add IPv6 update logic to applyUpdate**

Search for `!state.ReverseDNS.Equal` block in `applyUpdate`. Insert after that block, before the security groups block:

```go
	if !state.EnableIPv6.Equal(plan.EnableIPv6) {
		if plan.EnableIPv6.ValueBool() && !state.EnableIPv6.ValueBool() {
			if err := r.api.EnableIPv6Range(ctx, id); err != nil {
				return fmt.Errorf("enable ipv6: %w", err)
			}
		}
	}
```

- [ ] **Step 4: Add mock fields for EnableIPv6Range**

Search for the end of `mockInstancesAPI` field block in `instance_resource_mocks_test.go` (find `waitActiveErr` field), add after:

```go
	enableIPv6RangeCalls  int
	enableIPv6RangeID     string
	enableIPv6RangeErr    error
```

Add the method implementation at the end of the mock (after `WaitInstanceStopped`, before closing `}`):

```go
func (m *mockInstancesAPI) EnableIPv6Range(_ context.Context, id string) error {
	m.enableIPv6RangeCalls++
	m.enableIPv6RangeID = id
	return m.enableIPv6RangeErr
}
```

- [ ] **Step 5: Build check**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/provider/instance_resource.go internal/provider/instance_resource_mocks_test.go
git commit -m "feat: enable IPv6 post-create (remove RequiresReplace)"
```

---

## Task 5: Unit test — IPv6 update

**Files:**
- Modify: `internal/provider/instance_resource_update_test.go`

- [ ] **Step 1: Add IPv6 enable test**

Insert before the `errSentinel` type at end of `instance_resource_update_test.go`:

```go
// --- IPv6 ---

func TestUpdate_IPv6_Enable(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"}}
	state := InstanceResourceModel{ID: types.StringValue("i1"), EnableIPv6: types.BoolValue(false)}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), EnableIPv6: types.BoolValue(true)}
	runUpdate(t, mock, state, plan)
	if mock.enableIPv6RangeCalls != 1 {
		t.Fatalf("want enableIPv6RageCalls=1, got %d", mock.enableIPv6RangeCalls)
	}
	if mock.enableIPv6RangeID != "i1" {
		t.Fatalf("enableIPv6RangeID = %q, want i1", mock.enableIPv6RangeID)
	}
}

func TestUpdate_IPv6_DisableNoOp(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"}}
	state := InstanceResourceModel{ID: types.StringValue("i1"), EnableIPv6: types.BoolValue(true)}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), EnableIPv6: types.BoolValue(false)}
	runUpdate(t, mock, state, plan)
	if mock.enableIPv6RangeCalls != 0 {
		t.Fatalf("disable should be no-op, got %d calls", mock.enableIPv6RangeCalls)
	}
}

func TestUpdate_IPv6_AlreadyEnabled(t *testing.T) {
	mock := &mockInstancesAPI{getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"}}
	state := InstanceResourceModel{ID: types.StringValue("i1"), EnableIPv6: types.BoolValue(true)}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), EnableIPv6: types.BoolValue(true)}
	runUpdate(t, mock, state, plan)
	if mock.enableIPv6RangeCalls != 0 {
		t.Fatalf("already enabled: want 0 calls, got %d", mock.enableIPv6RangeCalls)
	}
}

func TestUpdate_IPv6_EnableError(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance:        &client.Instance{ID: "i1"},
		enableIPv6RangeErr: errSentinel("ipv6-error"),
	}
	state := InstanceResourceModel{ID: types.StringValue("i1"), EnableIPv6: types.BoolValue(false)}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), EnableIPv6: types.BoolValue(true)}
	r := &instanceResource{api: mock}
	if err := r.applyUpdate(context.Background(), &state, &plan); err == nil {
		t.Fatal("expected ipv6 enable error, got nil")
	}
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./internal/provider/ -run "TestUpdate_IPv6" -v
```

Expected: 4 tests PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/instance_resource_update_test.go
git commit -m "test: add IPv6 enable/disable unit tests"
```

---

## Task 6: Network interface — schema, model, reconcile logic

**Files:**
- Modify: `internal/provider/instance_resource.go`

- [ ] **Step 1: Add network_ids to schema**

Search for `"security_group_ids"` schema block. Insert after it:

```go
			"network_ids": schema.ListAttribute{
				MarkdownDescription: "Additional network IDs to attach to the instance. The default public network is managed automatically and excluded from this list.",
				ElementType:         types.StringType,
				Optional:            true,
			},
```

- [ ] **Step 2: Add network_ids to model**

Search for `SecurityGroupIDs` in `InstanceResourceModel` struct. Add after it:

```go
	NetworkIDs types.List `tfsdk:"network_ids"`
```

- [ ] **Step 3: Add interface methods to InstancesAPI**

In `InstancesAPI` interface, after `ListSecurityGroups`:

```go
	ListInterfaces(ctx context.Context, id string) ([]client.InterfaceGroup, error)
	AttachInterface(ctx context.Context, id, networkID string) error
	DetachInterface(ctx context.Context, id, interfaceID string) error
```

- [ ] **Step 4: Add reconcileNetworks call to applyUpdate**

Search for `reconcileSecurityGroups` call in `applyUpdate`. Insert after that block:

```go
	if !state.NetworkIDs.Equal(plan.NetworkIDs) {
		if err := r.reconcileNetworks(ctx, id, plan.NetworkIDs); err != nil {
			return err
		}
	}
```

- [ ] **Step 5: Add reconcileNetworks method**

Search for the end of `reconcileSecurityGroups` method (after its closing `}`). Insert after it:

```go
// reconcileNetworks brings the instance's attached networks in line with the
// plan list. Attach uses network_id; detach uses interface_id (one network
// may have multiple interfaces). The default public network is excluded.
func (r *instanceResource) reconcileNetworks(ctx context.Context, id string, planList types.List) error {
	if planList.IsNull() || planList.IsUnknown() {
		return nil
	}

	groups, err := r.api.ListInterfaces(ctx, id)
	if err != nil {
		return fmt.Errorf("list interfaces: %w", err)
	}

	planIDs := make(map[string]bool)
	var planSlice []string
	if diags := planList.ElementsAs(ctx, &planSlice, false); diags.HasError() {
		return fmt.Errorf("decode network_ids: %v", diags.Errors())
	}
	for _, nid := range planSlice {
		planIDs[nid] = true
	}

	currentNetworks := make(map[string]bool)
	currentInterfaces := make(map[string][]string) // networkID -> []interfaceID

	for _, group := range groups {
		for _, item := range group.Data {
			if item.IsDefault && group.IsPublic {
				continue
			}
			currentNetworks[item.NetworkID] = true
			currentInterfaces[item.NetworkID] = append(currentInterfaces[item.NetworkID], item.InterfaceID)
		}
	}

	for _, nid := range planSlice {
		if !currentNetworks[nid] {
			if err := r.api.AttachInterface(ctx, id, nid); err != nil {
				return fmt.Errorf("attach network %s: %w", nid, err)
			}
		}
	}

	for nid := range currentNetworks {
		if !planIDs[nid] {
			for _, ifID := range currentInterfaces[nid] {
				if err := r.api.DetachInterface(ctx, id, ifID); err != nil {
					return fmt.Errorf("detach interface %s: %w", ifID, err)
				}
			}
		}
	}

	return nil
}
```

- [ ] **Step 6: Build check**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/provider/instance_resource.go
git commit -m "feat: add network_ids attribute with reconcile logic"
```

---

## Task 7: Mock — interface management methods

**Files:**
- Modify: `internal/provider/instance_resource_mocks_test.go`

- [ ] **Step 1: Add mock fields**

In `mockInstancesAPI` struct, after the existing fields (after line 38):

```go
	attachNetworkID, detachInterfaceID string
	attachCalls, detachCalls           int
	listInterfacesReturn               []client.InterfaceGroup
	listInterfacesErr                  error
	attachErr, detachErr               error
```

- [ ] **Step 2: Add mock methods**

At the end of the mock (after `EnableIPv6Range` method):

```go
func (m *mockInstancesAPI) ListInterfaces(_ context.Context, id string) ([]client.InterfaceGroup, error) {
	return m.listInterfacesReturn, m.listInterfacesErr
}
func (m *mockInstancesAPI) AttachInterface(_ context.Context, id, networkID string) error {
	m.attachCalls++
	m.attachNetworkID = networkID
	return m.attachErr
}
func (m *mockInstancesAPI) DetachInterface(_ context.Context, id, interfaceID string) error {
	m.detachCalls++
	m.detachInterfaceID = interfaceID
	return m.detachErr
}
```

- [ ] **Step 3: Build check**

```bash
go build ./...
```

Expected: PASS (the mock methods must now compile against the updated interface).

- [ ] **Step 4: Commit**

```bash
git add internal/provider/instance_resource_mocks_test.go
git commit -m "test: add interface management mock support"
```

---

## Task 8: Unit test — network reconcile

**Files:**
- Modify: `internal/provider/instance_resource_update_test.go`

- [ ] **Step 1: Add network reconcile tests**

Insert before the `errSentinel` type at end of file:

```go
// --- Network reconcile ---

func TestUpdate_Network_AttachNew(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance:          &client.Instance{ID: "i1", Status: "ACTIVE"},
		listInterfacesReturn: nil,
	}
	state := InstanceResourceModel{ID: types.StringValue("i1")}
	plan := InstanceResourceModel{
		ID:         types.StringValue("i1"),
		NetworkIDs: strList(t, "net-1"),
	}
	runUpdate(t, mock, state, plan)
	if mock.attachCalls != 1 || mock.attachNetworkID != "net-1" || mock.detachCalls != 0 {
		t.Fatalf("attach: calls=%d id=%q detachCalls=%d, want calls=1 id=net-1 detachCalls=0",
			mock.attachCalls, mock.attachNetworkID, mock.detachCalls)
	}
}

func TestUpdate_Network_DetachRemoved(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"},
		listInterfacesReturn: []client.InterfaceGroup{{
			Data:        []client.InterfaceItem{{InterfaceID: "if-1", NetworkID: "net-1"}},
			IsPublic:    false,
		}},
	}
	state := InstanceResourceModel{ID: types.StringValue("i1")}
	plan := InstanceResourceModel{
		ID:         types.StringValue("i1"),
		NetworkIDs: strList(t),
	}
	runUpdate(t, mock, state, plan)
	if mock.detachCalls != 1 || mock.detachInterfaceID != "if-1" || mock.attachCalls != 0 {
		t.Fatalf("detach: calls=%d id=%q attachCalls=%d, want calls=1 id=if-1 attachCalls=0",
			mock.detachCalls, mock.detachInterfaceID, mock.attachCalls)
	}
}

func TestUpdate_Network_NoOpWhenUnchanged(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance:          &client.Instance{ID: "i1", Status: "ACTIVE"},
		listInterfacesReturn: nil,
	}
	netList := strList(t, "net-1")
	state := InstanceResourceModel{ID: types.StringValue("i1"), NetworkIDs: netList}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), NetworkIDs: netList}
	runUpdate(t, mock, state, plan)
	if mock.attachCalls+mock.detachCalls != 0 {
		t.Fatalf("unchanged: want 0 ops, got attach=%d detach=%d", mock.attachCalls, mock.detachCalls)
	}
}

func TestUpdate_Network_SkipsDefaultPublic(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"},
		listInterfacesReturn: []client.InterfaceGroup{{
			Data:     []client.InterfaceItem{{InterfaceID: "def-if", NetworkID: "net-public", IsDefault: true}},
			IsPublic: true,
		}},
	}
	planList := strList(t, "net-other")
	state := InstanceResourceModel{ID: types.StringValue("i1")}
	plan := InstanceResourceModel{ID: types.StringValue("i1"), NetworkIDs: planList}
	runUpdate(t, mock, state, plan)
	if mock.detachCalls != 0 {
		t.Fatalf("should not detach default public, got detach=%d", mock.detachCalls)
	}
	if mock.attachCalls != 1 || mock.attachNetworkID != "net-other" {
		t.Fatalf("should attach net-other: calls=%d id=%q", mock.attachCalls, mock.attachNetworkID)
	}
}

func TestUpdate_Network_DetachMultiInterface(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"},
		listInterfacesReturn: []client.InterfaceGroup{{
			Data: []client.InterfaceItem{
				{InterfaceID: "if-ipv4", NetworkID: "net-1", IPVersion: "IPv4"},
				{InterfaceID: "if-ipv6", NetworkID: "net-1", IPVersion: "IPv6"},
			},
		}},
	}
	state := InstanceResourceModel{ID: types.StringValue("i1")}
	plan := InstanceResourceModel{
		ID:         types.StringValue("i1"),
		NetworkIDs: strList(t),
	}
	runUpdate(t, mock, state, plan)
	if mock.detachCalls != 2 {
		t.Fatalf("multi-interface detach: want 2, got %d", mock.detachCalls)
	}
}

func TestUpdate_Network_NullPlanSkipped(t *testing.T) {
	mock := &mockInstancesAPI{
		getInstance: &client.Instance{ID: "i1", Status: "ACTIVE"},
	}
	state := InstanceResourceModel{ID: types.StringValue("i1")}
	plan := InstanceResourceModel{ID: types.StringValue("i1")}
	runUpdate(t, mock, state, plan)
	if mock.attachCalls+mock.detachCalls != 0 {
		t.Fatalf("null plan: want 0 ops, got attach=%d detach=%d", mock.attachCalls, mock.detachCalls)
	}
}
```

- [ ] **Step 2: Run tests**

```bash
go test ./internal/provider/ -run "TestUpdate_Network" -v
```

Expected: 6 tests PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/instance_resource_update_test.go
git commit -m "test: add network reconcile unit tests"
```

---

## Task 9: Backup schedule resource

**Files:**
- Create: `internal/provider/backup_schedule_resource.go`

- [ ] **Step 1: Create backup schedule resource file**

```go
// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

const (
	backupScheduleCreateTimeout = 2 * time.Minute
	backupSchedulePollInterval  = 5 * time.Second
)

type BackupScheduleAPI interface {
	CreateBackupSchedule(ctx context.Context, instanceID string, req client.BackupScheduleCreate) error
	ListBackupSchedules(ctx context.Context, instanceID string) ([]client.BackupSchedule, error)
	GetBackupSchedule(ctx context.Context, instanceID, scheduleID string) (*client.BackupSchedule, error)
	DeleteBackupSchedule(ctx context.Context, scheduleID int64) error
}

type backupScheduleResource struct {
	api BackupScheduleAPI
}

type BackupScheduleResourceModel struct {
	ID         types.String `tfsdk:"id"`
	InstanceID types.String `tfsdk:"instance_id"`
	Name       types.String `tfsdk:"name"`
	BackupType types.String `tfsdk:"backup_type"`
	Rotation   types.Int64  `tfsdk:"rotation"`
	RunAt      types.String `tfsdk:"run_at"`
}

func NewBackupScheduleResource() resource.Resource { return &backupScheduleResource{} }

func (r *backupScheduleResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "cloudfly_backup_schedule"
}

func (r *backupScheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a CloudFly instance backup schedule.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"instance_id": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"name":        schema.StringAttribute{Optional: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"backup_type": schema.StringAttribute{Optional: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"rotation":    schema.Int64Attribute{Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"run_at":      schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		},
	}
}

func (r *backupScheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected ProviderData type", "expected *client.Client")
		return
	}
	r.api = c
}

func (r *backupScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BackupScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	instID := plan.InstanceID.ValueString()
	backupType := plan.BackupType.ValueString()
	if backupType == "" {
		backupType = "weekly"
	}

	createReq := client.BackupScheduleCreate{
		Name:       plan.Name.ValueString(),
		BackupType: backupType,
	}

	if err := r.api.CreateBackupSchedule(ctx, instID, createReq); err != nil {
		resp.Diagnostics.AddError("Failed to create backup schedule", err.Error())
		return
	}

	found, err := waitForBackupSchedule(ctx, r.api, instID, backupType, plan.Name.ValueString(), backupScheduleCreateTimeout, backupSchedulePollInterval)
	if err != nil {
		resp.Diagnostics.AddError("Backup schedule did not appear", err.Error())
		return
	}

	backupScheduleToModel(found, &plan)
	plan.BackupType = types.StringValue(backupType)
	if plan.Name.ValueString() != "" {
		plan.Name = types.StringValue(plan.Name.ValueString())
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func waitForBackupSchedule(ctx context.Context, api BackupScheduleAPI, instID, backupType, name string, timeout, interval time.Duration) (*client.BackupSchedule, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		schedules, err := api.ListBackupSchedules(ctx, instID)
		if err != nil {
			return nil, fmt.Errorf("list backup schedules: %w", err)
		}
		for i := range schedules {
			if schedules[i].Instance == instID && schedules[i].BackupType == backupType {
				if name == "" || strings.Contains(schedules[i].BackupName, name) {
					return &schedules[i], nil
				}
			}
		}
		time.Sleep(interval)
	}
	return nil, fmt.Errorf("backup schedule (type=%q, name=%q) not found on instance %q within %s", backupType, name, instID, timeout)
}

func (r *backupScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BackupScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedule, err := r.api.GetBackupSchedule(ctx, state.InstanceID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read backup schedule", err.Error())
		return
	}
	backupScheduleToModel(schedule, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *backupScheduleResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "All backup schedule attributes use RequiresReplace; Update should not be reached")
}

func (r *backupScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BackupScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseScheduleID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid schedule id", err.Error())
		return
	}
	if err := r.api.DeleteBackupSchedule(ctx, id); err != nil {
		resp.Diagnostics.AddError("Failed to delete backup schedule", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *backupScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func backupScheduleToModel(bs *client.BackupSchedule, m *BackupScheduleResourceModel) {
	m.ID = types.StringValue(fmt.Sprintf("%d", bs.ID))
	m.Rotation = types.Int64Value(bs.Rotation)
	m.RunAt = types.StringValue(bs.RunAt)
	if bs.Instance != "" {
		m.InstanceID = types.StringValue(bs.Instance)
	}
	if bs.BackupName != "" {
		m.Name = types.StringValue(bs.BackupName)
	}
	if bs.BackupType != "" {
		m.BackupType = types.StringValue(bs.BackupType)
	}
}

func parseScheduleID(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	if err != nil {
		return 0, fmt.Errorf("invalid schedule id %q: %w", s, err)
	}
	return id, nil
}
```

- [ ] **Step 2: Register resource in provider.go**

In `provider.go`, add to `Resources` function after `NewSnapshotResource`:

```go
		NewBackupScheduleResource,
```

Also update the import if `time` is not already imported (it isn't, but `provider.go` doesn't use `time` — actually check: it doesn't import `time`. No import change needed in provider.go.)

- [ ] **Step 3: Build check**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/provider/backup_schedule_resource.go internal/provider/provider.go
git commit -m "feat: add cloudfly_backup_schedule resource"
```

---

## Task 10: Unit test — backup schedule

**Files:**
- Create: `internal/provider/backup_schedule_resource_test.go`

- [ ] **Step 1: Create backup schedule resource test file**

```go
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
```

- [ ] **Step 2: Run tests**

```bash
go test ./internal/provider/ -run "TestBackup|TestWaitForBackup|TestParseSchedule" -v
```

Expected: 7 tests PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/backup_schedule_resource_test.go
git commit -m "test: add backup schedule resource unit tests"
```

---

## Task 11: Acceptance test — backup schedule

**Files:**
- Modify: `internal/provider/phase3_acc_test.go`

- [ ] **Step 1: Add backup schedule acceptance test**

Insert before the `phase3BaseAttrs` constant (before line 295):

```go
// TestAccPhase3_BackupSchedule creates a backup schedule and verifies computed attrs.
func TestAccPhase3_BackupSchedule(t *testing.T) {
	testAccPreCheck(t)
	requireAccCreate(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPhase3BackupScheduleConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("cloudfly_backup_schedule.test", "id", regexp.MustCompile(`[0-9]+`)),
					resource.TestCheckResourceAttr("cloudfly_backup_schedule.test", "backup_type", "weekly"),
					resource.TestMatchResourceAttr("cloudfly_backup_schedule.test", "rotation", regexp.MustCompile(`[0-9]+`)),
					resource.TestMatchResourceAttr("cloudfly_backup_schedule.test", "run_at", regexp.MustCompile(`.+`)),
				),
			},
		},
	})
}

func testAccPhase3BackupScheduleConfig() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name = "tf-acc-backup"
%s
}

resource "cloudfly_backup_schedule" "test" {
  instance_id = cloudfly_instance.test.id
  backup_type = "weekly"
  name        = "tf-acc-backup"
}
`, phase3BaseAttrs)
}
```

- [ ] **Step 2: Add acceptance test — network_ids attribute**

Insert after backup schedule test:

```go
// TestAccPhase3_NetworkIDs verifies the network_ids attribute can be set.
func TestAccPhase3_NetworkIDs(t *testing.T) {
	testAccPreCheck(t)
	requireAccCreate(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPhase3NetworkIDsConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("cloudfly_instance.test", "id", regexp.MustCompile(`.+`)),
					resource.TestCheckResourceAttr("cloudfly_instance.test", "network_ids.#", "0"),
				),
			},
		},
	})
}

func testAccPhase3NetworkIDsConfig() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name        = "tf-acc-networks"
  network_ids = []
%s
}
`, phase3BaseAttrs)
}
```

- [ ] **Step 3: Add acceptance test — IPv6 enable (conditional)**

Insert after network IDs test:

```go
// TestAccPhase3_IPv6Enable creates an instance and enables IPv6 post-create.
// Skipped unless CLOUDFLY_ACC_CREATE=1 AND CLOUDFLY_ACC_IPV6=1 (IPv6 costs).
func TestAccPhase3_IPv6Enable(t *testing.T) {
	testAccPreCheck(t)
	if os.Getenv("CLOUDFLY_ACC_IPV6") == "" {
		t.Skip("CLOUDFLY_ACC_IPV6 not set; skipping IPv6 enable test")
	}
	requireAccCreate(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without IPv6
			{
				Config: testAccPhase3NoIPv6Config(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "enable_ipv6", "false"),
				),
			},
			// Enable IPv6
			{
				Config: testAccPhase3EnableIPv6Config(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "enable_ipv6", "true"),
				),
			},
		},
	})
}

func testAccPhase3NoIPv6Config() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name        = "tf-acc-ipv6"
  enable_ipv6 = false
%s
}
`, phase3BaseAttrs)
}

func testAccPhase3EnableIPv6Config() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name        = "tf-acc-ipv6"
  enable_ipv6 = true
%s
}
`, phase3BaseAttrs)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/provider/phase3_acc_test.go
git commit -m "test: add backup schedule and network_ids acceptance tests"
```

---

## Task 12: ROADMAP update

**Files:**
- Modify: `ROADMAP.md`

- [ ] **Step 1: Mark all 3 Phase 03 items as done**

Replace lines 61-62:

```markdown
- [ ] Network interface management (list interfaces endpoint commented out in API spec)
- [ ] IPv6 configuration (available at create time via RequiresReplace; post-create niche)
```

with:

```markdown
- [x] Network interface management
- [x] IPv6 configuration
```

Replace line 72:

```markdown
- [ ] Backup management (backup schedule create endpoint commented out; read-only data source available)
```

with:

```markdown
- [x] Backup management
```

- [ ] **Step 2: Commit**

```bash
git add ROADMAP.md
git commit -m "docs: mark Phase 03 network/backup/IPv6 as complete"
```

---

## Task 13: Documentation

**Files:**
- Modify: `docs/resources/instance.md`
- Create: `docs/resources/backup_schedule.md`

- [ ] **Step 1: Regenerate docs**

```bash
go generate ./...
```

Or if using tfplugindocs:

```bash
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest
```

- [ ] **Step 2: Verify docs**

```bash
ls docs/resources/backup_schedule.md
```

Expected: file exists.

- [ ] **Step 3: Commit**

```bash
git add docs/
git commit -m "docs: add backup_schedule resource doc and update instance doc"
```

---

## Task 14: Final verification

- [ ] **Step 1: Run all unit tests**

```bash
go test ./internal/provider/ -v 2>&1 | tail -50
```

Expected: All tests PASS.

- [ ] **Step 2: Run all unit tests across codebase**

```bash
go test ./... 2>&1 | tail -20
```

Expected: All tests PASS.

- [ ] **Step 3: Build**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 4: Format**

```bash
go fmt ./...
```

Expected: no changes or minimal.

---

## Task 15: Commit remaining changes and push

- [ ] **Step 1: Final commit**

```bash
git add -A
git status
```

If clean, skip. If changes remain:

```bash
git commit -m "chore: final tidy-up Phase 03 implementation"
```

- [ ] **Step 2: Verify full diff**

```bash
git log --oneline master..HEAD
```

Expected: List of all commits on this branch.
