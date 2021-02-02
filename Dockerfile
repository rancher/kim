ARG GOLANG=golang:1.15-alpine
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
    docker-cli \
 && docker --version

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
RUN make ORG=${ORG} PKG=${PKG} TAG=${TAG} bin/kim
RUN file bin/kim
RUN install -s bin/kim -m 0755 /usr/local/bin

FROM scratch AS release
COPY --from=build /usr/local/bin/kim /bin/kim
ENTRYPOINT ["kim"]
CMD ["--help"]
