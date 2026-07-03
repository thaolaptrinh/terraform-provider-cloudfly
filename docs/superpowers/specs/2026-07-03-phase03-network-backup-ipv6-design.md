# Phase 03: Network Interface, Backup Schedule, IPv6 Post-Create

## Summary

Implement 3 remaining Phase 03 roadmap items. API spec is outdated — live testing confirmed all endpoints are functional.

| # | Feature | Implementation | Scope |
|---|---------|---------------|-------|
| 1 | Network interface management | `network_ids` attribute on `cloudfly_instance` | Instance resource + client + tests |
| 2 | Backup schedule management | New `cloudfly_backup_schedule` resource | New resource + client extend + tests |
| 3 | IPv6 post-create | Remove `RequiresReplace`, update in `applyUpdate` | Instance resource + client + tests |

---

## 1. Network Interface Management

### API

| Method | Endpoint | Input | Output |
|--------|----------|-------|--------|
| GET | `/instances/{id}/interfaces` | - | `[{network_name, is_public, data: [{interface_id, network_id, ...}]}]` |
| POST | `/instances/{id}/attach-interface` | `{network_id}` | `{detail}` |
| POST | `/instances/{id}/detach-interface` | `{interface_id}` | `{detail}` |

Key constraint: attach uses `network_id`, detach uses `interface_id`.

### Schema

New attribute on `cloudfly_instance`:

```hcl
"network_ids": ListAttribute[String]
  Optional, no Computed
  Description: "Additional network IDs to attach. Default public network excluded from management."
```

### Client Types

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
```

### Reconcile Logic

```
reconcileNetworks(ctx, id, planNetworkIDs):
  if planNetworkIDs is null → skip
  groups = api.ListInterfaces(id)
  Filter out default public interfaces (is_default && is_public)
  Build currentNetworks set from remaining
  Attach: planIDs not in current
  Detach: current networkIDs not in plan → detach ALL interfaces in that network
```

### Design Decisions

- **No read-back**: `network_ids` not populated in `instanceToModel`. Source of truth = user config. Matches `security_group_ids` pattern.
- **No overlap with `enable_private_network`**: Separate mechanisms.
- **Default public network excluded**: Never detach the default public interface.
- **Detach all interfaces per network**: One network may have multiple interfaces (IPv4+IPv6).

---

## 2. Backup Schedule (`cloudfly_backup_schedule`)

### API

| Method | Endpoint | Input | Output |
|--------|----------|-------|--------|
| GET | `/instances/{id}/backup-server` | - | `[{id, instance, rotation, run_at, backup_name, backup_type}]` |
| GET | `/instances/{id}/backups` | - | `[{backup snaps}]` (data source potential) |
| POST | `/instances/{id}/create_backup_schedule` | `{name?, backup_type}` | No body (201) |
| DELETE | `/instances/backup-servers/{id}` | - | No body (204) |

No update API. Create does not return ID.

### Schema

```hcl
resource "cloudfly_backup_schedule" "example" {
  instance_id = cloudfly_instance.example.id  # Required, RequiresReplace
  backup_type = "weekly"                       # Optional, RequiresReplace, default "weekly"
  name        = "my-backup"                    # Optional, RequiresReplace
}
```

Computed attributes: `id` (string, converted from API int), `rotation` (int64), `run_at` (string).

All user-facing attributes use `RequiresReplace()` — no update API.

### Create Flow

```
1. POST /instances/{id}/create_backup_schedule
2. Poll GET /instances/{id}/backup-server every 5s, timeout 2min
3. Match by instance_id + backup_type + name
4. Set id, rotation, run_at
```

### Client (`internal/client/backups.go`)

Existing `BackupSchedule` struct reused. Add:
- `BackupScheduleCreate` struct
- `CreateBackupSchedule`, `GetBackupSchedule`, `ListBackupSchedules`, `DeleteBackupSchedule` methods

### File Layout

```
internal/client/backups.go                  (extend)
internal/provider/backup_schedule_resource.go  (new)
internal/provider/backup_schedule_resource_test.go  (new)
```

### Registration

Add `NewBackupScheduleResource` to `provider.go`.

---

## 3. IPv6 Post-Create

### API

| Method | Endpoint | Kết quả |
|--------|----------|---------|
| POST | `/instances/{id}/enable-ipv6-range` | 200 OK |
| POST | `/instances/{id}/enable-ipv6` | 400 (internal error, range needed first) |

### Change

Remove `RequiresReplace()` on `enable_ipv6`. Add to `applyUpdate`:

```
if !state.EnableIPv6.Equal(plan.EnableIPv6):
  if plan.EnableIPv6 && !state.EnableIPv6 → api.EnableIPv6Range(id)
  if !plan.EnableIPv6 && state.EnableIPv6 → no-op (cannot disable)
```

One-way only: false → true.

### Schema

```hcl
"enable_ipv6": BoolAttribute
  Optional
  # Removed: RequiresReplace()
```

### Client

Add `EnableIPv6Range(ctx, id string) error` — POST with empty body.

### Read-back

`instanceToModel` already maps `AccessIPv6`. No changes needed.

### Impact

- Non-breaking: existing `enable_ipv6: true` at create still works.
- User can now enable IPv6 post-create without destroy/recreate.

---

## Test Plan

### Unit Tests

| Feature | Test |
|---------|------|
| Network reconcile | `reconcileNetworks` add, remove, no-op, default public skip, multi-interface network detach |
| Backup schedule | `backupScheduleToModel` mapping, `waitForBackupSchedule` success/timeout/not-found |
| IPv6 update | `applyUpdate` calls `EnableIPv6Range` on false→true, no-op on true→false |

### Acceptance Tests

| Feature | Test |
|---------|------|
| Network | `network_ids` attribute round-trip (empty list), no crash on read |
| Backup schedule | Create with `cloudfly_backup_schedule`, verify computed attrs, delete |
| IPv6 | `enable_ipv6` false→true update (skip on CI without CLOUDFLY_ACC_IPV6 env) |

---

## File Changes Summary

| File | Action |
|------|--------|
| `internal/client/instances.go` | Add `InterfaceItem`, `InterfaceGroup`, interface methods, `EnableIPv6Range` |
| `internal/client/backups.go` | Add `BackupScheduleCreate`, CRUD methods |
| `internal/provider/instance_resource.go` | Add `network_ids` schema+model, `reconcileNetworks`, IPv6 update logic, extend `InstancesAPI` |
| `internal/provider/backup_schedule_resource.go` | New: resource CRUD |
| `internal/provider/backup_schedule_resource_test.go` | New: unit tests |
| `internal/provider/instance_resource_update_test.go` | Extend: network reconcile + IPv6 tests |
| `internal/provider/instance_resource_mocks_test.go` | Extend: interface/IPv6 mock fields |
| `internal/provider/provider.go` | Register `cloudfly_backup_schedule` |
| `internal/provider/phase3_acc_test.go` | Add acceptance tests |
| `ROADMAP.md` | Mark all 3 items done |
| `docs/resources/instance.md` | Document `network_ids` |
| `docs/resources/backup_schedule.md` | New: resource doc |
