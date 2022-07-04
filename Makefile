# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

BINARY_PATH         := bin/
REPO_ROOT           := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
COVERPROFILE        := test/output/coverprofile.out
REGISTRY            := eu.gcr.io/gardener-project/gardener
IMAGE_PREFIX        := $(REGISTRY)/extensions
NAME                := machine-controller-manager-provider-openstack
IMAGE_NAME          := $(IMAGE_PREFIX)/$(NAME)
VERSION             := $(shell cat VERSION)
CONTROL_NAMESPACE   := default
CONTROL_KUBECONFIG  := dev/control-kubeconfig.yaml
TARGET_KUBECONFIG   := dev/target-kubeconfig.yaml

# Below ones are used in tests
MACHINECLASS_V1 	:= dev/machineclassv1.yaml
MACHINECLASS_V2 	:= 
MCM_IMAGE			:= 
MC_IMAGE			:= 
# MCM_IMAGE			:= eu.gcr.io/gardener-project/gardener/machine-controller-manager:v0.42.0
# MC_IMAGE			:= $(IMAGE_NAME):v0.6.0
LEADER_ELECT 	    := "true"
# If Integration Test Suite is to be run locally against clusters then export the below variable
# with MCM deployment name in the cluster
MACHINE_CONTROLLER_MANAGER_DEPLOYMENT_NAME := machine-controller-manager
#################################################
# Rules for starting machine-controller locally
#################################################

.PHONY: start
start:
	@GO111MODULE=on go run \
			-mod=vendor \
			cmd/machine-controller/main.go \
			--control-kubeconfig=$(CONTROL_KUBECONFIG) \
			--target-kubeconfig=$(TARGET_KUBECONFIG) \
			--namespace=$(CONTROL_NAMESPACE) \
			--machine-creation-timeout=20m \
			--machine-drain-timeout=5m \
			--machine-health-timeout=10m \
			--machine-pv-detach-timeout=2m \
			--machine-safety-apiserver-statuscheck-timeout=30s \
			--machine-safety-apiserver-statuscheck-period=1m \
			--machine-safety-orphan-vms-period=30m \
			--leader-elect=$(LEADER_ELECT) \
			--v=3

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning
#####################################################################

.PHONY: install-requirements
install-requirements:
	@go install -mod=vendor $(REPO_ROOT)/vendor/github.com/ahmetb/gen-crd-api-reference-docs
	@go install -mod=vendor $(REPO_ROOT)/vendor/github.com/golang/mock/mockgen
	@go install -mod=vendor $(REPO_ROOT)/vendor/github.com/onsi/ginkgo/ginkgo
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/install-requirements.sh

.PHONY: install
install:
	@LD_FLAGS="-w -X github.com/gardener/$(NAME)/pkg/version.Version=$(VERSION)" \
	$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/install.sh ./...

.PHONY: generate
generate:
	@env GOMODULE111=on go generate -mod=vendor ./pkg/...

.PHONY: check-generate
check-generate:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check-generate.sh $(REPO_ROOT)

.PHONY: check
check:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/...
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check-charts.sh ./charts

.PHONY: format
format:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/format.sh ./cmd ./pkg 

.PHONY: test
test:
	@SKIP_FETCH_TOOLS=1 $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test.sh ./cmd/... ./pkg/...

.PHONY: test-cov
test-cov:
	@SKIP_FETCH_TOOLS=1 $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test-cover.sh ./cmd/... ./pkg/...

.PHONY: test-clean
test-clean:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test-cover-clean.sh

.PHONY: verify
verify: check format test

.PHONY: verify-extended
verify-extended: install-requirements check-generate check format test-cov test-clean

.PHONY: clean
clean:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/clean.sh ./cmd/... ./pkg/...

.PHONY: test-integration
test-integration:
	@if [[ -f $(PWD)/$(CONTROL_KUBECONFIG) ]]; then export CONTROL_KUBECONFIG=$(PWD)/$(CONTROL_KUBECONFIG); fi; \
	if [[ -f $(PWD)/$(TARGET_KUBECONFIG) ]]; then export TARGET_KUBECONFIG=$(PWD)/$(TARGET_KUBECONFIG); fi; \
	if [[ -f $(PWD)/$(MACHINECLASS_V1) ]]; then export MACHINECLASS_V1=$(PWD)/$(MACHINECLASS_V1); fi; \
	if [[ -f $(PWD)/$(MACHINECLASS_V2) ]]; then export MACHINECLASS_V2=$(PWD)/$(MACHINECLASS_V2); fi; \
	export MC_CONTAINER_IMAGE=$(MC_IMAGE); \
	export MCM_CONTAINER_IMAGE=$(MCM_IMAGE); \
	export CONTROL_CLUSTER_NAMESPACE=$(CONTROL_NAMESPACE); \
	export MACHINE_CONTROLLER_MANAGER_DEPLOYMENT_NAME=$(MACHINE_CONTROLLER_MANAGER_DEPLOYMENT_NAME); \
	.ci/local_integration_test

#########################################
# Rules for re-vendoring
#########################################

.PHONY: revendor
revendor:
	@env GO111MODULE=on go mod tidy -v
	@env GO111MODULE=on go mod vendor -v
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/*
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/.ci/*

.PHONY: update-dependencies
update-dependencies:
	@env GO111MODULE=on go get -u

#########################################
# Rules for build/release
#########################################

.PHONY: release
release: docker-image docker-push

.PHONY: docker-image
docker-image:
	docker image build -t $(IMAGE_NAME):$(VERSION) -t $(IMAGE_NAME):latest .

.PHONY: docker-login
docker-login:
	@gcloud auth login

.PHONY: docker-push
docker-push:
	@if ! docker images $(IMAGE_NAME) | awk '{ print $$2 }' | grep -q -F $(VERSION); then echo "$(IMAGE_NAME)/$(VERSION) is not yet built. Please run 'make docker-images'"; false; fi
	@docker image push $(IMAGE_NAME):$(VERSION)
