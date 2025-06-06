// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
//

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/machine-controller-manager-provider-openstack/pkg/client (interfaces: Compute,Network,Storage)
//
// Generated by this command:
//
//	mockgen -copyright_file=../../../hack/LICENSE_HEADER.txt -destination=./mocks.go -package=openstack github.com/gardener/machine-controller-manager-provider-openstack/pkg/client Compute,Network,Storage
//

// Package openstack is a generated GoMock package.
package openstack

import (
	context "context"
	reflect "reflect"

	volumes "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	servers "github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	images "github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	ports "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	subnets "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	gomock "go.uber.org/mock/gomock"
)

// MockCompute is a mock of Compute interface.
type MockCompute struct {
	ctrl     *gomock.Controller
	recorder *MockComputeMockRecorder
	isgomock struct{}
}

// MockComputeMockRecorder is the mock recorder for MockCompute.
type MockComputeMockRecorder struct {
	mock *MockCompute
}

// NewMockCompute creates a new mock instance.
func NewMockCompute(ctrl *gomock.Controller) *MockCompute {
	mock := &MockCompute{ctrl: ctrl}
	mock.recorder = &MockComputeMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCompute) EXPECT() *MockComputeMockRecorder {
	return m.recorder
}

// CreateServer mocks base method.
func (m *MockCompute) CreateServer(ctx context.Context, opts servers.CreateOptsBuilder, hintOpts servers.SchedulerHintOptsBuilder) (*servers.Server, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateServer", ctx, opts, hintOpts)
	ret0, _ := ret[0].(*servers.Server)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateServer indicates an expected call of CreateServer.
func (mr *MockComputeMockRecorder) CreateServer(ctx, opts, hintOpts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateServer", reflect.TypeOf((*MockCompute)(nil).CreateServer), ctx, opts, hintOpts)
}

// DeleteServer mocks base method.
func (m *MockCompute) DeleteServer(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteServer", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteServer indicates an expected call of DeleteServer.
func (mr *MockComputeMockRecorder) DeleteServer(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteServer", reflect.TypeOf((*MockCompute)(nil).DeleteServer), ctx, id)
}

// FlavorIDFromName mocks base method.
func (m *MockCompute) FlavorIDFromName(ctx context.Context, name string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FlavorIDFromName", ctx, name)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FlavorIDFromName indicates an expected call of FlavorIDFromName.
func (mr *MockComputeMockRecorder) FlavorIDFromName(ctx, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FlavorIDFromName", reflect.TypeOf((*MockCompute)(nil).FlavorIDFromName), ctx, name)
}

// GetServer mocks base method.
func (m *MockCompute) GetServer(ctx context.Context, id string) (*servers.Server, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServer", ctx, id)
	ret0, _ := ret[0].(*servers.Server)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetServer indicates an expected call of GetServer.
func (mr *MockComputeMockRecorder) GetServer(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServer", reflect.TypeOf((*MockCompute)(nil).GetServer), ctx, id)
}

// ImageIDFromName mocks base method.
func (m *MockCompute) ImageIDFromName(ctx context.Context, name string) (images.Image, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ImageIDFromName", ctx, name)
	ret0, _ := ret[0].(images.Image)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ImageIDFromName indicates an expected call of ImageIDFromName.
func (mr *MockComputeMockRecorder) ImageIDFromName(ctx, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ImageIDFromName", reflect.TypeOf((*MockCompute)(nil).ImageIDFromName), ctx, name)
}

// ListServers mocks base method.
func (m *MockCompute) ListServers(ctx context.Context, opts servers.ListOptsBuilder) ([]servers.Server, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListServers", ctx, opts)
	ret0, _ := ret[0].([]servers.Server)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListServers indicates an expected call of ListServers.
func (mr *MockComputeMockRecorder) ListServers(ctx, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListServers", reflect.TypeOf((*MockCompute)(nil).ListServers), ctx, opts)
}

// MockNetwork is a mock of Network interface.
type MockNetwork struct {
	ctrl     *gomock.Controller
	recorder *MockNetworkMockRecorder
	isgomock struct{}
}

// MockNetworkMockRecorder is the mock recorder for MockNetwork.
type MockNetworkMockRecorder struct {
	mock *MockNetwork
}

// NewMockNetwork creates a new mock instance.
func NewMockNetwork(ctrl *gomock.Controller) *MockNetwork {
	mock := &MockNetwork{ctrl: ctrl}
	mock.recorder = &MockNetworkMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNetwork) EXPECT() *MockNetworkMockRecorder {
	return m.recorder
}

// CreatePort mocks base method.
func (m *MockNetwork) CreatePort(ctx context.Context, opts ports.CreateOptsBuilder) (*ports.Port, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreatePort", ctx, opts)
	ret0, _ := ret[0].(*ports.Port)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreatePort indicates an expected call of CreatePort.
func (mr *MockNetworkMockRecorder) CreatePort(ctx, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatePort", reflect.TypeOf((*MockNetwork)(nil).CreatePort), ctx, opts)
}

// DeletePort mocks base method.
func (m *MockNetwork) DeletePort(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeletePort", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeletePort indicates an expected call of DeletePort.
func (mr *MockNetworkMockRecorder) DeletePort(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeletePort", reflect.TypeOf((*MockNetwork)(nil).DeletePort), ctx, id)
}

// GetSubnet mocks base method.
func (m *MockNetwork) GetSubnet(ctx context.Context, id string) (*subnets.Subnet, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSubnet", ctx, id)
	ret0, _ := ret[0].(*subnets.Subnet)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSubnet indicates an expected call of GetSubnet.
func (mr *MockNetworkMockRecorder) GetSubnet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSubnet", reflect.TypeOf((*MockNetwork)(nil).GetSubnet), ctx, id)
}

// GroupIDFromName mocks base method.
func (m *MockNetwork) GroupIDFromName(ctx context.Context, name string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GroupIDFromName", ctx, name)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GroupIDFromName indicates an expected call of GroupIDFromName.
func (mr *MockNetworkMockRecorder) GroupIDFromName(ctx, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GroupIDFromName", reflect.TypeOf((*MockNetwork)(nil).GroupIDFromName), ctx, name)
}

// ListPorts mocks base method.
func (m *MockNetwork) ListPorts(ctx context.Context, opts ports.ListOptsBuilder) ([]ports.Port, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListPorts", ctx, opts)
	ret0, _ := ret[0].([]ports.Port)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListPorts indicates an expected call of ListPorts.
func (mr *MockNetworkMockRecorder) ListPorts(ctx, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListPorts", reflect.TypeOf((*MockNetwork)(nil).ListPorts), ctx, opts)
}

// NetworkIDFromName mocks base method.
func (m *MockNetwork) NetworkIDFromName(ctx context.Context, name string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NetworkIDFromName", ctx, name)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NetworkIDFromName indicates an expected call of NetworkIDFromName.
func (mr *MockNetworkMockRecorder) NetworkIDFromName(ctx, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NetworkIDFromName", reflect.TypeOf((*MockNetwork)(nil).NetworkIDFromName), ctx, name)
}

// PortIDFromName mocks base method.
func (m *MockNetwork) PortIDFromName(ctx context.Context, name string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PortIDFromName", ctx, name)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PortIDFromName indicates an expected call of PortIDFromName.
func (mr *MockNetworkMockRecorder) PortIDFromName(ctx, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PortIDFromName", reflect.TypeOf((*MockNetwork)(nil).PortIDFromName), ctx, name)
}

// TagPort mocks base method.
func (m *MockNetwork) TagPort(ctx context.Context, id string, tags []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TagPort", ctx, id, tags)
	ret0, _ := ret[0].(error)
	return ret0
}

// TagPort indicates an expected call of TagPort.
func (mr *MockNetworkMockRecorder) TagPort(ctx, id, tags any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TagPort", reflect.TypeOf((*MockNetwork)(nil).TagPort), ctx, id, tags)
}

// UpdatePort mocks base method.
func (m *MockNetwork) UpdatePort(ctx context.Context, id string, opts ports.UpdateOptsBuilder) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdatePort", ctx, id, opts)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdatePort indicates an expected call of UpdatePort.
func (mr *MockNetworkMockRecorder) UpdatePort(ctx, id, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdatePort", reflect.TypeOf((*MockNetwork)(nil).UpdatePort), ctx, id, opts)
}

// MockStorage is a mock of Storage interface.
type MockStorage struct {
	ctrl     *gomock.Controller
	recorder *MockStorageMockRecorder
	isgomock struct{}
}

// MockStorageMockRecorder is the mock recorder for MockStorage.
type MockStorageMockRecorder struct {
	mock *MockStorage
}

// NewMockStorage creates a new mock instance.
func NewMockStorage(ctrl *gomock.Controller) *MockStorage {
	mock := &MockStorage{ctrl: ctrl}
	mock.recorder = &MockStorageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorage) EXPECT() *MockStorageMockRecorder {
	return m.recorder
}

// CreateVolume mocks base method.
func (m *MockStorage) CreateVolume(ctx context.Context, opts volumes.CreateOptsBuilder, hintOpts volumes.SchedulerHintOptsBuilder) (*volumes.Volume, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVolume", ctx, opts, hintOpts)
	ret0, _ := ret[0].(*volumes.Volume)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVolume indicates an expected call of CreateVolume.
func (mr *MockStorageMockRecorder) CreateVolume(ctx, opts, hintOpts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVolume", reflect.TypeOf((*MockStorage)(nil).CreateVolume), ctx, opts, hintOpts)
}

// DeleteVolume mocks base method.
func (m *MockStorage) DeleteVolume(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVolume", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVolume indicates an expected call of DeleteVolume.
func (mr *MockStorageMockRecorder) DeleteVolume(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVolume", reflect.TypeOf((*MockStorage)(nil).DeleteVolume), ctx, id)
}

// GetVolume mocks base method.
func (m *MockStorage) GetVolume(ctx context.Context, id string) (*volumes.Volume, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVolume", ctx, id)
	ret0, _ := ret[0].(*volumes.Volume)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVolume indicates an expected call of GetVolume.
func (mr *MockStorageMockRecorder) GetVolume(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVolume", reflect.TypeOf((*MockStorage)(nil).GetVolume), ctx, id)
}

// ListVolumes mocks base method.
func (m *MockStorage) ListVolumes(ctx context.Context, opts volumes.ListOptsBuilder) ([]volumes.Volume, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListVolumes", ctx, opts)
	ret0, _ := ret[0].([]volumes.Volume)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListVolumes indicates an expected call of ListVolumes.
func (mr *MockStorageMockRecorder) ListVolumes(ctx, opts any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListVolumes", reflect.TypeOf((*MockStorage)(nil).ListVolumes), ctx, opts)
}

// VolumeIDFromName mocks base method.
func (m *MockStorage) VolumeIDFromName(ctx context.Context, name string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VolumeIDFromName", ctx, name)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// VolumeIDFromName indicates an expected call of VolumeIDFromName.
func (mr *MockStorageMockRecorder) VolumeIDFromName(ctx, name any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VolumeIDFromName", reflect.TypeOf((*MockStorage)(nil).VolumeIDFromName), ctx, name)
}
