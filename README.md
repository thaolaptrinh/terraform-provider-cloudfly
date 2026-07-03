# Terraform Provider for CloudFly

> Community Terraform provider for the CloudFly Cloud Platform.

The CloudFly Terraform Provider allows you to provision and manage CloudFly cloud resources using Terraform.

## Features

### Resources

- `cloudfly_instance` — Manage compute instances
- `cloudfly_backup_schedule` — Manage instance backup schedules
- `cloudfly_snapshot` — Manage instance snapshots

### Data Sources

- `cloudfly_regions` — List available regions
- `cloudfly_images` — List available images
- `cloudfly_instance_options` — List available instance flavors
- `cloudfly_instance_price` — Query instance pricing
- `cloudfly_ssh_keys` — List SSH keys
- `cloudfly_instance_metrics` — Get instance metrics
- `cloudfly_instance_usage` — Get instance usage history
- `cloudfly_usage_summary` — Export usage summary
- `cloudfly_backup_schedules` — List backup schedules for an instance

See [ROADMAP.md](ROADMAP.md) for the complete development roadmap.

---

## Requirements

- Terraform >= 1.6
- Go >= 1.24

---

## Installation

```hcl
terraform {
  required_providers {
    cloudfly = {
      source  = "thaolaptrinh/cloudfly"
      version = "~> 0.1"
    }
  }
}
```

---

## Provider Configuration

```hcl
provider "cloudfly" {
  api_key = var.cloudfly_api_key
}
```

| Argument | Required | Description |
|---|---|---|
| `api_key` | Yes | CloudFly API key. Can also be set via `CLOUDFLY_API_KEY` environment variable. |
| `base_url` | No | API base URL. Defaults to `https://api.cloudfly.vn/backend/api`. Can also be set via `CLOUDFLY_BASE_URL`. |

---

## Examples

### Create an Instance

```hcl
data "cloudfly_regions" "all" {}

data "cloudfly_images" "all" {}

resource "cloudfly_instance" "web" {
  name        = "web-01"
  region      = "CLOUD-HN02"
  flavor_type = "Standard"
  image_name  = "CentOS-7.9"
  ram         = 1
  vcpus       = 1
  disk        = 20
  ssh_key_ids = [123]

  enable_ipv6            = true
  enable_private_network = true
  auto_backup            = true
}

output "instance_ip" {
  value = cloudfly_instance.web.access_ipv4
}
```

### Attach Security Groups and Networks

```hcl
resource "cloudfly_instance" "web" {
  name               = "web-01"
  region             = "CLOUD-HN02"
  flavor_type        = "Standard"
  image_name         = "CentOS-7.9"
  ram                = 1
  vcpus              = 1
  disk               = 20
  security_group_ids = ["sg-abc123"]
  network_ids        = ["net-xyz789"]
}
```

### Manage Backup Schedules

```hcl
resource "cloudfly_backup_schedule" "daily" {
  instance_id = cloudfly_instance.web.id
  backup_type = "daily"
}
```

### Create Snapshots

```hcl
resource "cloudfly_snapshot" "before_upgrade" {
  instance_id = cloudfly_instance.web.id
  name        = "before-upgrade"
  description = "Snapshot before system upgrade"
}
```

### Get Metrics

```hcl
data "cloudfly_instance_metrics" "cpu" {
  instance_id = cloudfly_instance.web.id
  metric_type = "vcpu"
  start_time  = "1h"
}
```

### Manage Instance State

```hcl
resource "cloudfly_instance" "web" {
  # ... other attributes ...
  power_state = "stopped" # Start/stop the instance
}

# Reboot an instance (set to true to trigger reboot, resets afterward)
resource "cloudfly_instance" "web" {
  # ... other attributes ...
  reboot = true
}

# Change password
resource "cloudfly_instance" "web" {
  # ... other attributes ...
  admin_password = "new-secure-password"
}
```

### Import Existing Resources

```sh
terraform import cloudfly_instance.example "<instance-id>"
terraform import cloudfly_backup_schedule.example "<instance-id>/<backup-schedule-id>"
terraform import cloudfly_snapshot.example "<instance-id>/<snapshot-id>"
```

---

## Building

Clone the repository.

```bash
git clone https://github.com/thaolaptrinh/terraform-provider-cloudfly.git
```

Build the provider.

```bash
go install
```

or

```bash
go build
```

---

## Development

Generate provider documentation.

```bash
make generate
```

Run unit tests.

```bash
make test
```

Run acceptance tests.

```bash
make testacc
```

> **Warning**
>
> Acceptance tests create real CloudFly resources and may incur charges.

---

## Project Structure

```text
terraform-provider-cloudfly
├── .github/
├── META.d/
├── docs/
├── examples/
├── internal/
├── tools/
├── CHANGELOG.md
├── LICENSE
├── README.md
├── ROADMAP.md
├── GNUmakefile
├── go.mod
├── go.sum
├── main.go
└── terraform-registry-manifest.json
```

---

## Contributing

Bug reports, feature requests, and pull requests are welcome.

Please open an issue before submitting significant changes.

---

## Disclaimer

This project is community-maintained and is **not affiliated with or officially maintained by CloudFly**.

CloudFly is a trademark of its respective owner.

---

## License

Licensed under the Mozilla Public License 2.0 (MPL-2.0).

See the `LICENSE` file for details.
