ARG BUILDER_IMAGE=golang:${GOLANG_VERSION}
FROM ${BUILDER_IMAGE} as builder

WORKDIR /workspace
# Copy the go source
COPY . .

# Build
RUN make build GO_BUILD_ENV='CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH}'