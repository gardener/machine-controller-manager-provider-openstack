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

package openstack

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MachineClassProviderConfig struct {
	metav1.TypeMeta

	Spec MachineClassSpec
}

type MachineClassSpec struct {
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
}
