//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Code generated by conversion-gen. DO NOT EDIT.

package v1alpha1

import (
	unsafe "unsafe"

	openstack "github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*MachineProviderConfig)(nil), (*openstack.MachineProviderConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_MachineProviderConfig_To_openstack_MachineProviderConfig(a.(*MachineProviderConfig), b.(*openstack.MachineProviderConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*openstack.MachineProviderConfig)(nil), (*MachineProviderConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_openstack_MachineProviderConfig_To_v1alpha1_MachineProviderConfig(a.(*openstack.MachineProviderConfig), b.(*MachineProviderConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MachineProviderConfigSpec)(nil), (*openstack.MachineProviderConfigSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_MachineProviderConfigSpec_To_openstack_MachineProviderConfigSpec(a.(*MachineProviderConfigSpec), b.(*openstack.MachineProviderConfigSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*openstack.MachineProviderConfigSpec)(nil), (*MachineProviderConfigSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_openstack_MachineProviderConfigSpec_To_v1alpha1_MachineProviderConfigSpec(a.(*openstack.MachineProviderConfigSpec), b.(*MachineProviderConfigSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*OpenStackNetwork)(nil), (*openstack.OpenStackNetwork)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_OpenStackNetwork_To_openstack_OpenStackNetwork(a.(*OpenStackNetwork), b.(*openstack.OpenStackNetwork), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*openstack.OpenStackNetwork)(nil), (*OpenStackNetwork)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_openstack_OpenStackNetwork_To_v1alpha1_OpenStackNetwork(a.(*openstack.OpenStackNetwork), b.(*OpenStackNetwork), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha1_MachineProviderConfig_To_openstack_MachineProviderConfig(in *MachineProviderConfig, out *openstack.MachineProviderConfig, s conversion.Scope) error {
	if err := Convert_v1alpha1_MachineProviderConfigSpec_To_openstack_MachineProviderConfigSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha1_MachineProviderConfig_To_openstack_MachineProviderConfig is an autogenerated conversion function.
func Convert_v1alpha1_MachineProviderConfig_To_openstack_MachineProviderConfig(in *MachineProviderConfig, out *openstack.MachineProviderConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_MachineProviderConfig_To_openstack_MachineProviderConfig(in, out, s)
}

func autoConvert_openstack_MachineProviderConfig_To_v1alpha1_MachineProviderConfig(in *openstack.MachineProviderConfig, out *MachineProviderConfig, s conversion.Scope) error {
	if err := Convert_openstack_MachineProviderConfigSpec_To_v1alpha1_MachineProviderConfigSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	return nil
}

// Convert_openstack_MachineProviderConfig_To_v1alpha1_MachineProviderConfig is an autogenerated conversion function.
func Convert_openstack_MachineProviderConfig_To_v1alpha1_MachineProviderConfig(in *openstack.MachineProviderConfig, out *MachineProviderConfig, s conversion.Scope) error {
	return autoConvert_openstack_MachineProviderConfig_To_v1alpha1_MachineProviderConfig(in, out, s)
}

func autoConvert_v1alpha1_MachineProviderConfigSpec_To_openstack_MachineProviderConfigSpec(in *MachineProviderConfigSpec, out *openstack.MachineProviderConfigSpec, s conversion.Scope) error {
	out.ImageID = in.ImageID
	out.ImageName = in.ImageName
	out.Region = in.Region
	out.AvailabilityZone = in.AvailabilityZone
	out.FlavorName = in.FlavorName
	out.KeyName = in.KeyName
	out.SecurityGroups = *(*[]string)(unsafe.Pointer(&in.SecurityGroups))
	out.Tags = *(*map[string]string)(unsafe.Pointer(&in.Tags))
	out.NetworkID = in.NetworkID
	out.SubnetID = (*string)(unsafe.Pointer(in.SubnetID))
	out.PodNetworkCidr = in.PodNetworkCidr
	out.RootDiskSize = in.RootDiskSize
	out.RootDiskType = (*string)(unsafe.Pointer(in.RootDiskType))
	out.UseConfigDrive = (*bool)(unsafe.Pointer(in.UseConfigDrive))
	out.ServerGroupID = (*string)(unsafe.Pointer(in.ServerGroupID))
	out.Networks = *(*[]openstack.OpenStackNetwork)(unsafe.Pointer(&in.Networks))
	return nil
}

// Convert_v1alpha1_MachineProviderConfigSpec_To_openstack_MachineProviderConfigSpec is an autogenerated conversion function.
func Convert_v1alpha1_MachineProviderConfigSpec_To_openstack_MachineProviderConfigSpec(in *MachineProviderConfigSpec, out *openstack.MachineProviderConfigSpec, s conversion.Scope) error {
	return autoConvert_v1alpha1_MachineProviderConfigSpec_To_openstack_MachineProviderConfigSpec(in, out, s)
}

func autoConvert_openstack_MachineProviderConfigSpec_To_v1alpha1_MachineProviderConfigSpec(in *openstack.MachineProviderConfigSpec, out *MachineProviderConfigSpec, s conversion.Scope) error {
	out.ImageID = in.ImageID
	out.ImageName = in.ImageName
	out.Region = in.Region
	out.AvailabilityZone = in.AvailabilityZone
	out.FlavorName = in.FlavorName
	out.KeyName = in.KeyName
	out.SecurityGroups = *(*[]string)(unsafe.Pointer(&in.SecurityGroups))
	out.Tags = *(*map[string]string)(unsafe.Pointer(&in.Tags))
	out.NetworkID = in.NetworkID
	out.SubnetID = (*string)(unsafe.Pointer(in.SubnetID))
	out.PodNetworkCidr = in.PodNetworkCidr
	out.RootDiskSize = in.RootDiskSize
	out.RootDiskType = (*string)(unsafe.Pointer(in.RootDiskType))
	out.UseConfigDrive = (*bool)(unsafe.Pointer(in.UseConfigDrive))
	out.ServerGroupID = (*string)(unsafe.Pointer(in.ServerGroupID))
	out.Networks = *(*[]OpenStackNetwork)(unsafe.Pointer(&in.Networks))
	return nil
}

// Convert_openstack_MachineProviderConfigSpec_To_v1alpha1_MachineProviderConfigSpec is an autogenerated conversion function.
func Convert_openstack_MachineProviderConfigSpec_To_v1alpha1_MachineProviderConfigSpec(in *openstack.MachineProviderConfigSpec, out *MachineProviderConfigSpec, s conversion.Scope) error {
	return autoConvert_openstack_MachineProviderConfigSpec_To_v1alpha1_MachineProviderConfigSpec(in, out, s)
}

func autoConvert_v1alpha1_OpenStackNetwork_To_openstack_OpenStackNetwork(in *OpenStackNetwork, out *openstack.OpenStackNetwork, s conversion.Scope) error {
	out.Id = in.Id
	out.Name = in.Name
	out.PodNetwork = in.PodNetwork
	return nil
}

// Convert_v1alpha1_OpenStackNetwork_To_openstack_OpenStackNetwork is an autogenerated conversion function.
func Convert_v1alpha1_OpenStackNetwork_To_openstack_OpenStackNetwork(in *OpenStackNetwork, out *openstack.OpenStackNetwork, s conversion.Scope) error {
	return autoConvert_v1alpha1_OpenStackNetwork_To_openstack_OpenStackNetwork(in, out, s)
}

func autoConvert_openstack_OpenStackNetwork_To_v1alpha1_OpenStackNetwork(in *openstack.OpenStackNetwork, out *OpenStackNetwork, s conversion.Scope) error {
	out.Id = in.Id
	out.Name = in.Name
	out.PodNetwork = in.PodNetwork
	return nil
}

// Convert_openstack_OpenStackNetwork_To_v1alpha1_OpenStackNetwork is an autogenerated conversion function.
func Convert_openstack_OpenStackNetwork_To_v1alpha1_OpenStackNetwork(in *openstack.OpenStackNetwork, out *OpenStackNetwork, s conversion.Scope) error {
	return autoConvert_openstack_OpenStackNetwork_To_v1alpha1_OpenStackNetwork(in, out, s)
}
