// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineProviderConfig contains OpenStack specific configuration for a machine.
type MachineProviderConfig struct {
	// +optional
	metav1.TypeMeta `json:",inline"`

	// +optional
	Spec MachineProviderConfigSpec `json:"spec,omitempty"`
}

// MachineProviderConfigSpec contains provider specific configuration for creating and managing machines.
type MachineProviderConfigSpec struct {
	// ImageID is the ID of image used by the machine.
	ImageID string `json:"imageID"`
	// ImageName is the name of the image used the machine. If ImageID is specified, it takes priority over ImageName.
	ImageName string `json:"imageName"`
	// Region is the region the machine should belong to.
	Region string `json:"region"`
	// AvailabilityZone is the availability zone the machine belongs.
	AvailabilityZone string `json:"availabilityZone"`
	// FlavorName is the flavor of the machine.
	FlavorName string `json:"flavorName"`
	// KeyName is the name of the key pair used for SSH access.
	KeyName string `json:"keyName"`
	// SecurityGroups is a list of security groups the instance should belong to.
	SecurityGroups []string `json:"securityGroups"`
	// Tags is a map of key-value pairs that annotate the instance. Tags are stored in the instance's Metadata field.
	Tags map[string]string `json:"tags,omitempty"`
	// NetworkID is the ID of the network the instance should belong to.
	NetworkID string `json:"networkID"`
	// SubnetID is the ID of the subnet the instance should belong to. If SubnetID is not specified
	// +optional
	SubnetID *string `json:"subnetID,omitempty"`
	// PodNetworkCidr is the CIDR range for the pods assigned to this instance.
	PodNetworkCidr string `json:"podNetworkCidr"`
	// The size of the root disk used for the instance.
	RootDiskSize int `json:"rootDiskSize,omitempty"` // in GB
	// The type of the root disk used for the instance.
	// +optional
	RootDiskType *string `json:"rootDiskType,omitempty"`
	// UseConfigDrive enables the use of configuration drives for the instance.
	UseConfigDrive *bool `json:"useConfigDrive,omitempty"`
	// ServerGroupID is the ID of the server group this instance should belong to.
	// +optional
	ServerGroupID *string `json:"serverGroupID,omitempty"`
	// Networks is a list of networks the instance should belong to. Networks is mutually exclusive with the NetworkID option
	// and only one should be specified.
	Networks []OpenStackNetwork `json:"networks,omitempty"`
}

// OpenStackNetwork describes a network this instance should belong to.
type OpenStackNetwork struct {
	// Id is the ID of a network the instance should belong to.
	Id string `json:"id,omitempty"` // takes priority before name
	// Name is the name of a network the instance should belong to. If Id is specified, it takes priority over Name.
	Name string `json:"name,omitempty"`
	// PodNetwork specifies whether this network is part of the pod network.
	PodNetwork bool `json:"podNetwork,omitempty"`
}
