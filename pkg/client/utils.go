// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

// onCall records a request to the specified service.
//
//nolint:unparam
func onCall(service string) {
	metrics.APIRequestCount.With(prometheus.Labels{"provider": "openstack", "service": service}).Inc()
}

// onFailure records a failure in the request to the specified service.
//
//nolint:unparam
func onFailure(service string) {
	metrics.APIFailedRequestCount.With(prometheus.Labels{"provider": "openstack", "service": service}).Inc()
}

func findSingleByName[T any](
	ctx context.Context,
	listFunc func(context.Context) ([]T, error),
	getName func(T) string,
	targetName string,
) (T, error) {
	var zero T // zero value to return on failure

	allItems, err := listFunc(ctx)
	if err != nil {
		return zero, fmt.Errorf("listing resources failed: %w", err)
	}

	var matches []T
	for _, item := range allItems {
		if getName(item) == targetName {
			matches = append(matches, item)
		}
	}

	switch len(matches) {
	case 0:
		return zero, fmt.Errorf("no resource found with name: %s", targetName)
	case 1:
		return matches[0], nil
	default:
		return zero, fmt.Errorf("multiple resources found with name: %s", targetName)
	}
}
