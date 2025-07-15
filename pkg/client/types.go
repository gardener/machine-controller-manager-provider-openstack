// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
)

// Compute is an interface for communication with Nova service.
type Compute interface {
	// CreateServer creates a server.
	CreateServer(ctx context.Context, opts servers.CreateOptsBuilder, hintOpts servers.SchedulerHintOptsBuilder) (*servers.Server, error)
	// GetServer fetches server data from the supplied ID.
	GetServer(ctx context.Context, id string) (*servers.Server, error)
	// ListServers lists all servers based on opts constraints.
	ListServers(ctx context.Context, opts servers.ListOptsBuilder) ([]servers.Server, error)
	// DeleteServer deletes a server with the supplied ID. If the server does not exist it returns nil.
	DeleteServer(ctx context.Context, id string) error

	// FlavorIDFromName resolves the given flavor name to a unique ID.
	FlavorIDFromName(ctx context.Context, name string) (string, error)
	// ImageIDFromName resolves the given image name to a unique ID.
	ImageIDFromName(ctx context.Context, name string) (images.Image, error)
}

// Network is an interface for communication with Neutron service.
type Network interface {
	// GetSubnet fetches the subnet data from the supplied ID.
	GetSubnet(ctx context.Context, id string) (*subnets.Subnet, error)

	// CreatePort creates a Neutron port.
	CreatePort(ctx context.Context, opts ports.CreateOptsBuilder) (*ports.Port, error)
	// ListPorts lists all ports.
	ListPorts(ctx context.Context, opts ports.ListOptsBuilder) ([]ports.Port, error)
	// UpdatePort updates the port from the supplied ID.
	UpdatePort(ctx context.Context, id string, opts ports.UpdateOptsBuilder) error
	// DeletePort deletes the port from the supplied ID.
	DeletePort(ctx context.Context, id string) error

	// NetworkIDFromName resolves the given network name to a unique ID.
	NetworkIDFromName(ctx context.Context, name string) (string, error)
	// GroupIDFromName resolves the given security group name to a unique ID.
	GroupIDFromName(ctx context.Context, name string) (string, error)
	// PortIDFromName resolves the given port name to a unique ID.
	PortIDFromName(ctx context.Context, name string) (string, error)
	// TagPort tags a port with the specified labels.
	TagPort(ctx context.Context, id string, tags []string) error
}

// Storage is an interface for communication with Cinder service.
type Storage interface {
	// CreateVolume creates a Cinder volume.
	CreateVolume(ctx context.Context, opts volumes.CreateOptsBuilder, hintOpts volumes.SchedulerHintOptsBuilder) (*volumes.Volume, error)
	// GetVolume retrieves information about a volume.
	GetVolume(ctx context.Context, id string) (*volumes.Volume, error)
	// DeleteVolume deletes a volume
	DeleteVolume(ctx context.Context, id string) error
	// VolumeIDFromName resolves the given volume name to a unique ID.
	VolumeIDFromName(ctx context.Context, name string) (string, error)
	// ListVolumes lists all volumes
	ListVolumes(ctx context.Context, opts volumes.ListOptsBuilder) ([]volumes.Volume, error)
}
