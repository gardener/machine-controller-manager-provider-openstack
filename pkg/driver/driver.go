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
	"fmt"
	"strings"
	"time"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
	api "github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/openstack"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/driver"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
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
	klog.V(2).Infof("Machine creation request has been received for %q", req.Machine.Name)
	defer klog.V(2).Infof("Machine creation request has been processed for %q", req.Machine.Name)

	providerConfig, err := p.decodeProviderSpec(req.MachineClass.ProviderSpec)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := p.validateRequest(providerConfig, req.Secret); err != nil {
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

	machID, nodeName, err := ex.createMachine(ctx, req.Machine.Name, req.Secret.Data[cloudprovider.UserData])
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &driver.CreateMachineResponse{
		ProviderID: machID,
		NodeName:   nodeName,
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

	providerConfig, err := p.decodeProviderSpec(&req.MachineClass.ProviderSpec)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := p.validateRequest(providerConfig, req.Secret); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err = p.deleteMachine(ctx, "")
	if err != nil {
		return nil, errorWrap(codes.Aborted, err, "")
	}
	return &driver.DeleteMachineResponse{}, status.Error(codes.Unimplemented, "")
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
	klog.V(2).Infof("Get request has been recieved for %q", req.Machine.Name)
	defer klog.V(2).Infof("Machine get request has been processed successfully for %q", req.Machine.Name)

	return &driver.GetMachineStatusResponse{}, status.Error(codes.Unimplemented, "")
}

// ListMachines lists all the machines possibilly created by a providerSpec
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
	klog.V(2).Infof("List machines request has been recieved for %q", req.MachineClass.Name)
	defer klog.V(2).Infof("List machines request has been recieved for %q", req.MachineClass.Name)

	return &driver.ListMachinesResponse{}, status.Error(codes.Unimplemented, "")
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
	defer klog.V(2).Infof("GetVolumeIDs request has been processed successfully for %q", req.PVSpecs)

	return &driver.GetVolumeIDsResponse{}, status.Error(codes.Unimplemented, "")
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
	defer klog.V(2).Infof("MigrateMachineClass request has been processed successfully for %q", req.ClassSpec)

	return &driver.GenerateMachineClassForMigrationResponse{}, status.Error(codes.Unimplemented, "")
}

func resourceInstanceBlockDevicesV2(rootDiskSize int, imageID string) ([]bootfromvolume.BlockDevice, error) {
	blockDeviceOpts := make([]bootfromvolume.BlockDevice, 1)
	blockDeviceOpts[0] = bootfromvolume.BlockDevice{
		UUID:                imageID,
		VolumeSize:          rootDiskSize,
		BootIndex:           0,
		DeleteOnTermination: true,
		SourceType:          "image",
		DestinationType:     "volume",
	}
	klog.V(2).Infof("[DEBUG] Block Device Options: %+v", blockDeviceOpts)
	return blockDeviceOpts, nil
}

func waitForStatus(c openstack.Compute, id string, pending []string, target []string, secs int) error {
	return wait.Poll(time.Second, 600*time.Second, func() (done bool, err error) {
		current, err := c.GetServer(id)
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok && strSliceContains(target, "DELETED") {
				return true, nil
			}
			return false, err
		}

		if strSliceContains(target, current.Status) {
			return true, nil
		}

		// if there is no pending statuses defined or current status is in the pending list, then continue polling
		if len(pending) == 0 || strSliceContains(pending, current.Status) {
			return false, nil
		}

		retErr := fmt.Errorf("unexpected status %q, wanted target %q", current.Status, strings.Join(target, ", "))
		if current.Status == "ERROR" {
			retErr = fmt.Errorf("%s, fault: %+v", retErr, current.Fault)
		}

		return false, retErr
	})
}

func (p *OpenstackDriver) deleteMachine(ctx context.Context, machineID string) error {
	return nil
}

func (p *OpenstackDriver) deleteMachine2(ctx context.Context, machineName, machineID string, providerSpec *api.MachineProviderConfig, compute openstack.Compute, network openstack.Network) error {
	res, err := p.getVMs(ctx, machineID, providerSpec, compute)
	if err != nil {
		return err
	} else if len(res) > 0 {
		instanceID := p.decodeProviderID(machineID)

		err = compute.DeleteServer(instanceID)
		//todo handleisnotfound
		if err != nil {
			klog.Errorf("Failed to delete machine with ID: %s", machineID)
			return err
		}

		// waiting for the machine to be deleted to release consumed quota resources, 5 minutes should be enough
		err = waitForStatus(compute, machineID, nil, []string{"DELETED", "SOFT_DELETED"}, 300)
		if err != nil {
			return fmt.Errorf("error waiting for the %q server to be deleted: %s", machineID, err)
		}
		klog.V(3).Infof("Deleted machine with ID: %s", machineID)

	} else {
		// No running instance exists with the given machine-ID
		klog.V(2).Infof("No VM matching the machine-ID found on the provider %q", machineID)
	}

	if err = p.deletePort(machineName, providerSpec, network); err != nil {
		return err
	}

	return nil
}

func (p *OpenstackDriver) getVMs(ctx context.Context, machineID string, providerSpec *api.MachineProviderConfig, compute openstack.Compute) (map[string]string, error) {
	listOfVMs := map[string]string{}

	searchClusterName := ""
	searchNodeRole := ""

	for key := range providerSpec.Spec.Tags {
		if strings.Contains(key, "kubernetes.io-cluster-") {
			searchClusterName = key
		} else if strings.Contains(key, "kubernetes.io-role-") {
			searchNodeRole = key
		}
	}

	if searchClusterName == "" || searchNodeRole == "" {
		return listOfVMs, nil
	}

	servers, err := compute.ListServers(servers.ListOpts{})
	if err != nil {
		klog.Errorf("Could not list instances. Error Message - %s", err)
		return nil, err
	}

	for _, server := range servers {
		clusterName := ""
		nodeRole := ""

		for key := range server.Metadata {
			if strings.Contains(key, "kubernetes.io-cluster-") {
				clusterName = key
			} else if strings.Contains(key, "kubernetes.io-role-") {
				nodeRole = key
			}
		}

		if clusterName == searchClusterName && nodeRole == searchNodeRole {
			instanceID := p.encodeProviderID(providerSpec.Spec.Region, server.ID)

			if machineID == "" {
				listOfVMs[instanceID] = server.Name
			} else if machineID == instanceID {
				listOfVMs[instanceID] = server.Name
				klog.V(3).Infof("Found machine with name: %q", server.Name)
				break
			}
		}

	}

	// Define an anonymous function to be executed on each page's iteration
	return listOfVMs, err
}

func (p *OpenstackDriver) deletePort(machineName string, provider *api.MachineProviderConfig, nwClient openstack.Network) error {
	if provider.Spec.SubnetID == nil || len(*provider.Spec.SubnetID) == 0 {
		return nil
	}

	portID, err := nwClient.PortIDFromName(machineName)
	if err != nil {
		if openstack.IsNotFoundError(err) {
			klog.V(3).Infof("port with name %q was not found", machineName)
			return nil
		}

		return fmt.Errorf("error deleting port with name %q: %s", machineName, err)
	}

	klog.V(3).Infof("deleting port with ID %s", portID)
	err = nwClient.DeletePort(portID)
	if err != nil {
		klog.Errorf("Failed to delete port with ID: %s", portID)
		return err
	}

	klog.V(3).Infof("Deleted port with ID: %s", portID)

	return nil
}
