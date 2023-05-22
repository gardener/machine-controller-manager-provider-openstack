// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package provider contains the cloud provider specific implementations to manage machines
package driver

import (
	"context"
	"fmt"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog/v2"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/validation"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/client"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/driver/executor"
)

const (
	cinderDriverName = "cinder.csi.openstack.org"
)

// NOTE
//
// The basic working of the controller will work with just implementing the CreateMachine() & DeleteMachine() methods.
// You can first implement these two methods and check the working of the controller.
// Leaving the other methods to NOT_IMPLEMENTED error status.
// Once this works you can implement the rest of the methods.
//
// Also make sure each method return appropriate errors mentioned in `https://github.com/gardener/machine-controller-manager/blob/master/docs/development/machine_error_codes.md`

// CreateMachine handles a machine creation request
//
// OPTIONAL IMPLEMENTATION LOGIC
// It is optionally expected by the safety controller to use an identification mechanism to map the VM Created by a providerSpec.
// These could be done using tag(s)/resource-groups etc.
// This logic is used by safety controller to delete orphan VMs which are not backed by any machine CRD
func (p *OpenstackDriver) CreateMachine(ctx context.Context, req *driver.CreateMachineRequest) (*driver.CreateMachineResponse, error) {
	klog.V(2).Infof("CreateMachine request has been received for %q", req.Machine.Name)
	defer klog.V(2).Infof("CreateMachine request has been processed for %q", req.Machine.Name)

	// Check if incoming provider in the MachineClass is a provider we support
	if req.MachineClass.Provider != openstackProvider {
		err := fmt.Errorf("Requested for Provider '%s', we only support '%s'", req.MachineClass.Provider, openstackProvider)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	providerConfig, err := p.decodeProviderSpec(req.MachineClass.ProviderSpec)
	if err != nil {
		klog.Errorf("decoding provider spec for machine class %q failed with: %v", req.MachineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := validation.ValidateRequest(providerConfig, req.Secret); err != nil {
		klog.Errorf("validating request for machine %q failed with: %v", req.Machine.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	factory, err := client.NewFactoryFromSecret(req.Secret)
	if err != nil {
		klog.Errorf("failed to construct OpenStack client: %v", err)
		return nil, status.Error(mapErrorToCode(err), fmt.Sprintf("failed to construct OpenStack client: %v", err))
	}

	ex, err := executor.NewExecutor(factory, providerConfig)
	if err != nil {
		klog.Errorf("failed to construct context for the request: %v", err)
		return nil, status.Error(mapErrorToCode(err), fmt.Sprintf("failed to construct context for the request: %v", err))
	}

	providerID, err := ex.CreateMachine(ctx, req.Machine.Name, req.Secret.Data[cloudprovider.UserData])
	if err != nil {
		klog.Errorf("machine creation for machine %q failed with: %v", req.Machine.Name, err)
		return nil, status.Error(mapErrorToCode(err), err.Error())
	}

	return &driver.CreateMachineResponse{
		ProviderID: providerID,
		NodeName:   req.Machine.Name,
	}, nil
}

// DeleteMachine handles a machine deletion request
func (p *OpenstackDriver) DeleteMachine(ctx context.Context, req *driver.DeleteMachineRequest) (*driver.DeleteMachineResponse, error) {
	// Log messages to track delete request
	klog.V(2).Infof("DeleteMachine request has been received for %q", req.Machine.Name)
	defer klog.V(2).Infof("DeleteMachine request has been processed for %q", req.Machine.Name)

	// Check if incoming provider in the MachineClass is a provider we support
	if req.MachineClass.Provider != openstackProvider {
		err := fmt.Errorf("Requested for Provider '%s', we only support '%s'", req.MachineClass.Provider, openstackProvider)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	providerConfig, err := p.decodeProviderSpec(req.MachineClass.ProviderSpec)
	if err != nil {
		klog.V(2).Infof("decoding provider spec for machine class %q failed with: %v", req.MachineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := validation.ValidateRequest(providerConfig, req.Secret); err != nil {
		klog.V(2).Infof("validating request for machine class %q failed with: %v", req.MachineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	factory, err := client.NewFactoryFromSecret(req.Secret)
	if err != nil {
		klog.Errorf("failed to construct OpenStack client: %v", err)
		return nil, status.Error(mapErrorToCode(err), fmt.Sprintf("failed to construct OpenStack client: %v", err))
	}

	ex, err := executor.NewExecutor(factory, providerConfig)
	if err != nil {
		klog.Errorf("failed to construct context for the request: %v", err)
		return nil, status.Error(mapErrorToCode(err), fmt.Sprintf("failed to construct context for the request: %v", err))
	}

	err = ex.DeleteMachine(ctx, req.Machine.Name, req.Machine.Spec.ProviderID)
	if err != nil {
		return nil, status.Error(mapErrorToCode(err), err.Error())
	}
	return &driver.DeleteMachineResponse{}, nil
}

// GetMachineStatus handles a machine get status request
func (p *OpenstackDriver) GetMachineStatus(ctx context.Context, req *driver.GetMachineStatusRequest) (*driver.GetMachineStatusResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("GetMachineStatus request has been received for %q", req.Machine.Name)
	defer klog.V(2).Infof("GetMachineStatus is not implemented")

	return nil, status.Error(codes.Unimplemented, "method not implemented")
}

// ListMachines lists all the machines possibly created by a providerSpec
// Identifying machines created by a given providerSpec depends on the OPTIONAL IMPLEMENTATION LOGIC
// you have used to identify machines created by a providerSpec. It could be tags/resource-groups etc
func (p *OpenstackDriver) ListMachines(ctx context.Context, req *driver.ListMachinesRequest) (*driver.ListMachinesResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("ListMachines request has been received for %q", req.MachineClass.Name)
	defer klog.V(2).Infof("ListMachines request has been processed for %q", req.MachineClass.Name)

	// Check if incoming provider in the MachineClass is a provider we support
	if req.MachineClass.Provider != openstackProvider {
		err := fmt.Errorf("Requested for Provider '%s', we only support '%s'", req.MachineClass.Provider, openstackProvider)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	providerConfig, err := p.decodeProviderSpec(req.MachineClass.ProviderSpec)
	if err != nil {
		klog.Errorf("decoding provider spec for machine class %q failed with: %v", req.MachineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := validation.ValidateRequest(providerConfig, req.Secret); err != nil {
		klog.Errorf("validating request for machine class %q failed with: %v", req.MachineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	factory, err := client.NewFactoryFromSecret(req.Secret)
	if err != nil {
		klog.Errorf("failed to construct OpenStack client: %v", err)
		return nil, status.Error(mapErrorToCode(err), fmt.Sprintf("failed to construct OpenStack client: %v", err))
	}

	ex, err := executor.NewExecutor(factory, providerConfig)
	if err != nil {
		klog.Errorf("failed to construct context for the request: %v", err)
		return nil, status.Error(mapErrorToCode(err), fmt.Sprintf("failed to construct context for the request: %v", err))
	}

	machines, err := ex.ListMachines(ctx)
	if err != nil {
		return nil, status.Error(mapErrorToCode(err), fmt.Sprintf("listing machines for machine class %q failed with: %v", req.MachineClass.Name, err))
	}
	if len(machines) == 0 {
		klog.V(3).Infof("no machines found for machine class: %q", req.MachineClass.Name)
	}

	return &driver.ListMachinesResponse{
		MachineList: machines,
	}, nil
}

// GetVolumeIDs returns a list of Volume IDs for all PV Specs for whom an provider volume was found
func (p *OpenstackDriver) GetVolumeIDs(_ context.Context, req *driver.GetVolumeIDsRequest) (*driver.GetVolumeIDsResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("GetVolumeIDs request has been received for %q", req.PVSpecs)
	defer klog.V(2).Infof("GetVolumeIDs request has been processed for %q", req.PVSpecs)

	names := make([]string, 0)
	for _, spec := range req.PVSpecs {
		if spec.Cinder != nil {
			name := spec.Cinder.VolumeID
			names = append(names, name)
		} else if spec.CSI != nil && spec.CSI.Driver == cinderDriverName && spec.CSI.VolumeHandle != "" {
			name := spec.CSI.VolumeHandle
			names = append(names, name)
		}
	}
	return &driver.GetVolumeIDsResponse{VolumeIDs: names}, nil
}
