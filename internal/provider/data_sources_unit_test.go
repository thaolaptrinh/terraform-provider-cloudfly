// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

// --- cloudfly_images (imagesToList) ---

func TestImagesToList(t *testing.T) {
	in := []client.Image{
		{ID: "img-1", Name: "CentOS-7.9"},
		{ID: "img-2", Name: "Ubuntu-22.04"},
	}
	l, diags := imagesToList(in)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if len(l.Elements()) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(l.Elements()))
	}
}

func TestImagesToList_Empty(t *testing.T) {
	l, diags := imagesToList(nil)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !l.IsNull() && len(l.Elements()) != 0 {
		t.Fatalf("expected empty list, got %v", l)
	}
}

// --- cloudfly_instance_metrics (readMetrics) ---

type mockMetricsAPI struct {
	result *client.MetricsResponse
	err    error
}

func (m *mockMetricsAPI) GetMetrics(context.Context, string, string, string) (*client.MetricsResponse, error) {
	return m.result, m.err
}

func TestReadMetrics(t *testing.T) {
	mock := &mockMetricsAPI{
		result: &client.MetricsResponse{Data: json.RawMessage(`{"cpu":[1,2,3]}`)},
	}
	m := &InstanceMetricsModel{
		InstanceID: types.StringValue("i1"),
		MetricType: types.StringValue("vcpu"),
		StartTime:  types.StringValue("1h"),
	}

	err := readMetrics(context.Background(), mock, m)
	if err != nil {
		t.Fatalf("readMetrics error: %v", err)
	}
	if m.Result.ValueString() != `{"cpu":[1,2,3]}` {
		t.Errorf("result = %q, want {\"cpu\":[1,2,3]}", m.Result.ValueString())
	}
}

func TestReadMetrics_Error(t *testing.T) {
	mock := &mockMetricsAPI{err: errSentinel("api error")}
	m := &InstanceMetricsModel{
		InstanceID: types.StringValue("i1"),
		MetricType: types.StringValue("vcpu"),
		StartTime:  types.StringValue("1h"),
	}

	err := readMetrics(context.Background(), mock, m)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- cloudfly_instance_usage (readUsage) ---

type mockUsageAPI struct {
	result []client.UsageItem
	err    error
}

func (m *mockUsageAPI) GetUsageHistory(context.Context, string) ([]client.UsageItem, error) {
	return m.result, m.err
}

func TestReadUsage(t *testing.T) {
	mock := &mockUsageAPI{
		result: []client.UsageItem{
			{"id": "u1", "amount": "100"},
		},
	}
	m := &InstanceUsageModel{InstanceID: types.StringValue("i1")}

	err := readUsage(context.Background(), mock, m)
	if err != nil {
		t.Fatalf("readUsage error: %v", err)
	}
	if m.Items.ValueString() == "" {
		t.Error("items should not be empty")
	}
}

func TestReadUsage_Error(t *testing.T) {
	mock := &mockUsageAPI{err: errSentinel("api error")}
	m := &InstanceUsageModel{InstanceID: types.StringValue("i1")}

	err := readUsage(context.Background(), mock, m)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReadUsage_Empty(t *testing.T) {
	mock := &mockUsageAPI{result: nil}
	m := &InstanceUsageModel{InstanceID: types.StringValue("i1")}

	err := readUsage(context.Background(), mock, m)
	if err != nil {
		t.Fatalf("readUsage error: %v", err)
	}
	if m.Items.ValueString() != "null" {
		t.Errorf("items = %q, want null for nil slice", m.Items.ValueString())
	}
}

// --- cloudfly_usage_summary (direct model assignment, no extraction needed) ---

func TestUsageSummaryDataSource_ModelAssignment(t *testing.T) {
	resp := &client.UsageSummaryResponse{CSVPath: "https://example.com/summary.csv"}
	m := &UsageSummaryModel{}
	m.CSVPath = types.StringValue(resp.CSVPath)

	if m.CSVPath.ValueString() != "https://example.com/summary.csv" {
		t.Errorf("csv_path = %q", m.CSVPath.ValueString())
	}
}
