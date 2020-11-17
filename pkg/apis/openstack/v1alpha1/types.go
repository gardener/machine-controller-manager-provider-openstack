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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MachineClassProviderConfig struct{
	// +optional
	metav1.TypeMeta `json:",inline"`

	// +optional
	Spec MachineClassSpec `json:"spec,omitempty"`
}

type MachineClassSpec struct {
	ImageID          string                  `json:"imageID"`
	ImageName        string                  `json:"imageName"`
	Region           string                  `json:"region"`
	AvailabilityZone string                  `json:"availabilityZone"`
	FlavorName       string                  `json:"flavorName"`
	KeyName          string                  `json:"keyName"`
	SecurityGroups   []string                `json:"securityGroups"`
	Tags             map[string]string       `json:"tags,omitempty"`
	NetworkID        string                  `json:"networkID"`
	SubnetID         *string                 `json:"subnetID,omitempty"`
	PodNetworkCidr   string                  `json:"podNetworkCidr"`
	RootDiskSize     int                     `json:"rootDiskSize,omitempty"` // in GB
	UseConfigDrive   *bool                   `json:"useConfigDrive,omitempty"`
	ServerGroupID    *string                 `json:"serverGroupID,omitempty"`
}

