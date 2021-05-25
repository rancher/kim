ifeq ($(GOARCH),)
GOARCH := $(shell go env GOARCH)
endif
GOARM := 7

ifeq ($(GOOS),)
GOOS := $(shell go env GOOS)
endif

DOCKER_BUILDKIT ?= 1
DOCKER_BUILDX   ?= docker buildx
DOCKER_IMAGE    ?= docker image
DOCKER_MANIFEST ?= docker manifest

ORG ?= rancher
PKG ?= github.com/rancher/kim
TAG ?= $(shell git describe --tags --always)
IMG := $(ORG)/kim:$(subst +,-,$(TAG))
REG ?= docker.io

ifeq ($(GO_BUILDTAGS),)
GO_BUILDTAGS := static_build,netgo,osusergo
#ifeq ($(GOOS),linux)
#GO_BUILDTAGS := $(GO_BUILDTAGS),seccomp,selinux
#endif
endif

GO_LDFLAGS ?= -w -extldflags=-static
GO_LDFLAGS += -X $(PKG)/pkg/version.GitCommit=$(shell git rev-parse HEAD)
GO_LDFLAGS += -X $(PKG)/pkg/version.Version=$(TAG)
GO_LDFLAGS += -X $(PKG)/pkg/server.DefaultAgentImage=$(REG)/$(ORG)/kim

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
	DOCKER_BUILDKIT=$(DOCKER_BUILDKIT) $(DOCKER_IMAGE) build \
		--build-arg GOLANG=$(GOLANG) \
		--build-arg ORG=$(ORG) \
		--build-arg PKG=$(PKG) \
		--build-arg TAG=$(TAG) \
		--tag $(IMG) \
		--tag $(IMG)-$(GOARCH) \
	.

.PHONY: image-dist
image-dist:
	$(DOCKER_BUILDX) build \
		--file Dockerfile.dist \
		--platform $(GOOS)/$(GOARCH) \
		--build-arg GOARCH=$(GOARCH) \
		--build-arg GOOS=$(GOOS) \
		--tag $(IMG)-$(GOARCH) \
	.

.PHONY: image-push
image-push:
	$(DOCKER_IMAGE) push $(IMG)-$(GOARCH)

.PHONY: image-manifest
image-manifest:
	DOCKER_CLI_EXPERIMENTAL=enabled $(DOCKER_MANIFEST) create --amend \
		$(IMG) \
		$(IMG)-$(GOARCH)
	DOCKER_CLI_EXPERIMENTAL=enabled $(DOCKER_MANIFEST) push \
		$(IMG)

.PHONY: image-manifest-all
image-manifest-all:
	DOCKER_CLI_EXPERIMENTAL=enabled $(DOCKER_MANIFEST) create --amend \
		$(IMG) \
		$(IMG)-amd64 \
		$(IMG)-arm64 \
		$(IMG)-arm \
		$(IMG)-ppc64le \
		$(IMG)-s390x
	DOCKER_CLI_EXPERIMENTAL=enabled $(DOCKER_MANIFEST) annotate \
		--arch arm \
		--variant v$(GOARM) \
		$(IMG) \
		$(IMG)-arm
	DOCKER_CLI_EXPERIMENTAL=enabled $(DOCKER_MANIFEST) push \
		$(IMG)

# use this target to test drone builds locally
.PHONY: drone-local
drone-local:
	DRONE_TAG=v0.0.0-dev.0+drone drone exec --trusted

.PHONY: dogfood
dogfood: build
	DOCKER_IMAGE="./bin/kim image" make image

.PHONY: symlinks
symlinks: build
	ln -nsf $(notdir $(BIN)) $(dir $(BIN))./kubectl-builder
	ln -nsf $(notdir $(BIN)) $(dir $(BIN))./kubectl-image

.PHONY: test-image-build-with-secret
test-image-build-with-secret: $(BIN)
	make KIM=$(shell pwd)/$(BIN) -C testdata/image-build-with-secret/.

.PHONY: test-image-build-with-ssh
test-image-build-with-ssh: $(BIN)
	make KIM=$(shell pwd)/$(BIN) -C testdata/image-build-with-ssh/.
