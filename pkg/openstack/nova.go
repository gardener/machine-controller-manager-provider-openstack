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
	"fmt"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/metrics"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/prometheus/client_golang/prometheus"
)

func newNovaV2(providerClient *gophercloud.ProviderClient, eo gophercloud.EndpointOpts) (*novaV2, error) {
	compute, err := openstack.NewComputeV2(providerClient, eo)
	if err != nil {
		return nil, fmt.Errorf("could not initialize compute client: %v", err)
	}
	return &novaV2{
		serviceClient: compute,
	}, nil
}

func (c *novaV2) ServiceClient() *gophercloud.ServiceClient {
	return c.serviceClient
}

func (c *novaV2) CreateServer(opts servers.CreateOptsBuilder) (*servers.Server, error) {
	server, err := servers.Create(c.serviceClient, opts).Extract()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		return nil, err
	}

	return server, nil
}

func (c *novaV2) BootFromVolume(opts servers.CreateOptsBuilder) (*servers.Server, error) {
	server, err := bootfromvolume.Create(c.serviceClient, opts).Extract()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		return nil, err
	}
	return server, nil
}

func (c *novaV2) GetServer(id string) (*servers.Server, error) {
	server, err := servers.Get(c.serviceClient, id).Extract()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil {
		if !IsNotFoundError(err) {
			metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		}
		return nil, err
	}
	return server, nil
}

func (c *novaV2) ListServers(opts servers.ListOptsBuilder) ([]servers.Server, error) {
	pages, err := servers.List(c.serviceClient, opts).AllPages()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		return nil, err
	}
	return servers.ExtractServers(pages)
}

func (c *novaV2) DeleteServer(id string) error {
	err := servers.Delete(c.serviceClient, id).ExtractErr()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil && !IsNotFoundError(err) {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		return err
	}
	return nil
}

func (c *novaV2) ImageIDFromName(name string) (string, error) {
	id, err := images.IDFromName(c.serviceClient, name)

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil {
		if !IsNotFoundError(err) {
			metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		}
		return "", err
	}

	return id, nil
}

func (c *novaV2) FlavorIDFromName(name string) (string, error) {
	id, err := flavors.IDFromName(c.serviceClient, name)

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
	if err != nil {
		if !IsNotFoundError(err) {
			metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "nova"}).Inc()
		}
		return "", err
	}

	return id, nil
}
