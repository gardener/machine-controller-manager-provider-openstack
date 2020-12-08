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

// Package provider contains the cloud provider specific implementations to manage machines
package driver

import (
	"context"
	"errors"

	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"k8s.io/klog"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
)

const (
	openStackMachineClassKind = "OpenStackMachineClass"
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
// REQUIRED METHOD
//
// REQUEST PARAMETERS (driver.CreateMachineRequest)
// Machine               *v1alpha1.Machine        Machine object from whom VM is to be created
// MachineClass          *v1alpha1.MachineClass   MachineClass backing the machine object
// Secret                *corev1.Secret           Kubernetes secret that contains any sensitive data/credentials
//
// RESPONSE PARAMETERS (driver.CreateMachineResponse)
// ProviderID            string                   Unique identification of the VM at the cloud provider. This could be the same/different from req.MachineName.
//                                                ProviderID typically matches with the node.Spec.ProviderID on the node object.
//                                                Eg: gce://project-name/region/vm-ProviderID
// NodeName              string                   Returns the name of the node-object that the VM register's with Kubernetes.
//                                                This could be different from req.MachineName as well
// LastKnownState        string                   (Optional) Last known state of VM during the current operation.
//                                                Could be helpful to continue operations in future requests.
//
// OPTIONAL IMPLEMENTATION LOGIC
// It is optionally expected by the safety controller to use an identification mechanism to map the VM Created by a providerSpec.
// These could be done using tag(s)/resource-groups etc.
// This logic is used by safety controller to delete orphan VMs which are not backed by any machine CRD
//
func (p *OpenstackDriver) CreateMachine(ctx context.Context, req *driver.CreateMachineRequest) (*driver.CreateMachineResponse, error) {
	// Log messages to track request
	klog.V(2).Infof("machine creation request has been received for %q", req.Machine.Name)
	defer klog.V(2).Infof("machine creation request has been processed for %q", req.Machine.Name)

	providerConfig, err := p.decodeProviderSpec(req.MachineClass.ProviderSpec)
	if err != nil {
		klog.V(2).Infof("decoding provider spec for machine class %q failed with: %v", req.MachineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := p.validateRequest(providerConfig, req.Secret); err != nil {
		klog.V(2).Infof("validating request for machine class %q failed with: %v", req.MachineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if !isEmptyString(req.Machine.Spec.ProviderID) {
		klog.V(1).Infof("AAAAAAAAAAAAAAAAAAAA create with providerID: %q", req.Machine.Spec.ProviderID)
		return &driver.CreateMachineResponse{
			ProviderID:     req.Machine.Spec.ProviderID,
			NodeName:       req.Machine.Name,
		}, nil
	}

	factory, err := p.clientConstructor(req.Secret)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	computeClient, err := factory.Compute()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	networkClient, err := factory.Network()

	ex := executor{
		compute: computeClient,
		network: networkClient,
		cfg:     *providerConfig,
	}

	providerID, err := ex.createMachine(ctx, req.Machine.Name, req.Secret.Data[cloudprovider.UserData])
	if err != nil {
		klog.V(2).Infof("machine creation for %q failed with: %v", req.Machine.Name, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &driver.CreateMachineResponse{
		ProviderID: providerID,
		NodeName:   req.Machine.Name,
	}, nil
}

// DeleteMachine handles a machine deletion request
//
// REQUEST PARAMETERS (driver.DeleteMachineRequest)
// Machine               *v1alpha1.Machine        Machine object from whom VM is to be deleted
// MachineClass          *v1alpha1.MachineClass   MachineClass backing the machine object
// Secret                *corev1.Secret           Kubernetes secret that contains any sensitive data/credentials
//
// RESPONSE PARAMETERS (driver.DeleteMachineResponse)
// LastKnownState        bytes(blob)              (Optional) Last known state of VM during the current operation.
//                                                Could be helpful to continue operations in future requests.
//
func (p *OpenstackDriver) DeleteMachine(ctx context.Context, req *driver.DeleteMachineRequest) (*driver.DeleteMachineResponse, error) {
	// Log messages to track delete request
	klog.V(2).Infof("Machine deletion request has been recieved for %q", req.Machine.Name)
	defer klog.V(2).Infof("Machine deletion request has been processed for %q", req.Machine.Name)

	providerConfig, err := p.decodeProviderSpec(req.MachineClass.ProviderSpec)
	if err != nil {
		klog.V(2).Infof("decoding provider spec for machine class %q failed with: %v", req.MachineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := p.validateRequest(providerConfig, req.Secret); err != nil {
		klog.V(2).Infof("validating request for machine class %q failed with: %v", req.MachineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	factory, err := p.clientConstructor(req.Secret)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	computeClient, err := factory.Compute()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	networkClient, err := factory.Network()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	ex := executor{
		compute: computeClient,
		network: networkClient,
		cfg:     *providerConfig,
	}

	err = ex.deleteMachine(ctx, req.Machine.Name, req.Machine.Spec.ProviderID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &driver.DeleteMachineResponse{}, nil
}

// GetMachineStatus handles a machine get status request
// OPTIONAL METHOD
//
// REQUEST PARAMETERS (driver.GetMachineStatusRequest)
// Machine               *v1alpha1.Machine        Machine object from whom VM status needs to be returned
// MachineClass          *v1alpha1.MachineClass   MachineClass backing the machine object
// Secret                *corev1.Secret           Kubernetes secret that contains any sensitive data/credentials
//
// RESPONSE PARAMETERS (driver.GetMachineStatueResponse)
// ProviderID            string                   Unique identification of the VM at the cloud provider. This could be the same/different from req.MachineName.
//                                                ProviderID typically matches with the node.Spec.ProviderID on the node object.
//                                                Eg: gce://project-name/region/vm-ProviderID
// NodeName             string                    Returns the name of the node-object that the VM register's with Kubernetes.
//                                                This could be different from req.MachineName as well
//
// The request should return a NOT_FOUND (5) status error code if the machine is not existing
func (p *OpenstackDriver) GetMachineStatus(ctx context.Context, req *driver.GetMachineStatusRequest) (*driver.GetMachineStatusResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("Get request has been received for %q", req.Machine.Name)

	providerConfig, err := p.decodeProviderSpec(req.MachineClass.ProviderSpec)
	if err != nil {
		klog.V(2).Infof("decoding provider spec for machine class %q failed with: %v", req.MachineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := p.validateRequest(providerConfig, req.Secret); err != nil {
		klog.V(2).Infof("validating request for machine class %q failed with: %v", req.MachineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}


	factory, err := p.clientConstructor(req.Secret)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	computeClient, err := factory.Compute()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	networkClient, err := factory.Network()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	ex := executor{
		compute: computeClient,
		network: networkClient,
		cfg:     *providerConfig,
	}

	// TODO(KA): We can could extrapolate the serverID for machine by doing a reverse name search and matching metadata tags with MachineClass.
	if isEmptyString(req.Machine.Spec.ProviderID) {
		klog.V(2).Infof("AAAAAAAAAAAAAAAAAAAAAa missing providerID from machine spec %v", req.Machine.Spec)
		prov, err := ex.getNoProvider(ctx, req.Machine.Name)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, status.Error(codes.NotFound, err.Error())
			} else {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}
		return &driver.GetMachineStatusResponse{
			ProviderID: prov,
			NodeName:   req.Machine.Name,
		}, nil
		// return nil, status.Error(codes.NotFound, fmt.Sprintf("missing providerID from machine spec"))
	}

	if err := ex.getMachineStatus(ctx, req.Machine.Name, req.Machine.Spec.ProviderID); err != nil {
		if errors.Is(err, ErrNotFound) {
			klog.V(2).Infof("machine %q not found: %v", req.Machine.Name, err)
			return nil, status.Error(codes.NotFound, err.Error())
		}
		klog.V(2).Infof("getMachineStatus request for %q failed with: %v", req.Machine.Name, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	klog.V(2).Infof("Machine get request has been processed for %q: %v", req.Machine.Name, err)
	return &driver.GetMachineStatusResponse{
		ProviderID: req.Machine.Spec.ProviderID,
		NodeName:   req.Machine.Name,
	}, nil
}

// ListMachines lists all the machines possibly created by a providerSpec
// Identifying machines created by a given providerSpec depends on the OPTIONAL IMPLEMENTATION LOGIC
// you have used to identify machines created by a providerSpec. It could be tags/resource-groups etc
// OPTIONAL METHOD
//
// REQUEST PARAMETERS (driver.ListMachinesRequest)
// MachineClass          *v1alpha1.MachineClass   MachineClass based on which VMs created have to be listed
// Secret                *corev1.Secret           Kubernetes secret that contains any sensitive data/credentials
//
// RESPONSE PARAMETERS (driver.ListMachinesResponse)
// MachineList           map<string,string>  A map containing the keys as the MachineID and value as the MachineName
//                                           for all machine's who where possibilly created by this ProviderSpec
//
func (p *OpenstackDriver) ListMachines(ctx context.Context, req *driver.ListMachinesRequest) (*driver.ListMachinesResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("list machines request has been received for %q", req.MachineClass.Name)
	defer klog.V(2).Infof("list machines request has been processed for %q", req.MachineClass.Name)

	providerConfig, err := p.decodeProviderSpec(req.MachineClass.ProviderSpec)
	if err != nil {
		klog.V(2).Infof("decoding provider spec for machine class %q failed with: %v", req.MachineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err := p.validateRequest(providerConfig, req.Secret); err != nil {
		klog.V(2).Infof("validating request for machine class %q failed with: %v", req.MachineClass.Name, err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	factory, err := p.clientConstructor(req.Secret)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	computeClient, err := factory.Compute()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	networkClient, err := factory.Network()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	ex := executor{
		compute: computeClient,
		network: networkClient,
		cfg:     *providerConfig,
	}

	machines, err := ex.listMachines(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if len(machines) == 0 {
		klog.V(2).Infof("no machines found for machine class: %q", req.MachineClass.Name)
	}

	return &driver.ListMachinesResponse{
		MachineList: machines,
	}, nil
}

// GetVolumeIDs returns a list of Volume IDs for all PV Specs for whom an provider volume was found
//
// REQUEST PARAMETERS (driver.GetVolumeIDsRequest)
// PVSpecList            []*corev1.PersistentVolumeSpec       PVSpecsList is a list PV specs for whom volume-IDs are required.
//
// RESPONSE PARAMETERS (driver.GetVolumeIDsResponse)
// VolumeIDs             []string                             VolumeIDs is a repeated list of VolumeIDs.
//
func (p *OpenstackDriver) GetVolumeIDs(ctx context.Context, req *driver.GetVolumeIDsRequest) (*driver.GetVolumeIDsResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("GetVolumeIDs request has been recieved for %q", req.PVSpecs)
	defer klog.V(2).Infof("GetVolumeIDs request has been processed for %q", req.PVSpecs)

	names := []string{}
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

// GenerateMachineClassForMigration helps in migration of one kind of machineClass CR to another kind.
// For instance an machineClass custom resource of `AWSMachineClass` to `MachineClass`.
// Implement this functionality only if something like this is desired in your setup.
// If you don't require this functionality leave is as is. (return Unimplemented)
//
// The following are the tasks typically expected out of this method
// 1. Validate if the incoming classSpec is valid one for migration (e.g. has the right kind).
// 2. Migrate/Copy over all the fields/spec from req.ProviderSpecificMachineClass to req.MachineClass
// For an example refer
//		https://github.com/prashanth26/machine-controller-manager-provider-gcp/blob/migration/pkg/gcp/machine_controller.go#L222-L233
//
// REQUEST PARAMETERS (driver.GenerateMachineClassForMigration)
// ProviderSpecificMachineClass    interface{}                             ProviderSpecificMachineClass is provider specfic machine class object (E.g. AWSMachineClass). Typecasting is required here.
// MachineClass 				   *v1alpha1.MachineClass                  MachineClass is the machine class object that is to be filled up by this method.
// ClassSpec                       *v1alpha1.ClassSpec                     Somemore classSpec details useful while migration.
//
// RESPONSE PARAMETERS (driver.GenerateMachineClassForMigration)
// NONE
//
func (p *OpenstackDriver) GenerateMachineClassForMigration(ctx context.Context, req *driver.GenerateMachineClassForMigrationRequest) (*driver.GenerateMachineClassForMigrationResponse, error) {
	// Log messages to track start and end of request
	klog.V(2).Infof("MigrateMachineClass request has been recieved for %q", req.ClassSpec)
	defer klog.V(2).Infof("MigrateMachineClass request has been processed for %q", req.ClassSpec)

	if req.ClassSpec.Kind != openStackMachineClassKind {
		return nil, status.Error(codes.Internal, "migration for this machineClass kind is not supported")
	}

	osMachineClass := req.ProviderSpecificMachineClass.(*v1alpha1.OpenStackMachineClass)
	err :=  migrateMachineClass(osMachineClass, req.MachineClass)
	if err != nil {
		err =  status.Error(codes.Internal, err.Error())
	}
	return &driver.GenerateMachineClassForMigrationResponse{}, err
}

