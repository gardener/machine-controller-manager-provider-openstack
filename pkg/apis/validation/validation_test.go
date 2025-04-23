// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"

	. "github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
	api "github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"
)

var _ = Describe("Validation", func() {
	Describe("#MachineProviderConfig", func() {
		var (
			machineProviderConfig *api.MachineProviderConfig
		)

		BeforeEach(func() {
			machineProviderConfig = &api.MachineProviderConfig{
				Spec: api.MachineProviderConfigSpec{
					ImageID:          "imageID",
					ImageName:        "imageName",
					Region:           "region",
					AvailabilityZone: "zone",
					FlavorName:       "flavor",
					KeyName:          "key",
					SecurityGroups:   nil,
					Tags: map[string]string{
						fmt.Sprintf("%s-foo", ServerTagRolePrefix):    "1",
						fmt.Sprintf("%s-foo", ServerTagClusterPrefix): "1",
					},
					NetworkID:       "networkID",
					SubnetID:        nil,
					PodNetworkCIDRs: []string{"10.0.0.1/8"},
					RootDiskSize:    0,
					UseConfigDrive:  nil,
					ServerGroupID:   nil,
					Networks:        nil,
				},
			}
		})

		Context("required fields", func() {
			It("should return no error", func() {
				err := validateMachineProviderConfig(machineProviderConfig).ToAggregate()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return error if required fields are missing", func() {
				spec := &machineProviderConfig.Spec
				spec.Region = ""
				spec.FlavorName = ""
				spec.AvailabilityZone = ""
				spec.KeyName = ""
				spec.PodNetworkCIDRs = nil
				err := validateMachineProviderConfig(machineProviderConfig)

				Expect(err).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  BeEquivalentTo("FieldValueRequired"),
						"Field": Equal("spec.region"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  BeEquivalentTo("FieldValueRequired"),
						"Field": Equal("spec.flavorName"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  BeEquivalentTo("FieldValueRequired"),
						"Field": Equal("spec.availabilityZone"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  BeEquivalentTo("FieldValueRequired"),
						"Field": Equal("spec.keyName"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  BeEquivalentTo("FieldValueRequired"),
						"Field": Equal("spec.PodNetworkCIDRs"),
					})),
				))
			})
		})

		Context("#Networks", func() {
			It("should not allow Networks and NetworkID data in the same request", func() {
				spec := &machineProviderConfig.Spec
				spec.Networks = []api.OpenStackNetwork{
					{
						Id:         "foo",
						Name:       "",
						PodNetwork: false,
					},
				}

				err := validateMachineProviderConfig(machineProviderConfig)
				Expect(err).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  BeEquivalentTo("FieldValueForbidden"),
						"Field": Equal("spec.networks"),
					})),
				))
			})

			It("should not allow missing Networks and NetworkID in the same request", func() {
				spec := &machineProviderConfig.Spec
				spec.NetworkID = ""

				err := validateMachineProviderConfig(machineProviderConfig)
				Expect(err).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  BeEquivalentTo("FieldValueForbidden"),
						"Field": Equal("spec.networkID"),
					})),
				))
			})

			It("should fail if Networks member are incorrect", func() {
				spec := &machineProviderConfig.Spec
				spec.NetworkID = ""
				spec.Networks = []api.OpenStackNetwork{
					{
						Id:         "foo",
						Name:       "foo",
						PodNetwork: false,
					},
				}
				err := validateMachineProviderConfig(machineProviderConfig)
				Expect(err).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  BeEquivalentTo("FieldValueForbidden"),
						"Field": Equal("spec.networks[0]"),
					})),
				))
			})
		})

		Context("#Tags", func() {
			It("should return an error if the cluster tags are missing", func() {
				spec := &machineProviderConfig.Spec
				spec.Tags = map[string]string{}

				err := validateMachineProviderConfig(machineProviderConfig)
				Expect(err).To(ConsistOf(
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  BeEquivalentTo("FieldValueRequired"),
						"Field": Equal("spec.tags"),
					})),
					PointTo(MatchFields(IgnoreExtras, Fields{
						"Type":  BeEquivalentTo("FieldValueRequired"),
						"Field": Equal("spec.tags"),
					})),
				))
			})
		})
	})

	Describe("#Secret", func() {

		var secret *corev1.Secret

		BeforeEach(func() {
			secret = &corev1.Secret{
				Data: map[string][]byte{
					OpenStackAuthURL:    []byte("auth"),
					OpenStackUsername:   []byte("user"),
					OpenStackPassword:   []byte("pwd"),
					OpenStackDomainName: []byte("domain"),
					OpenStackTenantName: []byte("tenant"),
				},
			}
		})

		It("should not fail", func() {
			err := validateSecret(secret).ToAggregate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail if required fields are missing", func() {
			secret = &corev1.Secret{}

			err := validateSecret(secret)
			Expect(err).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  BeEquivalentTo("FieldValueRequired"),
					"Field": Equal("data[authURL]"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  BeEquivalentTo("FieldValueRequired"),
					"Field": Equal("data[username]"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  BeEquivalentTo("FieldValueRequired"),
					"Field": Equal("data[password]"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  BeEquivalentTo("FieldValueRequired"),
					"Field": Equal("data[domainName]"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  BeEquivalentTo("FieldValueRequired"),
					"Field": Equal("data[tenantName]"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  BeEquivalentTo("FieldValueRequired"),
					"Field": Equal("data[applicationCredentialID]"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  BeEquivalentTo("FieldValueRequired"),
					"Field": Equal("data[applicationCredentialName]"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  BeEquivalentTo("FieldValueRequired"),
					"Field": Equal("data[applicationCredentialSecret]"),
				})),
			))
		})

		It("should fail if application credential id or name field are missing", func() {
			delete(secret.Data, OpenStackPassword)
			secret.Data[OpenStackApplicationCredentialSecret] = []byte("app-secret")

			err := validateSecret(secret)
			Expect(err).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  BeEquivalentTo("FieldValueRequired"),
					"Field": Equal("data[" + OpenStackApplicationCredentialID + "]"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  BeEquivalentTo("FieldValueRequired"),
					"Field": Equal("data[" + OpenStackApplicationCredentialName + "]"),
				})),
			))
		})

		It("should succeed if application credentials are used (id + secret)", func() {
			delete(secret.Data, OpenStackUsername)
			delete(secret.Data, OpenStackPassword)
			secret.Data[OpenStackApplicationCredentialID] = []byte("app-id")
			secret.Data[OpenStackApplicationCredentialSecret] = []byte("app-secret")

			err := validateSecret(secret).ToAggregate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should succeed if application credentials are used (id + name + secret)", func() {
			delete(secret.Data, OpenStackUsername)
			delete(secret.Data, OpenStackPassword)
			secret.Data[OpenStackApplicationCredentialID] = []byte("app-id")
			secret.Data[OpenStackApplicationCredentialName] = []byte("app-name")
			secret.Data[OpenStackApplicationCredentialSecret] = []byte("app-secret")

			err := validateSecret(secret).ToAggregate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should succeed if application credentials are used (username + name + secret)", func() {
			delete(secret.Data, OpenStackPassword)
			secret.Data[OpenStackApplicationCredentialName] = []byte("app-name")
			secret.Data[OpenStackApplicationCredentialSecret] = []byte("app-secret")

			err := validateSecret(secret).ToAggregate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail if Insecure has erroneous value", func() {
			secret.Data[OpenStackInsecure] = []byte("foo")

			err := validateSecret(secret).ToAggregate()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("#UserData", func() {
		var secret *corev1.Secret

		BeforeEach(func() {
			secret = &corev1.Secret{
				Data: map[string][]byte{
					UserData: []byte("foo"),
				},
			}
		})

		It("should fail if no user data found", func() {
			// empty secret
			secret = &corev1.Secret{}

			err := validateUserData(secret).ToAggregate()
			Expect(err).To(HaveOccurred())
		})

		It("should pass if user data found", func() {
			err := validateUserData(secret).ToAggregate()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
