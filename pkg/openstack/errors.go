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

