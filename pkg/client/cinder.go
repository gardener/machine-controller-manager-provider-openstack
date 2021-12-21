// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	utilGroups "github.com/gophercloud/utils/openstack/blockstorage/v3/volumes"
)

const (
	cinderService = "cinder"
	// VolumeStatusCreating is the status of a volume in creation.
	VolumeStatusCreating = "creating"
	// VolumeStatusInUse is the status of a volume that is currently used by an instance.
	VolumeStatusInUse = "in-use"
	// VolumeStatusAvailable denotees that he volume is available to be attached.
	VolumeStatusAvailable = "available"
	// VolumeStatusError denotes that the volume is in error state.
	VolumeStatusError = "error"
)

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

func (c *cinderV3) CreateVolume(opts volumes.CreateOptsBuilder) (*volumes.Volume, error) {
	v, err := volumes.Create(c.serviceClient, opts).Extract()
	onCall(cinderService)
	if err != nil {
		onFailure(cinderService)
		return nil, err
	}
	return v, nil
}

func (c *cinderV3) GetVolume(id string) (*volumes.Volume, error) {
	return volumes.Get(c.serviceClient, id).Extract()
}

func (c *cinderV3) DeleteVolume(id string) error {
	err := volumes.Delete(c.serviceClient, id, volumes.DeleteOpts{}).ExtractErr()
	onCall(cinderService)
	if err != nil {
		onFailure(cinderService)
		return err
	}
	return nil
}

// GroupIDFromName resolves the given security group name to a unique ID.
func (c *cinderV3) VolumeIDFromName(name string) (string, error) {
	id, err := utilGroups.IDFromName(c.serviceClient, name)

	onCall(cinderService)
	if err != nil {
		onFailure(cinderService)
		return "", err
	}
	return id, nil
}

func (c *cinderV3) ListVolumes(opts volumes.ListOptsBuilder) ([]volumes.Volume, error) {
	vols, err := volumes.List(c.serviceClient, opts).AllPages()
	onCall(cinderService)
	if err != nil {
		onFailure(cinderService)
		return nil, err
	}

	return volumes.ExtractVolumes(vols)
}
