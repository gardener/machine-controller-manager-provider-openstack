package validation_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	. "github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/cloudprovider"
	api "github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/openstack"
	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/apis/validation"
)

var ()

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
					NetworkID:      "networkID",
					SubnetID:       nil,
					PodNetworkCidr: "10.0.0.1/8",
					RootDiskSize:   0,
					UseConfigDrive: nil,
					ServerGroupID:  nil,
					Networks:       nil,
				},
			}
		})

		Context("required fields", func() {
			It("should return no error", func() {
				err := validation.ValidateMachineProviderConfig(machineProviderConfig).ToAggregate()
				Expect(err).To(BeNil())
			})

			It("should return error if required fields are missing", func() {
				spec := &machineProviderConfig.Spec
				spec.Region = ""
				spec.FlavorName = ""
				spec.AvailabilityZone = ""
				spec.KeyName = ""
				spec.PodNetworkCidr = ""
				err := validation.ValidateMachineProviderConfig(machineProviderConfig)

				expectedErr := field.ErrorList{
					{
						Type:     "FieldValueRequired",
						Field:    "spec.region",
						BadValue: "",
						Detail:   "Region is required",
					},
					{
						Type:     "FieldValueRequired",
						Field:    "spec.flavorName",
						BadValue: "",
						Detail:   "Flavor is required",
					},
					{
						Type:     "FieldValueRequired",
						Field:    "spec.availabilityZone",
						BadValue: "",
						Detail:   "AvailabilityZone name is required",
					},
					{
						Type:     "FieldValueRequired",
						Field:    "spec.keyName",
						BadValue: "",
						Detail:   "KeyName is required",
					},
					{
						Type:     "FieldValueRequired",
						Field:    "spec.podNetworkCidr",
						BadValue: "",
						Detail:   "PodNetworkCidr is required",
					},
				}

				Expect(err).To(HaveLen(len(expectedErr)))
				Expect(err).To(Equal(expectedErr))
			})
		})
		Context("#Networks", func() {
			It("should not allow Networks and NetworkID data in the same request", func(){
				spec := &machineProviderConfig.Spec
				spec.Networks = []api.OpenStackNetwork{
					{
						Id:         "foo",
						Name:       "",
						PodNetwork: false,
					},
				}
				expectedErr := field.ErrorList{
					{
						Type:     "FieldValueForbidden",
						Field:    "spec.networks",
						BadValue: "",
						Detail:   "\"networks\" list should not be specified along with \"providerConfig.Spec.NetworkID\"",
					},
				}
				err := validation.ValidateMachineProviderConfig(machineProviderConfig)
				Expect(err).To(HaveLen(len(expectedErr)))
				Expect(err).To(Equal(expectedErr))
			})

			It("should not allow missing Networks and NetworkID in the same request", func(){
				spec := &machineProviderConfig.Spec
				spec.NetworkID = ""
				expectedErr := field.ErrorList{
					{
							Type:     "FieldValueForbidden",
							Field:    "spec.networkID",
							BadValue: "",
							Detail:   "both \"networks\" and \"networkID\" should not be empty",
					},
				}
				err := validation.ValidateMachineProviderConfig(machineProviderConfig)
				Expect(err).To(HaveLen(len(expectedErr)))
				Expect(err).To(Equal(expectedErr))
			})

			It("should fail if Networks member are incorrect", func(){
				spec := &machineProviderConfig.Spec
				spec.NetworkID = ""
				spec.Networks = []api.OpenStackNetwork{
					{
						Id:         "foo",
						Name:       "foo",
						PodNetwork: false,
					},
				}
				expectedErr := field.ErrorList{
					{
						Type:     "FieldValueForbidden",
						Field:    "spec.networks[0]",
						BadValue: "",
						Detail:   "simultaneous use of network \"id\" and \"name\" is forbidden",
					},
				}
				err := validation.ValidateMachineProviderConfig(machineProviderConfig)
				Expect(err).To(HaveLen(len(expectedErr)))
				Expect(err).To(Equal(expectedErr))
			})
		})

		Context("#Tags", func() {
			It("should return an error if the cluster tags are missing", func() {
				spec := &machineProviderConfig.Spec
				spec.Tags = map[string]string{}

				err := validation.ValidateMachineProviderConfig(machineProviderConfig)
				expectedErr := field.ErrorList{
					{
						Type:     "FieldValueRequired",
						Field:    "spec.tags",
						BadValue: "",
						Detail:   "Tag required of the form kubernetes.io-cluster-****",
					},
					{
						Type:     "FieldValueRequired",
						Field:    "spec.tags",
						BadValue: "",
						Detail:   "Tag required of the form kubernetes.io-role-****",
					},
				}
				Expect(err).To(HaveLen(2))
				fmt.Printf("%v", err)
				Expect(err).To(Equal(expectedErr))
			})
		})
	})

	Describe("#Secret", func() {

		var secret *corev1.Secret

		BeforeEach(func(){
			secret = &corev1.Secret{
				Data: map[string][]byte{
					OpenStackAuthURL: []byte("auth"),
					OpenStackUsername: []byte("user"),
					OpenStackPassword: []byte("pwd"),
					OpenStackDomainName: []byte("domain"),
					OpenStackTenantName: []byte("tenant"),
				},
			}
		})

		It("should not fail", func(){
			err := validation.ValidateSecret(secret).ToAggregate()
			Expect(err).To(BeNil())
		})

		It("should fail is required fields are missing", func(){
			secret = &corev1.Secret{}

			expectedErr := field.ErrorList{
				{
					Type: "FieldValueRequired",
					Field: "data[authURL]",
					BadValue: "",
					Detail: "authURL is required",
				},
				{
					Type: "FieldValueRequired",
					Field: "data[username]",
					BadValue: "",
					Detail: "userDomainName is required",
				},
				{
					Type: "FieldValueRequired",
					Field: "data[password]",
					BadValue: "",
					Detail: "password is required",
				},
				{
					Type: "FieldValueRequired",
					Field: "data[domainName]",
					BadValue: "",
					Detail: "one of the following keys is required [domainName|domainID]",
				},
				{
					Type: "FieldValueRequired",
					Field: "data[tenantName]",
					BadValue: "",
					Detail: "one of the following keys is required [tenantName|tenantID]",
				},
			}

			err := validation.ValidateSecret(secret)
			Expect(err).To(Equal(expectedErr))
		})

		It("should fail if Insecure has erroneous value", func(){
			secret.Data[OpenStackInsecure] = []byte("foo")

			err := validation.ValidateSecret(secret).ToAggregate()
			Expect(err).NotTo(BeNil())
		})
	})

	Describe("#UserData", func(){
		var secret *corev1.Secret

		BeforeEach(func(){
			secret = &corev1.Secret{
				Data: map[string][]byte{
					UserData: []byte("foo"),
				},
			}
		})

		It("should fail if no user data found", func(){
			// empty secret
			secret = &corev1.Secret{}

			err := validation.ValidateUserData(secret).ToAggregate()
			Expect(err).To(Not(BeNil()))
		})

		It("should pass if user data found", func(){
			err := validation.ValidateUserData(secret).ToAggregate()
			Expect(err).To(BeNil())
		})
	})
})
