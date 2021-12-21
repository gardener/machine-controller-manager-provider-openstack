// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/client"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/driver"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/driver/executor"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
)

func getOrphanedInstances(machineClass *v1alpha1.MachineClass, factory *client.Factory) ([]string, error) {
	compute, err := factory.Compute()
	if err != nil {
		return nil, err
	}
	instances, err := compute.ListServers(&servers.ListOpts{})

	orphans := []string{}
	for _, instance := range instances {
		if _, ok := instance.Metadata[ITResourceTagKey]; ok {
			if err := compute.DeleteServer(instance.ID); err != nil {
				orphans = append(orphans, instance.ID)
			} else {
				fmt.Printf("deleted orphan VM: %s", instance.Name)
			}
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

func getOrphanedNICs(machineclass *v1alpha1.MachineClass, factory *client.Factory) ([]string, error) {
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
	orphans := []string{}
	for _, port := range ports {
		if err := network.DeletePort(port.ID); err != nil {
			orphans = append(orphans, port.Name)
		} else {
			fmt.Printf("deleted orphan port: %s", port.Name)
		}
	}
	return orphans, nil
}

func getOrphanedDisks(machineclass *v1alpha1.MachineClass, factory *client.Factory) ([]string, error) {
	storage, err := factory.Storage()
	if err != nil {
		return nil, err
	}

	vols, err := storage.ListVolumes(volumes.ListOpts{})
	if err != nil {
		return nil, err
	}

	orphans := []string{}
	for _, v := range vols {
		if _, ok := v.Metadata[ITResourceTagKey]; !ok {
			continue
		}
		if err := storage.DeleteVolume(v.ID); err != nil {
			orphans = append(orphans, v.Name)
		} else {
			fmt.Printf("deleted orphan port: %s", v.Name)
		}
	}
	return orphans, nil
}
