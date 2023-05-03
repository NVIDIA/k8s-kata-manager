ARG BUILDER_IMAGE=golang:${GOLANG_VERSION}
FROM ${BUILDER_IMAGE} as builder

WORKDIR /workspace
# Copy the go source
COPY . .

# Build
RUN make build-operand GO_BUILD_ENV='CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH}'
RUN install -m 755 bin/k8s-operand /usr/local/bin/k8s-operand

ENTRYPOINT [ "/usr/local/bin/k8s-operand" ]