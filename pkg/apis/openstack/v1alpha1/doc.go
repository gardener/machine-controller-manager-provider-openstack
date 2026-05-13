// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// +k8s:deepcopy-gen=package
// +k8s:conversion-gen=github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack
// +k8s:defaulter-gen=TypeMeta
// +k8s:openapi-gen=true
// +groupName=openstack.machine.gardener.cloud

//go:generate crd-ref-docs --source-path=. --config=../../../../hack/api-reference/api.yaml --renderer=markdown --templates-dir=$GARDENER_HACK_DIR/api-reference/template --log-level=ERROR --output-path=../../../../hack/api-reference/api.md

// Package v1alpha1 is a version of the API.
package v1alpha1
