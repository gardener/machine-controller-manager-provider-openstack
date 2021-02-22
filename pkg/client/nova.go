// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/metrics"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Server status source: https://docs.openstack.org/api-guide/compute/server_concepts.html
	// ServerStatusActive indicates that the server is active.
	ServerStatusActive = "ACTIVE"
	// ServerStatusBuild indicates tha the server has not yet finished the build process.
	ServerStatusBuild = "BUILD"
	// ServerStatusDeleted indicates that the server is deleted.
	ServerStatusDeleted = "DELETED"
	// ServerStatusError indicates that the server is in error.
	ServerStatusError = "ERROR"
)

var (
	_ Compute = &novaV2{}
)

// novaV2 is a NovaV2 client implementing the Compute interface.
type novaV2 struct {
	serviceClient *gophercloud.ServiceClient
}

func newNovaV2(providerClient *gophercloud.ProviderClient, eo gophercloud.EndpointOpts) (*novaV2, error) {
	compute, err := openstack.NewComputeV2(providerClient, eo)
	if err != nil {
		return nil, fmt.Errorf("could not initialize compute client: %v", err)
	}
	return &novaV2{
		serviceClient: compute,
	}, nil
}

// CreateServer creates a server.
func (c *novaV2) CreateServer(opts servers.CreateOptsBuilder) (*servers.Server, error) {
	server, err := servers.Create(c.serviceClient, opts).Extract()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		return nil, err
	}

	return server, nil
}

// BootFromVolume creates a server from a block device mapping.
func (c *novaV2) BootFromVolume(opts servers.CreateOptsBuilder) (*servers.Server, error) {
	server, err := bootfromvolume.Create(c.serviceClient, opts).Extract()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		return nil, err
	}
	return server, nil
}

// GetServer fetches server data from the supplied ID.
func (c *novaV2) GetServer(id string) (*servers.Server, error) {
	server, err := servers.Get(c.serviceClient, id).Extract()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil {
		if !IsNotFoundError(err) {
			metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		}
		return nil, err
	}
	return server, nil
}

// ListServers lists all servers based on opts constraints.
func (c *novaV2) ListServers(opts servers.ListOptsBuilder) ([]servers.Server, error) {
	pages, err := servers.List(c.serviceClient, opts).AllPages()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		return nil, err
	}
	return servers.ExtractServers(pages)
}

// DeleteServer deletes a server with the supplied ID. If the server does not exist it returns nil.
func (c *novaV2) DeleteServer(id string) error {
	err := servers.Delete(c.serviceClient, id).ExtractErr()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil && !IsNotFoundError(err) {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		return err
	}
	return nil
}

// ImageIDFromName resolves the given image name to a unique ID.
func (c *novaV2) ImageIDFromName(name string) (string, error) {
	id, err := images.IDFromName(c.serviceClient, name)

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil {
		if !IsNotFoundError(err) {
			metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		}
		return "", err
	}

	return id, nil
}

// FlavorIDFromName resolves the given flavor name to a unique ID.
func (c *novaV2) FlavorIDFromName(name string) (string, error) {
	id, err := flavors.IDFromName(c.serviceClient, name)

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil {
		if !IsNotFoundError(err) {
			metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		}
		return "", err
	}

	return id, nil
}
