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
	listOpts := networks.ListOpts{
		Name: name,
	}

	listFunc := func(ctx context.Context) ([]networks.Network, error) {
		allPages, err := networks.List(n.serviceClient, listOpts).AllPages(ctx)
		onCall("neutron")
		if err != nil {
			onFailure("neutron")
			return nil, err
		}
		return networks.ExtractNetworks(allPages)
	}

	getNameFunc := func(network networks.Network) string {
		return network.Name
	}

	network, err := findSingleByName(ctx, listFunc, getNameFunc, name)

	return network.ID, err
}

// GroupIDFromName resolves the given security group name to a unique ID.
func (n *neutronV2) GroupIDFromName(ctx context.Context, name string) (string, error) {
	listOpts := groups.ListOpts{
		Name: name,
	}

	listFunc := func(ctx context.Context) ([]groups.SecGroup, error) {
		allPages, err := groups.List(n.serviceClient, listOpts).AllPages(ctx)
		onCall("neutron")
		if err != nil {
			onFailure("neutron")
			return nil, err
		}
		return groups.ExtractGroups(allPages)
	}

	getNameFunc := func(sg groups.SecGroup) string {
		return sg.Name
	}

	sg, err := findSingleByName(ctx, listFunc, getNameFunc, name)

	return sg.ID, err
}

// PortIDFromName resolves the given port name to a unique ID.
func (n *neutronV2) PortIDFromName(ctx context.Context, name string) (string, error) {
	listOpts := ports.ListOpts{
		Name: name,
	}

	listFunc := func(ctx context.Context) ([]ports.Port, error) {
		allPages, err := ports.List(n.serviceClient, listOpts).AllPages(ctx)
		onCall("neutron")
		if err != nil {
			onFailure("neutron")
			return nil, err
		}
		return ports.ExtractPorts(allPages)
	}

	getNameFunc := func(port ports.Port) string {
		return port.Name
	}

	port, err := findSingleByName(ctx, listFunc, getNameFunc, name)

	return port.ID, err
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
