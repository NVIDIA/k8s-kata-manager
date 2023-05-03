.PHONY: all test  
.FORCE:

GO_CMD ?= go
GO_FMT ?= gofmt
GO_TEST_FLAGS ?= -race
# Use go.mod go version as a single source of truth of GO version.
GO_VERSION := $(shell awk '/^go /{print $$2}' go.mod|head -n1)

IMAGE_BUILD_CMD ?= docker build
IMAGE_BUILD_EXTRA_OPTS ?=
IMAGE_PUSH_CMD ?= docker push
CONTAINER_RUN_CMD ?= docker run
BUILDER_IMAGE ?= golang:$(GO_VERSION)
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

LDFLAGS ?=

all: image

build-all:
	@mkdir -p bin
	$(GO_CMD) build -v -o bin $(LDFLAGS) ./cmd/...

build-operand:
	@mkdir -p bin
	$(GO_CMD) build -v -o bin $(LDFLAGS) ./cmd/k8s-operand/...

build-cli:
	@mkdir -p bin
	$(GO_CMD) build -v -o bin $(LDFLAGS) ./cmd/kata-manager/...

install:
	$(GO_CMD) install -v $(LDFLAGS) ./cmd/...

.PHONY: image
image:
	$(IMAGE_BUILD_CMD) -t $(IMAGE_TAG) \
		--build-arg BUILDER_IMAGE=$(BUILDER_IMAGE) \
		$(IMAGE_BUILD_EXTRA_OPTS) .

gofmt:
	@$(GO_FMT) -w -l $$(find . -name '*.go')

gofmt-verify:
	@out=`$(GO_FMT) -w -l -d $$(find . -name '*.go')`; \
	if [ -n "$$out" ]; then \
	    echo "$$out"; \
	    exit 1; \
	fi

test:
	$(GO_CMD) test -covermode=atomic -coverprofile=coverage.out ./cmd/... ./pkg/...

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
