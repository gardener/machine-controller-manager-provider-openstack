// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/metrics"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	_ Network = &neutronV2{}
)

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
func (n *neutronV2) GetSubnet(id string) (*subnets.Subnet, error) {
	sn, err := subnets.Get(n.serviceClient, id).Extract()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return nil, err
	}
	return sn, nil
}

// CreatePort creates a Neutron port.
func (n *neutronV2) CreatePort(opts ports.CreateOptsBuilder) (*ports.Port, error) {
	p, err := ports.Create(n.serviceClient, opts).Extract()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return nil, err
	}
	return p, nil
}

// ListPorts lists all ports.
func (n *neutronV2) ListPorts(opts ports.ListOptsBuilder) ([]ports.Port, error) {
	pages, err := ports.List(n.serviceClient, opts).AllPages()
	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()

	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return nil, err
	}

	return ports.ExtractPorts(pages)
}

// UpdatePort updates the port from the supplied ID.
func (n *neutronV2) UpdatePort(id string, opts ports.UpdateOptsBuilder) error {
	_, err := ports.Update(n.serviceClient, id, opts).Extract()
	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()

	if err != nil {
		// skip registering not found errors as API errors
		if !IsNotFoundError(err) {
			metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		}
		return err
	}
	return nil
}

// DeletePort deletes the port from the supplied ID.
func (n *neutronV2) DeletePort(id string) error {
	err := ports.Delete(n.serviceClient, id).ExtractErr()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
	if err != nil && !IsNotFoundError(err) {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return err
	}
	return nil
}

// NetworkIDFromName resolves the given network name to a unique ID.
func (n *neutronV2) NetworkIDFromName(name string) (string, error) {
	id, err := networks.IDFromName(n.serviceClient, name)

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return "", err
	}
	return id, nil
}

// GroupIDFromName resolves the given security group name to a unique ID.
func (n *neutronV2) GroupIDFromName(name string) (string, error) {
	id, err := groups.IDFromName(n.serviceClient, name)

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return "", err
	}
	return id, nil
}

// PortIDFromName resolves the given port name to a unique ID.
func (n *neutronV2) PortIDFromName(name string) (string, error) {
	id, err := ports.IDFromName(n.serviceClient, name)
	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()

	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return "", err
	}
	return id, nil
}
