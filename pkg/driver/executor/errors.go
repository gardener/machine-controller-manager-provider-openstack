// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
)

var (
	ErrNotFound = fmt.Errorf("resource not found")
	ErrMultipleFound = fmt.Errorf("multiple resources found")
)