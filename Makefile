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

include $(CURDIR)/versions.mk

##### Go variables #####
MODULE := github.com/NVIDIA/k8s-kata-manager
GOOS ?= linux
LDFLAGS ?= -ldflags "-s -w"
GO_CMD ?= go
GO_FMT ?= gofmt
GO_TEST_FLAGS ?= -race
# Use go.mod go version as a single source of truth of GO version.
GOLANG_VERSION := $(shell awk '/^go /{print $$2}' go.mod|head -n1)

##### General make targets #####
CMDS := $(patsubst ./cmd/%/,%,$(sort $(dir $(wildcard ./cmd/*/))))
CMD_TARGETS := $(patsubst %,cmd-%, $(CMDS))

CHECK_TARGETS := assert-fmt vet lint ineffassign misspell
MAKE_TARGETS := cmds build install fmt test coverage $(CHECK_TARGETS)

TARGETS := $(MAKE_TARGETS)
DOCKER_TARGETS := $(pathsubst %, docker-%, $(TARGETS))
.PHONY: $(TARGETS) $(DOCKER_TARGETS)

##### Container image variables #####
BUILD_MULTI_ARCH_IMAGES ?= no
DOCKER ?= docker
BUILDX =
ifeq ($(BUILD_MULTI_ARCH_IMAGES),true)
BUILDX = buildx
endif

ifeq ($(IMAGE_NAME),)
REGISTRY ?= nvidia
IMAGE_NAME := $(REGISTRY)/k8s-kata-manager
endif

IMAGE_VERSION := $(VERSION)

DIST ?= ubi8

# Note: currently there is no need to build images for different distributions,
# so the distribution is omitted from the tag
#IMAGE_TAG ?= $(IMAGE_VERSION)-$(DIST)
IMAGE_TAG ?= $(IMAGE_VERSION)
IMAGE = $(IMAGE_NAME):$(IMAGE_TAG)

OUT_IMAGE_NAME ?= $(IMAGE_NAME)
OUT_IMAGE_VERSION ?= $(IMAGE_VERSION)
#OUT_IMAGE_TAG = $(OUT_IMAGE_VERSION)-$(DIST)
OUT_IMAGE_TAG = $(OUT_IMAGE_VERSION)
OUT_IMAGE = $(OUT_IMAGE_NAME):$(OUT_IMAGE_TAG)

##### Container image make targets #####
# Note: currently there is no need to build images for different distributions.
IMAGE_BUILD_TARGETS := build-image
IMAGE_PUSH_TARGETS := push-image
IMAGE_PULL_TARGETS := pull-image
.PHONY: $(IMAGE_BUILD_TARGETS) $(IMAGE_PUSH_TARGETS)

###### Target definitions #####
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

##### Devel image build and push targets #####
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

##### Image build and push targets #####
build-image:
	DOCKER_BUILDKIT=1 \
		$(DOCKER) $(BUILDX) build --pull \
			$(DOCKER_BUILD_OPTIONS) \
			$(DOCKER_BUILD_PLATFORM_OPTIONS) \
			--tag $(IMAGE) \
			--build-arg BASE_DIST="$(DIST)" \
			--build-arg CUDA_VERSION="$(CUDA_VERSION)" \
			--build-arg GOLANG_VERSION="$(GOLANG_VERSION)" \
			--build-arg VERSION="$(VERSION)" \
			--build-arg CVE_UPDATES="$(CVE_UPDATES)" \
			--file Dockerfile \
			$(CURDIR)

push-image:
	$(DOCKER) tag "$(IMAGE)" "$(OUT_IMAGE)"
	$(DOCKER) push "$(OUT_IMAGE)"

pull-image:
	$(DOCKER) pull "$(IMAGE)"
