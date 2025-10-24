# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

#############      builder                                  #############
FROM golang:1.25.2 AS builder

WORKDIR /go/src/github.com/gardener/machine-controller-manager-provider-openstack
COPY . .
RUN make install

#############      base                                     #############
FROM gcr.io/distroless/static-debian12:nonroot AS base


############# machine-controller-manager-provider-openstack #############
FROM base AS machine-controller
WORKDIR /

COPY --from=builder /go/bin/machine-controller /machine-controller
ENTRYPOINT ["/machine-controller"]
