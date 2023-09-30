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

BUILD_FLAGS := CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH}
BUILD_CMD := $(BUILD_FLAGS) go build \
                 -trimpath \
                 -ldflags="-X github.com/stackrox/stackrox/scanner/internal/version.Version=$(TAG)" \
                 -o image/scanner/bin/scanner \
                 ./cmd/scanner

DOCKERBUILD := $(CURDIR)/../scripts/docker-build.sh

.PHONY: all
all: deps scanner-go-build images

###############################################################
###### Binaries we depend on (need to be defined on top) ######
###############################################################

OSSLS_BIN := $(GOBIN)/ossls
$(OSSLS_BIN):
	@echo "+ $@"
	$(SILENT)cd tools/ && go install github.com/stackrox/ossls

#########
## Tag ##
#########

.PHONY: tag
tag:
	@echo $(TAG)

############
## Images ##
############

.PHONY: images
images: scanner-image db-image

.PHONY: scanner-go-build scanner-go-check-ver
scanner-go-check-ver:
ifneq ($(GO_VERSION),$(EXPECTED_GO_VERSION))
	$(error 'Go version "$(GO_VERSION)" does not match the expected "$(EXPECTED_GO_VERSION)" version.')
endif

scanner-go-build: deps
	@echo "+ $@"
	$(BUILD_CMD)

.PHONY: scanner-go-build-local
scanner-go-build-local:
	@echo "+ $@"
	$(SILENT)$(MAKE) GOOS=${HOST_OS} scanner-go-build

OSSLS_NOTICE_DEP := ossls-notice
ifdef CI
	OSSLS_NOTICE_DEP := ossls-notice-no-download
endif
.PHONY: scanner-image
scanner-image: scanner-go-build copy-scripts $(OSSLS_NOTICE_DEP)
	@echo "+ $@"
	@echo "Building scanner:$(TAG)"
	$(SILENT)$(DOCKERBUILD) -t stackrox/scanner-v4:$(TAG) -f image/scanner/Dockerfile image/scanner

.PHONY: db-image
db-image: copy-db-scripts copy-db-image-scripts copy-db-sigs
	@echo "+ $@"
	@echo "Building scanner-db:$(TAG)"
	$(SILENT)$(DOCKERBUILD) -t stackrox/scanner-v4-db:$(TAG) -f image/db/Dockerfile image/db

###########
## Tests ##
###########

###########
## Tools ##
###########

deps: ../go.mod scanner-go-check-ver
	@echo "+ $@"
	$(SILENT)go mod tidy
ifdef CI
	$(SILENT)git diff --exit-code -- go.mod go.sum || { \
		echo "go.mod/go.sum files were updated after running 'go mod tidy': run this command on your local machine and commit the results." ; \
		exit 1 ; \
	}
endif
	$(SILENT)go mod verify
	$(SILENT)touch $@

.PHONY: ossls-notice-no-download
ossls-notice-no-download: deps
	@echo "+ $@"
	$(SILENT)ossls version
	$(SILENT)ossls audit --export image/scanner/THIRD_PARTY_NOTICES

.PHONY: ossls-notice
ossls-notice: deps $(OSSLS_BIN)
	@echo "+ $@"
	$(SILENT)$(OSSLS_BIN) version
	$(SILENT)$(OSSLS_BIN) audit --export image/scanner/THIRD_PARTY_NOTICES

SCRIPTS := restore-all-dir-contents import-additional-cas save-dir-contents
.PHONY: copy-scripts
copy-scripts: $(addprefix ../image/rhel/static-bin/,$(SCRIPTS))
	@echo "+ $@"
	$(SILENT)cp $^ image/scanner/scripts

.PHONY: copy-db-scripts
copy-db-scripts: ../image/postgres/download.sh
	@echo "+ $@"
	$(SILENT)cp $^ image/db

.PHONY: copy-db-image-scripts
copy-db-image-scripts: ../image/postgres/scripts/init-entrypoint.sh
	@echo "+ $@"
	$(SILENT)cp $^ image/db/scripts

.PHONY: copy-db-sigs
copy-db-sigs: ../image/postgres/signatures
	@echo "+ $@"
	$(SILENT)cp -R $^ image/db/signatures

#########
## E2E ##
#########

# E2E vars
# --------

SCANNER_E2E_TAG       ?= $(TAG)
SCANNER_E2E_NAMESPACE ?= stackrox

e2e-chart     := e2etests/helmchart
e2e-namespace := $(SCANNER_E2E_NAMESPACE)
e2e-tag       := $(SCANNER_E2E_TAG)

# E2E helm files
# --------------

e2e-files   := db-postgresql.conf \
               db-pg_hba.conf

e2e-files-t := $(addprefix $(e2e-chart)/files/,$(e2e-files))

# E2E rules
# ---------

.PHONY: e2e-deploy e2e-clean

ifdef SILENT
.SILENT: e2e-deploy e2e $(e2e-files-t)
endif

$(e2e-chart)/files/db-%.conf: image/db/%.conf
	mkdir -p $(@D)
	cp $^ $@

e2e-deploy: $(e2e-files-t)
	@echo "+ $@"
	-kubectl create namespace $(e2e-namespace)
	helm upgrade scanner-v4-e2e $(e2e-chart) \
	    --install \
	    --namespace $(e2e-namespace) \
	    --set image.tag="$(e2e-tag)"

e2e-clean:
	rm -rf $(e2e-chart)/files/*

###########
## Clean ##
###########

.PHONY: clean
clean: clean-image clean-gobin clean-deps
	@echo "+ $@"

.PHONY: clean-image
clean-image:
	@echo "+ $@"
	$(SILENT)git clean -xdf image/scanner/bin

.PHONY: clean-gobin
clean-gobin:
	@echo "+ $@"
	$(SILENT)rm -rf $(GOBIN)

.PHONY: clean-deps
clean-deps:
	@echo "+ $@"
	$(SILENT)rm -f deps
