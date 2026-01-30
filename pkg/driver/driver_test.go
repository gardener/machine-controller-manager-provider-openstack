// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package driver

import (
	"errors"

	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/codes"
	"github.com/gardener/machine-controller-manager/pkg/util/provider/machinecodes/status"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/machine-controller-manager-provider-openstack/pkg/driver/executor"
)

var _ = Describe("Driver", func() {

	Context("mapErrorToCode", func() {
		It("should map executor.ErrFlavorNotFound errors to ResourceExhausted error code", func() {
			err1 := executor.ErrFlavorNotFound{Flavor: "flavor"}
			err2 := status.Error(mapErrorToCode(err1), err1.Error())
			Expect(err1).To(HaveOccurred())
			Expect(err2).To(HaveOccurred())
			Expect(mapErrorToCode(err1)).To(Equal(codes.ResourceExhausted))
			Expect(errors.Is(err2, executor.ErrNotFound)).To(BeFalse())
		})
	})

})
