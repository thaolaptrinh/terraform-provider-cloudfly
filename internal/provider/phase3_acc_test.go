// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// defaultTestSGID is the default security group CloudFly auto-attaches to
// new instances. Override with CLOUDFLY_ACC_SG_ID for a different account.
func defaultTestSGID() string {
	if id := os.Getenv("CLOUDFLY_ACC_SG_ID"); id != "" {
		return id
	}
	return "5493ddb2-585e-4b3b-8b0e-82a74b797370"
}

// requireAccCreate skips the test unless CLOUDFLY_ACC_CREATE=1 is set
// (instance creation costs money).
func requireAccCreate(t *testing.T) {
	t.Helper()
	if os.Getenv("CLOUDFLY_ACC_CREATE") == "" {
		t.Skip("CLOUDFLY_ACC_CREATE not set; skipping instance creation")
	}
}

// TestAccPhase3_InstanceLifecycle creates an instance and exercises all
// Phase 3 in-place update operations: stop, start, reboot, rename, password,
// reverse DNS, and data sources.
func TestAccPhase3_InstanceLifecycle(t *testing.T) {
	testAccPreCheck(t)
	requireAccCreate(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 0 — Create instance and verify Phase 2 + Phase 3 computed fields
			{
				Config: testAccPhase3CreateConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("cloudfly_instance.test", "id", regexp.MustCompile(`.+`)),
					resource.TestMatchResourceAttr("cloudfly_instance.test", "status", regexp.MustCompile(`ACTIVE|BUILDING`)),
					resource.TestMatchResourceAttr("cloudfly_instance.test", "access_ipv4", regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)),
					resource.TestCheckResourceAttr("cloudfly_instance.test", "power_state", "running"),
					resource.TestCheckResourceAttr("cloudfly_instance.test", "username", "root"),
				),
			},

			// Step 1 — Stop the instance
			{
				Config: testAccPhase3StopConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "power_state", "stopped"),
					resource.TestMatchResourceAttr("cloudfly_instance.test", "status", regexp.MustCompile(`STOPPED|SHUTOFF`)),
				),
			},

			// Step 2 — Start the instance back
			{
				Config: testAccPhase3StartConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "power_state", "running"),
				),
			},

			// Step 3 — Reboot the instance
			{
				Config: testAccPhase3RebootConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "power_state", "running"),
				),
			},

			// Step 4 — Rename + change password
			{
				Config: testAccPhase3RenamePasswordConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "name", "tf-acc-renamed"),
				),
			},

			// Step 5 — Reverse DNS update
			{
				Config: testAccPhase3ReverseDNSConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "reverse_dns", "tf-acc.example.com"),
				),
			},

			// Step 6 — Test all Phase 3 data sources
			{
				Config: testAccPhase3DataSourcesConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("data.cloudfly_instance_metrics.test", "result", regexp.MustCompile(`.+`)),
					resource.TestMatchResourceAttr("data.cloudfly_instance_usage.test", "items", regexp.MustCompile(`.+`)),
					resource.TestMatchResourceAttr("data.cloudfly_usage_summary.test", "csv_path", regexp.MustCompile(`.+`)),
					resource.TestMatchResourceAttr("data.cloudfly_backup_schedules.test", "schedules.#", regexp.MustCompile(`[0-9]+`)),
				),
			},
		},
	})
}

// TestAccPhase3_SecurityGroups tests the security_group_ids in-place update:
// add a group, remove all groups, add it back. Exercises both AddSecurityGroup
// and RemoveSecurityGroup client paths.
func TestAccPhase3_SecurityGroups(t *testing.T) {
	testAccPreCheck(t)
	requireAccCreate(t)

	sgID := defaultTestSGID()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 0 — Create instance (API auto-attaches default SG)
			{
				Config: testAccPhase3SGCreateConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("cloudfly_instance.test", "id", regexp.MustCompile(`.+`)),
				),
			},

			// Step 1 — Explicitly set security_group_ids to the default SG.
			// The API already has it attached, so this should be a no-op
			// (diff logic finds it in both current and plan sets).
			{
				Config: testAccPhase3SGAddConfig(sgID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "security_group_ids.#", "1"),
					resource.TestCheckResourceAttr("cloudfly_instance.test", "security_group_ids.0", sgID),
				),
			},

			// Step 2 — Remove all security groups (set to empty list).
			// This exercises the RemoveSecurityGroup path.
			{
				Config: testAccPhase3SGRemoveConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "security_group_ids.#", "0"),
				),
			},

			// Step 3 — Re-add the security group.
			// This exercises the AddSecurityGroup path on a clean instance.
			{
				Config: testAccPhase3SGAddConfig(sgID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "security_group_ids.#", "1"),
					resource.TestCheckResourceAttr("cloudfly_instance.test", "security_group_ids.0", sgID),
				),
			},
		},
	})
}

// TestAccPhase3_Idempotent verifies that re-applying the same configuration
// does not trigger redundant API calls or state drift:
//   - Stop an already-stopped instance (no-op)
//   - Start an already-running instance (no-op)
//   - reboot=true applied twice does not reboot twice
func TestAccPhase3_Idempotent(t *testing.T) {
	testAccPreCheck(t)
	requireAccCreate(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 0 — Create running instance
			{
				Config: testAccPhase3CreateConfig(),
			},

			// Step 1 — Stop the instance
			{
				Config: testAccPhase3StopConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "power_state", "stopped"),
				),
			},

			// Step 2 — Re-apply stopped config: must be a no-op (no error, no drift).
			// The framework verifies the plan is empty after apply.
			{
				Config:             testAccPhase3StopConfig(),
				ExpectNonEmptyPlan: false,
			},

			// Step 3 — Start the instance
			{
				Config: testAccPhase3StartConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "power_state", "running"),
				),
			},

			// Step 4 — Re-apply running config: must be a no-op.
			{
				Config:             testAccPhase3StartConfig(),
				ExpectNonEmptyPlan: false,
			},

			// Step 5 — Reboot (reboot=true)
			{
				Config: testAccPhase3RebootConfig(),
			},

			// Step 6 — Re-apply reboot=true: must NOT reboot again
			// (state.Reboot == plan.Reboot == true → condition is false).
			{
				Config:             testAccPhase3RebootConfig(),
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// TestAccPhase3_ConcurrentUpdate verifies that multiple attributes can be
// changed in a single Update call: power_state + name + password + reverse_dns.
func TestAccPhase3_ConcurrentUpdate(t *testing.T) {
	testAccPreCheck(t)
	requireAccCreate(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 0 — Create instance (running, default name)
			{
				Config: testAccPhase3CreateConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "name", "tf-acc-phase3"),
					resource.TestCheckResourceAttr("cloudfly_instance.test", "power_state", "running"),
				),
			},

			// Step 1 — Change everything at once: stop + rename + password + reverse DNS
			{
				Config: testAccPhase3ConcurrentConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "name", "tf-acc-concurrent"),
					resource.TestCheckResourceAttr("cloudfly_instance.test", "power_state", "stopped"),
					resource.TestCheckResourceAttr("cloudfly_instance.test", "reverse_dns", "concurrent.example.com"),
				),
			},
		},
	})
}

// TestAccPhase3_ImagesDataSource verifies the cloudfly_images data source
// returns a non-empty image list.
func TestAccPhase3_ImagesDataSource(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `data "cloudfly_images" "all" {}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestMatchResourceAttr("data.cloudfly_images.all", "images.#", regexp.MustCompile(`[0-9]+`)),
			),
		}},
	})
}

// TestAccPhase3_Snapshot creates an instance, takes a snapshot, reads it back.
func TestAccPhase3_Snapshot(t *testing.T) {
	testAccPreCheck(t)
	requireAccCreate(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPhase3SnapshotConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr("cloudfly_snapshot.test", "id", regexp.MustCompile(`.+`)),
					resource.TestMatchResourceAttr("cloudfly_snapshot.test", "name", regexp.MustCompile(`tf-acc-snapshot`)),
					resource.TestMatchResourceAttr("cloudfly_snapshot.test", "status", regexp.MustCompile(`.+`)),
					resource.TestMatchResourceAttr("cloudfly_snapshot.test", "size_in_gb", regexp.MustCompile(`.+`)),
					resource.TestMatchResourceAttr("cloudfly_snapshot.test", "created_at", regexp.MustCompile(`.+`)),
				),
			},
		},
	})
}

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
}
`, phase3BaseAttrs)
}

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
			{
				Config: testAccPhase3NoIPv6Config(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("cloudfly_instance.test", "enable_ipv6", "false"),
				),
			},
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

// --- Config helpers ---

const phase3BaseAttrs = `
  region      = "HN-Cloud01"
  flavor_type = "Standard"
  image_name  = "CentOS-7.9"
  ram         = 1
  vcpus       = 1
  disk        = 20
`

func testAccPhase3CreateConfig() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name = "tf-acc-phase3"
%s
}
`, phase3BaseAttrs)
}

func testAccPhase3StopConfig() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name        = "tf-acc-phase3"
  power_state = "stopped"
%s
}
`, phase3BaseAttrs)
}

func testAccPhase3StartConfig() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name        = "tf-acc-phase3"
  power_state = "running"
%s
}
`, phase3BaseAttrs)
}

func testAccPhase3RebootConfig() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name        = "tf-acc-phase3"
  power_state = "running"
  reboot      = true
%s
}
`, phase3BaseAttrs)
}

func testAccPhase3RenamePasswordConfig() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name           = "tf-acc-renamed"
  power_state    = "running"
  admin_password = "tf-acc-test-password"

%s
}
`, phase3BaseAttrs)
}

func testAccPhase3ReverseDNSConfig() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name        = "tf-acc-renamed"
  power_state = "running"
  reverse_dns = "tf-acc.example.com"
%s
}
`, phase3BaseAttrs)
}

func testAccPhase3ConcurrentConfig() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name           = "tf-acc-concurrent"
  power_state    = "stopped"
  admin_password = "tf-acc-test-password"
  reverse_dns    = "concurrent.example.com"
%s
}
`, phase3BaseAttrs)
}

func testAccPhase3DataSourcesConfig() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name = "tf-acc-phase3"
%s
}

data "cloudfly_instance_metrics" "test" {
  instance_id = cloudfly_instance.test.id
  metric_type = "vcpu"
  start_time  = "1h"
}

data "cloudfly_instance_usage" "test" {
  instance_id = cloudfly_instance.test.id
}

data "cloudfly_usage_summary" "test" {}

data "cloudfly_backup_schedules" "test" {
  instance_id = cloudfly_instance.test.id
}
`, phase3BaseAttrs)
}

func testAccPhase3SnapshotConfig() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name = "tf-acc-phase3-snap"
%s
}

resource "cloudfly_snapshot" "test" {
  instance_id = cloudfly_instance.test.id
  name        = "tf-acc-snapshot"
  description = "terraform acceptance test snapshot"
}
`, phase3BaseAttrs)
}

func testAccPhase3SGCreateConfig() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name = "tf-acc-phase3-sg"
%s
}
`, phase3BaseAttrs)
}

func testAccPhase3SGAddConfig(sgID string) string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name               = "tf-acc-phase3-sg"
  security_group_ids = ["%s"]
%s
}
`, sgID, phase3BaseAttrs)
}

func testAccPhase3SGRemoveConfig() string {
	return fmt.Sprintf(`
resource "cloudfly_instance" "test" {
  name               = "tf-acc-phase3-sg"
  security_group_ids = []
%s
}
`, phase3BaseAttrs)
}

func TestAccPhase3_Snapshot_Import(t *testing.T) {
	testAccPreCheck(t)
	requireAccCreate(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPhase3SnapshotConfig(),
			},
			{
				Config:            testAccPhase3SnapshotConfig(),
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "cloudfly_snapshot.test",
			},
		},
	})
}

func TestAccPhase3_BackupSchedule_Import(t *testing.T) {
	testAccPreCheck(t)
	requireAccCreate(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPhase3BackupScheduleConfig(),
			},
			{
				Config:            testAccPhase3BackupScheduleConfig(),
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "cloudfly_backup_schedule.test",
			},
		},
	})
}
