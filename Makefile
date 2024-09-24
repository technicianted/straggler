##################################################
# Variables                                      #
##################################################


# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.25.0

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

KUBEBUILDER_ASSETS ?= $(LOCALBIN)/kubebuilder
$(KUBEBUILDER_ASSETS):
	mkdir -p $(KUBEBUILDER_ASSETS)

ENVTEST ?= $(LOCALBIN)/setup-envtest

VERSION		   ?= latest
IMAGE_REGISTRY ?= technicianted
IMAGE_FULL = $(IMAGE_REGISTRY)/stagger:$(VERSION)

ARCH       		?=amd64
CGO        		?=0
TARGET_OS  		?=$(shell go env GOOS)

GIT_VERSION = $(shell git describe --always --abbrev=7)
GIT_COMMIT  = $(shell git rev-list -1 HEAD)

GIT_VERSION := $(shell git rev-parse HEAD)
ifndef BUILD_VERSION
	BUILD_VERSION := $(GIT_VERSION)
endif

.PHONY=default binaries build push

##################################################
# Build                                          #
##################################################
GO_BUILD_VARS=CGO_ENABLED=$(CGO) GOOS=$(TARGET_OS) GOARCH=$(ARCH) GOPRIVATE=$(GOPRIVATE)

.PHONY: binaries

binaries:
	rm -rf bin/ > /dev/null 2>&1
	mkdir bin/
	$(GO_BUILD_VARS) go build -ldflags "-X stagger/pkg/version.Build=$(BUILD_VERSION)" -o bin/stagger ./cmd/stagger/

docker-build: binaries
	docker build . -t stagger

docker-push: docker-build
	docker tag stagger ${IMAGE_FULL}
	docker push ${IMAGE_FULL}

gen:
	go generate ./...

deps:
	go install go.uber.org/mock/mockgen@latest
