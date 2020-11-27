/*
Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package validation - validation is used to validate cloud specific ProviderSpec
package validation

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"
	. "github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
)

// ValidateProviderSpecNSecret validates provider spec and secret to check if all fields are present and valid
func ValidateMachineProviderConfig(providerConfig *openstack.MachineProviderConfig) error {
	allErrs := field.ErrorList{}

	fldPath := field.NewPath("spec")

	if "" == providerConfig.Spec.ImageID{
		if "" == providerConfig.Spec.ImageName {
			allErrs = append(allErrs, field.Required(fldPath.Child("imageName"), "ImageName is required if no ImageID is given"))
		}
	}

	if "" == providerConfig.Spec.Region {
		allErrs = append(allErrs, field.Required(fldPath.Child("region"), "Region is required"))
	}
	if "" == providerConfig.Spec.FlavorName {
		allErrs = append(allErrs, field.Required(fldPath.Child("flavorName"), "Flavor is required"))
	}
	if "" == providerConfig.Spec.AvailabilityZone {
		allErrs = append(allErrs, field.Required(fldPath.Child("availabilityZone"), "AvailabilityZone Name is required"))
	}
	if "" == providerConfig.Spec.KeyName {
		allErrs = append(allErrs, field.Required(fldPath.Child("keyName"), "KeyName is required"))
	}
	if "" != providerConfig.Spec.NetworkID && len(providerConfig.Spec.Networks) > 0 {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("networks"), "\"networks\" list should not be providerConfig.Specified along with \"providerConfig.Spec.NetworkID\""))
	}
	if "" == providerConfig.Spec.NetworkID && len(providerConfig.Spec.Networks) == 0 {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("networks"), "both \"networks\" and \"networkID\" should not be empty"))
	}
	if "" == providerConfig.Spec.PodNetworkCidr {
		allErrs = append(allErrs, field.Required(fldPath.Child("podNetworkCidr"), "PodNetworkCidr is required"))
	}
	if providerConfig.Spec.RootDiskSize < 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("rootDiskSize"), "RootDiskSize can not be negative"))
	}

	allErrs = append(allErrs, validateOsNetworks(providerConfig.Spec.Networks, providerConfig.Spec.PodNetworkCidr, field.NewPath("spec.networks"))...)
	allErrs = append(allErrs, validateOSClassSpecTags(providerConfig.Spec.Tags, field.NewPath("spec.tags"))...)

	return allErrs.ToAggregate()
}

func validateOsNetworks(networks []openstack.OpenStackNetwork, podNetworkCidr string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for index, network := range networks {
		fldPath := fldPath.Index(index)
		if "" == network.Id && "" == network.Name {
			allErrs = append(allErrs, field.Required(fldPath, "at least one of network \"id\" or \"name\" is required"))
		}
		if "" != network.Id && "" != network.Name {
			allErrs = append(allErrs, field.Forbidden(fldPath, "simultaneous use of network \"id\" and \"name\" is forbidden"))
		}
		if "" == podNetworkCidr && network.PodNetwork {
			allErrs = append(allErrs, field.Required(fldPath.Child("podNetwork"), "\"podNetwork\" switch should not be used in absence of \"spec.podNetworkCidr\""))
		}
	}

	return allErrs
}

func validateOSClassSpecTags(tags map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	clusterName := ""
	nodeRole := ""

	for key := range tags {
		if strings.Contains(key, ServerTagClusterPrefix) {
			clusterName = key
		} else if strings.Contains(key, ServerTagRolePrefix) {
			nodeRole = key
		}
	}

	if clusterName == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child(ServerTagClusterPrefix), fmt.Sprintf("Tag required of the form %s-****", ServerTagClusterPrefix)))
	}
	if nodeRole == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child(ServerTagRolePrefix), fmt.Sprintf("Tag required of the form %s-****", ServerTagRolePrefix)))
	}

	return allErrs
}


func ValidateSecret(secret *corev1.Secret) error {
	var (
		ok, ok2 bool
	)
	data := secret.Data
	if b, ok := data[OpenStackAuthURL]; !ok || missingString(b) {
		return fmt.Errorf("missing %s in secret", OpenStackAuthURL)
	}
	if b, ok := data[OpenStackUsername]; !ok || missingString(b){
		return fmt.Errorf("missing %s in secret", OpenStackUsername)
	}
	if b, ok := data[OpenStackPassword]; !ok || missingString(b){
		return fmt.Errorf("missing %s in secret", OpenStackPassword)
	}

	domainName, ok := data[OpenStackDomainName]
	domainID, ok2 := data[OpenStackDomainID]
	if (!ok || missingString(domainName)) && (!ok2 || missingString(domainID)) {
		return fmt.Errorf("missing %s or %s in secret", OpenStackDomainName, OpenStackDomainID)
	}

	tenantName, ok := data[OpenStackTenantName]
	tenantID, ok2 := data[OpenStackTenantID]
	if (!ok || missingString(tenantName)) && (!ok2 || missingString(tenantID)) {
		return fmt.Errorf("missing %s or %s in secret", OpenStackTenantName, OpenStackTenantID)
	}

	var clientCert, clientKey []byte
	if clientCert, ok = data[OpenStackClientCert]; !ok {
		clientCert = nil
	}
	if clientKey, ok = data[OpenStackClientKey]; !ok {
		clientKey = nil
	}

	if len(clientCert) != 0 && len(clientKey) == 0 {
		return fmt.Errorf("missing %s in secret", OpenStackClientKey)
	}

	if insecureStr, ok := data[OpenStackInsecure]; ok {
		switch string(insecureStr) {
		case "true":
		case "false":
		default:
			return fmt.Errorf("invalid value for boolean field %s: %s ", OpenStackInsecure, string(insecureStr))
		}
	}

	if b, ok := data[UserData]; !ok || missingString(b){
		return fmt.Errorf("missing %s in secret", UserData)
	}

	return nil
}

func missingString(b []byte) bool {
	if len(b) == 0{
		return true
	}

	if len(strings.TrimSpace(string(b))) == 0 {
		return true
	}

	return false
}