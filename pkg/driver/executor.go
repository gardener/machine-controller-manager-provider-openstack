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
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/schedulerhints"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

const (
	cinderDriverName = "cinder.csi.openstack.org"
)

func (ex *executor) createMachine(ctx context.Context, machineName string, userData []byte) (string, error) {
	serverNetworks, podNetworkIDs, err := ex.resolveServerNetworks(machineName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve server networks: %v", err)
	}

	server, err := ex.deployServer(machineName, userData, serverNetworks)
	if err != nil {
		return "", fmt.Errorf("error deploying server: %w", err)
	}

	providerID := encodeProviderID(ex.cfg.Spec.Region, server.ID)
	deleteOnFail := func(err error) error {
		if errIn := ex.deleteMachine(ctx, machineName, providerID); errIn != nil {
			return fmt.Errorf("error deleting machine %q after unsuccessful creation attempt: %s. Original error: %s", providerID, errIn.Error(), err.Error())
		}
		return err
	}

	err = ex.waitForStatus(server.ID, []string{"BUILD"}, []string{"ACTIVE"}, 600)
	if err != nil {
		return "", deleteOnFail(fmt.Errorf("error waiting for the %q server status: %s", server.ID, err))
	}

	if err := ex.patchServerPortsForPodNetwork(server.ID, podNetworkIDs); err != nil {
		return "", deleteOnFail(fmt.Errorf("failed to patch server ports for server %s: %s", server.ID, err))
	}
	return providerID, nil
}

// resolveServerNetworks processes the networks for the server.
// It returns a list of networks that the server is part of, a map of network IDs that are part of the Pod network and
// the error if any occurred.
func (ex *executor) resolveServerNetworks(machineName string) ([]servers.Network, map[string]struct{}, error) {
	var (
		networkID      = ex.cfg.Spec.NetworkID
		subnetID       = ex.cfg.Spec.SubnetID
		networks       = ex.cfg.Spec.Networks
		serverNetworks = make([]servers.Network, 0, 0)
		podNetworkIDs  = make(map[string]struct{})
	)

	// If NetworkID is specified in the spec, we deploy the VMs in an existing Network.
	// If SubnetID is specified in addition to NetworkID, we have to preallocate a Neutron Port to force the VMs to get IP from the subnet's range.
	if !isEmptyString(networkID) {
		klog.V(3).Infof("deploying in existing network [ID=%q]", networkID)
		if isEmptyStringPtr(ex.cfg.Spec.SubnetID) {
			// if no SubnetID is specified, use only the NetworkID.
			serverNetworks = append(serverNetworks, servers.Network{UUID: ex.cfg.Spec.NetworkID})
		} else {
			klog.V(3).Infof("deploying in existing subnet [ID=%q]. Pre-allocating Neutron Port... ", *subnetID)
			if _, err := ex.network.GetSubnet(*subnetID); err != nil {
				return nil, nil, err
			}

			var securityGroupIDs []string
			for _, securityGroup := range ex.cfg.Spec.SecurityGroups {
				securityGroupID, err := ex.network.GroupIDFromName(securityGroup)
				if err != nil {
					return nil, nil, err
				}
				securityGroupIDs = append(securityGroupIDs, securityGroupID)
			}

			port, err := ex.network.CreatePort(&ports.CreateOpts{
				Name:                machineName,
				NetworkID:           ex.cfg.Spec.NetworkID,
				FixedIPs:            []ports.IP{{SubnetID: *ex.cfg.Spec.SubnetID}},
				AllowedAddressPairs: []ports.AddressPair{{IPAddress: ex.cfg.Spec.PodNetworkCidr}},
				SecurityGroups:      &securityGroupIDs,
			})
			if err != nil {
				return nil, nil, err
			}
			klog.V(3).Infof("port [ID=%q] successfully created", port.ID)
			serverNetworks = append(serverNetworks, servers.Network{UUID: ex.cfg.Spec.NetworkID, Port: port.ID})
		}
		podNetworkIDs[networkID] = struct{}{}
	} else {
		for _, network := range networks {
			var (
				resolvedNetworkID string
				err               error
			)
			if isEmptyString(network.Id) {
				resolvedNetworkID, err = ex.network.NetworkIDFromName(network.Name)
				if err != nil {
					return nil, nil, err
				}
			} else {
				resolvedNetworkID = networkID
			}
			serverNetworks = append(serverNetworks, servers.Network{UUID: resolvedNetworkID})
			if network.PodNetwork {
				podNetworkIDs[resolvedNetworkID] = struct{}{}
			}
		}
	}
	return serverNetworks, podNetworkIDs, nil
}

func (ex *executor) waitForStatus(serverID string, pending []string, target []string, secs int) error {
	return wait.Poll(time.Second, time.Duration(secs)*time.Second, func() (done bool, err error) {
		current, err := ex.compute.GetServer(serverID)
		if err != nil {
			if openstack.IsNotFoundError(err) && strSliceContains(target, openstack.StatusDeleted) {
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
		if current.Status == openstack.StatusError {
			retErr = fmt.Errorf("%s, fault: %+v", retErr, current.Fault)
		}

		return false, retErr
	})
}

func (ex *executor) deployServer(machineName string, userData []byte, nws []servers.Network) (*servers.Server, error) {
	keyName := ex.cfg.Spec.KeyName
	imageName := ex.cfg.Spec.ImageName
	imageID := ex.cfg.Spec.ImageID
	securityGroups := ex.cfg.Spec.SecurityGroups
	availabilityZone := ex.cfg.Spec.AvailabilityZone
	metadata := ex.cfg.Spec.Tags
	rootDiskSize := ex.cfg.Spec.RootDiskSize
	useConfigDrive := ex.cfg.Spec.UseConfigDrive
	flavorName := ex.cfg.Spec.FlavorName

	var (
		imageRef   string
		createOpts servers.CreateOptsBuilder
		err        error
	)

	// use imageID if provided, otherwise try to resolve the imageName to an imageID
	if imageID != "" {
		imageRef = imageID
	} else {
		imageRef, err = ex.compute.ImageIDFromName(imageName)
		if err != nil {
			return nil, fmt.Errorf("error resolving image ID from image name %s: %v", imageName, err)
		}
	}
	flavorRef, err := ex.compute.FlavorIDFromName(flavorName)
	if err != nil {
		return nil, fmt.Errorf("error resolving flavor ID from flavor name %s: %v", imageName, err)
	}

	createOpts = &servers.CreateOpts{
		// ServiceClient:    ex.compute.ServiceClient(),
		Name: machineName,
		// FlavorName:       flavorName,
		FlavorRef:        flavorRef,
		ImageRef:         imageRef,
		Networks:         nws,
		SecurityGroups:   securityGroups,
		Metadata:         metadata,
		UserData:         userData,
		AvailabilityZone: availabilityZone,
		ConfigDrive:      useConfigDrive,
	}

	createOpts = &keypairs.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		KeyName:           keyName,
	}

	if ex.cfg.Spec.ServerGroupID != nil {
		hints := schedulerhints.SchedulerHints{
			Group: *ex.cfg.Spec.ServerGroupID,
		}
		createOpts = schedulerhints.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			SchedulerHints:    hints,
		}
	}
	createOpts = &keypairs.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		KeyName:           keyName,
	}

	if ex.cfg.Spec.ServerGroupID != nil {
		hints := schedulerhints.SchedulerHints{
			Group: *ex.cfg.Spec.ServerGroupID,
		}
		createOpts = schedulerhints.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			SchedulerHints:    hints,
		}
	}

	var server *servers.Server
	// If a custom block_device (root disk size is provided) we need to boot from volume
	if rootDiskSize > 0 {
		blockDevices, err := resourceInstanceBlockDevicesV2(rootDiskSize, imageRef)
		if err != nil {
			return nil, err
		}

		createOpts = &bootfromvolume.CreateOptsExt{
			CreateOptsBuilder: createOpts,
			BlockDevice:       blockDevices,
		}
		server, err = ex.compute.BootFromVolume(createOpts)
	} else {
		server, err = ex.compute.CreateServer(createOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("error creating server: %s", err)
	}

	return server, err
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

func (ex *executor) patchServerPortsForPodNetwork(serverID string, podNetworkIDs map[string]struct{}) error {
	allPorts, err := ex.network.ListPorts(&ports.ListOpts{
		DeviceID: serverID,
	})
	if err != nil {
		return fmt.Errorf("failed to get ports: %s", err)
	}

	if len(allPorts) == 0 {
		return fmt.Errorf("got an empty port list for server ID %s", serverID)
	}

	for _, port := range allPorts {
		for id := range podNetworkIDs {
			if port.NetworkID == id {
				if err := ex.network.UpdatePort(port.ID, ports.UpdateOpts{
					AllowedAddressPairs: &[]ports.AddressPair{{IPAddress: ex.cfg.Spec.PodNetworkCidr}},
				}); err != nil {
					return fmt.Errorf("failed to update allowed address pair for port ID %s: %s", port.ID, err)
				}
			}
		}
	}
	return nil
}

func (ex *executor) deleteMachine(ctx context.Context, machineName, providerID string) error {
	serverID := decodeProviderID(providerID)

	klog.V(1).Infof("finding server with id %s", serverID)
	_, err := ex.getVM(ctx, serverID)
	if err != nil {
		if errors.Is(ErrNotFound, err) {
			return nil
		}
	}

	klog.V(1).Infof("deleting server with id %s", serverID)
	if err := ex.compute.DeleteServer(serverID); err != nil {
		return err
	}

	if err = ex.waitForStatus(serverID, nil, []string{openstack.StatusDeleted}, 300); err != nil {
		return fmt.Errorf("error waiting for server [ID=%q] to be deleted: %v", serverID, err)
	}

	//
	if !isEmptyStringPtr(ex.cfg.Spec.SubnetID) {
		return ex.deletePort(ctx, machineName)
	}

	return nil
}

func (ex *executor) getVM(ctx context.Context, serverID string) (*servers.Server, error) {
	server, err := ex.compute.GetServer(serverID)
	if err != nil {
		// mask NotFound errors
		klog.V(2).Infof("error fetching server %q: %v", serverID, err)
		if openstack.IsNotFoundError(err) {
			return nil, fmt.Errorf("%s%w", err, ErrNotFound)
		}
		return nil, err
	}

	// TODO(KA): is it necessary to scan for tags if we know the serverID ?
	var (
		searchClusterName string
		searchNodeRole    string
	)
	for key := range ex.cfg.Spec.Tags {
		if strings.Contains(key, cloudprovider.ServerTagClusterPrefix) {
			searchClusterName = key
		} else if strings.Contains(key, cloudprovider.ServerTagRolePrefix) {
			searchNodeRole = key
		}
	}

	if _, ok := server.Metadata[searchClusterName]; ok {
		if _, ok2 := server.Metadata[searchNodeRole]; ok2 {
			return server, nil
		}
	}
	klog.V(1).Infof("AAAAA Server found %q, but no matching tags %v", serverID, server.Metadata)

	return nil, fmt.Errorf("server [ID=%q] found, but cluster/role tags are missing%w", serverID, ErrNotFound)

}

func (ex *executor) deletePort(ctx context.Context, machineName string) error {
	portID, err := ex.network.PortIDFromName(machineName)
	if err != nil {
		if openstack.IsNotFoundError(err) {
			klog.V(3).Infof("port with name %q was not found", machineName)
			return nil
		}
		return fmt.Errorf("error deleting port with name %q: %s", machineName, err)
	}

	klog.V(3).Infof("deleting port with ID %s", portID)

	err = ex.network.DeletePort(portID)
	if err != nil {
		klog.Errorf("failed to delete port with ID: %s", portID)
		return err
	}
	klog.V(3).Infof("deleted port with ID: %s", portID)

	return nil
}

func (ex *executor) getNoProvider(ctx context.Context, machineName string) (string, error) {
	vms, err := ex.listMachines(ctx)
	if err != nil {
		return "", err
	}

	result := []string{}
	for key, val := range vms {
		if val == machineName {
			klog.V(1).Infof("BBBBBBBBBBBBBBB no providerID but server found %q", machineName)
			result = append(result, key)
		}
	}

	if len(result) > 1 {
		klog.V(1).Infof("BBBBBBBBBBBBBB a lot of providerIDs found for name %q: %v", machineName, result)
		return "", fmt.Errorf("JPDSAJOJDSJDJDAS")
	} else if len(result) == 0 {
		return "", fmt.Errorf("fdasfdsfdsda %w", ErrNotFound)
	}

	return result[0], nil
}

func (ex *executor) getMachineStatus(ctx context.Context, machineName, providerID string) error {
	serverID := decodeProviderID(providerID)

	server, err := ex.getVM(ctx, serverID)
	if err != nil {
		return err
	}

	if server.Name != machineName {
		return fmt.Errorf("server with ID %q found, but server name %q does not match expected name %q%w", serverID, server.Name, machineName, ErrNotFound)
	}

	return nil
}

func (ex *executor) listMachines(ctx context.Context) (map[string]string, error) {
	searchClusterName := ""
	searchNodeRole := ""

	for key := range ex.cfg.Spec.Tags {
		if strings.Contains(key, cloudprovider.ServerTagClusterPrefix) {
			searchClusterName = key
		} else if strings.Contains(key, cloudprovider.ServerTagRolePrefix) {
			searchNodeRole = key
		}
	}

	// TODO(KA) better tag handling ? Should it return nil if no tags are found (should be blocked by validation)
	if searchClusterName == "" || searchNodeRole == "" {
		klog.V(1).Infof("AAAAAAAAAAAAAAAAA No cluster tag keys found: %q, %q", searchClusterName, searchNodeRole)
		return nil, nil
	}

	servers, err := ex.compute.ListServers(&servers.ListOpts{})
	if err != nil {
		return nil, err
	}

	result := map[string]string{}
	for _, server := range servers {
		if _, ok := server.Metadata[searchClusterName]; ok {
			if _, ok2 := server.Metadata[searchNodeRole]; ok2 {
				providerID := encodeProviderID(ex.cfg.Spec.Region, server.ID)
				result[providerID] = server.Name
			}
		}
	}

	return result, nil
}
