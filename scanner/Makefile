include $(CURDIR)/../make/env.mk

# Store tooling in a location that does not affect the system.
GOBIN := $(CURDIR)/.gobin
PATH := $(GOBIN):"$(PATH)"
export PATH

# Set to empty string to echo some command lines which are hidden by default.
SILENT ?= @

TAG := # make sure tag is never injectable as an env var

ifdef CI
ifneq ($(NIGHTLY_TAG),)
	TAG := $(NIGHTLY_TAG)
else ifneq ($(RELEASE_TAG),)
	TAG := $(RELEASE_TAG)
endif
endif

ifeq ($(TAG),)
	TAG := $(shell $(MAKE) -C ../ --quiet --no-print-directory tag)
endif

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

GOOS := $(DEFAULT_GOOS)
HOST_OS := linux
ifeq ($(UNAME_S),Darwin)
	HOST_OS := darwin
endif

GO_VERSION := $(firstword $(subst go version ,,$(shell go version)))
EXPECTED_GO_VERSION := $(shell cat ../EXPECTED_GO_VERSION)
ifneq ($(GO_VERSION),$(EXPECTED_GO_VERSION))
$(error 'Go version "$(GO_VERSION)" does not match the expected "$(EXPECTED_GO_VERSION)" version.')
endif

BUILD_FLAGS := CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH}
BUILD_CMD := go build -trimpath -ldflags="-X github.com/stackrox/stackrox/scanner/internal/version.Version=$(TAG)" -o image/scanner/bin/scanner ./cmd/scanner

DOCKERBUILD := $(CURDIR)/../scripts/docker-build.sh

.PHONY: all
all: deps scanner-go-build images

###############################################################
###### Binaries we depend on (need to be defined on top) ######
###############################################################

OSSLS_BIN := $(GOBIN)/ossls
$(OSSLS_BIN):
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

.PHONY: scanner-go-build
scanner-go-build: deps
	$(SILENT)echo "+ $@"
	$(BUILD_FLAGS) $(BUILD_CMD)

.PHONY: scanner-go-build-local
scanner-go-build-local:
	$(SILENT)echo "+ $@"
	$(SILENT)$(MAKE) GOOS=${HOST_OS} scanner-go-build

OSSLS_NOTICE_DEP := ossls-notice
ifdef CI
	OSSLS_NOTICE_DEP := ossls-notice-no-download
endif
.PHONY: scanner-image
scanner-image: scanner-go-build copy-scripts $(OSSLS_NOTICE_DEP)
	$(SILENT)echo "+ $@"
	$(SILENT)echo "Building scanner:$(TAG)"
	$(SILENT)$(DOCKERBUILD) -t stackrox/scanner-v4:$(TAG) -f image/scanner/Dockerfile image/scanner

.PHONY: db-image
db-image: copy-db-scripts copy-db-image-scripts copy-db-sigs
	$(SILENT)echo "+ $@"
	$(SILENT)echo "Building scanner-db:$(TAG)"
	$(SILENT)$(DOCKERBUILD) -t stackrox/scanner-v4-db:$(TAG) -f image/db/Dockerfile image/db

###########
## Tests ##
###########

###########
## Tools ##
###########

deps: ../go.mod
	$(SILENT)echo "+ $@"
	$(SILENT)go mod tidy
ifdef CI
	$(SILENT)git diff --exit-code -- go.mod go.sum || { \
		echo "go.mod/go.sum files were updated after running 'go mod tidy': run this command on your local machine and commit the results." ; \
		exit 1 ; \
	}
endif
	$(SILENT)go mod verify
	$(SILENT)touch deps

.PHONY: ossls-notice-no-download
ossls-notice-no-download: deps
	$(SILENT)echo "+ $@"
	$(SILENT)ossls version
	$(SILENT)ossls audit --export image/scanner/THIRD_PARTY_NOTICES

.PHONY: ossls-notice
ossls-notice: deps $(OSSLS_BIN)
	$(SILENT)echo "+ $@"
	$(SILENT)$(OSSLS_BIN) version
	$(SILENT)$(OSSLS_BIN) audit --export image/scanner/THIRD_PARTY_NOTICES

SCRIPTS := restore-all-dir-contents import-additional-cas save-dir-contents
.PHONY: copy-scripts
copy-scripts: $(addprefix ../image/rhel/static-bin/,$(SCRIPTS))
	$(SILENT)echo "+ $@"
	$(SILENT)cp $^ image/scanner/scripts

.PHONY: copy-db-scripts
copy-db-scripts: ../image/postgres/download.sh
	$(SILENT)echo "+ $@"
	$(SILENT)cp $^ image/db

.PHONY: copy-db-image-scripts
copy-db-image-scripts: ../image/postgres/scripts/init-entrypoint.sh
	$(SILENT)echo "+ $@"
	$(SILENT)cp $^ image/db/scripts

.PHONY: copy-db-sigs
copy-db-sigs: ../image/postgres/signatures
	$(SILENT)echo "+ $@"
	$(SILENT)cp -R $^ image/db/signatures

###########
## Clean ##
###########

.PHONY: clean
clean: clean-image clean-gobin clean-deps
	$(SILENT)echo "+ $@"

.PHONY: clean-image
clean-image:
	$(SILENT)echo "+ $@"
	$(SILENT)git clean -xdf image/scanner/bin

.PHONY: clean-gobin
clean-gobin:
	$(SILENT)echo "+ $@"
	$(SILENT)rm -rf $(GOBIN)

.PHONY: clean-deps
clean-deps:
	$(SILENT)echo "+ $@"
	$(SILENT)rm -f deps
