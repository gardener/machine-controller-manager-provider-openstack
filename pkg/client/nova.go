// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
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

var _ Compute = &novaV2{}

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
func (c *novaV2) CreateServer(ctx context.Context, opts servers.CreateOptsBuilder, hintOpts servers.SchedulerHintOptsBuilder) (*servers.Server, error) {
	server, err := servers.Create(ctx, c.serviceClient, opts, hintOpts).Extract()
	onCall("nova")
	if err != nil {
		onFailure("nova")
		return nil, err
	}
	return server, nil
}

// GetServer fetches server data from the supplied ID.
func (c *novaV2) GetServer(ctx context.Context, id string) (*servers.Server, error) {
	server, err := servers.Get(ctx, c.serviceClient, id).Extract()

	onCall("nova")
	if err != nil {
		if !IsNotFoundError(err) {
			onFailure("nova")
		}
		return nil, err
	}
	return server, nil
}

// ListServers lists all servers based on opts constraints.
func (c *novaV2) ListServers(ctx context.Context, opts servers.ListOptsBuilder) ([]servers.Server, error) {
	pages, err := servers.List(c.serviceClient, opts).AllPages(ctx)

	onCall("nova")
	if err != nil {
		onFailure("nova")
		return nil, err
	}
	return servers.ExtractServers(pages)
}

// DeleteServer deletes a server with the supplied ID. If the server does not exist it returns nil.
func (c *novaV2) DeleteServer(ctx context.Context, id string) error {
	err := servers.Delete(ctx, c.serviceClient, id).ExtractErr()

	onCall("nova")
	if err != nil && !IsNotFoundError(err) {
		onFailure("nova")
		return err
	}
	return nil
}

// ImageIDFromName resolves the given image name to a unique ID.
func (c *novaV2) ImageIDFromName(ctx context.Context, name string) (images.Image, error) {
	listOpts := images.ListOpts{
		Name: name,
	}

	listFunc := func(ctx context.Context) ([]images.Image, error) {
		allPages, err := images.List(c.serviceClient, listOpts).AllPages(ctx)
		onCall("nova")
		if err != nil {
			onFailure("nova")
			return nil, err
		}
		return images.ExtractImages(allPages)
	}

	getNameFunc := func(image images.Image) string {
		return image.Name
	}

	return findSingleByName(ctx, listFunc, getNameFunc, name, "image")
}

// FlavorIDFromName resolves the given flavor name to a unique ID.
func (c *novaV2) FlavorIDFromName(ctx context.Context, name string) (string, error) {
	listFunc := func(ctx context.Context) ([]flavors.Flavor, error) {
		allPages, err := flavors.ListDetail(c.serviceClient, nil).AllPages(ctx)
		onCall("nova")
		if err != nil {
			onFailure("nova")
			return nil, err
		}
		return flavors.ExtractFlavors(allPages)
	}

	getNameFunc := func(flavor flavors.Flavor) string {
		return flavor.Name
	}

	flavor, err := findSingleByName(ctx, listFunc, getNameFunc, name, "flavor")

	return flavor.ID, err
}
