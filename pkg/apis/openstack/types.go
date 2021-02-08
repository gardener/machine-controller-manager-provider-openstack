// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// MachineProviderConfig contains OpenStack specific configuration for a machine.
type MachineProviderConfig struct {
	metav1.TypeMeta

	Spec MachineProviderConfigSpec
}

// MachineProviderConfigSpec is the specification for an OpenStack instance.
type MachineProviderConfigSpec struct {
	// ImageID is the ID of image used by the machine.
	ImageID string
	// ImageName is the name of the image used the machine. If ImageID is specified, it takes priority over ImageName.
	ImageName string
	// Region is the region the machine should belong to.
	Region string
	// AvailabilityZone is the availability zone the machine belongs.
	AvailabilityZone string
	// FlavorName is the flavor of the machine.
	FlavorName string
	// KeyName is the name of the key pair used for SSH access.
	KeyName string
	// SecurityGroups is a list of security groups the instance should belong to.
	SecurityGroups []string
	// Tags is a map of key-value pairs that annotate the instance. Tags are stored in the instance's Metadata field.
	Tags map[string]string
	// NetworkID is the ID of the network the instance should belong to.
	NetworkID string
	// SubnetID is the ID of the subnet the instance should belong to. If SubnetID is not specified
	SubnetID *string
	// PodNetworkCidr is the CIDR range for the pods assigned to this instance.
	PodNetworkCidr string
	// The size of the root disk used for the instance.
	RootDiskSize int
	// UseConfigDrive enables the use of configuration drives for the instance.
	UseConfigDrive *bool
	// ServerGroupID is the ID of the server group this instance should belong to.
	ServerGroupID *string
	// Networks is a list of networks the instance should belong to. Networks is mutually exclusive with the NetworkID option
	// and only one should be specified.
	Networks []OpenStackNetwork
}

// OpenStacknetwork describes an network this instance should belong to.
type OpenStackNetwork struct {
	// Id is the ID of a network the instance should belong to.
	Id string
	// Name is the name of a network the instance should belong to. If Id is specified, it takes priority over Name.
	Name string
	// PodNetwork specifies whether this network is part of the pod network.
	PodNetwork bool
}
