// Copyright (c) Thao Nguyen. Individual contributor.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccInstance_basic verifies the cloudfly_instance resource can be created,
// read, and destroyed against the live CloudFly API. Skipped unless both
// CLOUDFLY_API_KEY and CLOUDFLY_ACC_CREATE are set (creation costs money).
func TestAccInstance_basic(t *testing.T) {
	testAccPreCheck(t)
	if os.Getenv("CLOUDFLY_ACC_CREATE") == "" {
		t.Skip("CLOUDFLY_ACC_CREATE not set; skipping instance creation (costs money)")
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: testAccInstanceConfig(),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestMatchResourceAttr("cloudfly_instance.test", "id", regexp.MustCompile(`.+`)),
				resource.TestMatchResourceAttr("cloudfly_instance.test", "status", regexp.MustCompile(`ACTIVE|BUILDING`)),
			),
		}},
	})
}

// TestAccInstance_import verifies terraform import works for cloudfly_instance.
func TestAccInstance_import(t *testing.T) {
	testAccPreCheck(t)
	if os.Getenv("CLOUDFLY_ACC_CREATE") == "" {
		t.Skip("CLOUDFLY_ACC_CREATE not set; skipping")
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:            testAccInstanceConfig(),
				ImportState:       true,
				ImportStateVerify: true,
				ResourceName:      "cloudfly_instance.test",
			},
		},
	})
}

func testAccInstanceConfig() string {
	return `
resource "cloudfly_instance" "test" {
  name        = "tf-acc-test"
  region      = "HN-Cloud01"
  flavor_type = "Standard"
  image_name  = "CentOS-7.9"
  ram         = 1
  vcpus       = 1
  disk        = 20
}
`
}
