// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package provider contains the cloud provider specific implementations to manage machines
package driver

import (
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/openstack"
)

var (
	_ driver.Driver = &OpenstackDriver{}
)

// OpenstackDriver is the struct that implements the driver interface
// It is used to implement the basic driver functionalities
type OpenstackDriver struct {
	decoder           runtime.Decoder
	clientConstructor openstack.ClientConstructor
}

// NewOpenstackDriver returns an empty provider object
func NewOpenstackDriver(decoder runtime.Decoder, constructor openstack.ClientConstructor) driver.Driver {
	return &OpenstackDriver{
		decoder:           decoder,
		clientConstructor: constructor,
	}
}

