#!/bin/bash

# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

set -o errexit
set -o nounset
set -o pipefail

# setup virtual GOPATH
source "$GARDENER_HACK_DIR"/vgopath-setup.sh

CODE_GEN_DIR=$(go list -m -f '{{.Dir}}' k8s.io/code-generator)

rm -f $GOPATH/bin/*-gen

bash "${CODE_GEN_DIR}/kube_codegen.sh" \
  deepcopy,defaulter \
  github.com/gardener/machine-controller-manager-provider-openstack/pkg/client \
  github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis \
  github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis \
  "openstack:v1alpha1" \
  --go-header-file "${GARDENER_HACK_DIR}/LICENSE_BOILERPLATE.txt"

bash "${CODE_GEN_DIR}/kube_codegen.sh" \
  conversion \
  github.com/gardener/machine-controller-manager-provider-openstack/pkg/client \
  github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis \
  github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis \
  "openstack:v1alpha1" \
  --extra-peer-dirs=github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack,github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/conversion,k8s.io/apimachinery/pkg/runtime \
  --go-header-file "${GARDENER_HACK_DIR}/LICENSE_BOILERPLATE.txt"
