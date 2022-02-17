// SPDX-FileCopyrightText: 2014 The Kubernetes Authors.
// SPDX-FileCopyrightText: modifications 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
//
// This file was copied and modified from the kubernetes/kubernetes project
// https://github.com/kubernetes/kubernetes/release-1.8/cmd/kube-controller-manager/controller_manager.go

package main

import (
	_ "github.com/gardener/machine-controller-manager/pkg/util/client/metrics/prometheus" // for client metric registration
	"github.com/gardener/machine-controller-manager/pkg/util/provider/app"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/app/options"
	_ "github.com/gardener/machine-controller-manager/pkg/util/reflector/prometheus" // for reflector metric registration
	_ "github.com/gardener/machine-controller-manager/pkg/util/workqueue/prometheus" // for workqueue metric registration
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack/install"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/driver"
)

func main() {
	s := options.NewMCServer()
	s.AddFlags(pflag.CommandLine)

	flag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	scheme := runtime.NewScheme()
	if err := install.AddToScheme(scheme); err != nil {
		klog.Fatalf("failed to install scheme: %v", err)
	}

	provider := driver.NewOpenstackDriver(serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder())

	if err := app.Run(s, provider); err != nil {
		klog.Fatalf("failed to run application: %v", err)
	}
}
