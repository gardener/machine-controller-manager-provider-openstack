// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
)

// encodeProviderID encodes the ID of a server.
func encodeProviderID(region string, machineID string) string {
	return fmt.Sprintf("openstack:///%s/%s", region, machineID)
}

// decodeProviderID decodes a provider-encoded ID into the ID of the server.
func decodeProviderID(id string) string {
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

func findMandatoryTags(tags map[string]string) (string, string, bool) {
	var (
		searchClusterName = ""
		searchNodeRole    = ""
	)

	for key := range tags {
		if strings.Contains(key, cloudprovider.ServerTagClusterPrefix) {
			searchClusterName = key
		} else if strings.Contains(key, cloudprovider.ServerTagRolePrefix) {
			searchNodeRole = key
		}
	}

	if searchNodeRole == "" || searchClusterName == "" {
		return searchClusterName, searchNodeRole, false
	}
	return searchClusterName, searchNodeRole, true
}
