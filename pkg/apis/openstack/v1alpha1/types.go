// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MachineProviderConfig struct {
	// +optional
	metav1.TypeMeta `json:",inline"`

	// +optional
	Spec MachineProviderConfigSpec `json:"spec,omitempty"`
}

type MachineProviderConfigSpec struct {
	ImageID          string             `json:"imageID"`
	ImageName        string             `json:"imageName"`
	Region           string             `json:"region"`
	AvailabilityZone string             `json:"availabilityZone"`
	FlavorName       string             `json:"flavorName"`
	KeyName          string             `json:"keyName"`
	SecurityGroups   []string           `json:"securityGroups"`
	Tags             map[string]string  `json:"tags,omitempty"`
	NetworkID        string             `json:"networkID"`
	SubnetID         *string            `json:"subnetID,omitempty"`
	PodNetworkCidr   string             `json:"podNetworkCidr"`
	RootDiskSize     int                `json:"rootDiskSize,omitempty"` // in GB
	UseConfigDrive   *bool              `json:"useConfigDrive,omitempty"`
	ServerGroupID    *string            `json:"serverGroupID,omitempty"`
	Networks         []OpenStackNetwork `json:"networks,omitempty"`
}

type OpenStackNetwork struct {
	Id         string `json:"id,omitempty"` // takes priority before name
	Name       string `json:"name,omitempty"`
	PodNetwork bool   `json:"podNetwork,omitempty"`
}
