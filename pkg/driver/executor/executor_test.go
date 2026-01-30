// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/client"
	mocks "github.com/gardener/machine-controller-manager-provider-openstack/pkg/mock/openstack"
)

var _ = Describe("Executor", func() {
	const (
		region    = "eu-nl-1"
		networkID = "networkID"
	)
	var (
		ctrl    *gomock.Controller
		compute *mocks.MockCompute
		network *mocks.MockNetwork
		storage *mocks.MockStorage
		tags    map[string]string
		cfg     *openstack.MachineProviderConfig
		ctx     context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		ctrl = gomock.NewController(GinkgoT())
		compute = mocks.NewMockCompute(ctrl)
		network = mocks.NewMockNetwork(ctrl)
		storage = mocks.NewMockStorage(ctrl)

		tags = map[string]string{
			fmt.Sprintf("%sfoo", cloudprovider.ServerTagClusterPrefix): "1",
			fmt.Sprintf("%sfoo", cloudprovider.ServerTagRolePrefix):    "1",
		}

		cfg = &openstack.MachineProviderConfig{
			Spec: openstack.MachineProviderConfigSpec{
				Tags:      tags,
				Region:    region,
				NetworkID: networkID,
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
			serverIPv4  = "10.250.0.5"
			serverIPv6  = "2000:db0::1"
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
				Config:  cfg,
			}

			compute.EXPECT().ListServers(ctx, &servers.ListOpts{Name: machineName}).Return([]servers.Server{}, nil)
			compute.EXPECT().ImageIDFromName(ctx, imageName).Return(images.Image{ID: "imageID"}, nil)
			compute.EXPECT().FlavorIDFromName(ctx, flavorName).Return("flavorID", nil)
			compute.EXPECT().CreateServer(ctx, gomock.Any(), gomock.Any()).Return(&servers.Server{
				ID: serverID,
			}, nil)
			gomock.InOrder(
				compute.EXPECT().GetServer(ctx, serverID).Return(&servers.Server{
					ID:     serverID,
					Status: client.ServerStatusBuild,
				}, nil),
				compute.EXPECT().GetServer(ctx, serverID).Return(&servers.Server{
					ID:     serverID,
					Status: client.ServerStatusActive,
					Addresses: map[string]any{
						"private": []any{
							map[string]any{
								"addr":    serverIPv4,
								"version": 4,
							},
						},
					},
				}, nil))
			network.EXPECT().ListPorts(ctx, &ports.ListOpts{
				DeviceID: serverID,
			}).Return([]ports.Port{{NetworkID: networkID, ID: portID}}, nil)
			network.EXPECT().UpdatePort(ctx, portID, ports.UpdateOpts{
				AllowedAddressPairs: &[]ports.AddressPair{{IPAddress: podCidr}},
			}).Return(nil)

			server, err := ex.CreateMachine(ctx, machineName, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(server.InternalIPs).To(HaveLen(1))
			Expect(server.InternalIPs[0]).To(Equal(serverIPv4))
		})

		It("should succeed when spec contains subnet", func() {
			subnetID := "subnetID"

			cfg.Spec.SubnetID = &subnetID
			ex := &Executor{
				Compute: compute,
				Network: network,
				Config:  cfg,
			}

			compute.EXPECT().ListServers(ctx, &servers.ListOpts{Name: machineName}).Return([]servers.Server{}, nil)
			network.EXPECT().GetSubnet(ctx, subnetID).Return(&subnets.Subnet{}, nil)
			network.EXPECT().PortIDFromName(ctx, machineName).Return("", gophercloud.ErrResourceNotFound{})
			network.EXPECT().CreatePort(ctx, gomock.Any()).Return(&ports.Port{ID: portID, Name: machineName}, nil)
			network.EXPECT().TagPort(ctx, gomock.Any(), gomock.Any()).Return(nil)
			compute.EXPECT().ImageIDFromName(ctx, imageName).Return(images.Image{ID: "imageID"}, nil)
			compute.EXPECT().FlavorIDFromName(ctx, flavorName).Return("flavorID", nil)
			compute.EXPECT().CreateServer(ctx, gomock.Any(), gomock.Any()).Return(&servers.Server{ID: serverID}, nil)
			gomock.InOrder(
				compute.EXPECT().GetServer(ctx, serverID).Return(&servers.Server{ID: serverID, Status: client.ServerStatusBuild}, nil),
				compute.EXPECT().GetServer(ctx, serverID).Return(&servers.Server{ID: serverID, Status: client.ServerStatusActive}, nil),
			)
			network.EXPECT().ListPorts(ctx, &ports.ListOpts{DeviceID: serverID}).Return([]ports.Port{{NetworkID: networkID, ID: portID}}, nil)
			network.EXPECT().UpdatePort(ctx, portID, ports.UpdateOpts{
				AllowedAddressPairs: &[]ports.AddressPair{{IPAddress: podCidr}},
			}).Return(nil)

			server, err := ex.CreateMachine(ctx, machineName, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(server.ProviderID).To(Equal(encodeProviderID(region, serverID)))
		})

		It("should succeed when spec contains rootDisksize", func() {
			var (
				diskType = "standard_hdd"
				diskSize = 50
				volumeID = "volumeID"
			)
			cfg.Spec.RootDiskType = &diskType
			cfg.Spec.RootDiskSize = diskSize
			ex := &Executor{
				Compute: compute,
				Network: network,
				Storage: storage,
				Config:  cfg,
			}

			compute.EXPECT().ListServers(ctx, &servers.ListOpts{Name: machineName}).Return([]servers.Server{}, nil)
			compute.EXPECT().ImageIDFromName(ctx, imageName).Return(images.Image{ID: "imageID"}, nil)
			compute.EXPECT().FlavorIDFromName(ctx, flavorName).Return("flavorID", nil)
			storage.EXPECT().VolumeIDFromName(ctx, machineName).Return("", gophercloud.ErrResourceNotFound{})
			gomock.InOrder(
				storage.EXPECT().GetVolume(ctx, volumeID).Return(&volumes.Volume{ID: volumeID, Status: client.VolumeStatusCreating}, nil),
				storage.EXPECT().GetVolume(ctx, volumeID).Return(&volumes.Volume{ID: volumeID, Status: client.VolumeStatusAvailable}, nil),
			)
			storage.EXPECT().CreateVolume(ctx, gomock.Any(), gomock.Any()).Return(&volumes.Volume{ID: volumeID}, nil)
			compute.EXPECT().CreateServer(ctx, gomock.Any(), gomock.Any()).Return(&servers.Server{
				ID: serverID,
			}, nil)
			gomock.InOrder(
				compute.EXPECT().GetServer(ctx, serverID).Return(&servers.Server{ID: serverID, Status: client.ServerStatusBuild}, nil),
				compute.EXPECT().GetServer(ctx, serverID).Return(&servers.Server{ID: serverID, Status: client.ServerStatusActive}, nil),
			)
			network.EXPECT().ListPorts(ctx, &ports.ListOpts{DeviceID: serverID}).Return([]ports.Port{{NetworkID: networkID, ID: portID}}, nil)
			network.EXPECT().UpdatePort(ctx, portID, ports.UpdateOpts{
				AllowedAddressPairs: &[]ports.AddressPair{{IPAddress: podCidr}},
			}).Return(nil)

			server, err := ex.CreateMachine(ctx, machineName, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(server.ProviderID).To(Equal(encodeProviderID(region, serverID)))
		})

		It("should raise a ErrResourceNotFound error when called with a missing flavor", func() {
			ex := &Executor{
				Compute: compute,
				Network: network,
				Config:  cfg,
			}

			compute.EXPECT().ListServers(ctx, &servers.ListOpts{Name: machineName}).Return([]servers.Server{}, nil)
			compute.EXPECT().ImageIDFromName(ctx, imageName).Return(images.Image{ID: "imageID"}, nil)
			compute.EXPECT().FlavorIDFromName(ctx, flavorName).Return(flavorName, gophercloud.ErrResourceNotFound{Name: flavorName, ResourceType: "flavor"})
			compute.EXPECT().ListServers(ctx, &servers.ListOpts{Name: machineName}).Return([]servers.Server{}, nil)

			_, err := ex.CreateMachine(ctx, machineName, nil)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrFlavorNotFound{Flavor: "flavor"})).To(BeTrue())
		})

		It("should delete the server on failure", func() {
			ex := &Executor{
				Compute: compute,
				Network: network,
				Config:  cfg,
			}

			server := &servers.Server{
				Metadata: tags,
				ID:       serverID,
				Name:     machineName,
			}

			compute.EXPECT().ListServers(ctx, &servers.ListOpts{Name: machineName}).Return([]servers.Server{}, nil)
			compute.EXPECT().ImageIDFromName(ctx, imageName).Return(images.Image{ID: "imageID"}, nil)
			compute.EXPECT().FlavorIDFromName(ctx, flavorName).Return("flavorID", nil)
			compute.EXPECT().CreateServer(ctx, gomock.Any(), gomock.Any()).Return(&servers.Server{
				ID: serverID,
			}, nil)

			gomock.InOrder(
				// we return an error to avoid waiting for the wait.Poll timeout
				compute.EXPECT().GetServer(ctx, serverID).Return(nil, fmt.Errorf("error fetching server")),
				compute.EXPECT().ListServers(ctx, &servers.ListOpts{Name: machineName}).Return([]servers.Server{*server}, nil),
				compute.EXPECT().DeleteServer(ctx, serverID).Return(nil),
				compute.EXPECT().GetServer(ctx, serverID).Do(func(_ context.Context, _ string) { server.Status = client.ServerStatusDeleted }).Return(server, nil),
			)

			_, err := ex.CreateMachine(ctx, machineName, nil)
			Expect(err).To(HaveOccurred())
		})

		It("should accept multiple internal IPs", func() {
			ex := &Executor{
				Compute: compute,
				Network: network,
				Config:  cfg,
			}

			compute.EXPECT().ListServers(ctx, &servers.ListOpts{Name: machineName}).Return([]servers.Server{}, nil)
			compute.EXPECT().ImageIDFromName(ctx, imageName).Return(images.Image{ID: "imageID"}, nil)
			compute.EXPECT().FlavorIDFromName(ctx, flavorName).Return("flavorID", nil)
			compute.EXPECT().CreateServer(ctx, gomock.Any(), gomock.Any()).Return(&servers.Server{
				ID: serverID,
			}, nil)
			gomock.InOrder(
				compute.EXPECT().GetServer(ctx, serverID).Return(&servers.Server{
					ID:     serverID,
					Status: client.ServerStatusBuild,
				}, nil),
				compute.EXPECT().GetServer(ctx, serverID).Return(&servers.Server{
					ID:     serverID,
					Status: client.ServerStatusActive,
					Addresses: map[string]any{
						"private": []any{
							map[string]any{
								"addr":    serverIPv4,
								"version": 4,
							},
							map[string]any{
								"addr":    serverIPv6,
								"version": 6,
							},
						},
					},
				}, nil))
			network.EXPECT().ListPorts(ctx, &ports.ListOpts{
				DeviceID: serverID,
			}).Return([]ports.Port{{NetworkID: networkID, ID: portID}}, nil)
			network.EXPECT().UpdatePort(ctx, portID, ports.UpdateOpts{
				AllowedAddressPairs: &[]ports.AddressPair{{IPAddress: podCidr}},
			}).Return(nil)

			server, err := ex.CreateMachine(ctx, machineName, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(server.InternalIPs).To(HaveLen(2))
			Expect(server.InternalIPs).To(ConsistOf(serverIPv4, serverIPv6))
		})
	})

	Context("List", func() {
		It("should filter the instances based on tags", func() {
			compute.EXPECT().ListServers(ctx, gomock.Any()).Return(
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
				Config:  cfg,
			}

			res, err := ex.ListMachines(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveLen(2))
			Expect(res).To(Equal(map[string]string{
				encodeProviderID(region, "id1"): "foo",
				encodeProviderID(region, "id2"): "bar",
			}))
		})
	})

	Context("#GetMachineStatus", func() {
		var serverList []servers.Server

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

		DescribeTable("#Status",
			func(name string, expectedID string, expectedErr error) {
				compute.EXPECT().ListServers(ctx, &servers.ListOpts{Name: name}).Return(serverList, nil)
				ex := Executor{
					Compute: compute,
					Network: network,
					Config:  cfg,
				}
				server, err := ex.getMachineByName(ctx, name)
				if expectedErr != nil {
					Expect(errors.Is(err, expectedErr)).To(BeTrue())
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(server.ID).To(Equal(expectedID))
				}
			},
			Entry("Should find the entry with matching metadata", "foo", "id1", nil),
			Entry("Should return not found if name not exists", "unknown", "", ErrNotFound),
			Entry("Should return not found if name exists without matching metadata", "baz", "", ErrNotFound),
			Entry("Should detect multiple matching servers", "lorem", "", ErrMultipleFound),
		)
	})

	Context("Delete", func() {
		var serverList []servers.Server

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
			compute.EXPECT().ListServers(ctx, &servers.ListOpts{Name: "unknown"}).Return(serverList, nil)
			ex := Executor{
				Compute: compute,
				Network: network,
				Config:  cfg,
			}
			err := ex.DeleteMachine(ctx, "unknown", "")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return no error if delete is successful", func() {
			compute.EXPECT().ListServers(ctx, &servers.ListOpts{Name: "foo"}).Return(serverList, nil)
			compute.EXPECT().DeleteServer(ctx, "id1").Return(nil)
			compute.EXPECT().GetServer(ctx, "id1").Return(&servers.Server{Status: client.ServerStatusDeleted}, nil)
			ex := Executor{
				Compute: compute,
				Network: network,
				Config:  cfg,
			}
			err := ex.DeleteMachine(ctx, "foo", "")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should try to find by ProviderID if supplied", func() {
			id := "id"
			gomock.InOrder(
				compute.EXPECT().GetServer(ctx, id).Return(&servers.Server{ID: id, Status: client.ServerStatusActive, Metadata: tags}, nil),
				compute.EXPECT().DeleteServer(ctx, id).Return(nil),
				compute.EXPECT().GetServer(ctx, id).Return(&servers.Server{ID: id, Status: client.ServerStatusDeleted, Metadata: tags}, nil),
			)
			ex := Executor{
				Compute: compute,
				Network: network,
				Config:  cfg,
			}
			err := ex.DeleteMachine(ctx, "", encodeProviderID(region, id))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should try to delete the port if we use specific subnetID", func() {
			var (
				subnetID    = "subID1"
				portID      = "portID"
				machineName = "foo"
			)

			cfg.Spec.SubnetID = ptr.To(subnetID)
			gomock.InOrder(
				compute.EXPECT().ListServers(ctx, &servers.ListOpts{Name: machineName}).Return(serverList, nil),
				compute.EXPECT().DeleteServer(ctx, "id1").Return(nil),
				compute.EXPECT().GetServer(ctx, "id1").Return(&servers.Server{Status: client.ServerStatusDeleted}, nil),
			)
			gomock.InOrder(
				network.EXPECT().ListPorts(ctx, ports.ListOpts{Name: machineName}).Return([]ports.Port{{ID: portID}}, nil),
				network.EXPECT().DeletePort(ctx, portID).Return(nil),
			)

			ex := Executor{
				Compute: compute,
				Network: network,
				Config:  cfg,
			}
			err := ex.DeleteMachine(ctx, machineName, "")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete all ports if multiple are found", func() {
			var (
				subnetID    = "subID1"
				portID1     = "portID1"
				portID2     = "portID2"
				machineName = "foo"
			)

			cfg.Spec.SubnetID = ptr.To(subnetID)
			gomock.InOrder(
				compute.EXPECT().ListServers(ctx, &servers.ListOpts{Name: machineName}).Return(serverList, nil),
				compute.EXPECT().DeleteServer(ctx, "id1").Return(nil),
				compute.EXPECT().GetServer(ctx, "id1").Return(&servers.Server{Status: client.ServerStatusDeleted}, nil),
			)
			gomock.InOrder(
				network.EXPECT().ListPorts(ctx, ports.ListOpts{Name: machineName}).Return([]ports.Port{{ID: portID1}, {ID: portID2}}, nil),
				network.EXPECT().DeletePort(ctx, portID1).Return(nil),
				network.EXPECT().DeletePort(ctx, portID2).Return(nil),
			)

			ex := Executor{
				Compute: compute,
				Network: network,
				Config:  cfg,
			}
			err := ex.DeleteMachine(ctx, machineName, "")
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
