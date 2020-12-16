// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
	api "github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/schedulerhints"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
)

type Executor struct {
	Compute openstack.Compute
	Network openstack.Network
	Config  api.MachineProviderConfig
}

func NewExecutor(factory openstack.Factory, config api.MachineProviderConfig) (*Executor, error) {
	computeClient, err := factory.Compute()
	if err != nil {
		klog.Errorf("failed to create compute client for executor")
		return nil, err
	}
	networkClient, err := factory.Network()
	if err != nil {
		klog.Errorf("failed to create network client for executor")
		return nil, err
	}

	ex := &Executor{
		Compute: computeClient,
		Network: networkClient,
		Config:  config,
	}
	return ex, nil
}

func (ex *Executor) CreateMachine(ctx context.Context, machineName string, userData []byte) (string, error) {
	serverNetworks, podNetworkIDs, err := ex.resolveServerNetworks(machineName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve server networks: %v", err)
	}

	server, err := ex.deployServer(machineName, userData, serverNetworks)
	if err != nil {
		return "", fmt.Errorf("failed to deploy server for machine %q: %w", machineName, err)
	}

	providerID := EncodeProviderID(ex.Config.Spec.Region, server.ID)

	// if we fail in the creation post-processing step we have to delete the server we created
	deleteOnFail := func(err error) error {
		if errIn := ex.DeleteMachine(ctx, machineName, providerID); errIn != nil {
			return fmt.Errorf("error deleting machine %q after unsuccessful creation attempt: %s. Original error: %s", providerID, errIn.Error(), err.Error())
		}
		return err
	}

	err = ex.waitForStatus(server.ID, []string{openstack.StatusBuild}, []string{openstack.StatusActive}, 600)
	if err != nil {
		return "", deleteOnFail(fmt.Errorf("error waiting for server [ID=%q] to reach target status: %s", server.ID, err))
	}

	if err := ex.patchServerPortsForPodNetwork(server.ID, podNetworkIDs); err != nil {
		return "", deleteOnFail(fmt.Errorf("failed to patch server [ID=%q] ports: %s", server.ID, err))
	}
	return providerID, nil
}

// resolveServerNetworks processes the networks for the server.
// It returns a list of networks that the server is part of, a map of Network IDs that are part of the Pod Network and
// the error if any occurred.
func (ex *Executor) resolveServerNetworks(machineName string) ([]servers.Network, map[string]struct{}, error) {
	var (
		networkID      = ex.Config.Spec.NetworkID
		subnetID       = ex.Config.Spec.SubnetID
		networks       = ex.Config.Spec.Networks
		serverNetworks = make([]servers.Network, 0, 0)
		podNetworkIDs  = make(map[string]struct{})
	)

	// If NetworkID is specified in the spec, we deploy the VMs in an existing Network.
	// If SubnetID is specified in addition to NetworkID, we have to preallocate a Neutron Port to force the VMs to get IP from the subnet's range.
	if !isEmptyString(pointer.StringPtr(networkID)) {
		klog.V(3).Infof("deploying in existing network [ID=%q]", networkID)
		if isEmptyString(ex.Config.Spec.SubnetID) {
			// if no SubnetID is specified, use only the NetworkID for the network attachments.
			serverNetworks = append(serverNetworks, servers.Network{UUID: ex.Config.Spec.NetworkID})
		} else {
			klog.V(3).Infof("deploying in existing subnet [ID=%q]. Pre-allocating Neutron Port... ", *subnetID)
			if _, err := ex.Network.GetSubnet(*subnetID); err != nil {
				return nil, nil, err
			}

			var securityGroupIDs []string
			for _, securityGroup := range ex.Config.Spec.SecurityGroups {
				securityGroupID, err := ex.Network.GroupIDFromName(securityGroup)
				if err != nil {
					return nil, nil, err
				}
				securityGroupIDs = append(securityGroupIDs, securityGroupID)
			}

			port, err := ex.Network.CreatePort(&ports.CreateOpts{
				Name:                machineName,
				NetworkID:           ex.Config.Spec.NetworkID,
				FixedIPs:            []ports.IP{{SubnetID: *ex.Config.Spec.SubnetID}},
				AllowedAddressPairs: []ports.AddressPair{{IPAddress: ex.Config.Spec.PodNetworkCidr}},
				SecurityGroups:      &securityGroupIDs,
			})
			if err != nil {
				return nil, nil, err
			}
			klog.V(3).Infof("port [ID=%q] successfully created", port.ID)
			serverNetworks = append(serverNetworks, servers.Network{UUID: ex.Config.Spec.NetworkID, Port: port.ID})
		}
		podNetworkIDs[networkID] = struct{}{}
	} else {
		for _, network := range networks {
			var (
				resolvedNetworkID string
				err               error
			)
			if isEmptyString(pointer.StringPtr(network.Id)) {
				resolvedNetworkID, err = ex.Network.NetworkIDFromName(network.Name)
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

func (ex *Executor) waitForStatus(serverID string, pending []string, target []string, secs int) error {
	return wait.Poll(time.Second, time.Duration(secs)*time.Second, func() (done bool, err error) {
		current, err := ex.Compute.GetServer(serverID)
		if err != nil {
			if openstack.IsNotFoundError(err) && strSliceContains(target, openstack.StatusDeleted) {
				return true, nil
			}
			return false, err
		}

		klog.V(5).Infof("waiting for server [ID=%q] status %v. current status %v", serverID, target, current.Status)
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

func (ex *Executor) deployServer(machineName string, userData []byte, nws []servers.Network) (*servers.Server, error) {
	keyName := ex.Config.Spec.KeyName
	imageName := ex.Config.Spec.ImageName
	imageID := ex.Config.Spec.ImageID
	securityGroups := ex.Config.Spec.SecurityGroups
	availabilityZone := ex.Config.Spec.AvailabilityZone
	metadata := ex.Config.Spec.Tags
	rootDiskSize := ex.Config.Spec.RootDiskSize
	useConfigDrive := ex.Config.Spec.UseConfigDrive
	flavorName := ex.Config.Spec.FlavorName

	var (
		imageRef   string
		createOpts servers.CreateOptsBuilder
		err        error
	)

	// use imageID if provided, otherwise try to resolve the imageName to an imageID
	if imageID != "" {
		imageRef = imageID
	} else {
		imageRef, err = ex.Compute.ImageIDFromName(imageName)
		if err != nil {
			return nil, fmt.Errorf("error resolving image ID from image name %q: %v", imageName, err)
		}
	}
	flavorRef, err := ex.Compute.FlavorIDFromName(flavorName)
	if err != nil {
		return nil, fmt.Errorf("error resolving flavor ID from flavor name %q: %v", imageName, err)
	}

	createOpts = &servers.CreateOpts{
		// ServiceClient:    ex.Compute.ServiceClient(),
		// FlavorName:       flavorName,
		Name:             machineName,
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

	if ex.Config.Spec.ServerGroupID != nil {
		hints := schedulerhints.SchedulerHints{
			Group: *ex.Config.Spec.ServerGroupID,
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
		server, err = ex.Compute.BootFromVolume(createOpts)
	} else {
		server, err = ex.Compute.CreateServer(createOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("error creating server: %v", err)
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

func (ex *Executor) patchServerPortsForPodNetwork(serverID string, podNetworkIDs map[string]struct{}) error {
	allPorts, err := ex.Network.ListPorts(&ports.ListOpts{
		DeviceID: serverID,
	})
	if err != nil {
		return fmt.Errorf("failed to get ports: %v", err)
	}

	if len(allPorts) == 0 {
		return fmt.Errorf("got an empty port list for server %q", serverID)
	}

	for _, port := range allPorts {
		for id := range podNetworkIDs {
			if port.NetworkID == id {
				if err := ex.Network.UpdatePort(port.ID, ports.UpdateOpts{
					AllowedAddressPairs: &[]ports.AddressPair{{IPAddress: ex.Config.Spec.PodNetworkCidr}},
				}); err != nil {
					return fmt.Errorf("failed to update allowed address pair for port [ID=%q]: %v", port.ID, err)
				}
			}
		}
	}
	return nil
}

func (ex *Executor) DeleteMachine(ctx context.Context, machineName, providerID string) error {
	var (
		err    error
		server *servers.Server
	)

	if isEmptyString(pointer.StringPtr(providerID)) {
		server, err = ex.getMachineByName(ctx, machineName)
	} else {
		server, err = ex.getMachineByProviderID(ctx, machineName, providerID)
	}
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}

	klog.V(1).Infof("deleting server with id %s", server.ID)
	if err := ex.Compute.DeleteServer(server.ID); err != nil {
		return err
	}

	if err = ex.waitForStatus(server.ID, nil, []string{openstack.StatusDeleted}, 300); err != nil {
		return fmt.Errorf("error waiting for server [ID=%q] to be deleted: %v", server.ID, err)
	}

	if !isEmptyString(ex.Config.Spec.SubnetID) {
		return ex.deletePort(ctx, machineName)
	}

	return nil
}

func (ex *Executor) deletePort(_ context.Context, machineName string) error {
	portID, err := ex.Network.PortIDFromName(machineName)
	if err != nil {
		if openstack.IsNotFoundError(err) {
			klog.V(3).Infof("port with name %q was not found", machineName)
			return nil
		}
		return fmt.Errorf("error deleting port with name %q: %s", machineName, err)
	}

	klog.V(3).Infof("deleting port [ID=%q]", portID)
	err = ex.Network.DeletePort(portID)
	if err != nil {
		klog.Errorf("failed to delete [ID=%q]", portID)
		return err
	}
	klog.V(3).Infof("deleted port [ID=%q]", portID)

	return nil
}

func (ex *Executor) getMachineByProviderID(_ context.Context, machineName, providerID string) (*servers.Server, error) {
	klog.V(2).Infof("finding server with providerID %s", providerID)
	serverID := DecodeProviderID(providerID)
	if isEmptyString(pointer.StringPtr(serverID)) {
		return nil, fmt.Errorf("could not parse serverID from providerID %q", providerID)
	}

	server, err := ex.Compute.GetServer(serverID)
	if err != nil {
		klog.V(2).Infof("error finding server %q: %v", serverID, err)
		if openstack.IsNotFoundError(err) {
			// normalize errors by wrapping not found error
			return nil, fmt.Errorf("could not find server [ID=%q]: %w", serverID, ErrNotFound)
		}
		return nil, err
	}

	if machineName != server.Name {
		klog.Warningf("found server [ID=%q] with matching ID, but names mismatch with machine object %q", serverID, machineName)
		// TODO(KA): return an error if names mismatch ?
	}

	var (
		searchClusterName string
		searchNodeRole    string
	)
	for key := range ex.Config.Spec.Tags {
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

	klog.Warningf("server [ID=%q] found, but cluster/role tags are missing", serverID)
	return nil, fmt.Errorf("could not find server [ID=%q]: %w", serverID, ErrNotFound)
}

// getMachineByName returns a server that matches the following criteria:
// a) has the same name as machineName
// b) has the cluster and role tags as set in the machineClass
// The current approach is weak because the tags are currently stored as server metadata. Later Nova versions allow
// to store tags in a respective field and do a server-side filtering. To avoid incompatibility with older versions
// we will continue making the filtering clientside.
func (ex *Executor) getMachineByName(_ context.Context, machineName string) (*servers.Server, error) {
	var (
		searchClusterName string
		searchNodeRole    string
	)

	for key := range ex.Config.Spec.Tags {
		if strings.Contains(key, cloudprovider.ServerTagClusterPrefix) {
			searchClusterName = key
		} else if strings.Contains(key, cloudprovider.ServerTagRolePrefix) {
			searchNodeRole = key
		}
	}

	// TODO(KA) better tag handling ? Should it return nil if no tags are found (should be blocked by validation)
	if searchClusterName == "" || searchNodeRole == "" {
		return nil, nil
	}

	listedServers, err := ex.Compute.ListServers(&servers.ListOpts{
		Name: machineName,
	})
	if err != nil {
		return nil, err
	}

	matchingServers := []servers.Server{}
	for _, server := range listedServers {
		if server.Name == machineName {
			if _, ok := server.Metadata[searchClusterName]; ok {
				if _, ok2 := server.Metadata[searchNodeRole]; ok2 {
					matchingServers = append(matchingServers, server)
				}
			}
		}
	}

	if len(matchingServers) > 1 {
		return nil, fmt.Errorf("failed to find machine by name %q: %w", machineName, ErrMultipleFound)
	} else if len(matchingServers) == 0 {
		return nil, fmt.Errorf("failed to find machine by name %q: %w", machineName, ErrNotFound)
	}

	return &matchingServers[0], nil
}

func (ex *Executor) GetMachineStatus(ctx context.Context, machineName string) (string, error) {
	server, err := ex.getMachineByName(ctx, machineName)
	if err != nil {
		return "", err
	}

	return EncodeProviderID(ex.Config.Spec.Region, server.ID), nil
}

func (ex *Executor) ListMachines(_ context.Context) (map[string]string, error) {
	searchClusterName := ""
	searchNodeRole := ""

	for key := range ex.Config.Spec.Tags {
		if strings.Contains(key, cloudprovider.ServerTagClusterPrefix) {
			searchClusterName = key
		} else if strings.Contains(key, cloudprovider.ServerTagRolePrefix) {
			searchNodeRole = key
		}
	}

	// TODO(KA) better tag handling ? Should it return nil if no tags are found (should be blocked by validation)
	if searchClusterName == "" || searchNodeRole == "" {
		return nil, nil
	}

	servers, err := ex.Compute.ListServers(&servers.ListOpts{})
	if err != nil {
		return nil, err
	}

	result := map[string]string{}
	for _, server := range servers {
		if _, ok := server.Metadata[searchClusterName]; ok {
			if _, ok2 := server.Metadata[searchNodeRole]; ok2 {
				providerID := EncodeProviderID(ex.Config.Spec.Region, server.ID)
				result[providerID] = server.Name
			}
		}
	}

	return result, nil
}
