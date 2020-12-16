# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

#############      builder                                  #############
FROM eu.gcr.io/gardener-project/3rd/golang:1.15.5 AS builder

WORKDIR /go/src/github.com/gardener/machine-controller-manager-provider-openstack
COPY . .
RUN make install
RUN ls -la /go/bin

#############      base                                     #############
FROM eu.gcr.io/gardener-project/3rd/alpine:3.12.1 AS base

RUN apk add --update bash curl tzdata
WORKDIR /

#############      machine-controller               #############
FROM base AS machine-controller

COPY --from=builder /go/bin/machine-controller /machine-controller
ENTRYPOINT ["/machine-controller"]
