ifeq ($(GOARCH),)
GOARCH := $(shell go env GOARCH)
endif
ifeq ($(GOARCH),arm)
ifeq ($(GOARM),)
GOARM := 7
endif
endif

ifeq ($(GOOS),)
GOOS := $(shell go env GOOS)
endif

DOCKER_BUILDKIT ?= 1

ORG ?= rancher
PKG ?= github.com/rancher/kim
TAG ?= $(shell git describe --tags --always)
IMG := $(ORG)/kim:$(subst +,-,$(TAG))

ifeq ($(GO_BUILDTAGS),)
GO_BUILDTAGS := static_build,netgo,osusergo
#ifeq ($(GOOS),linux)
#GO_BUILDTAGS := $(GO_BUILDTAGS),seccomp,selinux
#endif
endif

GO_LDFLAGS ?= -w -extldflags=-static
GO_LDFLAGS += -X $(PKG)/pkg/version.GitCommit=$(shell git rev-parse HEAD)
GO_LDFLAGS += -X $(PKG)/pkg/version.Version=$(TAG)
GO_LDFLAGS += -X $(PKG)/pkg/server.DefaultAgentImage=docker.io/$(ORG)/kim

GO ?= go
GOLANG ?= golang:1.16-alpine3.12

BIN ?= bin/kim
ifeq ($(GOOS),windows)
BINSUFFIX := .exe
endif
BIN := $(BIN)$(BINSUFFIX)

.PHONY: build image package publish validate
build: $(BIN)
package: | dist image
publish: | image image-push image-manifest
validate:

.PHONY: $(BIN)
$(BIN):
	$(GO) build -ldflags "$(GO_LDFLAGS)" -tags "$(GO_BUILDTAGS)" -o $@ .

.PHONY: dist
dist:
	@mkdir -p dist/artifacts
	@make GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) BIN=dist/artifacts/kim-$(GOOS)-$(GOARCH)$(BINSUFFIX) -C .

.PHONY: clean
clean:
	rm -rf bin dist vendor

.PHONY: image
image:
	DOCKER_BUILDKIT=$(DOCKER_BUILDKIT) docker build \
		--build-arg GOLANG=$(GOLANG) \
		--build-arg ORG=$(ORG) \
		--build-arg PKG=$(PKG) \
		--build-arg TAG=$(TAG) \
		--tag $(IMG) \
		--tag $(IMG)-$(GOARCH) \
	.

.PHONY: image-push
image-push:
	docker push $(IMG)-$(GOARCH)

.PHONY: image-manifest
image-manifest:
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest create --amend \
		$(IMG) \
		$(IMG)-$(GOARCH)
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest push \
		$(IMG)

.PHONY: image-manifest-all
image-manifest-all:
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest create --amend \
		$(IMG) \
		$(IMG)-amd64 \
		$(IMG)-arm64 \
		$(IMG)-arm
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest push \
		$(IMG)

# use this target to test drone builds locally
.PHONY: drone-local
drone-local:
	DRONE_TAG=v0.0.0-dev.0+drone drone exec --trusted
