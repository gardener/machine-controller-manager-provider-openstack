# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

ENSURE_GARDENER_MOD := $(shell go get github.com/gardener/gardener@$$(go list -m -f "{{.Version}}" github.com/gardener/gardener))
GARDENER_HACK_DIR   := $(shell go list -m -f "{{.Dir}}" github.com/gardener/gardener)/hack


REPO_ROOT           := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
HACK_DIR            := $(REPO_ROOT)/hack
COVERPROFILE        := test/output/coverprofile.out
REGISTRY            := europe-docker.pkg.dev/gardener-project/public
IMAGE_PREFIX        := $(REGISTRY)/gardener/extensions
NAME                := machine-controller-manager-provider-openstack
IMAGE_NAME          := $(IMAGE_PREFIX)/$(NAME)
VERSION             := $(shell cat VERSION)

LEADER_ELECT 	    := "true"
# If Integration Test Suite is to be run locally against clusters then export the below variable
# with MCM deployment name in the cluster
MACHINE_CONTROLLER_MANAGER_DEPLOYMENT_NAME := machine-controller-manager

#########################################
# Tools & Cleanup
#########################################

TOOLS_DIR := $(HACK_DIR)/tools
include $(GARDENER_HACK_DIR)/tools.mk
-include .env

.PHONY: tidy
tidy:
	@go mod tidy
	@mkdir -p $(REPO_ROOT)/.ci/hack && cp $(GARDENER_HACK_DIR)/.ci/* $(REPO_ROOT)/.ci/hack/ && chmod +xw $(REPO_ROOT)/.ci/hack/*
	@GARDENER_HACK_DIR=$(GARDENER_HACK_DIR) bash $(REPO_ROOT)/hack/update-github-templates.sh
	@cp $(GARDENER_HACK_DIR)/cherry-pick-pull.sh $(HACK_DIR)/cherry-pick-pull.sh && chmod +xw $(HACK_DIR)/cherry-pick-pull.sh

#################################################
# Rules for starting machine-controller locally
#################################################

.PHONY: start
start:
	go run \
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

.PHONY: install
install:
	@LD_FLAGS="-w -X github.com/gardener/$(NAME)/pkg/version.Version=$(VERSION)" \
	bash $(GARDENER_HACK_DIR)/install.sh ./...

.PHONY: generate
generate: $(VGOPATH) $(CONTROLLER_GEN) $(GEN_CRD_API_REFERENCE_DOCS) $(HELM) $(MOCKGEN)
	@REPO_ROOT=$(REPO_ROOT) VGOPATH=$(VGOPATH) GARDENER_HACK_DIR=$(GARDENER_HACK_DIR) bash $(GARDENER_HACK_DIR)/generate-sequential.sh ./charts/... ./cmd/... ./example/... ./pkg/...
	$(MAKE) format

.PHONY: check-generate
check-generate:
	@bash $(GARDENER_HACK_DIR)/check-generate.sh $(REPO_ROOT)

.PHONY: format
format: $(GOIMPORTS) $(GOIMPORTSREVISER)
	@bash $(GARDENER_HACK_DIR)/format.sh ./cmd ./pkg ./test

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT)
	@bash $(GARDENER_HACK_DIR)/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/... ./test/...
	@bash $(GARDENER_HACK_DIR)/check-charts.sh ./charts

.PHONY: test
test:
	@SKIP_FETCH_TOOLS=1 bash $(GARDENER_HACK_DIR)/test.sh ./cmd/... ./pkg/...

.PHONY: test-cov
test-cov:
	@SKIP_FETCH_TOOLS=1 bash $(GARDENER_HACK_DIR)/test-cover.sh ./cmd/... ./pkg/...

.PHONY: test-clean
test-clean:
	@bash $(GARDENER_HACK_DIR)/test-cover-clean.sh

.PHONY: verify
verify: check format test

.PHONY: verify-extended
verify-extended: check-generate check format test-cov test-clean

.PHONY: clean
clean:
	@bash $(GARDENER_HACK_DIR)/clean.sh ./cmd/... ./pkg/...

.PHONY: test-integration
test-integration:
	@if [[ -f $(PWD)/$(CONTROL_KUBECONFIG) ]]; then export CONTROL_KUBECONFIG=$(PWD)/$(CONTROL_KUBECONFIG); elif [[ -f $(CONTROL_KUBECONFIG) ]]; then export CONTROL_KUBECONFIG=$(CONTROL_KUBECONFIG);else echo "No such file exists for CONTROL_KUBECONFIG";exit 1; fi; \
	if [[ -f $(PWD)/$(TARGET_KUBECONFIG) ]]; then export TARGET_KUBECONFIG=$(PWD)/$(TARGET_KUBECONFIG); elif [[ -f $(TARGET_KUBECONFIG) ]]; then export TARGET_KUBECONFIG=$(TARGET_KUBECONFIG);else echo "No such file exists for TARGET_KUBECONFIG";exit 1; fi; \
	if [[ -f "$(PWD)/$(MACHINECLASS_V1)" ]]; then export MACHINECLASS_V1="$(PWD)/$(MACHINECLASS_V1)"; elif [[ -f "$(MACHINECLASS_V1)" ]]; then export MACHINECLASS_V1="$(MACHINECLASS_V1)"; fi; \
	if [[ -f "$(PWD)/$(MACHINECLASS_V2)" ]]; then export MACHINECLASS_V2="$(PWD)/$(MACHINECLASS_V2)"; elif [[ -f "$(MACHINECLASS_V2)" ]]; then export MACHINECLASS_V2="$(MACHINECLASS_V2)"; fi; \
	export MC_CONTAINER_IMAGE=$(MC_IMAGE); \
	export MCM_CONTAINER_IMAGE=$(MCM_IMAGE); \
	export CONTROL_CLUSTER_NAMESPACE=$(CONTROL_NAMESPACE); \
	export MACHINE_CONTROLLER_MANAGER_DEPLOYMENT_NAME=$(MACHINE_CONTROLLER_MANAGER_DEPLOYMENT_NAME); \
	.ci/local_integration_test

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
