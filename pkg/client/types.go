// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

// Compute is an interface for communication with Nova service.
type Compute interface {
	// CreateServer creates a server.
	CreateServer(opts servers.CreateOptsBuilder) (*servers.Server, error)
	// BootFromVolume creates a server from a block device mapping.
	BootFromVolume(opts servers.CreateOptsBuilder) (*servers.Server, error)
	// GetServer fetches server data from the supplied ID.
	GetServer(id string) (*servers.Server, error)
	// ListServers lists all servers based on opts constraints.
	ListServers(opts servers.ListOptsBuilder) ([]servers.Server, error)
	// DeleteServer deletes a server with the supplied ID. If the server does not exist it returns nil.
	DeleteServer(id string) error

	// FlavorIDFromName resolves the given flavor name to a unique ID.
	FlavorIDFromName(name string) (string, error)
	// ImageIDFromName resolves the given image name to a unique ID.
	ImageIDFromName(name string) (string, error)
}

// Network is an interface for communication with Neutron service.
type Network interface {
	// GetSubnet fetches the subnet data from the supplied ID.
	GetSubnet(id string) (*subnets.Subnet, error)

	// CreatePort creates a Neutron port.
	CreatePort(opts ports.CreateOptsBuilder) (*ports.Port, error)
	// ListPorts lists all ports.
	ListPorts(opts ports.ListOptsBuilder) ([]ports.Port, error)
	// UpdatePort updates the port from the supplied ID.
	UpdatePort(id string, opts ports.UpdateOptsBuilder) error
	// DeletePort deletes the port from the supplied ID.
	DeletePort(id string) error

	// NetworkIDFromName resolves the given network name to a unique ID.
	NetworkIDFromName(name string) (string, error)
	// GroupIDFromName resolves the given security group name to a unique ID.
	GroupIDFromName(name string) (string, error)
	// PortIDFromName resolves the given port name to a unique ID.
	PortIDFromName(name string) (string, error)
	// TagPort tags a port with the specified labels.
	TagPort(id string, tags []string) error
}
