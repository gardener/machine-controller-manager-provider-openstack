// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
)

const (
	// NoValidHost is a part of the error message returned when there is no valid host in the zone to deploy a VM.
	// Matches:
	//
	//	"No valid host was found."
	//	"No valid host was found. There are not enough hosts available."
	NoValidHost = "No valid host was found"

	// FlavorNotFound is part of the error message returned when a flavor cannot be resolved.
	// This string-matching fallback is needed because FlavorIDFromName (from executor.go deployServer)
	// only wraps the typed ErrFlavorNotFound when the underlying gophercloud error is gophercloud.ErrResourceNotFound.
	// If gophercloud returns any other error type, the typed-error path in mapErrorToCode will not match, and
	// we fall back to inspecting the error message here so the failure is still classified as codes.ResourceExhausted.
	FlavorNotFound = "error resolving flavor"
)

var (
	// ErrNotFound is returned when the requested resource could not be found.
	ErrNotFound = fmt.Errorf("resource not found")

	// ErrMultipleFound is returned when a resource that is expected to be unique has multiple matches.
	// For example, reverse lookups from names to IDs may yield multiple matches because names are not unique in most
	// OpenStack resources. In case this case, where a unique ID could not be determined an ErrMultipleFound is returned.
	ErrMultipleFound = fmt.Errorf("multiple resources found")
)

// ErrFlavorNotFound is returned when there is no flavor can be matched with the specified flavor name.
// It can happen when certain flavor is not available in specified region and needs to be treated as ResourceExhausted
// to allow fallback to other flavors.
type ErrFlavorNotFound struct {
	Flavor string
}

func (e ErrFlavorNotFound) Error() string {
	return fmt.Sprintf("Unable to find flavor with name %s", e.Flavor)
}
