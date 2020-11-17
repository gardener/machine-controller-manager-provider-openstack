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
	corev1 "k8s.io/api/core/v1"
)

// ServiceProvider can construct a factory for openstack clients.
type ServiceFactory interface {
	// ServiceClientFactory can construct a factory for OpenStack clients using the credentials inside <secret>.
	ServiceClientFactory(secret *corev1.Secret) (ServiceClientFactory, error)
}

type ServiceClientFactory interface {
	// Compute() (Compute, error)
	// Network() (Network, error)
}

type Compute interface {}
type Network interface {}

type ServiceClientFactoryImpl struct{
	providerClient *gophercloud.ProviderClient
}

// ComputeClient implements the Compute interface. It can be used to interact with OpenStack Compute services.
type ComputeClient struct{}
type NetworkClient struct{}