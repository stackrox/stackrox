# Store tooling in a location that does not affect the system.
GOBIN := $(CURDIR)/.gobin
export GOBIN
PATH := $(GOBIN):"$(PATH)"
export PATH

# Set to empty string to echo some command lines which are hidden by default.
SILENT ?= @

SHELL = env GOBIN=$(GOBIN) PATH=$(PATH) /bin/bash
BASE_DIR=$(CURDIR)

TAG := # make sure tag is never injectable as an env var

ifdef CI
ifneq ($(NIGHTLY_TAG),)
TAG := $(NIGHTLY_TAG)
else ifneq ($(RELEASE_TAG),)
TAG := $(RELEASE_TAG)
endif
endif

ifeq ($(TAG),)
TAG=$(shell git describe --tags --abbrev=10 --dirty --long)
endif

DEFAULT_IMAGE_REGISTRY := quay.io/stackrox-io
BUILD_IMAGE_VERSION=$(shell sed 's/\s*\#.*//' BUILD_IMAGE_VERSION)
BUILD_IMAGE := $(DEFAULT_IMAGE_REGISTRY)/apollo-ci:$(BUILD_IMAGE_VERSION)

ifeq ($(shell uname -ms),Darwin arm64)
	# TODO(ROX-12064) build these images in the CI pipeline
	# Uncomment the line below to enable native arm64 builder images
	# BUILD_IMAGE = quay.io/rhacs-eng/sandbox:apollo-ci-stackrox-build-0.3.49-arm64
	ARCH := aarch64
	GOARCH := arm64
else
	ARCH := x86_64
	GOARCH := amd64
endif

LOCAL_VOLUME_ARGS := -v$(CURDIR):/src:delegated -v $(GOPATH):/go:delegated
GOPATH_WD_OVERRIDES := -w /src -e GOPATH=/go
IMAGE_BUILD_FLAGS := -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=${GOARCH}
BUILD_FLAGS := CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH}
BUILD_CMD := go build -trimpath -ldflags="-X github.com/stackrox/scanner/v4/internal/version.Version=$(TAG)" -o image/scanner/bin/scanner ./cmd

#####################################################################
###### Binaries we depend on (need to be defined on top) ############
#####################################################################

OSSLS_BIN := $(GOBIN)/ossls
$(OSSLS_BIN): deps
	$(SILENT)echo "+ $@"
	$(SILENT)cd tools/ && go install github.com/stackrox/ossls

#########
## Tag ##
#########

.PHONY: tag
tag:
	$(SILENT)echo $(TAG)

############
## Images ##
############

.PHONY: images
images: scanner-image db-image

.PHONY: scanner-build-dockerized
scanner-build-dockerized: deps
	$(SILENT)echo "+ $@"
ifdef CI
	docker container create --name builder $(BUILD_IMAGE) $(BUILD_CMD)
	docker cp $(GOPATH) builder:/
	docker start -i builder
	docker cp builder:/go/src/github.com/stackrox/scanner/image/scanner/bin/scanner image/scanner/bin/scanner
else
	docker run $(IMAGE_BUILD_FLAGS) $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) $(BUILD_CMD)
endif

.PHONY: scanner-build-nodeps
scanner-build-nodeps:
	$(BUILD_FLAGS) $(BUILD_CMD)

.PHONY: $(CURDIR)/image/scanner/bundle.tar.gz
$(CURDIR)/image/scanner/bundle.tar.gz:
	$(CURDIR)/image/scanner/create-bundle.sh $(CURDIR)/image/scanner $(CURDIR)/image/scanner

.PHONY: $(CURDIR)/image/db/bundle.tar.gz
$(CURDIR)/image/db/bundle.tar.gz:
	$(CURDIR)/image/db/create-bundle.sh $(CURDIR)/image/db $(CURDIR)/image/db

.PHONY: scanner-image
scanner-image: scanner-build-dockerized ossls-notice $(CURDIR)/image/scanner/bundle.tar.gz
	$(SILENT)echo "+ $@"
	$(SILENT)docker build -t scanner:$(TAG) -f image/scanner/Dockerfile image/scanner

.PHONY: db-image
db-image: $(CURDIR)/image/db/bundle.tar.gz
	$(SILENT)echo "+ $@"
	$(SILENT)docker build -t scanner-db:$(TAG) --build-arg POSTGRESQL_ARCH=${ARCH} -f image/db/Dockerfile image/db

.PHONY: ossls-notice
ossls-notice: deps $(OSSLS_BIN)
	$(OSSLS_BIN) version
	$(OSSLS_BIN) audit --export image/scanner/THIRD_PARTY_NOTICES

###########
## Tests ##
###########

####################
## Generated Srcs ##
####################

###########
## Style ##
###########

deps: go.mod
	$(SILENT)echo "+ $@"
	$(SILENT)go mod tidy
ifdef CI
	$(SILENT)git diff --exit-code -- go.mod go.sum || { echo "go.mod/go.sum files were updated after running 'go mod tidy', run this command on your local machine and commit the results." ; exit 1 ; }
	go mod verify
endif
	$(SILENT)touch deps

###########
## Clean ##
###########

.PHONY: clean
clean: clean-image
	$(SILENT)echo "+ $@"

.PHONY: clean-image
clean-image:
	$(SILENT)echo "+ $@"
	$(SILENT)git clean -xdf image/scanner/bin

###########
## Tools ##
###########
