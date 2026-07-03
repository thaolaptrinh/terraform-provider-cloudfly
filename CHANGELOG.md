## 0.1.0 (Unreleased)

FEATURES:

* **New Resource:** `cloudfly_instance` — manage CloudFly compute instances with support for power management, reboot, rename, password change, reverse DNS, security groups, network interfaces, IPv6, auto-backup, and SSH keys
* **New Resource:** `cloudfly_backup_schedule` — manage automatic backup schedules for instances
* **New Resource:** `cloudfly_snapshot` — manage instance snapshots
* **New Data Source:** `cloudfly_regions` — list available CloudFly regions
* **New Data Source:** `cloudfly_images` — list available instance images
* **New Data Source:** `cloudfly_instance_options` — list available instance flavors with pricing
* **New Data Source:** `cloudfly_instance_price` — query instance configuration pricing
* **New Data Source:** `cloudfly_ssh_keys` — list SSH keys in your account
* **New Data Source:** `cloudfly_instance_metrics` — retrieve CPU, memory, disk, interface, and packet metrics
* **New Data Source:** `cloudfly_instance_usage` — retrieve usage history for an instance
* **New Data Source:** `cloudfly_usage_summary` — export a usage summary CSV for all instances
* **New Data Source:** `cloudfly_backup_schedules` — list backup schedules for an instance
* **New Provider:** `cloudfly` provider with `api_key` and `base_url` configuration

ENHANCEMENTS:

* Provider built on `terraform-plugin-framework` v1.19.0 (protocol 6.0)
* All 3 resources support `import` (`terraform import`)
* Token-based API authentication via `Authorization: Token` header
* API key can be set via provider config or `CLOUDFLY_API_KEY` environment variable
* Base URL configurable via provider config or `CLOUDFLY_BASE_URL` environment variable
* Retryable HTTP client with configurable backoff
* Auto-generated documentation via `tfplugindocs`
* Example configurations for all resources and data sources
* Acceptance tests gated behind `TF_ACC` and `CLOUDFLY_ACC_CREATE` flags
* Automated releases via GoReleaser with multiplatform builds
* CI/CD via GitHub Actions
