// Copyright (c) Thao Nguyen. Individual contributor.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccProvider verifies the provider initialises and configures its client
// without error. It exercises the acceptance-test harness
// (testAccProtoV6ProviderFactories + testAccPreCheck) and is skipped unless
// CLOUDFLY_API_TOKEN is set.
func TestAccProvider(t *testing.T) {
	testAccPreCheck(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `provider "cloudfly" {}`,
			},
		},
	})
}
