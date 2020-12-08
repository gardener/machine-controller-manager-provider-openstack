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

package driver

import (
	"fmt"
	"strings"

	mcmv1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack/v1alpha1"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/validation"
)

const (
	openstackProvider = "openstack"
)

func (p *OpenstackDriver) decodeProviderSpec(raw runtime.RawExtension) (*openstack.MachineProviderConfig, error) {
	json, err := raw.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to decode provider spec: %v", err)
	}

	cfg := &openstack.MachineProviderConfig{}
	_, _, err = p.decoder.Decode(json, nil, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode provider spec: %v", err)
	}

	return cfg, nil
}

func (p *OpenstackDriver) validateRequest(config *openstack.MachineProviderConfig, secret *corev1.Secret) error {
	if err := validation.ValidateMachineProviderConfig(config); err != nil {
		return err
	}

	return validation.ValidateSecret(secret)
}

func errorWrap(code codes.Code, err error, message string, args ...interface{}) error {
	args = append(args, err)
	return status.Error(code, fmt.Sprintf(message+": %v", args...))
}

func strSliceContains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func isEmptyStringPtr(ptr *string) bool {
	if ptr == nil {
		return true
	}

	if len(strings.TrimSpace(*ptr)) == 0 {
		return true
	}

	return false
}

func isEmptyString(str string) bool {
	return len(strings.TrimSpace(str)) == 0
}

func encodeProviderID(region string, machineID string) string {
	return fmt.Sprintf("openstack:///%s/%s", region, machineID)
}

func decodeProviderID(id string) string {
	splitProviderID := strings.Split(id, "/")
	return splitProviderID[len(splitProviderID)-1]
}

func migrateMachineClass(os *mcmv1alpha1.OpenStackMachineClass, machineClass *mcmv1alpha1.MachineClass) error {
	migratedNetworks := []v1alpha1.OpenStackNetwork{}
	for _, nw := range os.Spec.Networks {
		migratedNetworks = append(migratedNetworks, v1alpha1.OpenStackNetwork{
			Id:         nw.Id,
			Name:       nw.Name,
			PodNetwork: nw.PodNetwork,
		})
	}

	cfg := &v1alpha1.MachineProviderConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineProviderConfig",
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		Spec: v1alpha1.MachineProviderConfigSpec{
			ImageID:          os.Spec.ImageID,
			ImageName:        os.Spec.ImageName,
			Region:           os.Spec.Region,
			AvailabilityZone: os.Spec.AvailabilityZone,
			FlavorName:       os.Spec.FlavorName,
			KeyName:          os.Spec.KeyName,
			SecurityGroups:   os.Spec.SecurityGroups,
			Tags:             os.Spec.Tags,
			NetworkID:        os.Spec.NetworkID,
			SubnetID:         os.Spec.SubnetID,
			PodNetworkCidr:   os.Spec.PodNetworkCidr,
			RootDiskSize:     os.Spec.RootDiskSize,
			UseConfigDrive:   os.Spec.UseConfigDrive,
			ServerGroupID:    os.Spec.ServerGroupID,
			Networks:         migratedNetworks,
		},
	}

	machineClass.Name = os.Name
	machineClass.Labels = os.Labels
	machineClass.Annotations = os.Annotations
	//TODO(KA): finalizers necessary ?
	machineClass.Finalizers = os.Finalizers
	machineClass.ProviderSpec = runtime.RawExtension{
		Object: cfg,
	}
	machineClass.SecretRef = os.Spec.SecretRef
	machineClass.Provider = openstackProvider

	return nil
}
