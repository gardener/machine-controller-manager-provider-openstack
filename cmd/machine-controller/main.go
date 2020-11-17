/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This file was copied and modified from the kubernetes/kubernetes project
https://github.com/kubernetes/kubernetes/release-1.8/cmd/kube-controller-manager/controller_manager.go

Modifications Copyright (c) 2017 SAP SE or an SAP affiliate company. All rights reserved.
*/

package main

import (
	"fmt"
	"os"

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

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack/install"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/provider"
)

func main() {

	s := options.NewMCServer()
	s.AddFlags(pflag.CommandLine)

	flag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	scheme := runtime.NewScheme()
	if err := install.AddToScheme(scheme); err != nil {
		fatal(err, "failed to install scheme")
	}

	provider := provider.NewProvider(serializer.NewCodecFactory(scheme).UniversalDecoder())

	if err := app.Run(s, provider); err != nil {
		fatal(err, "failed to run application")
	}
}


func fatal(err error, msg string) {
	fmt.Fprintf(os.Stderr, "[FATAL] %s: %v", msg, err)
	os.Exit(1)
}