# Copyright (c) NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

.PHONY: all test  
.FORCE:

MODULE := github.com/NVIDIA/k8s-kata-manager
GOOS ?= linux
LDFLAGS ?= -ldflags "-s -w"
GO_CMD ?= go
GO_FMT ?= gofmt
GO_TEST_FLAGS ?= -race
# Use go.mod go version as a single source of truth of GO version.
GOLANG_VERSION := $(shell awk '/^go /{print $$2}' go.mod|head -n1)

DOCKER ?= docker

IMAGE_BUILD_CMD ?= docker build
IMAGE_BUILD_EXTRA_OPTS ?=
IMAGE_PUSH_CMD ?= docker push
CONTAINER_RUN_CMD ?= docker run
#BUILDER_IMAGE ?= golang:$(GO_VERSION)
BASE_IMAGE_FULL ?= debian:bullseye-slim
BASE_IMAGE_MINIMAL ?= gcr.io/distroless/base

VERSION := $(shell git describe --tags --dirty --always)

IMAGE_REGISTRY ?= nvidia
IMAGE_TAG_NAME ?= $(VERSION)
IMAGE_EXTRA_TAG_NAMES ?=

IMAGE_NAME := k8s-kata-manager
IMAGE_REPO := $(IMAGE_REGISTRY)/$(IMAGE_NAME)
IMAGE_TAG := $(IMAGE_REPO):$(IMAGE_TAG_NAME)
IMAGE_EXTRA_TAGS := $(foreach tag,$(IMAGE_EXTRA_TAG_NAMES),$(IMAGE_REPO):$(tag))

KUBECONFIG ?= ${HOME}/.kube/config
E2E_TEST_CONFIG ?=
E2E_PULL_IF_NOT_PRESENT ?= false

CMDS := $(patsubst ./cmd/%/,%,$(sort $(dir $(wildcard ./cmd/*/))))
CMD_TARGETS := $(patsubst %,cmd-%, $(CMDS))

CHECK_TARGETS := assert-fmt vet lint ineffassign misspell
MAKE_TARGETS := cmds build install fmt test coverage $(CHECK_TARGETS)

TARGETS := $(MAKE_TARGETS)
DOCKER_TARGETS := $(pathsubst %, docker-%, $(TARGETS))
.PHONY: $(TARGETS) $(DOCKER_TARGETS)

all: image

cmds: $(CMD_TARGETS)
$(CMD_TARGETS): cmd-%:
	@mkdir -p bin
	GOOS=$(GOOS) $(GO_CMD) build -v -o bin $(LDLAGS) $(COMMAND_BUILD_OPTIONS) $(MODULE)/cmd/$(*)

build:
	GOOS=$(GOOS) $(GO_CMD) build -v $(LDFLAGS) $(MODULE)/...

install:
	$(GO_CMD) install -v $(LDFLAGS) $(MODULE)/cmd/...

fmt:
	$(GO_CMD) list -f '{{.Dir}}' $(MODULE)/... \
		| xargs $(GO_FMT) -s -l -w

assert-fmt:
	$(GO_CMD) list -f '{{.Dir}}' $(MODULE)/... \
		| xargs $(GO_FMT) -s -l | ( grep -v /vendor/ || true ) > fmt.out
	@if [ -s fmt.out ]; then \
		echo "\nERROR: The following files are not formatted:\n"; \
		cat fmt.out; \
		rm fmt.out; \
		exit 1; \
	else \
		rm fmt.out; \
	fi

ineffassign:
	ineffassign $(MODULE)/...

lint:
# We use `go list -f '{{.Dir}}' $(MODULE)/...` to skip the `vendor` folder.
	$(GO_CMD) list -f '{{.Dir}}' $(MODULE)/... | xargs golint -set_exit_status

misspell:
	misspell $(MODULE)/...

vet:
	$(GO_CMD) vet $(MODULE)/...

COVERAGE_FILE := coverage.out
test: build
	$(GO_CMD) test -v -coverprofile=$(COVERAGE_FILE) $(MODULE)/...

coverage: test
	cat $(COVERAGE_FILE) | grep -v "_mock.go" > $(COVERAGE_FILE).no-mocks
	$(GO_CMD) tool cover -func=$(COVERAGE_FILE).no-mocks

# Targets used to build a golang devel container used in CI pipelines
.PHONY: .build-image .pull-build-image .push-build-image
BUILDIMAGE ?= k8s-kata-manager-devel
.build-image: Dockerfile.devel
	if [ x"$(SKIP_IMAGE_BUILD)" = x"" ]; then \
		$(DOCKER) build \
			--progress=plain \
			--build-arg GOLANG_VERSION="$(GOLANG_VERSION)" \
			--tag $(BUILDIMAGE) \
			-f $(^) \
			.; \
	fi

.pull-build-image:
	$(DOCKER) pull $(BUILDIMAGE)

.push-build-image:
	$(DOCKER) push $(BUILDIMAGE)

$(DOCKER_TARGETS): docker-%: .build-image
	@echo "Running 'make $(*)' in docker container $(BUILDIMAGE)"
	$(DOCKER) run \
		--rm \
		-e GOCACHE=/tmp/.cache \
		-v $(PWD):$(PWD) \
		-w $(PWD) \
		--user $$(id -u):$$(id -g) \
		$(BUILDIMAGE) \
			make $(*)

.PHONY: image
image:
	$(IMAGE_BUILD_CMD) -t $(IMAGE_TAG) \
		--build-arg GOLANG_VERSION=$(GOLANG_VERSION) \
		$(IMAGE_BUILD_EXTRA_OPTS) .

push:
	$(IMAGE_PUSH_CMD) $(IMAGE_TAG)
	$(IMAGE_PUSH_CMD) $(IMAGE_TAG)-minimal
	$(IMAGE_PUSH_CMD) $(IMAGE_TAG)-full
	for tag in $(IMAGE_EXTRA_TAGS); do \
	    $(IMAGE_PUSH_CMD) $$tag; \
	    $(IMAGE_PUSH_CMD) $$tag-minimal; \
	    $(IMAGE_PUSH_CMD) $$tag-full; \
	done

push-all: ensure-buildx yamls
	$(IMAGE_BUILDX_CMD) --push $(IMAGE_BUILD_ARGS) $(IMAGE_BUILD_ARGS_FULL)
	$(IMAGE_BUILDX_CMD) --push $(IMAGE_BUILD_ARGS) $(IMAGE_BUILD_ARGS_MINIMAL)
