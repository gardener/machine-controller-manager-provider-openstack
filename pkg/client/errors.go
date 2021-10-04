// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"errors"

	"github.com/gophercloud/gophercloud"
)

// IsNotFoundError checks if an error returned by OpenStack service calls is caused by HTTP 404 status code.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if errors.As(err, &gophercloud.ErrDefault404{}) {
		return true
	}

	var e gophercloud.Err404er
	if errors.As(err, &e) {
		return true
	}

	if errors.As(err, &gophercloud.ErrResourceNotFound{}) {
		return true
	}

	return false
}

// IsUnauthenticated checks if an error returned by OpenStack service calls is caused by HTTP 401 status code.
func IsUnauthenticated(err error) bool {
	if err == nil {
		return false
	}

	if errors.As(err, &gophercloud.ErrDefault401{}) {
		return true
	}

	var e gophercloud.Err401er
	return errors.As(err, &e)
}

// IsUnauthorized checks if an error returned by OpenStack service calls is caused by HTTP 403 status code.
func IsUnauthorized(err error) bool {
	if err == nil {
		return false
	}

	if errors.As(err, &gophercloud.ErrDefault403{}) {
		return true
	}

	var e gophercloud.Err403er
	return errors.As(err, &e)
}
