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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace         = "openstack"
	cloudAPISubsystem = "cloud_api"
)

var (
	// APIRequestCount Number of Cloud Service API requests, partitioned by provider, and service.
	APIRequestCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: cloudAPISubsystem,
		Name:      "requests_total",
		Help:      "Number of Cloud Service API requests, partitioned by provider, and service.",
	}, []string{"provider", "service"},
	)

	// APIFailedRequestCount Number of Failed Cloud Service API requests, partitioned by provider, and service.
	APIFailedRequestCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: cloudAPISubsystem,
		Name:      "requests_failed_total",
		Help:      "Number of Failed Cloud Service API requests, partitioned by provider, and service.",
	}, []string{"provider", "service"},
	)
)

func init() {
	prometheus.MustRegister(APIRequestCount)
	prometheus.MustRegister(APIFailedRequestCount)
}
