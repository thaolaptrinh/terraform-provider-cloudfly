// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"time"

	"github.com/thaolaptrinh/terraform-provider-cloudfly/internal/client"
)

// mockInstancesAPI implements InstancesAPI for unit testing the instance
// resource without a live HTTP server. Each method records its call args
// and returns the configured error (nil by default).
type mockInstancesAPI struct {
	// Recorded call args.
	startID, stopID, rebootID, renameID, renameName string
	reverseID, reverseDNS, reverseIP                string
	addSGID, removeSGID, listSGID                   string
	passwordID, passwordValue                       string

	// Call counters.
	startCalls, stopCalls, rebootCalls      int
	renameCalls, passwordCalls              int
	reverseCalls, addSGCalls, removeSGCalls int
	listSGCalls                             int
	getInstanceOptionsCalls                 int

	// Return values.
	getInstance                        *client.Instance
	getInstanceOptions                 []client.InstanceOption
	currentSGs                         []client.SecurityGroup
	startErr, stopErr, rebootErr       error
	renameErr, passwordErr             error
	reverseErr                         error
	addSGErr, removeSGErr              error
	listSGErr                          error
	getInstanceErr                     error
	getInstanceOptionsErr              error
	waitActiveErr, waitStopErr         error
	enableIPv6RangeCalls               int
	enableIPv6RangeID                  string
	enableIPv6RangeErr                 error
	attachNetworkID, detachInterfaceID string
	attachCalls, detachCalls           int
	listInterfacesReturn               []client.InterfaceGroup
	listInterfacesErr                  error
	attachErr, detachErr               error
}

func (m *mockInstancesAPI) CreateInstance(context.Context, client.InstanceCreate) (string, error) {
	return "", nil
}
func (m *mockInstancesAPI) GetInstance(ctx context.Context, id string) (*client.Instance, error) {
	if m.getInstanceErr != nil {
		return nil, m.getInstanceErr
	}
	return m.getInstance, nil
}
func (m *mockInstancesAPI) GetInstanceOptions(context.Context) ([]client.InstanceOption, error) {
	m.getInstanceOptionsCalls++
	if m.getInstanceOptionsErr != nil {
		return nil, m.getInstanceOptionsErr
	}
	return m.getInstanceOptions, nil
}
func (m *mockInstancesAPI) DeleteInstance(context.Context, string) error { return nil }
func (m *mockInstancesAPI) WaitInstanceActive(context.Context, string, time.Duration, time.Duration) error {
	return m.waitActiveErr
}
func (m *mockInstancesAPI) WaitInstanceDeleted(context.Context, string, time.Duration, time.Duration) error {
	return nil
}
func (m *mockInstancesAPI) WaitInstanceStopped(context.Context, string, time.Duration, time.Duration) error {
	return m.waitStopErr
}

func (m *mockInstancesAPI) StartInstance(_ context.Context, id string) error {
	m.startCalls++
	m.startID = id
	return m.startErr
}
func (m *mockInstancesAPI) StopInstance(_ context.Context, id string) error {
	m.stopCalls++
	m.stopID = id
	return m.stopErr
}
func (m *mockInstancesAPI) RebootInstance(_ context.Context, id string) error {
	m.rebootCalls++
	m.rebootID = id
	return m.rebootErr
}
func (m *mockInstancesAPI) RenameInstance(_ context.Context, id, name string) error {
	m.renameCalls++
	m.renameID, m.renameName = id, name
	return m.renameErr
}
func (m *mockInstancesAPI) ChangePassword(_ context.Context, id, pwd string) error {
	m.passwordCalls++
	m.passwordID, m.passwordValue = id, pwd
	return m.passwordErr
}
func (m *mockInstancesAPI) UpdateReverseDNS(_ context.Context, id, dns, ip string) error {
	m.reverseCalls++
	m.reverseID, m.reverseDNS, m.reverseIP = id, dns, ip
	return m.reverseErr
}
func (m *mockInstancesAPI) AddSecurityGroup(_ context.Context, id, sgID string) error {
	m.addSGCalls++
	m.addSGID = sgID
	return m.addSGErr
}
func (m *mockInstancesAPI) RemoveSecurityGroup(_ context.Context, id, sgID string) error {
	m.removeSGCalls++
	m.removeSGID = sgID
	return m.removeSGErr
}
func (m *mockInstancesAPI) ListSecurityGroups(_ context.Context, id string) ([]client.SecurityGroup, error) {
	m.listSGCalls++
	m.listSGID = id
	return m.currentSGs, m.listSGErr
}
func (m *mockInstancesAPI) EnableIPv6Range(_ context.Context, id string) error {
	m.enableIPv6RangeCalls++
	m.enableIPv6RangeID = id
	return m.enableIPv6RangeErr
}
func (m *mockInstancesAPI) ListInterfaces(_ context.Context, id string) ([]client.InterfaceGroup, error) {
	return m.listInterfacesReturn, m.listInterfacesErr
}
func (m *mockInstancesAPI) AttachInterface(_ context.Context, id, networkID string) error {
	m.attachCalls++
	m.attachNetworkID = networkID
	return m.attachErr
}
func (m *mockInstancesAPI) DetachInterface(_ context.Context, id, interfaceID string) error {
	m.detachCalls++
	m.detachInterfaceID = interfaceID
	return m.detachErr
}
