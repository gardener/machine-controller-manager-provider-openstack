// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// Package provider contains the cloud provider specific implementations to manage machines
package driver

import (
	"context"
	"errors"
	"fmt"

	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	corev1 "k8s.io/api/core/v1"
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
// The other methods should be implemented according to specs or return the NOT_IMPLEMENTED error status.
//
// Make sure each method returns the appropriate errors mentioned in
// `https://github.com/gardener/machine-controller-manager/blob/master/docs/development/machine_error_codes.md`

// setupExecutor handles the common validation and setup, returning a ready-to-use executor.
func (p *OpenstackDriver) setupExecutor(ctx context.Context, machineClass *v1alpha1.MachineClass, secret *corev1.Secret) (*executor.Executor, error) {
	// Check if incoming provider in the MachineClass is a provider we support
	if machineClass.Provider != openstackProvider {
		err := fmt.Errorf("requested for Provider '%s', we only support '%s'", machineClass.Provider, openstackProvider)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	providerConfig, err := p.decodeProviderSpec(machineClass.ProviderSpec)
	if err != nil {
		klog.Errorf("decoding provider spec for machine class %q failed with: %v", machineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := validation.ValidateRequest(providerConfig, secret); err != nil {
		klog.Errorf("validating request failed with: %v", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	factory, err := client.NewFactoryFromSecret(ctx, secret)
	if err != nil {
		klog.Errorf("failed to construct OpenStack client: %v", err)
		return nil, status.Error(mapErrorToCode(err), fmt.Sprintf("failed to construct OpenStack client: %v", err))
	}

	ex, err := executor.NewExecutor(factory, providerConfig)
	if err != nil {
		klog.Errorf("failed to construct context for the request: %v", err)
		return nil, status.Error(mapErrorToCode(err), fmt.Sprintf("failed to construct context for the request: %v", err))
	}

	return ex, nil
}

// CreateMachine handles a machine creation request
//
// OPTIONAL IMPLEMENTATION LOGIC
// It is optionally expected by the safety controller to use an identification mechanism to map the VM Created by a providerSpec.
// These could be done using tag(s)/resource-groups etc.
// This logic is used by safety controller to delete orphan VMs which are not backed by any machine CRD
func (p *OpenstackDriver) CreateMachine(ctx context.Context, req *driver.CreateMachineRequest) (*driver.CreateMachineResponse, error) {
	klog.V(2).Infof("CreateMachine request has been received for %q", req.Machine.Name)
	defer klog.V(2).Infof("CreateMachine request has been processed for %q", req.Machine.Name)

	ex, err := p.setupExecutor(ctx, req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
	}

	server, err := ex.CreateMachine(ctx, req.Machine.Name, req.Secret.Data[cloudprovider.UserData])
	if err != nil {
		klog.Errorf("machine creation for machine %q failed with: %v", req.Machine.Name, err)
		return nil, status.Error(mapErrorToCode(err), err.Error())
	}

	response := driver.CreateMachineResponse{
		ProviderID: server.ProviderID,
		NodeName:   req.Machine.Name,
	}

	if len(server.InternalIPs) > 0 {
		addresses := make([]corev1.NodeAddress, 0, len(server.InternalIPs))

		for _, ip := range server.InternalIPs {
			addresses = append(addresses, corev1.NodeAddress{
				Type:    corev1.NodeInternalIP,
				Address: ip,
			})
		}
		response.Addresses = addresses
	}

	return &response, nil
}

// InitializeMachine handles VM initialization for openstack VM's. Currently, un-implemented.
func (p *OpenstackDriver) InitializeMachine(_ context.Context, _ *driver.InitializeMachineRequest) (*driver.InitializeMachineResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Openstack Provider does not yet implement InitializeMachine")
}

// DeleteMachine handles a machine deletion request
func (p *OpenstackDriver) DeleteMachine(ctx context.Context, req *driver.DeleteMachineRequest) (*driver.DeleteMachineResponse, error) {
	// Log messages to track delete request
	klog.V(2).Infof("DeleteMachine request has been received for %q", req.Machine.Name)
	defer klog.V(2).Infof("DeleteMachine request has been processed for %q", req.Machine.Name)

	ex, err := p.setupExecutor(ctx, req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
	}

	err = ex.DeleteMachine(ctx, req.Machine.Name, req.Machine.Spec.ProviderID)
	if err != nil {
		return nil, status.Error(mapErrorToCode(err), err.Error())
	}
	return &driver.DeleteMachineResponse{}, nil
}

// GetMachineStatus handles a machine get status request
func (p *OpenstackDriver) GetMachineStatus(ctx context.Context, req *driver.GetMachineStatusRequest) (response *driver.GetMachineStatusResponse, err error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("GetMachineStatus request has been received for %q", req.Machine.Name)
	defer klog.V(2).Infof("GetMachineStatus request has been processed for: %q", req.Machine.Name)

	ex, err := p.setupExecutor(ctx, req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
	}

	var machine *servers.Server
	// Finding by ProviderID should be the common path, by name fallback for pre-creation
	if req.Machine.Spec.ProviderID != "" {
		klog.V(2).Infof("Finding Machine (%q) by ProviderID: %q", req.Machine.Name, req.Machine.Spec.ProviderID)
		machine, err = ex.GetMachineByProviderID(ctx, req.Machine.Spec.ProviderID)
	} else {
		klog.V(2).Infof("Finding Machine by Tags and Name: %q", req.Machine.Name)
		machine, err = ex.GetMachineByName(ctx, req.Machine.Name)
	}

	if err != nil {
		if errors.Is(err, executor.ErrNotFound) {
			klog.V(2).Infof("Machine status: did not find Machine: %q", req.Machine.Name)
		} else {
			klog.Errorf("Failed to get Machine %q: %v", req.Machine.Name, err)
		}
		return nil, status.Error(mapErrorToCode(err), err.Error())
	}

	if machine.Name != req.Machine.Name {
		klog.Errorf("Name of server with ProviderID %q (%q) does not match req.Machine.Name %q",
			req.Machine.Spec.ProviderID, machine.Name, req.Machine.Name)
		return nil, status.Error(codes.Internal, "Name and request machine name mismatch")
	}

	return &driver.GetMachineStatusResponse{
		ProviderID: req.Machine.Spec.ProviderID,
		NodeName:   machine.Name,
	}, nil
}

// ListMachines lists all the machines possibly created by a providerSpec
// Identifying machines created by a given providerSpec depends on the OPTIONAL IMPLEMENTATION LOGIC
// you have used to identify machines created by a providerSpec. It could be tags/resource-groups etc
func (p *OpenstackDriver) ListMachines(ctx context.Context, req *driver.ListMachinesRequest) (*driver.ListMachinesResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("ListMachines request has been received for %q", req.MachineClass.Name)
	defer klog.V(2).Infof("ListMachines request has been processed for %q", req.MachineClass.Name)

	ex, err := p.setupExecutor(ctx, req.MachineClass, req.Secret)
	if err != nil {
		return nil, err
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
