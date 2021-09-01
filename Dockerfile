ARG GOLANG=golang:1.16-alpine3.14
FROM ${GOLANG} AS base
RUN set -x \
 && apk --no-cache add \
    binutils \
    file \
    git \
    make \
 && git --version

FROM base AS docker
RUN set -x \
 && apk --no-cache add \
    curl \
    docker-cli \
 && docker --version
ARG BUILDX_RELEASE=v0.5.1
RUN set -x \
 && export BUILDX_ARCH=$(go env GOARCH) \
 && if [ "${BUILDX_ARCH}" = "arm" ]; then export BUILDX_ARCH="arm-v7"; fi \
 && mkdir -p /usr/libexec/docker/cli-plugins/ \
 && curl -fsSL --output /usr/libexec/docker/cli-plugins/docker-buildx \
    "https://github.com/docker/buildx/releases/download/${BUILDX_RELEASE}/buildx-${BUILDX_RELEASE}.linux-${BUILDX_ARCH}" \
 && file /usr/libexec/docker/cli-plugins/docker-buildx \
 && chmod -v +x /usr/libexec/docker/cli-plugins/docker-buildx \
 && docker buildx version

FROM base AS gobuild
RUN apk --no-cache add \
    gcc \
    libseccomp-dev \
    libselinux-dev \
    musl-dev \
    protobuf-dev \
    protoc
RUN GO111MODULE=on go get github.com/gogo/protobuf/protoc-gen-gofast@v1.3.2
COPY . /go/src/kim
WORKDIR /go/src/kim

FROM gobuild AS build
RUN go mod vendor
RUN go generate -x
ARG ORG=rancher
ARG PKG=github.com/rancher/kim
ARG TAG=0.0.0-dev+possible
ARG GOOS=linux
ARG GOARCH=amd64
RUN make GOOS=${GOOS} GOARCH=${GOARCH} ORG=${ORG} PKG=${PKG} TAG=${TAG} bin/kim
RUN file bin/kim
RUN install -s bin/kim -m 0755 /usr/local/bin || cp -vf bin/kim /usr/local/bin/

FROM scratch AS release
COPY --from=build /usr/local/bin/kim /bin/kim
ENTRYPOINT ["kim"]
CMD ["--help"]
