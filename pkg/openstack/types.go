// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	corev1 "k8s.io/api/core/v1"
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

// ClientConstructor can create a client factory from a secret.
type ClientConstructor func(secret *corev1.Secret) (Factory, error)

// Option can modify client parameters by manipulating EndpointOpts.
type Option func(opts gophercloud.EndpointOpts) gophercloud.EndpointOpts

// Factory is a client factory for OpenStack services and can construct individual service clients (e.g. Nova, Neutron).
type Factory interface {
	Compute(...Option) (Compute, error)
	Network(...Option) (Network, error)
}

// Compute is an interface for communication with Nova service.
type Compute interface {
	// ServiceClient returns the gophercloud.ServiceClient client for the service.
	// By itself is an internal implementation detail and should not really be exposed. In the in-tree implementation it is
	// used as an argument to CreateOpts to resolve Image and Flavor names to IDs (assuming they were provided by names and not IDs).
	// In this implementation we fetch the IDs before the create call and supply them directly, hence this is superfluous.
	// TODO(KA): Remove with further testing.
	// ServiceClient() *gophercloud.ServiceClient

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
}

// ClientFactory is an implementation of the Factory interface. It implements clients for NovaV2 and NeutronV2 OpenStack services.
type ClientFactory struct {
	providerClient *gophercloud.ProviderClient
}

// novaV2 is a NovaV2 client implementing the Compute interface.
type novaV2 struct {
	serviceClient *gophercloud.ServiceClient
}

// neutronV2 is a NeutronV2 client implementing the Network interface.
type neutronV2 struct {
	serviceClient *gophercloud.ServiceClient
}
