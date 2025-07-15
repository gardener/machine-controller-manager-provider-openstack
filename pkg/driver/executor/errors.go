// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
)

// NoValidHost is a part of the error message returned when there is no valid host in the zone to deploy a VM.
// Matches:
//
//	"No valid host was found."
//	"No valid host was found. There are not enough hosts available."
const NoValidHost = "No valid host was found"

var (
	// ErrNotFound is returned when the requested resource could not be found.
	ErrNotFound = fmt.Errorf("resource not found")

	// ErrMultipleFound is returned when a resource that is expected to be unique has multiple matches.
	// For example, reverse lookups from names to IDs may yield multiple matches because names are not unique in most
	// OpenStack resources. In case this case, where a unique ID could not be determined an ErrMultipleFound is returned.
	ErrMultipleFound = fmt.Errorf("multiple resources found")
)
