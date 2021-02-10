// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"
	. "github.com/gardener/machine-controller-manager-provider-openstack/pkg/driver/executor"
	mocks "github.com/gardener/machine-controller-manager-provider-openstack/pkg/mock/openstack"
	client "github.com/gardener/machine-controller-manager-provider-openstack/pkg/openstack"
)

var _ = Describe("Executor", func() {
	const (
		region = "eu-nl-1"
	)
	var (
		ctrl    *gomock.Controller
		factory *mocks.MockFactory
		compute *mocks.MockCompute
		network *mocks.MockNetwork
		tags    map[string]string
		cfg     *openstack.MachineProviderConfig
		ctx     context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		ctrl = gomock.NewController(GinkgoT())
		factory = mocks.NewMockFactory(ctrl)
		compute = mocks.NewMockCompute(ctrl)
		network = mocks.NewMockNetwork(ctrl)

		factory.EXPECT().Compute().AnyTimes().Return(compute, nil)
		factory.EXPECT().Network().AnyTimes().Return(network, nil)

		tags = map[string]string{
			fmt.Sprintf("%sfoo", cloudprovider.ServerTagClusterPrefix): "1",
			fmt.Sprintf("%sfoo", cloudprovider.ServerTagRolePrefix):    "1",
		}

		cfg = &openstack.MachineProviderConfig{
			Spec: openstack.MachineProviderConfigSpec{
				Tags:   tags,
				Region: region,
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Create", func() {
		var (
			machineName = "name"
			imageName   = "image"
			flavorName  = "flavor"
			serverID    = "server"
			networkID   = "networkID"
			portID      = "portID"
			podCidr     = "10.0.0.0/16"
		)
		BeforeEach(func() {
			cfg = &openstack.MachineProviderConfig{
				Spec: openstack.MachineProviderConfigSpec{
					ImageName:      imageName,
					Region:         region,
					FlavorName:     flavorName,
					SecurityGroups: nil,
					Tags:           tags,
					NetworkID:      networkID,
					RootDiskSize:   0,
					PodNetworkCidr: podCidr,
				},
			}

		})

		It("should take the happy path", func() {
			ex := &Executor{
				Compute: compute,
				Network: network,
				Config:  *cfg,
			}

			compute.EXPECT().ImageIDFromName(imageName).Return("imageID", nil)
			compute.EXPECT().FlavorIDFromName(flavorName).Return("flavorID", nil)
			compute.EXPECT().CreateServer(gomock.Any()).Return(&servers.Server{
				ID: serverID,
			}, nil)
			gomock.InOrder(
				compute.EXPECT().GetServer(serverID).Return(&servers.Server{
					ID:     serverID,
					Status: client.ServerStatusBuild,
				}, nil),
				compute.EXPECT().GetServer(serverID).Return(&servers.Server{
					ID:     serverID,
					Status: client.ServerStatusActive,
				}, nil))
			network.EXPECT().ListPorts(&ports.ListOpts{
				DeviceID: serverID,
			}).Return([]ports.Port{{NetworkID: networkID, ID: portID}}, nil)
			network.EXPECT().UpdatePort(portID, ports.UpdateOpts{
				AllowedAddressPairs: &[]ports.AddressPair{{IPAddress: podCidr}},
			}).Return(nil)

			providerId, err := ex.CreateMachine(ctx, machineName, nil)
			Expect(err).To(BeNil())
			Expect(providerId).To(Equal(EncodeProviderID(region, serverID)))
		})

		It("should delete the server on failure", func() {
			ex := &Executor{
				Compute: compute,
				Network: network,
				Config:  *cfg,
			}

			server := &servers.Server{
				Metadata: tags,
				ID:       serverID,
				Name:     machineName,
			}

			compute.EXPECT().ImageIDFromName(imageName).Return("imageID", nil)
			compute.EXPECT().FlavorIDFromName(flavorName).Return("flavorID", nil)
			compute.EXPECT().CreateServer(gomock.Any()).Return(&servers.Server{
				ID: serverID,
			}, nil)

			gomock.InOrder(
				// we return an error to avoid waiting for the wait.Poll timeout
				compute.EXPECT().GetServer(serverID).Return(nil, fmt.Errorf("error fetching server")),
				compute.EXPECT().GetServer(serverID).Return(server, nil),
				compute.EXPECT().DeleteServer(serverID).Return(nil),
				compute.EXPECT().GetServer(serverID).Do(func(_ string) { server.Status = client.ServerStatusDeleted }).Return(server, nil),
			)

			_, err := ex.CreateMachine(ctx, machineName, nil)
			Expect(err).NotTo(BeNil())
		})
	})

	Context("List", func() {
		It("should filter the instances based on tags", func() {
			compute.EXPECT().ListServers(gomock.Any()).Return(
				[]servers.Server{
					{
						Metadata: tags,
						ID:       "id1",
						Name:     "foo",
					},
					{
						Metadata: tags,
						ID:       "id2",
						Name:     "bar",
					},
					{
						ID:   "baz",
						Name: "baz",
					},
				},
				nil)

			ex := Executor{
				Compute: compute,
				Network: network,
				Config:  *cfg,
			}

			res, err := ex.ListMachines(ctx)
			Expect(err).To(BeNil())
			Expect(res).To(HaveLen(2))
			Expect(res).To(Equal(map[string]string{
				EncodeProviderID(region, "id1"): "foo",
				EncodeProviderID(region, "id2"): "bar",
			}))
		})
	})

	Context("#GetMachineStatus", func() {
		var (
			serverList []servers.Server
		)

		BeforeEach(func() {
			serverList = []servers.Server{
				{
					Metadata: tags,
					ID:       "id1",
					Name:     "foo",
				},
				{
					ID:   "id2",
					Name: "foo",
				},
				{
					ID:       "id3",
					Name:     "bar",
					Metadata: tags,
				},
				{
					ID:   "id4",
					Name: "baz",
				},
				{
					ID:       "id5",
					Name:     "lorem",
					Metadata: tags,
				},
				{
					ID:       "id6",
					Name:     "lorem",
					Metadata: tags,
				},
			}
		})

		table.DescribeTable("#Status", func(name string, expectedID string, expectedErr error) {
			compute.EXPECT().ListServers(&servers.ListOpts{Name: name}).Return(serverList, nil)
			ex := Executor{
				Compute: compute,
				Network: network,
				Config:  *cfg,
			}
			providerID, err := ex.GetMachineStatus(ctx, name)
			if expectedErr != nil {
				Expect(err).ToNot(BeNil())
				Expect(errors.Is(err, expectedErr)).To(BeTrue())
			} else {
				Expect(err).To(BeNil())
				Expect(providerID).To(Equal(EncodeProviderID(region, expectedID)))
			}
		},
			table.Entry("Should find the entry with matching metadata", "foo", "id1", nil),
			table.Entry("Should return not found if name not exists", "unknown", "", ErrNotFound),
			table.Entry("Should return not found if name exists without matching metadata", "baz", "", ErrNotFound),
			table.Entry("Should detect multiple matching servers", "lorem", "", ErrMultipleFound),
		)
	})

	Context("Delete", func() {

		var (
			serverList []servers.Server
		)

		BeforeEach(func() {
			serverList = []servers.Server{
				{
					Metadata: tags,
					ID:       "id1",
					Name:     "foo",
				},
				{
					ID:   "id2",
					Name: "foo",
				},
			}
		})

		It("should return no error if NotFound", func() {
			compute.EXPECT().ListServers(&servers.ListOpts{Name: "unknown"}).Return(serverList, nil)
			ex := Executor{
				Compute: compute,
				Network: network,
				Config:  *cfg,
			}
			err := ex.DeleteMachine(ctx, "unknown", "")
			Expect(err).To(BeNil())
		})

		It("should return no error if delete is successful", func() {
			compute.EXPECT().ListServers(&servers.ListOpts{Name: "foo"}).Return(serverList, nil)
			compute.EXPECT().DeleteServer("id1").Return(nil)
			compute.EXPECT().GetServer("id1").Return(&servers.Server{Status: client.ServerStatusDeleted}, nil)
			ex := Executor{
				Compute: compute,
				Network: network,
				Config:  *cfg,
			}
			err := ex.DeleteMachine(ctx, "foo", "")
			Expect(err).To(BeNil())
		})

		It("should try to find by ProviderID if supplied", func() {
			var (
				id = "id"
			)
			gomock.InOrder(
				compute.EXPECT().GetServer(id).Return(&servers.Server{ID: id, Status: client.ServerStatusActive, Metadata: tags}, nil),
				compute.EXPECT().DeleteServer(id).Return(nil),
				compute.EXPECT().GetServer(id).Return(&servers.Server{ID: id, Status: client.ServerStatusDeleted, Metadata: tags}, nil),
			)
			ex := Executor{
				Compute: compute,
				Network: network,
				Config:  *cfg,
			}
			err := ex.DeleteMachine(ctx, "", EncodeProviderID(region, id))
			Expect(err).To(BeNil())
		})

		It("should try to delete the port if we use specific subnetID", func() {
			var (
				subnetID    = "subID1"
				portID      = "portID"
				machineName = "foo"
			)

			cfg.Spec.SubnetID = pointer.StringPtr(subnetID)
			gomock.InOrder(
				compute.EXPECT().ListServers(&servers.ListOpts{Name: machineName}).Return(serverList, nil),
				compute.EXPECT().DeleteServer("id1").Return(nil),
				compute.EXPECT().GetServer("id1").Return(&servers.Server{Status: client.ServerStatusDeleted}, nil),
			)
			gomock.InOrder(
				network.EXPECT().PortIDFromName(machineName).Return(portID, nil),
				network.EXPECT().DeletePort(portID).Return(nil),
			)

			ex := Executor{
				Compute: compute,
				Network: network,
				Config:  *cfg,
			}
			err := ex.DeleteMachine(ctx, machineName, "")
			Expect(err).To(BeNil())
		})
	})
})
