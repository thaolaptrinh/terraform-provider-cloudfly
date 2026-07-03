// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type FlexString string

func (f *FlexString) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == "null" || s == "" {
		*f = ""
		return nil
	}
	if data[0] == '"' {
		var t string
		if err := json.Unmarshal(data, &t); err != nil {
			return err
		}
		*f = FlexString(t)
		return nil
	}
	*f = FlexString(data)
	return nil
}

type InstanceCreate struct {
	Name          string `json:"name,omitempty"`
	FlavorType    string `json:"flavor_type"`
	Region        string `json:"region"`
	ImageName     string `json:"image_name"`
	RAM           int    `json:"ram"`
	Disk          int    `json:"disk"`
	VCPUs         int    `json:"vcpus"`
	EnableIPv6    bool   `json:"enable_ipv6"`
	EnablePrivNet bool   `json:"enable_private_network"`
	AutoBackup    bool   `json:"auto_backup"`
	SSHKeyIDs     []int  `json:"ssh_key_ids,omitempty"`
}

type Instance struct {
	ID                    string     `json:"id"`
	DisplayName           string     `json:"display_name"`
	Name                  string     `json:"name"`
	Status                string     `json:"status"`
	Region                RegionRef  `json:"region"`
	AccessIPv4            string     `json:"accessIPv4"`
	AccessIPv6            string     `json:"accessIPv6"`
	Created               string     `json:"created"`
	Flavor                Flavor     `json:"flavor"`
	Image                 Image      `json:"image"`
	Username              string     `json:"username"`
	TaskState             string     `json:"task_state"`
	BackupServer          FlexString `json:"backup_server"`
	HostName              string     `json:"host_name"`
	StoppedByCloudfly     bool       `json:"stopped_by_cloudfly"`
	CurrentMonthTraffic   FlexString `json:"current_month_traffic"`
	CurrentMonthTrafficMB FlexString `json:"current_month_traffic_mb"`
	RemainMaxIPAddon      FlexString `json:"remain_max_ip_addon"`
}

// RegionRef is the nested region object returned by the instances list/detail
// endpoints. The OpenAPI spec types `region` as a string, but the live API
// returns this object — verified during Phase 2 acceptance testing.
type RegionRef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Flavor is the nested flavor object. RAM/VCPUs/Disk come from here because
// the detail endpoint does not echo them as top-level scalars.
type Flavor struct {
	Name        string      `json:"name"`
	MemoryMB    int         `json:"memory_mb"`
	VCPUs       int         `json:"vcpus"`
	RootGB      int         `json:"root_gb"`
	FlavorGroup FlavorGroup `json:"flavor_group"`
}

// (FlavorGroup is shared with options.go; Image is local to instances.)

// Image is the nested image object; the configured image name is Image.Name.
type Image struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type SecurityGroup struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type InterfaceItem struct {
	InterfaceID string `json:"interface_id"`
	NetworkID   string `json:"network_id"`
	SubnetID    string `json:"subnet_id"`
	IPVersion   string `json:"ip_version"`
	IsDefault   bool   `json:"is_default"`
	Gateway     string `json:"gateway"`
	IPAddress   string `json:"ip_address"`
}

type InterfaceGroup struct {
	Data        []InterfaceItem `json:"data"`
	NetworkName string          `json:"network_name"`
	IsPublic    bool            `json:"is_public"`
	IPV6Range   []interface{}   `json:"ipv6_range"`
}

type attachInterfaceRequest struct {
	NetworkID string `json:"network_id"`
}

type detachInterfaceRequest struct {
	InterfaceID string `json:"interface_id"`
}

type rebuildRequest struct {
	ImageID string `json:"image_id"`
}

type rebootRequest struct {
	RebootType string `json:"reboot_type"`
}

type renameRequest struct {
	Name string `json:"name"`
}

type passwordRequest struct {
	AdminPassword string `json:"admin_password"`
}

type reverseDNSRequest struct {
	ReverseDNS string `json:"reverse_dns"`
	IPAddress  string `json:"ip_address"`
}

type securityGroupRequest struct {
	SecurityGroupID string `json:"security_group_id"`
}

type listInstancesResponse struct {
	Count   int        `json:"count"`
	Results []Instance `json:"results"`
}

// CreateInstance POSTs and then polls the list (filtered by name) until the
// instance appears, because the POST response contains only {detail} (no ID).
func (c *Client) CreateInstance(ctx context.Context, req InstanceCreate) (string, error) {
	resp, err := c.Do(ctx, http.MethodPost, "/instances", req, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return "", err
	}
	// POST returns only {detail}; poll list filtered by name until found.
	if req.Name == "" {
		return "", fmt.Errorf("create returned no id and no name was supplied to search for")
	}
	deadline := time.Now().Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		list, err := c.ListInstances(ctx, req.Name)
		if err != nil {
			return "", err
		}
		var match *Instance
		for i := range list {
			if list[i].DisplayName == req.Name {
				if match != nil {
					return "", fmt.Errorf("ambiguous name %q: multiple instances match", req.Name)
				}
				match = &list[i]
			}
		}
		if match != nil {
			return match.ID, nil
		}
		time.Sleep(10 * time.Second)
	}
	return "", fmt.Errorf("timed out waiting for instance %q to appear in list", req.Name)
}

func (c *Client) GetInstance(ctx context.Context, id string) (*Instance, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/instances/"+id, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out Instance
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode instance: %w", err)
	}
	return &out, nil
}

func (c *Client) DeleteInstance(ctx context.Context, id string) error {
	resp, err := c.Do(ctx, http.MethodDelete, "/instances/"+id, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) ListInstances(ctx context.Context, search string) ([]Instance, error) {
	path := "/instances"
	if search != "" {
		path += "?search=" + search
	}
	resp, err := c.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out listInstancesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode instances list: %w", err)
	}
	return out.Results, nil
}

// WaitInstanceActive polls GetInstance until Status == "ACTIVE" (UPPERCASE per API).
func (c *Client) WaitInstanceActive(ctx context.Context, id string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		inst, err := c.GetInstance(ctx, id)
		if err == nil && inst.Status == "ACTIVE" {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("timed out waiting for instance %s to become ACTIVE", id)
}

// WaitInstanceDeleted polls GetInstance until HTTP 404.
func (c *Client) WaitInstanceDeleted(ctx context.Context, id string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		resp, err := c.Do(ctx, http.MethodGet, "/instances/"+id, nil, nil)
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("timed out waiting for instance %s to be deleted", id)
}

func (c *Client) StartInstance(ctx context.Context, id string) error {
	resp, err := c.Do(ctx, http.MethodGet, "/instances/"+id+"/start", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) StopInstance(ctx context.Context, id string) error {
	resp, err := c.Do(ctx, http.MethodGet, "/instances/"+id+"/stop", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) RebootInstance(ctx context.Context, id string) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+id+"/reboot", rebootRequest{RebootType: "soft"}, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) RenameInstance(ctx context.Context, id, name string) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+id+"/rename", renameRequest{Name: name}, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) ChangePassword(ctx context.Context, id, password string) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+id+"/change-password", passwordRequest{AdminPassword: password}, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) UpdateReverseDNS(ctx context.Context, id, dns, ip string) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+id+"/update-reverse-dns", reverseDNSRequest{ReverseDNS: dns, IPAddress: ip}, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) AddSecurityGroup(ctx context.Context, id, sgID string) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+id+"/add-security-group", securityGroupRequest{SecurityGroupID: sgID}, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) RemoveSecurityGroup(ctx context.Context, id, sgID string) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+id+"/remove-security-group", securityGroupRequest{SecurityGroupID: sgID}, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) ListSecurityGroups(ctx context.Context, id string) ([]SecurityGroup, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/instances/"+id+"/security-groups", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out []SecurityGroup
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode security groups: %w", err)
	}
	return out, nil
}

func (c *Client) WaitInstanceStopped(ctx context.Context, id string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		inst, err := c.GetInstance(ctx, id)
		if err == nil && (inst.Status == "SHUTOFF" || inst.Status == "STOPPED") {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("timed out waiting for instance %s to stop", id)
}

type imagesResponse struct {
	Count   int     `json:"count"`
	Results []Image `json:"results"`
}

func (c *Client) ListImages(ctx context.Context) ([]Image, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/images", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out imagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode images: %w", err)
	}
	return out.Results, nil
}

func (c *Client) RebuildInstance(ctx context.Context, id, imageID string) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+id+"/rebuild", rebuildRequest{ImageID: imageID}, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) EnableIPv6Range(ctx context.Context, id string) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+id+"/enable-ipv6-range", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) ListInterfaces(ctx context.Context, id string) ([]InterfaceGroup, error) {
	resp, err := c.Do(ctx, http.MethodGet, "/instances/"+id+"/interfaces", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := AsError(resp); err != nil {
		return nil, err
	}
	var out []InterfaceGroup
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode interfaces: %w", err)
	}
	return out, nil
}

func (c *Client) AttachInterface(ctx context.Context, id, networkID string) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+id+"/attach-interface", attachInterfaceRequest{NetworkID: networkID}, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}

func (c *Client) DetachInterface(ctx context.Context, id, interfaceID string) error {
	resp, err := c.Do(ctx, http.MethodPost, "/instances/"+id+"/detach-interface", detachInterfaceRequest{InterfaceID: interfaceID}, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return AsError(resp)
}
