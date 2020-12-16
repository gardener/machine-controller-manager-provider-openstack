#!/bin/bash

# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

rm -f $GOPATH/bin/*-gen

PROJECT_ROOT=$(dirname $0)/..

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter \
  github.com/gardener/machine-controller-manager-provider-openstack/pkg/client \
  github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis \
  github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis \
  "openstack:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/vendor/github.com/gardener/gardener/hack/LICENSE_BOILERPLATE.txt"

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  conversion \
  github.com/gardener/machine-controller-manager-provider-openstack/pkg/client \
  github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis \
  github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis \
  "openstack:v1alpha1" \
  --extra-peer-dirs=github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack,github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/conversion,k8s.io/apimachinery/pkg/runtime \
  --go-header-file "${PROJECT_ROOT}/vendor/github.com/gardener/gardener/hack/LICENSE_BOILERPLATE.txt"
