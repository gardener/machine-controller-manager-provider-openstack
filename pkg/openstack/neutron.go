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

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/metrics"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/prometheus/client_golang/prometheus"
)

func newNeutronV2(providerClient *gophercloud.ProviderClient, eo gophercloud.EndpointOpts) (*neutronV2, error) {
	nw, err := openstack.NewNetworkV2(providerClient, eo)
	if err != nil {
		return nil, fmt.Errorf("could not initialize network client: %v", err)
	}
	return &neutronV2{
		serviceClient: nw,
	}, nil
}

func (n *neutronV2) GetSubnet(id string) (*subnets.Subnet, error) {
	sn, err := subnets.Get(n.serviceClient, id).Extract()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return nil, err
	}
	return sn, nil
}

func (n *neutronV2) CreatePort(opts ports.CreateOptsBuilder) (*ports.Port, error) {
	p, err := ports.Create(n.serviceClient, opts).Extract()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return nil, err
	}
	return p, nil
}

func (n *neutronV2) ListPorts(opts ports.ListOptsBuilder) ([]ports.Port, error) {
	pages, err := ports.List(n.serviceClient, opts).AllPages()
	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()

	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return nil, err
	}

	return ports.ExtractPorts(pages)
}

func (n *neutronV2) UpdatePort(id string, opts ports.UpdateOptsBuilder) error {
	_, err := ports.Update(n.serviceClient, id, opts).Extract()
	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()

	if err != nil {
		// skip registering not found errors as API errors
		if !IsNotFoundError(err) {
			metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		}
		return err
	}
	return nil
}

func (n *neutronV2) DeletePort(id string) error {
	err := ports.Delete(n.serviceClient, id).ExtractErr()

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
	if err != nil && !IsNotFoundError(err) {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return err
	}
	return nil
}

func (n *neutronV2) NetworkIDFromName(name string) (string, error) {
	id, err := networks.IDFromName(n.serviceClient, name)

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return "", err
	}
	return id, nil
}

func (n *neutronV2) GroupIDFromName(name string) (string, error) {
	id, err := groups.IDFromName(n.serviceClient, name)

	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return "", err
	}
	return id, nil
}

func (n *neutronV2) PortIDFromName(name string) (string, error) {
	id, err := ports.IDFromName(n.serviceClient, name)
	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()

	if err != nil {
		metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": "neutron"}).Inc()
		return "", err
	}
	return id, nil
}
