// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Acceptance tests for the four Phase 2 data sources. Read-only against the
// live API; require only CLOUDFLY_API_TOKEN (no resource creation).

func TestAccRegionsDataSource(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `data "cloudfly_regions" "all" {}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestMatchResourceAttr("data.cloudfly_regions.all", "regions.#", regexp.MustCompile(`[0-9]+`)),
			),
		}},
	})
}

func TestAccSSHKeysDataSource(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `data "cloudfly_ssh_keys" "all" {}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestMatchResourceAttr("data.cloudfly_ssh_keys.all", "ssh_keys.#", regexp.MustCompile(`[0-9]+`)),
			),
		}},
	})
}

func TestAccInstanceOptionsDataSource(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `data "cloudfly_instance_options" "all" {}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestMatchResourceAttr("data.cloudfly_instance_options.all", "options.#", regexp.MustCompile(`[0-9]+`)),
			),
		}},
	})
}

func TestAccInstancePriceDataSource(t *testing.T) {
	testAccPreCheck(t)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
data "cloudfly_instance_price" "test" {
  flavor_type = "Standard"
  ram         = 1
  disk        = 20
  vcpus       = 1
  region      = "HN-Cloud01"
  image_name  = "CentOS-7.9"
}
`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestMatchResourceAttr("data.cloudfly_instance_price.test", "price_per_month", regexp.MustCompile(`[0-9]+`)),
				resource.TestMatchResourceAttr("data.cloudfly_instance_price.test", "price_per_hour", regexp.MustCompile(`[0-9]+`)),
			),
		}},
	})
}
