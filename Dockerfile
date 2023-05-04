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

ARG BASE_DIST=ubi8
ARG CUDA_VERSION
ARG GOLANG_VERSION
ARG VERSION="N/A"

FROM golang:${GOLANG_VERSION} as builder

WORKDIR /build
# Copy the go source
COPY . .
# Build
RUN make cmds GO_BUILD_ENV='CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH}'

FROM nvcr.io/nvidia/cuda:${CUDA_VERSION}-base-${BASE_DIST}
COPY --from=builder /build/bin/k8s-operand /usr/local/bin/k8s-operand

LABEL version="${VERSION}"
LABEL release="N/A"
LABEL vendor="NVIDIA"
LABEL io.k8s.display-name="NVIDIA Kata Manager for Kubernetes"
LABEL name="NVIDIA Kata Manager for Kubernetes"
LABEL summary="NVIDIA Kata Manager for Kubernetes"
LABEL description="See summary"

ENTRYPOINT [ "/usr/local/bin/k8s-operand" ]
