// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"github.com/gophercloud/gophercloud"
)

// IsNotFoundError checks if an error returned by OpenStack is caused by HTTP 404 status code.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if _, ok := err.(gophercloud.ErrDefault404); ok {
		return true
	}

	if _, ok := err.(gophercloud.Err404er); ok {
		return true
	}

	if _, ok := err.(gophercloud.ErrResourceNotFound); ok {
		return true
	}

	return false
}
