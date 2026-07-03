// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

// Mapper coverage for cloudfly_instance_price is provided end-to-end by the
// acceptance test (TestAccInstancePrice via the real client). The Read path
// is a straight pass-through to the client GetPrice method plus two
// types.Float64Value assignments, with no non-trivial mapping to unit-test.
