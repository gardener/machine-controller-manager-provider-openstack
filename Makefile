# Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

BINARY_PATH         := bin/
REPO_ROOT 			:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
COVERPROFILE        := test/output/coverprofile.out
IMAGE_REPOSITORY    := <link-to-image-repo>
IMAGE_TAG           := $(shell cat VERSION)
PROVIDER_NAME       := SampleProvider
PROJECT_NAME        := gardener
CONTROL_NAMESPACE  	:= default
CONTROL_KUBECONFIG 	:= dev/target-kubeconfig.yaml
TARGET_KUBECONFIG  	:= dev/target-kubeconfig.yaml

#########################################
# Rules for starting machine-controller locally
#########################################

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
			--v=3

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################

.PHONY: install-requirements
install-requirements:
	@go install -mod=vendor $(REPO_ROOT)/vendor/github.com/ahmetb/gen-crd-api-reference-docs
	@go install -mod=vendor $(REPO_ROOT)/vendor/github.com/golang/mock/mockgen
	@go install -mod=vendor $(REPO_ROOT)/vendor/github.com/onsi/ginkgo/ginkgo
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/install-requirements.sh

.PHONY: check-generate
check-generate:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check-generate.sh $(REPO_ROOT)

.PHONY: check
check:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/... ./test/...
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check-charts.sh ./charts

.PHONY: generate
generate:
	@env GOMODULE111=on go generate -mod=vendor ./pkg/...

.PHONY: format
format:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/format.sh ./cmd ./pkg ./test

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

#########################################
# Rules for re-vendoring
#########################################

.PHONY: revendor
revendor:
	@env GO111MODULE=on go mod vendor -v
	@env GO111MODULE=on go mod tidy -v
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/*
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/.ci/*

.PHONY: update-dependencies
update-dependencies:
	@env GO111MODULE=on go get -u

#########################################
# Rules for testing
#########################################

.PHONY: test-unit
test-unit:
	.ci/test

#########################################
# Rules for build/release
#########################################

.PHONY: release
release: build-local build docker-image docker-login docker-push rename-binaries

.PHONY: build-local
build-local:
	@env LOCAL_BUILD=1 .ci/build

.PHONY: build
build:
	@.ci/build

.PHONY: docker-image
docker-image:
	@docker build -t $(IMAGE_REPOSITORY):$(IMAGE_TAG) .

.PHONY: docker-login
docker-login:
	@gcloud auth login

.PHONY: docker-push
docker-push:
	@if ! docker images $(IMAGE_REPOSITORY) | awk '{ print $$2 }' | grep -q -F $(IMAGE_TAG); then echo "$(IMAGE_REPOSITORY) version $(IMAGE_TAG) is not yet built. Please run 'make docker-images'"; false; fi
	@gcloud docker -- push $(IMAGE_REPOSITORY):$(IMAGE_TAG)

.PHONY: rename-binaries
rename-binaries:
	@if [[ -f bin/machine-controller ]]; then cp bin/machine-controller machine-controller-darwin-amd64; fi
	@if [[ -f bin/rel/machine-controller ]]; then cp bin/rel/machine-controller machine-controller-linux-amd64; fi

.PHONY: clean
clean:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/clean.sh ./cmd/... ./pkg/... ./test/...
	@rm -rf bin/
	@rm -f *linux-amd64
	@rm -f *darwin-amd64
