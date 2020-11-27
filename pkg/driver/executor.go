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
	"fmt"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/schedulerhints"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

func (ex *executor) createMachine(ctx context.Context, machineName string, userData []byte) (string, string, error) {
	serverNetworks, podNetworkIDs, err := ex.resolveServerNetworks(machineName)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve server networks: %v", err)
	}

	server, err := ex.deployServer(machineName, userData, serverNetworks)
	if err != nil {
		return "", "", fmt.Errorf("error deploying server: %w", err)
	}

	machineID := encodeProviderID(ex.cfg.Spec.Region, server.ID)
	deleteOnFail := func(err error) error {
		if errIn := ex.deleteMachine(ctx, machineID); errIn != nil {
			return fmt.Errorf("error deleting machine %q after unsuccessful creation attempt: %s. Original error: %s", machineID, errIn.Error(), err.Error())
		}
		return err
	}

	err = ex.waitForStatus(server.ID, []string{"BUILD"}, []string{"ACTIVE"}, 600)
	if err != nil {
		return "", "", deleteOnFail(fmt.Errorf("error waiting for the %q server status: %s", server.ID, err))
	}

	allPorts, err := ex.network.ListPorts(&ports.ListOpts{
		DeviceID: server.ID,
	})
	if err != nil {
		return "", "", deleteOnFail(fmt.Errorf("failed to get ports: %s", err))
	}

	if len(allPorts) == 0 {
		return "", "", deleteOnFail(fmt.Errorf("got an empty port list for server ID %s", server.ID))
	}

	for _, port := range allPorts {
		for id := range podNetworkIDs {
			if port.NetworkID == id {
				if err := ex.network.UpdatePort(port.ID, ports.UpdateOpts{

					AllowedAddressPairs: &[]ports.AddressPair{{IPAddress: id}},
				}); err != nil {
					return "", "", deleteOnFail(fmt.Errorf("failed to update allowed address pair for port ID %s: %s", port.ID, err))
				}
			}
		}
	}

	if err := ex.patchServerPortsForPodNetwork(server.ID, podNetworkIDs); err != nil {
		return "", "", deleteOnFail(fmt.Errorf("failed to patch server ports for server %s: %s", server.ID, err))
	}
	return machineID, machineName, nil
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
		// if no SubnetID is specified, use only the NetworkID.
		if isEmptyStringPtr(ex.cfg.Spec.SubnetID) {
			klog.V(3).Infof("deploying in existing network [ID=%q]", networkID)
			serverNetworks = append(serverNetworks, servers.Network{UUID: ex.cfg.Spec.NetworkID})
		} else {
			klog.V(3).Infof("deploying in existing network [ID=%q] and subnet [ID=%q]. Pre-allocating Neutron Port... ", networkID, *subnetID)
			if _, err := ex.network.GetSubnet(*subnetID); err != nil {
				return nil, nil, err
			}

			klog.V(3).Infof("creating port in subnet %s", *ex.cfg.Spec.SubnetID)
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

func (ex *executor) waitForStatus(id string, pending []string, target []string, secs int) error {
	return wait.Poll(time.Second, time.Duration(secs)*time.Second, func() (done bool, err error) {
		current, err := ex.compute.GetServer(id)
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
		server, err = ex.compute.BootFromVolume(createOpts)
	} else {
		server, err = ex.compute.CreateServer(createOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("error creating server: %s", err)
	}

	return server, err
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

func (ex *executor) deleteMachine(ctx context.Context, machineID string) error {
	return nil
}
