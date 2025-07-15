// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/client"
)

// ITResourceTagKey is specifically used for integration test
// primarily to avoid orphan collection of resources when the control cluster is non-seed cluster
var ITResourceTagKey = "kubernetes.io-role-integration-test"

// ResourcesTrackerImpl type keeps a note of resources which are
// initialized in MCM IT suite and are used in provider IT
type ResourcesTrackerImpl struct {
	MachineClass *v1alpha1.MachineClass
	SecretData   map[string][]byte
}

// InitializeResourcesTracker initializes the type ResourcesTrackerImpl variable and tries
// to delete the orphan resources present before the actual IT runs.
// create a cleanup function to delete the list of orphan resources.
// 1. get list of orphan resources.
// 2. Mark them for deletion and call cleanup.
// 3. Print the orphan resources which got error in deletion.
func (r *ResourcesTrackerImpl) InitializeResourcesTracker(machineClass *v1alpha1.MachineClass, secretData map[string][]byte, _ string) error {
	r.MachineClass = machineClass
	r.SecretData = secretData

	initialVMs, initialNICs, initialVolumes, initialMachines, err := r.probeResources(context.Background())
	if err != nil {
		fmt.Printf("Error in initial probe of orphaned resources: %s", err.Error())
		return err
	}

	delErrOrphanVMs, delErrOrphanVolumes, delErrOrphanNICs := cleanOrphanResources(context.Background(), initialVMs, initialVolumes, initialNICs, r.MachineClass, r.SecretData)
	if len(delErrOrphanVMs) != 0 || len(delErrOrphanVolumes) != 0 || len(initialMachines) != 0 || len(delErrOrphanNICs) != 0 {
		err = fmt.Errorf("error in cleaning the following orphan resources. Clean them up before proceeding with the test.\nvirtual machines: %v\ndisks: %v\nmcm machines: %v\nnics: %v", delErrOrphanVMs, delErrOrphanVolumes, initialMachines, delErrOrphanNICs)
		return err
	}
	return nil
}

// probeResources will look for resources currently available and returns them
func (r *ResourcesTrackerImpl) probeResources(ctx context.Context) ([]string, []string, []string, []string, error) {
	factory, err := client.NewFactoryFromSecretData(ctx, r.SecretData)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	availMachines, err := getMachines(r.MachineClass, factory)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to find available machines: %s", err)
	}

	orphanVMs, err := getOrphanedInstances(ctx, factory)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to find orphaned instances: %s", err)
	}

	orphanNICs, err := getOrphanedNICs(ctx, factory)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to find available ports: %s", err)
	}

	orphanDisks, err := getOrphanedDisks(ctx, factory)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to find available disks: %s", err)
	}

	return orphanVMs, orphanNICs, orphanDisks, availMachines, nil
}

// IsOrphanedResourcesAvailable checks whether there are any orphaned resources left.
// If yes, then prints them and returns true. If not, then returns false
func (r *ResourcesTrackerImpl) IsOrphanedResourcesAvailable() bool {
	afterTestExecutionVMs, afterTestExecutionNICs, afterTestExecutionDisks, afterTestExecutionAvailmachines, err := r.probeResources(context.Background())
	if err != nil {
		fmt.Printf("Error probing orphaned resources: %s", err.Error())
		return true
	}

	if len(afterTestExecutionVMs) != 0 || len(afterTestExecutionAvailmachines) != 0 || len(afterTestExecutionNICs) != 0 || len(afterTestExecutionDisks) != 0 {
		fmt.Printf("The following resources are orphans ... waiting for them to be deleted \n")
		fmt.Printf("Virtual Machines: %v\nNICs: %v\nMCM Machines: %v\n", afterTestExecutionVMs, afterTestExecutionNICs, afterTestExecutionAvailmachines)
		return true
	}

	return false
}
