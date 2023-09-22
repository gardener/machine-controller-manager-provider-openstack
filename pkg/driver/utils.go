// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"errors"
	"fmt"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack/install"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/client"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/driver/executor"
)

const (
	openstackProvider = "OpenStack"
)

// Decoder is a decoder for a scheme containing the mcm-openstack APIs.
var Decoder = serializer.NewCodecFactory(install.Install(runtime.NewScheme())).UniversalDecoder()

// DecodeProviderSpec can decode raw to a MachineProviderConfig.
func DecodeProviderSpec(decoder runtime.Decoder, raw runtime.RawExtension) (*openstack.MachineProviderConfig, error) {
	json, err := raw.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to decode provider spec: %v", err)
	}

	cfg := &openstack.MachineProviderConfig{}
	_, _, err = decoder.Decode(json, nil, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to decode provider spec: %v", err)
	}

	return cfg, nil
}

func (p *OpenstackDriver) decodeProviderSpec(raw runtime.RawExtension) (*openstack.MachineProviderConfig, error) {
	return DecodeProviderSpec(p.decoder, raw)
}

func mapErrorToCode(err error) codes.Code {
	if errors.Is(err, executor.ErrNoValidHost) {
		return codes.ResourceExhausted
	}

	if errors.Is(err, executor.ErrNotFound) {
		return codes.NotFound
	}

	if errors.Is(err, executor.ErrMultipleFound) {
		return codes.OutOfRange
	}

	if client.IsUnauthenticated(err) {
		return codes.Unauthenticated
	}

	if client.IsUnauthorized(err) {
		return codes.PermissionDenied
	}

	return codes.Internal
}
