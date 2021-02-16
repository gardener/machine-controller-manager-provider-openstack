// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package provider contains the cloud provider specific implementations to manage machines
package driver

import (
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/client"
)

var (
	_ driver.Driver = &OpenstackDriver{}
)

// OpenstackDriver implements and handles requests via the Driver interface.
type OpenstackDriver struct {
	decoder           runtime.Decoder
	clientConstructor client.ClientConstructor
}

// NewOpenstackDriver returns a new instance of the Openstack driver.
func NewOpenstackDriver(decoder runtime.Decoder, constructor client.ClientConstructor) driver.Driver {
	return &OpenstackDriver{
		decoder:           decoder,
		clientConstructor: constructor,
	}
}
