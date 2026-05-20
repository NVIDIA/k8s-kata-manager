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

# Build a static busybox layout: one binary plus applet symlinks (sh, rm,
# ln, sleep, cat, ...) so PATH-resolved commands in init-container wrappers
# and lifecycle hooks keep working on the non-*-dev* distroless base.
FROM debian:trixie-slim AS shell
RUN apt-get update \
 && apt-get install -y --no-install-recommends busybox-static \
 && rm -rf /var/lib/apt/lists/* \
 && mkdir /busybox \
 && cp /bin/busybox /busybox/busybox \
 && /busybox/busybox --install -s /busybox

# This Dockerfile is also used to define the golang version used in this project
# This allows dependabot to manage this version in addition to other images.
FROM nvcr.io/nvidia/distroless/cc:v4.0.6

COPY --from=shell /busybox /busybox
USER 0:0
RUN ["/busybox/ln", "-s", "/busybox/sh", "/bin/sh"]
ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/busybox

ENTRYPOINT ["/busybox/sh", "-c", "while true; do sleep 86400; done"]
