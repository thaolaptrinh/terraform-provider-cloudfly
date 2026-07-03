# Terraform Provider for CloudFly

> Community Terraform provider for CloudFly Cloud Platform.

The CloudFly Terraform Provider allows you to provision and manage CloudFly cloud resources using Terraform.

> **Status:** 🚧 Early Development

## Features

Current development focuses on providing first-class Terraform support for CloudFly public APIs.

### Planned Resources

- `cloudfly_instance`

### Planned Data Sources

- `cloudfly_regions`
- `cloudfly_instance_options`
- `cloudfly_instance_price`
- `cloudfly_ssh_keys`

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

---

## Example

```hcl
terraform {
  required_providers {
    cloudfly = {
      source = "thaolaptrinh/cloudfly"
    }
  }
}

provider "cloudfly" {
  api_key = var.cloudfly_api_key
}

resource "cloudfly_instance" "example" {
  name   = "web-01"
  region = "hn"
}
```

> **Note**
>
> The provider is under active development. Resource schemas and capabilities may evolve until the first stable release.

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
