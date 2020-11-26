/*
 * Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package openstack

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	corev1 "k8s.io/api/core/v1"
)

type ClientConstructor func (secret *corev1.Secret) (Factory, error)

type Factory interface {
	Compute() (Compute, error)
	Network() (Network, error)
}

type Compute interface{
	ServiceClient() *gophercloud.ServiceClient

	CreateServer(opts servers.CreateOptsBuilder) (*servers.Server, error)
	BootFromVolume(opts servers.CreateOptsBuilder) (*servers.Server, error)
	GetServer(id string)(*servers.Server, error)
	ListServers(opts servers.ListOptsBuilder) ([]servers.Server, error)
	DeleteServer(id string) error

	FlavorIDFromName(name string) (string, error)
	ImageIDFromName(name string) (string, error)
}

type Network interface{
	GetSubnet(id string)(*subnets.Subnet, error)

	CreatePort(opts ports.CreateOptsBuilder) (*ports.Port, error)
	ListPorts(opts ports.ListOptsBuilder)([]ports.Port, error)
	UpdatePort(id string, opts ports.UpdateOptsBuilder) error
	DeletePort(id string) error

	NetworkIDFromName(name string) (string, error)
	GroupIDFromName(name string) (string, error)
	PortIDFromName(name string) (string, error)
}

type ClientFactory struct{
	providerClient *gophercloud.ProviderClient
}

// novaV2 is a Nova client implementing the Compute interface.
type novaV2 struct{
	serviceClient *gophercloud.ServiceClient
}

// neutronV2 is a Neutron client implementing the Network interface.
type neutronV2 struct{
	serviceClient *gophercloud.ServiceClient
}
