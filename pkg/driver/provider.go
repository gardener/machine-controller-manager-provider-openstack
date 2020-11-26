/*
Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package provider contains the cloud provider specific implementations to manage machines
package driver

import (
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"k8s.io/apimachinery/pkg/runtime"

	api "github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/openstack"
)

// OpenstackDriver is the struct that implements the driver interface
// It is used to implement the basic driver functionalities
type OpenstackDriver struct {
	decoder runtime.Decoder
	clientConstructor openstack.ClientConstructor
}

// NewOpenstackDriver returns an empty provider object
func NewOpenstackDriver(decoder runtime.Decoder, constructor openstack.ClientConstructor) driver.Driver {
	return &OpenstackDriver{
		decoder: decoder,
		clientConstructor: constructor,
	}
}

type executor struct {
	compute openstack.Compute
	network openstack.Network
	cfg *api.MachineProviderConfig
}