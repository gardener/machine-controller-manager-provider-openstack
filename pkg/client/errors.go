// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"errors"
	"net/http"

	"github.com/gophercloud/gophercloud/v2"
)

// IsNotFoundError checks if an error returned by OpenStack service calls is caused by HTTP 404 status code.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return true
	}

	if errors.As(err, &gophercloud.ErrResourceNotFound{}) {
		return true
	}

	return false
}

// IsUnauthorized checks if an error returned by OpenStack service calls is caused by HTTP 401 status code.
func IsUnauthorized(err error) bool {
	if err == nil {
		return false
	}

	return gophercloud.ResponseCodeIs(err, http.StatusUnauthorized)
}

// IsForbidden checks if an error returned by OpenStack service calls is caused by HTTP 403 status code.
func IsForbidden(err error) bool {
	if err == nil {
		return false
	}

	return gophercloud.ResponseCodeIs(err, http.StatusForbidden)
}
