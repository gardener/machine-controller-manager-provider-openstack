// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/attributestags"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
)

var _ Network = &neutronV2{}

// neutronV2 is a NeutronV2 client implementing the Network interface.
type neutronV2 struct {
	serviceClient *gophercloud.ServiceClient
}

func newNeutronV2(providerClient *gophercloud.ProviderClient, eo gophercloud.EndpointOpts) (*neutronV2, error) {
	nw, err := openstack.NewNetworkV2(providerClient, eo)
	if err != nil {
		return nil, fmt.Errorf("could not initialize network client: %v", err)
	}
	return &neutronV2{
		serviceClient: nw,
	}, nil
}

// GetSubnet fetches the subnet data from the supplied ID.
func (n *neutronV2) GetSubnet(ctx context.Context, id string) (*subnets.Subnet, error) {
	sn, err := subnets.Get(ctx, n.serviceClient, id).Extract()
	onCall("neutron")

	if err != nil {
		onFailure("neutron")
		return nil, err
	}
	return sn, nil
}

// CreatePort creates a Neutron port.
func (n *neutronV2) CreatePort(ctx context.Context, opts ports.CreateOptsBuilder) (*ports.Port, error) {
	p, err := ports.Create(ctx, n.serviceClient, opts).Extract()
	onCall("neutron")

	if err != nil {
		onFailure("neutron")
		return nil, err
	}
	return p, nil
}

// ListPorts lists all ports.
func (n *neutronV2) ListPorts(ctx context.Context, opts ports.ListOptsBuilder) ([]ports.Port, error) {
	pages, err := ports.List(n.serviceClient, opts).AllPages(ctx)
	onCall("neutron")

	if err != nil {
		onFailure("neutron")
		return nil, err
	}

	return ports.ExtractPorts(pages)
}

// UpdatePort updates the port from the supplied ID.
func (n *neutronV2) UpdatePort(ctx context.Context, id string, opts ports.UpdateOptsBuilder) error {
	_, err := ports.Update(ctx, n.serviceClient, id, opts).Extract()
	onCall("neutron")

	if err != nil {
		// skip registering not found errors as API errors
		if !IsNotFoundError(err) {
			onFailure("neutron")
		}
		return err
	}
	return nil
}

// DeletePort deletes the port from the supplied ID.
func (n *neutronV2) DeletePort(ctx context.Context, id string) error {
	err := ports.Delete(ctx, n.serviceClient, id).ExtractErr()

	onCall("neutron")
	if err != nil && !IsNotFoundError(err) {
		onFailure("neutron")
		return err
	}
	return nil
}

// NetworkIDFromName resolves the given network name to a unique ID.
func (n *neutronV2) NetworkIDFromName(ctx context.Context, name string) (string, error) {
	opts := networks.ListOpts{
		Name: name,
	}

	allPages, err := networks.List(n.serviceClient, opts).AllPages(ctx)
	onCall("neutron")
	if err != nil {
		onFailure("neutron")
		return "", fmt.Errorf("failed to list networks: %w", err)
	}

	allNetworks, err := networks.ExtractNetworks(allPages)
	if err != nil {
		onFailure("neutron")
		return "", fmt.Errorf("failed to extract networks: %w", err)
	}

	for _, net := range allNetworks {
		if net.Name == name {
			return net.ID, nil
		}
	}

	return "", fmt.Errorf("no network found with name: %s", name)
}

// GroupIDFromName resolves the given security group name to a unique ID.
func (n *neutronV2) GroupIDFromName(ctx context.Context, name string) (string, error) {
	listOpts := groups.ListOpts{
		Name: name,
	}

	allPages, err := groups.List(n.serviceClient, listOpts).AllPages(ctx)
	onCall("neutron")
	if err != nil {
		onFailure("neutron")
		return "", fmt.Errorf("failed to list security groups: %w", err)
	}

	allGroups, err := groups.ExtractGroups(allPages)
	if err != nil {
		return "", fmt.Errorf("failed to extract security groups: %w", err)
	}

	for _, group := range allGroups {
		if group.Name == name {
			return group.ID, nil
		}
	}

	return "", fmt.Errorf("no security group found with name: %s", name)
}

// PortIDFromName resolves the given port name to a unique ID.
func (n *neutronV2) PortIDFromName(ctx context.Context, name string) (string, error) {
	opts := ports.ListOpts{
		Name: name,
	}

	allPages, err := ports.List(n.serviceClient, opts).AllPages(ctx)
	onCall("neutron")
	if err != nil {
		onFailure("neutron")
		return "", fmt.Errorf("failed to list ports: %w", err)
	}

	allPorts, err := ports.ExtractPorts(allPages)
	if err != nil {
		onFailure("neutron")
		return "", fmt.Errorf("failed to extract ports: %w", err)
	}

	for _, port := range allPorts {
		if port.Name == name {
			return port.ID, nil
		}
	}

	return "", fmt.Errorf("no port found with name: %s", name)
}

func (n *neutronV2) TagPort(ctx context.Context, id string, tags []string) error {
	if len(tags) == 0 {
		return nil
	}
	tagOpts := attributestags.ReplaceAllOpts{Tags: tags}
	_, err := attributestags.ReplaceAll(ctx, n.serviceClient, "ports", id, tagOpts).Extract()
	onCall("neutron")
	if err != nil {
		onFailure("neutron")
		return err
	}
	return nil
}
