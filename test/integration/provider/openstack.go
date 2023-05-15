// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/client"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/driver"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/driver/executor"
)

func getOrphanedInstances(factory *client.Factory) ([]string, error) {
	compute, err := factory.Compute()
	if err != nil {
		return nil, err
	}

	instances, err := compute.ListServers(&servers.ListOpts{})
	if err != nil {
		return nil, err
	}

	var orphans []string
	for _, instance := range instances {
		if _, ok := instance.Metadata[ITResourceTagKey]; ok {
			orphans = append(orphans, instance.ID)
		}
	}
	return orphans, nil
}

func getMachines(machineClass *v1alpha1.MachineClass, factory *client.Factory) ([]string, error) {
	providerConfig, err := driver.DecodeProviderSpec(driver.Decoder, machineClass.ProviderSpec)
	if err != nil {
		return nil, err
	}
	ex, err := executor.NewExecutor(factory, providerConfig)
	if err != nil {
		return nil, err
	}
	machineList, err := ex.ListMachines(context.Background())
	if err != nil {
		return nil, err
	}

	machines := []string{}
	fmt.Printf("\nAvailable Machines: ")
	for _, name := range machineList {
		fmt.Printf("%s,", name)
		machines = append(machines, name)
	}
	fmt.Println("")
	return machines, nil
}

func getOrphanedNICs(factory *client.Factory) ([]string, error) {
	network, err := factory.Network()
	if err != nil {
		return nil, err
	}

	ports, err := network.ListPorts(&ports.ListOpts{
		Tags: ITResourceTagKey,
	})
	if err != nil {
		return nil, err
	}
	var orphans []string
	for _, port := range ports {
		orphans = append(orphans, port.ID)
	}
	return orphans, nil
}

func getOrphanedDisks(factory *client.Factory) ([]string, error) {
	storage, err := factory.Storage()
	if err != nil {
		return nil, err
	}

	vols, err := storage.ListVolumes(volumes.ListOpts{})
	if err != nil {
		return nil, err
	}

	var orphans []string
	for _, v := range vols {
		if _, ok := v.Metadata[ITResourceTagKey]; !ok {
			continue
		}
		orphans = append(orphans, v.ID)
	}
	return orphans, nil
}

func cleanOrphanResources(orphanVms []string, orphanVolumes []string, orphanNICs []string, machineClass *v1alpha1.MachineClass, secretData map[string][]byte) (delErrOrphanVms []string, delErrOrphanVolumes []string, delErrOrphanNICs []string) {
	factory, err := client.NewFactoryFromSecretData(secretData)
	if err != nil {
		fmt.Printf("failed to create Openstack client: %v", err)
		if len(orphanVms) != 0 {
			delErrOrphanVms = orphanVms
		}
		if len(orphanNICs) != 0 {
			delErrOrphanNICs = orphanNICs
		}
		if len(orphanVolumes) != 0 {
			delErrOrphanVolumes = orphanVolumes
		}
		return
	}

	if len(orphanVms) != 0 {
		compute, err := factory.Compute()
		if err == nil {
			for _, instanceID := range orphanVms {
				if err := compute.DeleteServer(instanceID); err != nil {
					fmt.Printf("failed to delete instance %v: %v", instanceID, err)
					delErrOrphanVms = append(delErrOrphanVms, instanceID)
				}
			}
		} else {
			delErrOrphanVms = orphanVms
		}
	}

	if len(orphanNICs) != 0 {
		network, err := factory.Network()
		if err == nil {
			for _, portID := range orphanNICs {
				if err := network.DeletePort(portID); err != nil {
					fmt.Printf("failed to delete port %v: %v", portID, err)
					delErrOrphanNICs = append(delErrOrphanNICs, portID)
				}
			}
		} else {
			delErrOrphanNICs = orphanNICs
		}
	}

	if len(orphanVolumes) != 0 {
		storage, err := factory.Storage()
		if err == nil {
			for _, volumeID := range orphanVolumes {
				if err := storage.DeleteVolume(volumeID); err != nil {
					fmt.Printf("failed to delete volume %v: %v", volumeID, err)
					delErrOrphanNICs = append(delErrOrphanVolumes, volumeID)
				}
			}
		} else {
			delErrOrphanVolumes = orphanVolumes
		}
	}

	return
}
