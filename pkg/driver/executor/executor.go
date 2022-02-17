// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/bootfromvolume"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/schedulerhints"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"

	api "github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/client"
)

// Executor concretely handles the execution of requests to the machine controller. Executor is responsible
// for communicating with OpenStack services and orchestrates the operations.
type Executor struct {
	Compute client.Compute
	Network client.Network
	Storage client.Storage
	Config  *api.MachineProviderConfig
}

// NewExecutor returns a new instance of Executor.
func NewExecutor(factory *client.Factory, config *api.MachineProviderConfig) (*Executor, error) {
	computeClient, err := factory.Compute(client.WithRegion(config.Spec.Region))
	if err != nil {
		klog.Errorf("failed to create compute client for executor: %v", err)
		return nil, err
	}
	networkClient, err := factory.Network(client.WithRegion(config.Spec.Region))
	if err != nil {
		klog.Errorf("failed to create network client for executor: %v", err)
		return nil, err
	}
	storageClient, err := factory.Storage(client.WithRegion(config.Spec.Region))
	if err != nil {
		klog.Errorf("failed to create storage client for executor: %v", err)
		return nil, err
	}

	ex := &Executor{
		Compute: computeClient,
		Network: networkClient,
		Storage: storageClient,
		Config:  config,
	}
	return ex, nil
}

// CreateMachine creates a new OpenStack server instance and waits until it reports "ACTIVE".
// If there is an error during the build process, or if the building phase timeouts, it will delete any artifacts created.
func (ex *Executor) CreateMachine(ctx context.Context, machineName string, userData []byte) (string, error) {
	var (
		server *servers.Server
		err    error
	)

	deleteOnFail := func(err error) error {
		klog.Infof("attempting to delete server [Name=%q] after unsuccessful create operation with error: %v", machineName, err)
		if errIn := ex.DeleteMachine(ctx, machineName, ""); errIn != nil {
			return fmt.Errorf("error deleting server [Name=%q] after unsuccessful creation attempt: %v. Original error: %w", machineName, errIn, err)
		}
		return err
	}

	server, err = ex.getMachineByName(ctx, machineName)
	if err == nil {
		klog.Infof("found existing server [Name=%q, ID=%q]", machineName, server.ID)
	} else if !errors.Is(err, ErrNotFound) {
		return "", err
	} else {
		// clean-up function when creation fails in an intermediate step
		serverNetworks, err := ex.resolveServerNetworks(ctx, machineName)
		if err != nil {
			return "", deleteOnFail(fmt.Errorf("failed to resolve server [Name=%q] networks: %w", machineName, err))
		}

		server, err = ex.deployServer(machineName, userData, serverNetworks)
		if err != nil {
			return "", deleteOnFail(fmt.Errorf("failed to deploy server [Name=%q]: %w", machineName, err))
		}
	}

	err = ex.waitForServerStatus(server.ID, []string{client.ServerStatusBuild}, []string{client.ServerStatusActive}, 600)
	if err != nil {
		return "", deleteOnFail(fmt.Errorf("error waiting for server [ID=%q] to reach target status: %w", server.ID, err))
	}

	if err := ex.patchServerPortsForPodNetwork(server.ID); err != nil {
		return "", deleteOnFail(fmt.Errorf("failed to patch server [ID=%q] ports: %s", server.ID, err))
	}

	return encodeProviderID(ex.Config.Spec.Region, server.ID), nil
}

// resolveServerNetworks resolves the network configuration for the server.
func (ex *Executor) resolveServerNetworks(ctx context.Context, machineName string) ([]servers.Network, error) {
	var (
		networkID      = ex.Config.Spec.NetworkID
		subnetID       = ex.Config.Spec.SubnetID
		networks       = ex.Config.Spec.Networks
		serverNetworks = make([]servers.Network, 0)
	)

	klog.V(3).Infof("resolving network setup for machine [Name=%q]", machineName)
	// If SubnetID is specified in addition to NetworkID, we have to preallocate a Neutron Port to force the VMs to get IP from the subnet's range.
	if ex.isUserManagedNetwork() {
		// check if the subnet exists
		if _, err := ex.Network.GetSubnet(*subnetID); err != nil {
			return nil, err
		}

		klog.V(3).Infof("deploying machine [Name=%q] in subnet [ID=%q]", machineName, *subnetID)
		portID, err := ex.getOrCreatePort(ctx, machineName)
		if err != nil {
			return nil, err
		}

		serverNetworks = append(serverNetworks, servers.Network{UUID: ex.Config.Spec.NetworkID, Port: portID})
		return serverNetworks, nil
	}

	if !isEmptyString(pointer.StringPtr(networkID)) {
		klog.V(3).Infof("deploying in network [ID=%q]", networkID)
		serverNetworks = append(serverNetworks, servers.Network{UUID: ex.Config.Spec.NetworkID})
		return serverNetworks, nil
	}

	for _, network := range networks {
		var (
			resolvedNetworkID string
			err               error
		)
		if isEmptyString(pointer.StringPtr(network.Id)) {
			resolvedNetworkID, err = ex.Network.NetworkIDFromName(network.Name)
			if err != nil {
				return nil, err
			}
		} else {
			resolvedNetworkID = network.Id
		}
		serverNetworks = append(serverNetworks, servers.Network{UUID: resolvedNetworkID})
	}
	return serverNetworks, nil
}

// waitForServerStatus blocks until the server with the specified ID reaches one of the target status.
// waitForServerStatus will fail if an error occurs, the operation it timeouts after the specified time, or the server status is not in the pending list.
func (ex *Executor) waitForServerStatus(serverID string, pending []string, target []string, secs int) error {
	return wait.Poll(time.Second, time.Duration(secs)*time.Second, func() (done bool, err error) {
		current, err := ex.Compute.GetServer(serverID)
		if err != nil {
			if client.IsNotFoundError(err) && strSliceContains(target, client.ServerStatusDeleted) {
				return true, nil
			}
			return false, err
		}

		klog.V(5).Infof("waiting for server [ID=%q] and current status %v, to reach status %v.", serverID, current.Status, target)
		if strSliceContains(target, current.Status) {
			return true, nil
		}

		// if there is no pending statuses defined or current status is in the pending list, then continue polling
		if len(pending) == 0 || strSliceContains(pending, current.Status) {
			return false, nil
		}

		retErr := fmt.Errorf("server [ID=%q] reached unexpected status %q", serverID, current.Status)
		if current.Status == client.ServerStatusError {
			retErr = fmt.Errorf("%s, fault: %+v", retErr, current.Fault)
		}

		return false, retErr
	})
}

// deployServer handles creating the server instance.
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

	// If a custom block_device (root disk size is provided) we need to boot from volume
	if rootDiskSize > 0 {
		return ex.bootFromVolume(machineName, imageID, createOpts)
	}

	return ex.Compute.CreateServer(createOpts)
}

func (ex *Executor) bootFromVolume(machineName, imageID string, createOpts servers.CreateOptsBuilder) (*servers.Server, error) {
	blockDeviceOpts := make([]bootfromvolume.BlockDevice, 1)

	if ex.Config.Spec.RootDiskType != nil {
		volumeID, err := ex.ensureVolume(machineName, imageID)
		if err != nil {
			return nil, fmt.Errorf("failed to ensure volume [Name=%q]: %s", machineName, err)
		}

		blockDeviceOpts[0] = bootfromvolume.BlockDevice{
			UUID:                volumeID,
			VolumeSize:          ex.Config.Spec.RootDiskSize,
			BootIndex:           0,
			DeleteOnTermination: true,
			SourceType:          "volume",
			DestinationType:     "volume",
		}
	} else {
		blockDeviceOpts[0] = bootfromvolume.BlockDevice{
			UUID:                imageID,
			VolumeSize:          ex.Config.Spec.RootDiskSize,
			BootIndex:           0,
			DeleteOnTermination: true,
			SourceType:          "image",
			DestinationType:     "volume",
		}
	}

	klog.V(3).Infof("[DEBUG] Block Device Options: %+v", blockDeviceOpts)
	createOpts = &bootfromvolume.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		BlockDevice:       blockDeviceOpts,
	}
	return ex.Compute.BootFromVolume(createOpts)
}

func (ex *Executor) ensureVolume(name, imageID string) (string, error) {
	var (
		volumeID string
		err      error
	)

	volumeID, err = ex.Storage.VolumeIDFromName(name)
	if err != nil && !client.IsNotFoundError(err) {
		return "", err
	}

	if client.IsNotFoundError(err) {
		volume, err := ex.Storage.CreateVolume(volumes.CreateOpts{
			Name:             name,
			VolumeType:       *ex.Config.Spec.RootDiskType,
			Size:             ex.Config.Spec.RootDiskSize,
			ImageID:          imageID,
			AvailabilityZone: ex.Config.Spec.AvailabilityZone,
			Metadata:         ex.Config.Spec.Tags,
		})
		if err != nil {
			return "", fmt.Errorf("failed to created volume [Name=%s]: %v", name, err)
		}
		volumeID = volume.ID
	}

	if err := ex.waitForVolumeStatus(volumeID, []string{client.VolumeStatusCreating}, []string{client.VolumeStatusAvailable}, 600); err != nil {
		return "", err
	}

	return volumeID, nil
}

func (ex *Executor) waitForVolumeStatus(volumeID string, pending, target []string, secs int) error {
	return wait.Poll(time.Second, time.Duration(secs)*time.Second, func() (done bool, err error) {
		current, err := ex.Storage.GetVolume(volumeID)
		if err != nil {
			if client.IsNotFoundError(err) {
				return true, nil
			}
			return false, err
		}

		klog.V(3).Infof("waiting for volume[ID=%q] with current status %v, to reach status %v.", volumeID, current.Status, target)
		if strSliceContains(target, current.Status) {
			return true, nil
		}

		if len(pending) == 0 || strSliceContains(pending, current.Status) {
			return false, nil
		}

		retErr := fmt.Errorf("volume [ID=%q] reached status %q. Retrying until status reaches %q", volumeID, current.Status, target)
		if current.Status == client.VolumeStatusError {
			retErr = fmt.Errorf("%s, fault: %+v", retErr, current.Status)
		}

		return false, retErr
	})
}

// patchServerPortsForPodNetwork updates a server's ports with rules for whitelisting the pod network CIDR.
func (ex *Executor) patchServerPortsForPodNetwork(serverID string) error {
	allPorts, err := ex.Network.ListPorts(&ports.ListOpts{
		DeviceID: serverID,
	})
	if err != nil {
		return fmt.Errorf("failed to get ports: %v", err)
	}

	if len(allPorts) == 0 {
		return fmt.Errorf("got an empty port list for server %q", serverID)
	}

	podNetworkIDs, err := ex.resolveNetworkIDsForPodNetwork()
	if err != nil {
		return fmt.Errorf("failed to resolve network IDs for the pod network %v", err)
	}

	for _, port := range allPorts {
		if podNetworkIDs.Has(port.NetworkID) {
			addressPairFound := false

			for _, pair := range port.AllowedAddressPairs {
				if pair.IPAddress == ex.Config.Spec.PodNetworkCidr {
					klog.V(3).Infof("port [ID=%q] already allows pod network CIDR range. Skipping update...", port.ID)
					addressPairFound = true
					// break inner loop if target found
					break
				}
			}
			// continue outer loop if target found
			if addressPairFound {
				continue
			}

			if err := ex.Network.UpdatePort(port.ID, ports.UpdateOpts{
				AllowedAddressPairs: &[]ports.AddressPair{{IPAddress: ex.Config.Spec.PodNetworkCidr}},
			}); err != nil {
				return fmt.Errorf("failed to update allowed address pair for port [ID=%q]: %v", port.ID, err)
			}
		}
	}
	return nil
}

// resolveNetworkIDsForPodNetwork resolves the networks that accept traffic from the pod CIDR range.
func (ex *Executor) resolveNetworkIDsForPodNetwork() (sets.String, error) {
	var (
		networkID     = ex.Config.Spec.NetworkID
		networks      = ex.Config.Spec.Networks
		podNetworkIDs = sets.NewString()
	)

	if !isEmptyString(pointer.StringPtr(networkID)) {
		podNetworkIDs.Insert(networkID)
		return podNetworkIDs, nil
	}

	for _, network := range networks {
		var (
			resolvedNetworkID string
			err               error
		)
		if isEmptyString(pointer.StringPtr(network.Id)) {
			resolvedNetworkID, err = ex.Network.NetworkIDFromName(network.Name)
			if err != nil {
				return nil, err
			}
		} else {
			resolvedNetworkID = network.Id
		}
		if network.PodNetwork {
			podNetworkIDs.Insert(resolvedNetworkID)
		}
	}
	return podNetworkIDs, nil
}

// DeleteMachine deletes a server based on the supplied machineName. If a providerID is supplied it is used instead of the
// machineName to locate the server.
func (ex *Executor) DeleteMachine(ctx context.Context, machineName, providerID string) error {
	var (
		server *servers.Server
		err    error
	)

	if !isEmptyString(pointer.StringPtr(providerID)) {
		serverID := decodeProviderID(providerID)
		server, err = ex.getMachineByID(ctx, serverID)
	} else {
		server, err = ex.getMachineByName(ctx, machineName)
	}

	if err == nil {
		klog.V(1).Infof("deleting server [Name=%s, ID=%s]", server.Name, server.ID)
		if err := ex.Compute.DeleteServer(server.ID); err != nil {
			return err
		}

		if err = ex.waitForServerStatus(server.ID, nil, []string{client.ServerStatusDeleted}, 300); err != nil {
			return fmt.Errorf("error while waiting for server [ID=%q] to be deleted: %v", server.ID, err)
		}
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}

	if ex.isUserManagedNetwork() {
		err := ex.deletePort(ctx, machineName)
		if err != nil {
			return err
		}
	}

	if ex.Config.Spec.RootDiskType != nil {
		return ex.deleteVolume(ctx, machineName)
	}

	return nil
}

func (ex *Executor) getOrCreatePort(_ context.Context, machineName string) (string, error) {
	var (
		err              error
		securityGroupIDs []string
	)

	portID, err := ex.Network.PortIDFromName(machineName)
	if err == nil {
		klog.V(2).Infof("found port [Name=%q, ID=%q]... skipping creation", machineName, portID)
		return portID, nil
	}

	if !client.IsNotFoundError(err) {
		klog.V(5).Infof("error fetching port [Name=%q]: %s", machineName, err)
		return "", fmt.Errorf("error fetching port [Name=%q]: %s", machineName, err)
	}

	klog.V(5).Infof("port [Name=%q] does not exist", machineName)
	klog.V(3).Infof("creating port [Name=%q]... ", machineName)

	for _, securityGroup := range ex.Config.Spec.SecurityGroups {
		securityGroupID, err := ex.Network.GroupIDFromName(securityGroup)
		if err != nil {
			return "", err
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
		return "", err
	}

	searchClusterName, searchNodeRole, ok := findMandatoryTags(ex.Config.Spec.Tags)
	if !ok {
		klog.Warningf("operation can not proceed: cluster/role tags are missing")
		return "", fmt.Errorf("operation can not proceed: cluster/role tags are missing")
	}

	portTags := []string{searchClusterName, searchNodeRole}
	if err := ex.Network.TagPort(port.ID, portTags); err != nil {
		return "", err
	}

	klog.V(3).Infof("port [Name=%q] successfully created", port.Name)
	return port.ID, nil
}

func (ex *Executor) deletePort(_ context.Context, machineName string) error {
	portID, err := ex.Network.PortIDFromName(machineName)
	if err != nil {
		if client.IsNotFoundError(err) {
			klog.V(3).Infof("port [Name=%q] was not found", machineName)
			return nil
		}
		return fmt.Errorf("error deleting port [Name=%q]: %s", machineName, err)
	}

	klog.V(2).Infof("deleting port [Name=%q]", machineName)
	err = ex.Network.DeletePort(portID)
	if err != nil {
		klog.Errorf("failed to delete port [Name=%q]: %s", machineName, err)
		return err
	}

	klog.V(3).Infof("deleted port [Name=%q]", machineName)
	return nil
}

func (ex *Executor) deleteVolume(_ context.Context, machineName string) error {
	volumeID, err := ex.Storage.VolumeIDFromName(machineName)
	if err != nil {
		if client.IsNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("error deleting [Name=%q]: %s", machineName, err)
	}

	klog.V(2).Infof("deleting volume [Name=%q]", machineName)
	err = ex.Storage.DeleteVolume(volumeID)
	if err != nil {
		klog.Errorf("failed to delete port [Name=%q]", machineName)
		return err
	}
	return nil
}

// getMachineByProviderID fetches the data for a server based on a provider-encoded ID.
func (ex *Executor) getMachineByID(_ context.Context, serverID string) (*servers.Server, error) {
	klog.V(2).Infof("finding server with [ID=%q]", serverID)
	server, err := ex.Compute.GetServer(serverID)
	if err != nil {
		klog.V(2).Infof("error finding server [ID=%q]: %v", serverID, err)
		if client.IsNotFoundError(err) {
			// normalize errors by wrapping not found error
			return nil, fmt.Errorf("could not find server [ID=%q]: %w", serverID, ErrNotFound)
		}
		return nil, err
	}

	searchClusterName, searchNodeRole, ok := findMandatoryTags(ex.Config.Spec.Tags)
	if !ok {
		klog.Warningf("operation can not proceed: cluster/role tags are missing")
		return nil, fmt.Errorf("operation can not proceed: cluster/role tags are missing")
	}

	if _, nameOk := server.Metadata[searchClusterName]; nameOk {
		if _, roleOk := server.Metadata[searchNodeRole]; roleOk {
			return server, nil
		}
	}

	klog.Warningf("server [ID=%q] found, but cluster/role tags are missing/not matching", serverID)
	return nil, fmt.Errorf("could not find server [ID=%q]: %w", serverID, ErrNotFound)
}

// getMachineByName returns a server that matches the following criteria:
// a) has the same name as machineName
// b) has the cluster and role tags as set in the machineClass
// The current approach is weak because the tags are currently stored as server metadata. Later Nova versions allow
// to store tags in a respective field and do a server-side filtering. To avoid incompatibility with older versions
// we will continue making the filtering clientside.
func (ex *Executor) getMachineByName(_ context.Context, machineName string) (*servers.Server, error) {
	searchClusterName, searchNodeRole, ok := findMandatoryTags(ex.Config.Spec.Tags)
	if !ok {
		klog.Warningf("getMachineByName operation can not proceed: cluster/role tags are missing for machine [Name=%q]", machineName)
		return nil, fmt.Errorf("getMachineByName operation can not proceed: cluster/role tags are missing for machine [Name=%q]", machineName)
	}

	listedServers, err := ex.Compute.ListServers(&servers.ListOpts{
		Name: machineName,
	})
	if err != nil {
		return nil, err
	}

	var matchingServers []servers.Server
	for _, server := range listedServers {
		if server.Name == machineName {
			if _, nameOk := server.Metadata[searchClusterName]; nameOk {
				if _, roleOk := server.Metadata[searchNodeRole]; roleOk {
					matchingServers = append(matchingServers, server)
				}
			}
		}
	}

	if len(matchingServers) > 1 {
		return nil, fmt.Errorf("failed to find server [Name=%q]: %w", machineName, ErrMultipleFound)
	} else if len(matchingServers) == 0 {
		return nil, fmt.Errorf("failed to find server [Name=%q]: %w", machineName, ErrNotFound)
	}

	return &matchingServers[0], nil
}

// ListMachines lists returns a map from the server's encoded provider ID to the server name.
func (ex *Executor) ListMachines(ctx context.Context) (map[string]string, error) {
	allServers, err := ex.listServers(ctx)
	if err != nil {
		return nil, err
	}

	result := map[string]string{}
	for _, server := range allServers {
		providerID := encodeProviderID(ex.Config.Spec.Region, server.ID)
		result[providerID] = server.Name
	}

	return result, nil
}

// ListServers lists all servers with the appropriate tags.
func (ex *Executor) listServers(_ context.Context) ([]servers.Server, error) {
	searchClusterName, searchNodeRole, ok := findMandatoryTags(ex.Config.Spec.Tags)
	if !ok {
		klog.Warningf("list operation can not proceed: cluster/role tags are missing")
		return nil, fmt.Errorf("list operation can not proceed: cluster/role tags are missing")
	}

	allServers, err := ex.Compute.ListServers(&servers.ListOpts{})
	if err != nil {
		return nil, err
	}

	var result []servers.Server
	for _, server := range allServers {
		if _, nameOk := server.Metadata[searchClusterName]; nameOk {
			if _, roleOk := server.Metadata[searchNodeRole]; roleOk {
				result = append(result, server)
			}
		}
	}

	return result, nil
}

// isUserManagedNetwork returns true if the port used by the machine will be created and managed by MCM.
func (ex *Executor) isUserManagedNetwork() bool {
	return !isEmptyString(pointer.StringPtr(ex.Config.Spec.NetworkID)) && !isEmptyString(ex.Config.Spec.SubnetID)
}
