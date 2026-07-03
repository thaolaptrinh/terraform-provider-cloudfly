# Roadmap

This document outlines the planned development roadmap for the Terraform Provider for CloudFly.

The roadmap is organized around Terraform resources and capabilities rather than individual REST API endpoints, following conventions used by official Terraform providers.

---

# Phase 1 — Provider Foundation

## Provider

- [x] Provider configuration
- [x] API authentication
- [x] HTTP client
- [x] Diagnostics
- [x] Logging
- [x] Documentation generation
- [x] Acceptance testing
- [x] CI/CD
- [x] Automated releases

---

# Phase 2 — Compute

## Resources

- [x] `cloudfly_instance`

## Data Sources

- [x] `cloudfly_regions`
- [x] `cloudfly_instance_options`
- [x] `cloudfly_instance_price`
- [x] `cloudfly_ssh_keys`

---

# Phase 3 — Instance Management

Implement CloudFly instance capabilities using Terraform resource lifecycle operations where appropriate.

### Lifecycle

- [x] Create
- [x] Read
- [x] Update
- [x] Delete
- [x] Import

### Operations

- [x] Power management
- [ ] Rebuild (API requires image_id — no list-images endpoint available)
- [x] Rename
- [x] Password management

### Networking

- [ ] Network interface management (list interfaces endpoint commented out in API spec)
- [ ] IPv6 configuration (available at create time via RequiresReplace; post-create niche)
- [x] Reverse DNS

### Security

- [x] Security group management

### Backup

- [x] Snapshot management
- [ ] Backup management (backup schedule create endpoint commented out; read-only data source available)

### Monitoring

- [x] Metrics
- [x] Usage history
- [x] Usage summary

---

# Phase 4 — Documentation

- [ ] Provider documentation
- [ ] Resource documentation
- [ ] Data source documentation
- [ ] Example configurations

---

# Phase 5 — Testing

- [ ] Unit tests
- [ ] Acceptance tests
- [ ] Import tests

---

# Future

Support additional CloudFly public APIs as they become available.

Future resources may include networking, storage, DNS, load balancing, and other CloudFly services once they are publicly exposed.

---

# Versioning

This project follows Semantic Versioning.

- **v0.x** — Active development
- **v1.x** — Stable releases
- **v2.x** — Breaking changes
