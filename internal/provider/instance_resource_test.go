package provider

import (
	"context"
	"testing"
	"time"

	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

type fakeInstancesAPI struct {
	createdID string
	inst      *client.Instance
	err       error
}

func (f *fakeInstancesAPI) CreateInstance(ctx context.Context, req client.InstanceCreate) (string, error) {
	return f.createdID, f.err
}
func (f *fakeInstancesAPI) GetInstance(ctx context.Context, id string) (*client.Instance, error) {
	return f.inst, f.err
}
func (f *fakeInstancesAPI) DeleteInstance(ctx context.Context, id string) error { return f.err }
func (f *fakeInstancesAPI) WaitInstanceActive(ctx context.Context, id string, timeout, interval time.Duration) error {
	return f.err
}
func (f *fakeInstancesAPI) WaitInstanceDeleted(ctx context.Context, id string, timeout, interval time.Duration) error {
	return f.err
}

func TestInstanceToModel(t *testing.T) {
	m := &InstanceResourceModel{}
	instanceToModel(&client.Instance{ID: "i9", Status: "ACTIVE", AccessIPv4: "1.2.3.4", Created: "2026-01-01"}, m)
	if m.ID.ValueString() != "i9" || m.Status.ValueString() != "ACTIVE" || m.AccessIPv4.ValueString() != "1.2.3.4" {
		t.Fatalf("unexpected: %+v", m)
	}
}

func TestInstanceCreateFromModel(t *testing.T) {
	m := InstanceResourceModel{}
	// Leave SSHKeyIDs null to skip the ElementsAs path.
	req, diags := instanceCreateFromModel(context.Background(), m)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if req.SSHKeyIDs != nil {
		t.Errorf("expected nil SSHKeyIDs for null list, got %v", req.SSHKeyIDs)
	}
}
