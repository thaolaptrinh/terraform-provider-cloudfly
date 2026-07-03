# Roadmap

This document outlines the planned development roadmap for the Terraform Provider for CloudFly.

The roadmap is organized around Terraform resources and capabilities rather than individual REST API endpoints, following conventions used by official Terraform providers.

---

# Phase 1 — Provider Foundation

## Provider

- [ ] Provider configuration
- [ ] API authentication
- [ ] HTTP client
- [ ] Diagnostics
- [ ] Logging
- [ ] Documentation generation
- [ ] Acceptance testing
- [ ] CI/CD
- [ ] Automated releases

---

# Phase 2 — Compute

## Resources

- [ ] `cloudfly_instance`

## Data Sources

- [ ] `cloudfly_regions`
- [ ] `cloudfly_instance_options`
- [ ] `cloudfly_instance_price`
- [ ] `cloudfly_ssh_keys`

---

# Phase 3 — Instance Management

Implement CloudFly instance capabilities using Terraform resource lifecycle operations where appropriate.

### Lifecycle

- [ ] Create
- [ ] Read
- [ ] Update
- [ ] Delete
- [ ] Import

### Operations

- [ ] Power management
- [ ] Rebuild
- [ ] Rename
- [ ] Password management

### Networking

- [ ] Network interface management
- [ ] IPv6 configuration
- [ ] Reverse DNS

### Security

- [ ] Security group management

### Backup

- [ ] Snapshot management
- [ ] Backup management

### Monitoring

- [ ] Metrics
- [ ] Usage history
- [ ] Usage summary

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
