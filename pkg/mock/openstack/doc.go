// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

//go:generate mockgen -copyright_file=../../../hack/LICENSE_HEADER.txt -destination=./mocks.go -package=openstack github.com/gardener/machine-controller-manager-provider-openstack/pkg/client Compute,Network,Storage
package openstack
