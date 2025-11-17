// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"

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

// CreateMachineResult represents the result of a CreateMachine call (internal IP addresses + provider ID of VM).
type CreateMachineResult struct {
	ProviderID  string
	InternalIPs []string
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

// getServerIPs assumes the server has exactly one network interface
// and extracts its internal IP addresses.
func getServerIPs(server *servers.Server) ([]string, error) {
	ips := make([]string, 0)

	if len(server.Addresses) != 1 {
		return nil, fmt.Errorf("expected 1 network, but found %d", len(server.Addresses))
	}

	// Format of the addresses field: https://docs.openstack.org/api-ref/compute/#list-servers-detailed.
	for _, networkAddresses := range server.Addresses {
		addrList, ok := networkAddresses.([]any)
		if !ok {
			return nil, fmt.Errorf("could not assert network addresses to slice")
		}

		// Iterate through the addresses (may be IPv4, IPv6).
		for _, addrData := range addrList {
			addressMap, ok := addrData.(map[string]any)
			if !ok {
				continue
			}

			if ipAddress, ok := addressMap["addr"].(string); ok {
				ips = append(ips, ipAddress)
			}
		}
	}

	return ips, nil
}

// CreateMachine creates a new OpenStack server instance and waits until it reports "ACTIVE".
// If there is an error during the build process, or if the building phase timeouts, it will delete any artifacts created.
func (ex *Executor) CreateMachine(ctx context.Context, machineName string, userData []byte) (*CreateMachineResult, error) {
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
		return nil, err
	} else {
		// clean-up function when creation fails in an intermediate step
		serverNetworks, err := ex.resolveServerNetworks(ctx, machineName)
		if err != nil {
			return nil, deleteOnFail(fmt.Errorf("failed to resolve server [Name=%q] networks: %w", machineName, err))
		}

		server, err = ex.deployServer(ctx, machineName, userData, serverNetworks)
		if err != nil {
			return nil, deleteOnFail(fmt.Errorf("failed to deploy server [Name=%q]: %w", machineName, err))
		}
	}

	// The server information when status is ACTIVE has addresses field populated
	var activeServer *servers.Server
	activeServer, err = ex.waitForServerStatus(ctx,
		server.ID,
		[]string{client.ServerStatusBuild},
		[]string{client.ServerStatusActive}, 1200)
	if err != nil {
		return nil, deleteOnFail(fmt.Errorf("error waiting for server [ID=%q] to reach target status: %w", server.ID, err))
	}

	if err := ex.patchServerPortsForPodNetwork(ctx, activeServer.ID); err != nil {
		return nil, deleteOnFail(fmt.Errorf("failed to patch server [ID=%q] ports: %s", server.ID, err))
	}

	var internalIPs []string
	internalIPs, err = getServerIPs(activeServer)
	if err != nil {
		klog.Infof("failed to extract internal IPs [ID=%q] ports: %s", activeServer.ID, err)
	}

	return &CreateMachineResult{
		ProviderID:  encodeProviderID(ex.Config.Spec.Region, activeServer.ID),
		InternalIPs: internalIPs,
	}, nil
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
		if _, err := ex.Network.GetSubnet(ctx, *subnetID); err != nil {
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

	if !isEmptyString(ptr.To(networkID)) {
		klog.V(3).Infof("deploying in network [ID=%q]", networkID)
		serverNetworks = append(serverNetworks, servers.Network{UUID: ex.Config.Spec.NetworkID})
		return serverNetworks, nil
	}

	for _, network := range networks {
		var (
			resolvedNetworkID string
			err               error
		)
		if isEmptyString(ptr.To(network.Id)) {
			resolvedNetworkID, err = ex.Network.NetworkIDFromName(ctx, network.Name)
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

// waitForServerStatus blocks until the server with the specified ID reaches one of the target status and returns the server after reaching this status.
// waitForServerStatus will fail if an error occurs, the operation it timeouts after the specified time, or the server status is not in the pending list.
func (ex *Executor) waitForServerStatus(ctx context.Context, serverID string, pending []string, target []string, secs int) (*servers.Server, error) {
	var server *servers.Server
	return server, wait.PollUntilContextTimeout(
		ctx,
		10*time.Second,
		time.Duration(secs)*time.Second,
		true,
		func(_ context.Context) (done bool, err error) {
			current, err := ex.Compute.GetServer(ctx, serverID)
			if err != nil {
				if client.IsNotFoundError(err) && strSliceContains(target, client.ServerStatusDeleted) {
					return true, nil
				}
				return false, err
			}

			klog.V(5).Infof("waiting for server [ID=%q] and current status %v, to reach status %v.", serverID, current.Status, target)
			if strSliceContains(target, current.Status) {
				server = current
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
func (ex *Executor) deployServer(ctx context.Context, machineName string, userData []byte, nws []servers.Network) (*servers.Server, error) {
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
		imageRef       string
		err            error
		serverHintOpts servers.SchedulerHintOpts
	)

	// use imageID if provided, otherwise try to resolve the imageName to an imageID
	if imageID != "" {
		imageRef = imageID
	} else {
		image, err := ex.Compute.ImageIDFromName(ctx, imageName)
		if err != nil {
			return nil, fmt.Errorf("error resolving image ID from image name %q: %v", imageName, err)
		}
		imageRef = image.ID
	}
	flavorRef, err := ex.Compute.FlavorIDFromName(ctx, flavorName)
	if err != nil {
		return nil, fmt.Errorf("error resolving flavor ID from flavor name %q: %v", imageName, err)
	}

	createOpts := &servers.CreateOpts{
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

	if ex.Config.Spec.ServerGroupID != nil {
		serverHintOpts = servers.SchedulerHintOpts{
			Group: *ex.Config.Spec.ServerGroupID,
		}
	}

	// If a custom block_device (root disk size is provided) we need to boot from volume
	if rootDiskSize > 0 {
		createOpts, err = ex.addBlockDeviceOpts(ctx, machineName, imageRef, createOpts)
		if err != nil {
			return nil, fmt.Errorf("error adding block device opts %w", err)
		}
	}

	createOptsBuilder := &keypairs.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		KeyName:           keyName,
	}

	return ex.Compute.CreateServer(ctx, createOptsBuilder, serverHintOpts)
}

func (ex *Executor) addBlockDeviceOpts(ctx context.Context, machineName,
	imageID string, createOpts *servers.CreateOpts) (*servers.CreateOpts, error) {
	createOpts.BlockDevice = make([]servers.BlockDevice, 1)

	if ex.Config.Spec.RootDiskType != nil {
		volumeID, err := ex.ensureVolume(ctx, machineName, imageID, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to ensure volume [Name=%q]: %s", machineName, err)
		}

		createOpts.BlockDevice[0] = servers.BlockDevice{
			UUID:                volumeID,
			VolumeSize:          ex.Config.Spec.RootDiskSize,
			BootIndex:           0,
			DeleteOnTermination: false,
			SourceType:          "volume",
			DestinationType:     "volume",
		}
	} else {
		createOpts.BlockDevice[0] = servers.BlockDevice{
			UUID:                imageID,
			VolumeSize:          ex.Config.Spec.RootDiskSize,
			BootIndex:           0,
			DeleteOnTermination: true,
			SourceType:          "image",
			DestinationType:     "volume",
		}
	}

	klog.V(3).Infof("[DEBUG] Block Device Options: %+v", createOpts.BlockDevice[0])

	return createOpts, nil
}

func (ex *Executor) ensureVolume(ctx context.Context, name, imageID string,
	hintOpts volumes.SchedulerHintOptsBuilder) (string, error) {
	var (
		volumeID string
		err      error
	)

	volumeID, err = ex.Storage.VolumeIDFromName(ctx, name)
	if err != nil && !client.IsNotFoundError(err) {
		return "", err
	}

	if client.IsNotFoundError(err) {
		volume, err := ex.Storage.CreateVolume(ctx, volumes.CreateOpts{
			Name:             name,
			VolumeType:       *ex.Config.Spec.RootDiskType,
			Size:             ex.Config.Spec.RootDiskSize,
			ImageID:          imageID,
			AvailabilityZone: ex.Config.Spec.AvailabilityZone,
			Metadata:         ex.Config.Spec.Tags,
		}, hintOpts)
		if err != nil {
			return "", fmt.Errorf("failed to created volume [Name=%s]: %v", name, err)
		}
		volumeID = volume.ID
	}

	pendingStatuses := []string{client.VolumeStatusCreating, client.VolumeStatusDownloading}
	targetStatuses := []string{client.VolumeStatusAvailable}
	if err := ex.waitForVolumeStatus(ctx, volumeID, pendingStatuses, targetStatuses, 1200); err != nil {
		return "", err
	}

	return volumeID, nil
}

func (ex *Executor) waitForVolumeStatus(ctx context.Context, volumeID string, pending, target []string, secs int) error {
	return wait.PollUntilContextTimeout(
		ctx,
		10*time.Second,
		time.Duration(secs)*time.Second,
		true,
		func(_ context.Context) (done bool, err error) {
			current, err := ex.Storage.GetVolume(ctx, volumeID)
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
func (ex *Executor) patchServerPortsForPodNetwork(ctx context.Context, serverID string) error {
	allPorts, err := ex.Network.ListPorts(ctx, &ports.ListOpts{
		DeviceID: serverID,
	})
	if err != nil {
		return fmt.Errorf("failed to get ports: %v", err)
	}

	if len(allPorts) == 0 {
		return fmt.Errorf("got an empty port list for server %q", serverID)
	}

	podNetworkIDs, err := ex.resolveNetworkIDsForPodNetwork(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve network IDs for the pod network %v", err)
	}

	// coalesce all pod network CIDRs into a single slice.
	podCIDRs := sets.NewString(ex.Config.Spec.PodNetworkCIDRs...)
	if ex.Config.Spec.PodNetworkCidr != "" {
		podCIDRs.Insert(ex.Config.Spec.PodNetworkCidr)
	}

	for _, port := range allPorts {
		// if the port is not part of the networks we care about, continue.
		if !podNetworkIDs.Has(port.NetworkID) {
			continue
		}

		for _, cidr := range podCIDRs.List() {
			if err := func() error {
				for _, pair := range port.AllowedAddressPairs {
					if pair.IPAddress == cidr {
						klog.V(3).Infof("port [ID=%q] already allows pod network CIDR range. Skipping update...", port.ID)
						return nil
					}
				}
				if err := ex.Network.UpdatePort(ctx, port.ID, ports.UpdateOpts{
					AllowedAddressPairs: &[]ports.AddressPair{{IPAddress: cidr}},
				}); err != nil {
					return fmt.Errorf("failed to update allowed address pair for port [ID=%q]: %v", port.ID, err)
				}
				return nil
			}(); err != nil {
				return err
			}
		}
	}
	return nil
}

// resolveNetworkIDsForPodNetwork resolves the networks that accept traffic from the pod CIDR range.
func (ex *Executor) resolveNetworkIDsForPodNetwork(ctx context.Context) (sets.Set[string], error) {
	var (
		networkID     = ex.Config.Spec.NetworkID
		networks      = ex.Config.Spec.Networks
		podNetworkIDs = sets.New[string]()
	)

	if !isEmptyString(ptr.To(networkID)) {
		podNetworkIDs.Insert(networkID)
		return podNetworkIDs, nil
	}

	for _, network := range networks {
		var (
			resolvedNetworkID string
			err               error
		)
		if isEmptyString(ptr.To(network.Id)) {
			resolvedNetworkID, err = ex.Network.NetworkIDFromName(ctx, network.Name)
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

	if !isEmptyString(ptr.To(providerID)) {
		serverID := decodeProviderID(providerID)
		server, err = ex.getMachineByID(ctx, serverID)
	} else {
		server, err = ex.getMachineByName(ctx, machineName)
	}

	if err == nil {
		klog.V(1).Infof("deleting server [Name=%s, ID=%s]", server.Name, server.ID)
		if err := ex.Compute.DeleteServer(ctx, server.ID); err != nil {
			return err
		}

		if _, err = ex.waitForServerStatus(ctx, server.ID, nil, []string{client.ServerStatusDeleted}, 1200); err != nil {
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

func (ex *Executor) getOrCreatePort(ctx context.Context, machineName string) (string, error) {
	var (
		err              error
		securityGroupIDs []string
	)

	portID, err := ex.Network.PortIDFromName(ctx, machineName)
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
		securityGroupID, err := ex.Network.GroupIDFromName(ctx, securityGroup)
		if err != nil {
			return "", err
		}
		securityGroupIDs = append(securityGroupIDs, securityGroupID)
	}

	port, err := ex.Network.CreatePort(ctx, &ports.CreateOpts{
		Name:           machineName,
		NetworkID:      ex.Config.Spec.NetworkID,
		FixedIPs:       []ports.IP{{SubnetID: *ex.Config.Spec.SubnetID}},
		SecurityGroups: &securityGroupIDs,
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
	if err := ex.Network.TagPort(ctx, port.ID, portTags); err != nil {
		return "", err
	}

	klog.V(3).Infof("port [Name=%q] successfully created", port.Name)
	return port.ID, nil
}

func (ex *Executor) deletePort(ctx context.Context, machineName string) error {
	portList, err := ex.Network.ListPorts(ctx, ports.ListOpts{
		Name: machineName,
	})
	if err != nil {
		return fmt.Errorf("error deleting port [Name=%q]: %s", machineName, err)
	}
	if len(portList) == 0 {
		klog.V(2).Infof("port [Name=%q] was not found", machineName)
		return nil
	}

	klog.V(2).Infof("deleting ports for machine [Name=%q]", machineName)
	for _, p := range portList {
		klog.V(2).Infof("deleting port [ID=%q]", p.ID)
		err = ex.Network.DeletePort(ctx, p.ID)
		if err != nil {
			klog.Errorf("failed to delete port [ID=%q]: %s", p.ID, err)
			return err
		}
		klog.V(3).Infof("deleted port [ID=%q]", p.ID)
	}

	return nil
}

func (ex *Executor) deleteVolume(ctx context.Context, machineName string) error {
	volumeID, err := ex.Storage.VolumeIDFromName(ctx, machineName)
	if err != nil {
		if client.IsNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("error deleting [Name=%q]: %s", machineName, err)
	}

	klog.V(2).Infof("deleting volume [Name=%q]", machineName)
	err = ex.Storage.DeleteVolume(ctx, volumeID)
	if err != nil {
		klog.Errorf("failed to delete port [Name=%q]", machineName)
		return err
	}
	return nil
}

// getMachineByProviderID fetches the data for a server based on a provider-encoded ID.
func (ex *Executor) getMachineByID(ctx context.Context, serverID string) (*servers.Server, error) {
	klog.V(2).Infof("finding server with [ID=%q]", serverID)
	server, err := ex.Compute.GetServer(ctx, serverID)
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
func (ex *Executor) getMachineByName(ctx context.Context, machineName string) (*servers.Server, error) {
	searchClusterName, searchNodeRole, ok := findMandatoryTags(ex.Config.Spec.Tags)
	if !ok {
		klog.Warningf("getMachineByName operation can not proceed: cluster/role tags are missing for machine [Name=%q]", machineName)
		return nil, fmt.Errorf("getMachineByName operation can not proceed: cluster/role tags are missing for machine [Name=%q]", machineName)
	}

	listedServers, err := ex.Compute.ListServers(ctx, &servers.ListOpts{
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
func (ex *Executor) listServers(ctx context.Context) ([]servers.Server, error) {
	searchClusterName, searchNodeRole, ok := findMandatoryTags(ex.Config.Spec.Tags)
	if !ok {
		klog.Warningf("list operation can not proceed: cluster/role tags are missing")
		return nil, fmt.Errorf("list operation can not proceed: cluster/role tags are missing")
	}

	allServers, err := ex.Compute.ListServers(ctx, &servers.ListOpts{})
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
	return !isEmptyString(ptr.To(ex.Config.Spec.NetworkID)) && !isEmptyString(ex.Config.Spec.SubnetID)
}
