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

	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/validation"
)

func (p *OpenstackDriver) decodeProviderSpec(raw runtime.RawExtension) (cfg *openstack.MachineProviderConfig, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to decode provider spec: %v", err)
		}
	}()

	json, err := raw.MarshalJSON()
	if err != nil {
		return nil, err
	}

	cfg = &openstack.MachineProviderConfig{}
	_, _, err = p.decoder.Decode(json, nil, cfg)
	if err != nil {
		return nil, err
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

