// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Logger", func() {
	Context("#formatHeaders", func() {
		It("should format headers", func() {
			header := http.Header{
				"Test-Header-A": []string{"test-value-A"},
				"Test-Header-B": []string{"test-value-B"},
			}
			res := formatHeaders(header, ",")
			Expect(res).To(Equal("Test-Header-A: test-value-A,Test-Header-B: test-value-B"))
		})
	})

	Context("#hideSensitiveHeadersData", func() {
		It("should hide sensitive data", func() {
			header := http.Header{
				"Test-Header":  []string{"test-value"},
				"x-auth-token": []string{"secret-token"},
			}
			res := hideSensitiveHeadersData(header)
			Expect(res).To(ContainElements("x-auth-token: ***", "Test-Header: test-value"))
		})
	})
})
