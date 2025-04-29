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
	// TODO The CreateOptsExt struct has been removed and a BlockDevice field added to the CreateOpts struct in openstack/compute/v2/servers
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
	page, err := images.List(c.serviceClient, listOpts).AllPages(ctx)
	onCall("nova")
	if err != nil {
		if !IsNotFoundError(err) {
			onFailure("nova")
		}
		return images.Image{}, fmt.Errorf("unable to list images: %w", err)
	}
	foundImages, err := images.ExtractImages(page)
	if err != nil {
		return images.Image{}, fmt.Errorf("unable to extract images: %w", err)
	}

	if len(foundImages) == 0 {
		return images.Image{}, fmt.Errorf("image with name %s not found", name)
	}

	return foundImages[0], nil
}

// FlavorIDFromName resolves the given flavor name to a unique ID.
func (c *novaV2) FlavorIDFromName(ctx context.Context, name string) (string, error) {
	allPages, err := flavors.ListDetail(c.serviceClient, nil).AllPages(ctx)
	onCall("nova")
	if err != nil {
		if !IsNotFoundError(err) {
			onFailure("nova")
		}
		return "", fmt.Errorf("unable to list flavors: %w", err)
	}
	allFlavors, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return "", fmt.Errorf("unable to extract flavors: %w", err)
	}

	for _, flavor := range allFlavors {
		if flavor.Name == name {
			return flavor.ID, nil
		}
	}

	return "", fmt.Errorf("flavor with name %q not found", name)
}
