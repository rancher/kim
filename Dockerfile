ARG GOLANG=golang:1.15-alpine
FROM ${GOLANG} AS base
RUN apk --no-cache add \
    file \
    gcc \
    git \
    libseccomp-dev \
    libselinux-dev \
    make \
    musl-dev \
    protobuf-dev \
    protoc
RUN GO111MODULE=on go get github.com/gogo/protobuf/protoc-gen-gofast@v1.3.2
COPY . /go/src/kim
WORKDIR /go/src/kim

FROM base AS dapper
RUN apk --no-cache add docker-cli
ENV DAPPER_ENV GOLANG GODEBUG GOARCH GOOS ORG TAG DRONE_TAG DRONE_BUILD_EVENT
ARG DAPPER_HOST_ARCH
ENV GOARCH $DAPPER_HOST_ARCH
ENV DAPPER_SOURCE /go/src/kim
ENV DAPPER_OUTPUT ./dist ./bin
ENV DAPPER_DOCKER_SOCKET true
ENV DAPPER_TARGET dapper
ENV DAPPER_RUN_ARGS "--privileged --network host -v kim-pkg:/go/pkg -v kim-cache:/root/.cache/go-build"
RUN go version

FROM base AS build
RUN go mod vendor
RUN go generate -x
ARG ORG=rancher
ARG PKG=github.com/rancher/kim
ARG TAG=0.0.0-dev
RUN make bin/kim ORG=${ORG} PKG=${PKG} TAG=${TAG}
RUN file bin/kim
RUN install -s bin/kim -m 0755 /usr/local/bin

FROM scratch AS release
COPY --from=build /usr/local/bin/kim /bin/kim
ENTRYPOINT ["kim"]
CMD ["--help"]
