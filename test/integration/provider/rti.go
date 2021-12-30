// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/client"
	v1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
)

// ITResourceTagKey is specifically used for integration test
// primarily to avoid orphan collection of resources when the control cluster is
// non seed cluster
var ITResourceTagKey = "kubernetes.io-role-integration-test"

// ResourcesTrackerImpl type keeps a note of resources which are initialized in MCM IT suite and are used in provider IT
type ResourcesTrackerImpl struct {
	MachineClass *v1alpha1.MachineClass
	SecretData   map[string][]byte
}

// InitializeResourcesTracker initializes the type ResourcesTrackerImpl variable and tries
// to delete the orphan resources present before the actual IT runs.
func (r *ResourcesTrackerImpl) InitializeResourcesTracker(machineClass *v1alpha1.MachineClass, secretData map[string][]byte, _ string) error {
	r.MachineClass = machineClass
	r.SecretData = secretData

	initialVMs, initialNICs, initialDisks, initialMachines, err := r.probeResources()
	if err != nil {
		fmt.Printf("Error in initial probe of orphaned resources: %s", err.Error())
		return err
	}

	if len(initialVMs) != 0 || len(initialMachines) != 0 || len(initialNICs) != 0 || len(initialDisks) != 0 {
		err := fmt.Errorf("orphan resources are available. Clean them up before proceeding with the test.\nvirtual machines: %v\nmcm machines: %v\nnics: %v", initialVMs, initialMachines, initialNICs)
		return err
	}
	return nil
}

// probeResources will look for resources currently available and returns them
func (r *ResourcesTrackerImpl) probeResources() ([]string, []string, []string, []string, error) {
	factory, err := client.NewFactoryFromSecretData(r.SecretData)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	availMachines, err := getMachines(r.MachineClass, factory)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to find available machines: %s", err)
	}

	orphanVMs, err := getOrphanedInstances(r.MachineClass, factory)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to find orphaned instances: %s", err)
	}

	orphanNICs, err := getOrphanedNICs(r.MachineClass, factory)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to find available ports: %s", err)
	}

	orphanDisks, err := getOrphanedDisks(r.MachineClass, factory)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to find available disks: %s", err)
	}

	return orphanVMs, orphanNICs, orphanDisks, availMachines, nil
}

// IsOrphanedResourcesAvailable checks whether there are any orphaned resources left.
// If yes, then prints them and returns true. If not, then returns false
func (r *ResourcesTrackerImpl) IsOrphanedResourcesAvailable() bool {
	afterTestExecutionVMs, afterTestExecutionNICs, afterTestExecutionDisks, afterTestExecutionAvailmachines, err := r.probeResources()
	if err != nil {
		fmt.Printf("Error probing orphaned resources: %s", err.Error())
		return true
	}

	if len(afterTestExecutionVMs) != 0 || len(afterTestExecutionAvailmachines) != 0 || len(afterTestExecutionNICs) != 0 || len(afterTestExecutionDisks) != 0 {
		fmt.Printf("Virtual Machines: %v\nNICs: %v\nMCM Machines: %v\n", afterTestExecutionVMs, afterTestExecutionNICs, afterTestExecutionAvailmachines)
		return true
	}

	return false
}
