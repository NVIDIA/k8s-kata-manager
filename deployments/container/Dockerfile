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

ARG GOLANG_VERSION=x.x.x
ARG VERSION="N/A"

FROM nvcr.io/nvidia/cuda:12.9.1-base-ubi9 as build

RUN yum install -y wget make git gcc

ARG GOLANG_VERSION=0.0.0
RUN set -eux; \
    \
    arch="$(uname -m)"; \
    case "${arch##*-}" in \
        x86_64 | amd64) ARCH='amd64' ;; \
        ppc64el | ppc64le) ARCH='ppc64le' ;; \
        aarch64 | arm64) ARCH='arm64' ;; \
        *) echo "unsupported architecture" ; exit 1 ;; \
    esac; \
    wget -nv -O - https://storage.googleapis.com/golang/go${GOLANG_VERSION}.linux-${ARCH}.tar.gz \
    | tar -C /usr/local -xz

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

WORKDIR /build
COPY . .

RUN mkdir /artifacts
ARG VERSION="N/A"
ARG GIT_COMMIT="unknown"
RUN make PREFIX=/artifacts cmds

FROM nvcr.io/nvidia/distroless/go:v3.1.11-dev

USER 0:0

SHELL ["/busybox/sh", "-c"]
RUN ln -s /busybox/sh /bin/sh
COPY --from=build /artifacts/k8s-kata-manager /usr/bin/k8s-kata-manager

LABEL version="${VERSION}"
LABEL release="N/A"
LABEL vendor="NVIDIA"
LABEL io.k8s.display-name="NVIDIA Kata Manager for Kubernetes"
LABEL name="NVIDIA Kata Manager for Kubernetes"
LABEL summary="NVIDIA Kata Manager for Kubernetes"
LABEL description="See summary"

ENTRYPOINT [ "/usr/bin/k8s-kata-manager" ]
