// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package validation - validation is used to validate cloud specific ProviderSpec
package validation

import (
	"fmt"
	"strings"

	. "github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateRequest validates a request received by the OpenStack driver.
func ValidateRequest(providerConfig *openstack.MachineProviderConfig, secret *corev1.Secret) error {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateMachineProviderConfig(providerConfig)...)
	allErrs = append(allErrs, validateSecret(secret)...)
	allErrs = append(allErrs, validateUserData(secret)...)

	return allErrs.ToAggregate()
}

// validateMachineProviderConfig validates a MachineProviderConfig object.
func validateMachineProviderConfig(providerConfig *openstack.MachineProviderConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	fldPath := field.NewPath("spec")

	if "" == providerConfig.Spec.ImageID {
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
		allErrs = append(allErrs, field.Required(fldPath.Child("availabilityZone"), "AvailabilityZone name is required"))
	}
	if "" == providerConfig.Spec.KeyName {
		allErrs = append(allErrs, field.Required(fldPath.Child("keyName"), "KeyName is required"))
	}
	if "" != providerConfig.Spec.NetworkID && len(providerConfig.Spec.Networks) > 0 {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("networks"), "\"networks\" list should not be specified along with \"providerConfig.Spec.NetworkID\""))
	}
	if "" == providerConfig.Spec.NetworkID && len(providerConfig.Spec.Networks) == 0 {
		allErrs = append(allErrs, field.Forbidden(fldPath.Child("networkID"), "both \"networks\" and \"networkID\" should not be empty"))
	}
	if "" == providerConfig.Spec.PodNetworkCidr {
		allErrs = append(allErrs, field.Required(fldPath.Child("podNetworkCidr"), "PodNetworkCidr is required"))
	}
	if providerConfig.Spec.RootDiskSize < 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("rootDiskSize"), "RootDiskSize can not be negative"))
	}

	allErrs = append(allErrs, validateNetworks(providerConfig.Spec.Networks, providerConfig.Spec.PodNetworkCidr, field.NewPath("spec.networks"))...)
	allErrs = append(allErrs, validateClassSpecTags(providerConfig.Spec.Tags, field.NewPath("spec.tags"))...)

	return allErrs
}

func validateNetworks(networks []openstack.OpenStackNetwork, podNetworkCidr string, fldPath *field.Path) field.ErrorList {
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

func validateClassSpecTags(tags map[string]string, fldPath *field.Path) field.ErrorList {
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
		allErrs = append(allErrs, field.Required(fldPath, fmt.Sprintf("Tag required of the form %s****", ServerTagClusterPrefix)))
	}
	if nodeRole == "" {
		allErrs = append(allErrs, field.Required(fldPath, fmt.Sprintf("Tag required of the form %s****", ServerTagRolePrefix)))
	}

	return allErrs
}

// validateSecret validates that the secret contain data to authenticate with an Openstack provider.
func validateSecret(secret *corev1.Secret) field.ErrorList {
	var (
		ok, ok2 bool
		allErrs = field.ErrorList{}
	)

	root := field.NewPath("data")
	data := secret.Data
	if b, ok := data[OpenStackAuthURL]; !ok || isEmptyStringByteSlice(b) {
		allErrs = append(allErrs, field.Required(root.Key(OpenStackAuthURL), fmt.Sprintf("%s is required", OpenStackAuthURL)))
	}
	if _, ok := data[OpenStackApplicationCredentialID]; !ok {
		if b, ok := data[OpenStackUsername]; !ok || isEmptyStringByteSlice(b) {
			allErrs = append(allErrs, field.Required(root.Key(OpenStackUsername), fmt.Sprintf("%s is required", OpenStackUsername)))
		}
		if b, ok := data[OpenStackPassword]; !ok || isEmptyStringByteSlice(b) {
			allErrs = append(allErrs, field.Required(root.Key(OpenStackPassword), fmt.Sprintf("%s is required", OpenStackPassword)))
		}
	} else {
		if b, ok := data[OpenStackApplicationCredentialID]; !ok || isEmptyStringByteSlice(b) {
			allErrs = append(allErrs, field.Required(root.Key(OpenStackApplicationCredentialID), fmt.Sprintf("%s is required", OpenStackApplicationCredentialID)))
		}
		if b, ok := data[OpenStackApplicationCredentialSecret]; !ok || isEmptyStringByteSlice(b) {
			allErrs = append(allErrs, field.Required(root.Key(OpenStackApplicationCredentialSecret), fmt.Sprintf("%s is required", OpenStackApplicationCredentialSecret)))
		}
	}

	domainName, ok := data[OpenStackDomainName]
	domainID, ok2 := data[OpenStackDomainID]
	if (!ok || isEmptyStringByteSlice(domainName)) && (!ok2 || isEmptyStringByteSlice(domainID)) {
		allErrs = append(allErrs, field.Required(root.Key(OpenStackDomainName), fmt.Sprintf("one of the following keys is required [%s|%s]", OpenStackDomainName, OpenStackDomainID)))
	}

	tenantName, ok := data[OpenStackTenantName]
	tenantID, ok2 := data[OpenStackTenantID]
	if (!ok || isEmptyStringByteSlice(tenantName)) && (!ok2 || isEmptyStringByteSlice(tenantID)) {
		allErrs = append(allErrs, field.Required(root.Key(OpenStackTenantName), fmt.Sprintf("one of the following keys is required [%s|%s]", OpenStackTenantName, OpenStackTenantID)))
	}

	var clientCert, clientKey []byte
	if clientCert, ok = data[OpenStackClientCert]; !ok {
		clientCert = nil
	}
	if clientKey, ok = data[OpenStackClientKey]; !ok {
		clientKey = nil
	}

	if len(clientCert) != 0 && len(clientKey) == 0 {
		allErrs = append(allErrs, field.Required(root.Key(OpenStackClientKey), fmt.Sprintf("%s is required, if %s is present", OpenStackClientKey, OpenStackClientCert)))
	}

	if insecureStr, ok := data[OpenStackInsecure]; ok {
		switch string(insecureStr) {
		case "true":
		case "false":
		default:
			allErrs = append(allErrs, field.Invalid(root.Key(OpenStackInsecure), string(insecureStr), "value does not match expected boolean value [\"true\"|\"false\"]"))
		}
	}

	return allErrs
}

// validateUserData validates that a secret contains user data.
func validateUserData(secret *corev1.Secret) field.ErrorList {
	allErrs := field.ErrorList{}
	root := field.NewPath("data")
	if b, ok := secret.Data[UserData]; !ok || isEmptyStringByteSlice(b) {
		allErrs = append(allErrs, field.Required(root.Key(UserData), fmt.Sprintf("%s is required", UserData)))
	}

	return allErrs
}

func isEmptyStringByteSlice(b []byte) bool {
	if len(b) == 0 {
		return true
	}

	if len(strings.TrimSpace(string(b))) == 0 {
		return true
	}

	return false
}
