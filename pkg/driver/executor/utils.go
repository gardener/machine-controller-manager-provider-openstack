// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"
)

// EncodeProviderID encodes the ID of a server.
func EncodeProviderID(region string, machineID string) string {
	return fmt.Sprintf("openstack:///%s/%s", region, machineID)
}

// DecodeProviderID decodes a provider-encoded ID into the ID of the server.
func DecodeProviderID(id string) string {
	splitProviderID := strings.Split(id, "/")
	return splitProviderID[len(splitProviderID)-1]
}

func strSliceContains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func isEmptyString(ptr *string) bool {
	if ptr == nil {
		return true
	}

	if len(strings.TrimSpace(*ptr)) == 0 {
		return true
	}

	return false
}
