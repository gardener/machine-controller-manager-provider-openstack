// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
)

const (
	cinderService = "cinder"
	// VolumeStatusAvailable indicates that he volume is available to be attached.
	VolumeStatusAvailable = "available"
	// VolumeStatusCreating indicates that the volume is being created.
	VolumeStatusCreating = "creating"
	// VolumeStatusDownloading indicates that the volume is in downloading state.
	VolumeStatusDownloading = "downloading"
	// VolumeStatusDeleting indicates that the volume is in the process of being deleted.
	VolumeStatusDeleting = "deleting"
	// VolumeStatusError indicates that the volume is in error state.
	VolumeStatusError = "error"
	// VolumeStatusInUse indicates that the volume is currently in use.
	VolumeStatusInUse = "in-use"
)

var _ Storage = &cinderV3{}

type cinderV3 struct {
	serviceClient *gophercloud.ServiceClient
}

func newCinderV3(providerClient *gophercloud.ProviderClient, eo gophercloud.EndpointOpts) (*cinderV3, error) {
	storage, err := openstack.NewBlockStorageV3(providerClient, eo)
	if err != nil {
		return nil, fmt.Errorf("could not initialize storage client: %v", err)
	}

	return &cinderV3{
		serviceClient: storage,
	}, nil
}

// CreateVolume creates a Cinder volume.
func (c *cinderV3) CreateVolume(ctx context.Context, opts volumes.CreateOptsBuilder, hintOpts volumes.SchedulerHintOptsBuilder) (*volumes.Volume, error) {
	v, err := volumes.Create(ctx, c.serviceClient, opts, hintOpts).Extract()
	onCall(cinderService)
	if err != nil {
		onFailure(cinderService)
		return nil, err
	}
	return v, nil
}

// GetVolume retrieves information about a volume.
func (c *cinderV3) GetVolume(ctx context.Context, id string) (*volumes.Volume, error) {
	return volumes.Get(ctx, c.serviceClient, id).Extract()
}

// DeleteVolume deletes a volume
func (c *cinderV3) DeleteVolume(ctx context.Context, id string) error {
	err := volumes.Delete(ctx, c.serviceClient, id, volumes.DeleteOpts{}).ExtractErr()
	onCall(cinderService)
	if err != nil {
		onFailure(cinderService)
		return err
	}
	return nil
}

// VolumeIDFromName resolves the given volume name to a unique ID.
func (c *cinderV3) VolumeIDFromName(ctx context.Context, name string) (string, error) {
	listOpts := volumes.ListOpts{
		Name: name,
	}

	listFunc := func(ctx context.Context) ([]volumes.Volume, error) {
		allPages, err := volumes.List(c.serviceClient, listOpts).AllPages(ctx)
		onCall(cinderService)
		if err != nil {
			onFailure(cinderService)
			return nil, err
		}
		return volumes.ExtractVolumes(allPages)
	}

	getNameFunc := func(volume volumes.Volume) string {
		return volume.Name
	}

	volume, err := findSingleByName(ctx, listFunc, getNameFunc, name, "volume")

	return volume.ID, err
}

// ListVolumes lists all volumes
func (c *cinderV3) ListVolumes(ctx context.Context, opts volumes.ListOptsBuilder) ([]volumes.Volume, error) {
	vols, err := volumes.List(c.serviceClient, opts).AllPages(ctx)
	onCall(cinderService)
	if err != nil {
		onFailure(cinderService)
		return nil, err
	}

	return volumes.ExtractVolumes(vols)
}
