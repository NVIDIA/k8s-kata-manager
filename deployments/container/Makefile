# Copyright (c) 2024, NVIDIA CORPORATION.  All rights reserved.
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

BUILD_MULTI_ARCH_IMAGES ?= no
DOCKER ?= docker
MKDIR    ?= mkdir

##### Global variables #####
include $(CURDIR)/versions.mk

ifeq ($(IMAGE_NAME),)
IMAGE_NAME := $(REGISTRY)/$(NAME)
endif

IMAGE_TAG ?= $(VERSION)
IMAGE = $(IMAGE_NAME):$(IMAGE_TAG)

OUT_IMAGE_NAME ?= $(IMAGE_NAME)
OUT_IMAGE_VERSION ?= $(VERSION)
OUT_IMAGE = $(OUT_IMAGE_NAME):$(OUT_IMAGE_VERSION)

##### Public rules #####

IMAGE_TARGETS := image
BUILD_TARGETS := build-image
PUSH_TARGETS := push-image
TEST_TARGETS := test

.PHONY: $(IMAGE_TARGETS) $(TEST_TARGETS) $(BUILD_TARGETS) $(PUSH_TARGETS) 

ifneq ($(BUILD_MULTI_ARCH_IMAGES),true)
include $(CURDIR)/deployments/container/native-only.mk
else
include $(CURDIR)/deployments/container/multi-arch.mk
endif

build-image: DOCKERFILE = $(CURDIR)/deployments/container/Dockerfile

# Use a generic build target to build the relevant images
$(IMAGE_TARGETS):
	$(DOCKER) build --pull \
		$(DOCKER_BUILD_OPTIONS) \
		$(DOCKER_BUILD_PLATFORM_OPTIONS) \
		--tag $(IMAGE) \
		--build-arg GOLANG_VERSION="$(GOLANG_VERSION)" \
		--build-arg VERSION="$(VERSION)" \
		--build-arg GIT_COMMIT="$(GIT_COMMIT)" \
		$(if $(LABEL_IMAGE_SOURCE),--label "org.opencontainers.image.source=$(LABEL_IMAGE_SOURCE)",) \
		-f $(DOCKERFILE) \
		$(CURDIR)

build: build-image
REGCTL ?= regctl
$(PUSH_TARGETS):
	$(REGCTL) \
		image copy \
		$(IMAGE) $(OUT_IMAGE)

