// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"fmt"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	"github.com/gophercloud/gophercloud/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/driver/executor"
)

var _ = Describe("Driver", func() {

	Context("mapErrorToCode", func() {
		It("should map executor.ErrFlavorNotFound error to ResourceExhausted error code", func() {
			err1 := executor.ErrFlavorNotFound{}
			err2 := status.Error(mapErrorToCode(err1), err1.Error())
			Expect(err1).To(HaveOccurred())
			Expect(err2).To(HaveOccurred())
			Expect(mapErrorToCode(err1)).To(Equal(codes.ResourceExhausted))
		})
		It("should map executor.ErrFlavorNotFound error with specific flavor to ResourceExhausted error code", func() {
			err1 := executor.ErrFlavorNotFound{Flavor: "flavor"}
			err2 := status.Error(mapErrorToCode(err1), err1.Error())
			Expect(err1).To(HaveOccurred())
			Expect(err2).To(HaveOccurred())
			Expect(mapErrorToCode(err1)).To(Equal(codes.ResourceExhausted))
		})
		It("should map executor.ErrNotFound error to NotFound error code", func() {
			err1 := executor.ErrNotFound
			err2 := status.Error(mapErrorToCode(err1), err1.Error())
			Expect(err1).To(HaveOccurred())
			Expect(err2).To(HaveOccurred())
			Expect(mapErrorToCode(err1)).To(Equal(codes.NotFound))
		})
		It("should map gophercloud.ErrResourceNotFound error to Internal error code", func() {
			err1 := gophercloud.ErrResourceNotFound{}
			err2 := status.Error(mapErrorToCode(err1), err1.Error())
			Expect(err1).To(HaveOccurred())
			Expect(err2).To(HaveOccurred())
			Expect(mapErrorToCode(err1)).To(Equal(codes.Internal))
		})
		It("should map error containing executor.NoValidHost to ResourceExhausted error code", func() {
			err1 := fmt.Errorf("error: %s", executor.NoValidHost)
			err2 := status.Error(mapErrorToCode(err1), err1.Error())
			Expect(err1).To(HaveOccurred())
			Expect(err2).To(HaveOccurred())
			Expect(mapErrorToCode(err1)).To(Equal(codes.ResourceExhausted))
		})
		It("should map error containing random string to Internal error code", func() {
			err1 := fmt.Errorf("error: Random provider issue")
			err2 := status.Error(mapErrorToCode(err1), err1.Error())
			Expect(err1).To(HaveOccurred())
			Expect(err2).To(HaveOccurred())
			Expect(mapErrorToCode(err1)).To(Equal(codes.Internal))
		})
	})
})
