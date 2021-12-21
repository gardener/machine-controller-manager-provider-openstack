// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"github.com/gardener/machine-controller-manager/pkg/util/provider/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// onCall records a request to the specified service.
func onCall(service string) {
	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": service}).Inc()
}

// onFailure records a failure in the request to the specified service.
func onFailure(service string) {
	metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": service}).Inc()
}
