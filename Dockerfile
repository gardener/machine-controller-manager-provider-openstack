# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

#############      builder                                  #############
FROM golang:1.17.9 AS builder

WORKDIR /go/src/github.com/gardener/machine-controller-manager-provider-openstack
COPY . .
RUN make install
RUN ls -la /go/bin

#############      base                                     #############
FROM alpine:3.15.4 AS base

RUN apk add --update bash curl tzdata
WORKDIR /

#############      machine-controller               #############
FROM base AS machine-controller

COPY --from=builder /go/bin/machine-controller /machine-controller
ENTRYPOINT ["/machine-controller"]
