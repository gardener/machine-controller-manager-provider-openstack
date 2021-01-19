// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MachineProviderConfig struct {
	metav1.TypeMeta

	Spec MachineProviderConfigSpec
}

type MachineProviderConfigSpec struct {
	ImageID          string
	ImageName        string
	Region           string
	AvailabilityZone string
	FlavorName       string
	KeyName          string
	SecurityGroups   []string
	Tags             map[string]string
	NetworkID        string
	SubnetID         *string
	PodNetworkCidr   string
	RootDiskSize     int
	UseConfigDrive   *bool
	ServerGroupID    *string
	Networks         []OpenStackNetwork
}

type OpenStackNetwork struct {
	Id         string
	Name       string
	PodNetwork bool
}
