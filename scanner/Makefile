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

GOOS := $(DEFAULT_GOOS)
HOST_OS := linux
ifeq ($(shell uname -s),Darwin)
	HOST_OS := darwin
endif

GO_VERSION := $(firstword $(subst go version ,,$(shell go version)))
EXPECTED_GO_VERSION := $(shell cat ../EXPECTED_GO_VERSION)

GO_BUILD_FLAGS := CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH}
GO_BUILD_CMD := $(GO_BUILD_FLAGS) go build \
                 -trimpath \
                 -ldflags="-X github.com/stackrox/rox/scanner/internal/version.Version=$(TAG)"
GO_TEST_CMD := $(GO_BUILD_FLAGS) go test

DOCKERBUILD := $(CURDIR)/../scripts/docker-build.sh

# The namespace to use for targets that require it (E2E, certs, etc).
SCANNER_NAMESPACE ?= stackrox

.PHONY: all
all: images

###############################################################
###### Binaries we depend on (need to be defined on top) ######
###############################################################

OSSLS_BIN := $(GOBIN)/ossls
$(OSSLS_BIN): | scanner-go-check-ver
	@echo "+ $@"
	$(SILENT)cd tools/ && go install github.com/stackrox/ossls

#########
## Tag ##
#########

.PHONY: tag
tag:
	@echo $(TAG)

###############
## Go Builds ##
###############

build-cmd := $(GO_BUILD_CMD)
build-d   := bin
build-t   := $(addprefix $(build-d)/,$(notdir $(wildcard cmd/*)))

.PHONY: build
build: $(build-t)

.PHONY: $(build-t)
$(build-t): deps
	@echo "+ $@"
	$(build-cmd) -o $@ ./cmd/$(@F)

.PHONY: build-local
build-local:
	@echo "+ $@"
	$(MAKE) SILENT=$(SILENT) GOOS=${HOST_OS} build

.PHONY: build-local-scanner
build-local-scanner:
	@echo "+ $@"
	$(MAKE) SILENT=$(SILENT) GOOS=${HOST_OS} bin/scanner

.PHONY: build-local-scannerctl
build-local-scannerctl:
	@echo "+ $@"
	$(MAKE) SILENT=$(SILENT) GOOS=${HOST_OS} bin/scannerctl

.PHONY: build-local-updater
build-local-updater:
	@echo "+ $@"
	$(MAKE) SILENT=$(SILENT) GOOS=${HOST_OS} bin/updater

.PHONY: clean-build
clean-build:
	@echo "+ $@"
	$(SILENT)rm -rf bin/

############
## Images ##
############

image-prefix := scanner-v4

.PHONY: images
images: scanner-image db-image

image/scanner/bin/scanner: bin/scanner
	@echo "+ $@"
	cp $< $@

OSSLS_NOTICE_DEP := ossls-notice
ifdef CI
	OSSLS_NOTICE_DEP := ossls-notice-no-download
endif
.PHONY: scanner-image
scanner-image: image/scanner/bin/scanner copy-scripts $(OSSLS_NOTICE_DEP)
	@echo "+ $@"
	$(DOCKERBUILD) -t stackrox/$(image-prefix):$(TAG) -f image/scanner/Dockerfile image/scanner

.PHONY: db-image
db-image: copy-db-scripts copy-db-image-scripts copy-db-sigs
	@echo "+ $@"
	$(DOCKERBUILD) -t stackrox/$(image-prefix)-db:$(TAG) -f image/db/Dockerfile image/db

.PHONY: clean-image
clean-image:
	@echo "+ $@"
	$(SILENT)git clean -xdf image/scanner/bin

###########
## Tests ##
###########

###########
## Tools ##
###########

.PHONY: scanner-go-check-ver
scanner-go-check-ver:
ifdef CI
ifneq ($(GO_VERSION),$(EXPECTED_GO_VERSION))
	$(error 'Go version "$(GO_VERSION)" does not match the expected "$(EXPECTED_GO_VERSION)" version.')
endif
endif

deps: ../go.mod | scanner-go-check-ver
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

##################
## Certificates ##
##################

# Certificates vars
# =================

SCANNER_CERTS_NAMESPACE ?= $(SCANNER_NAMESPACE)

certs-d         := certs
certs-names     := scanner-v4 scanner-v4-db scannerctl
certs-namespace := $(SCANNER_CERTS_NAMESPACE)

# Targets.
certs-t := $(foreach name,$(certs-names),$(certs-d)/$(name)/.verified)

# Subject prefix.
certs-subj := /C=US/ST=North Carolina/L=Raleigh/O=Red Hat, Inc./OU=Scanner

# DNS name or CN of the current cert target.
cert-dns = $(notdir $(@D)).$(certs-namespace)

# Certificates rules
# ==================

.PHONY: certs clean-certs

certs: $(certs-t)

clean-certs:
	@echo "+ $@"
	$(SILENT)rm -rf certs/

# Generate keys.
#
.PRECIOUS: $(certs-d)/%/key.pem
$(certs-d)/%/key.pem:
	@echo "+ $@"
	$(SILENT)mkdir -p $(@D)
	$(SILENT) openssl genpkey  \
	    -algorithm RSA \
	    -out $@

# Generate root certificate authority.
#
$(certs-d)/ca/root.pem: $(certs-d)/ca/key.pem
	@echo "+ $@"
	$(SILENT) openssl req \
	    -new \
	    -x509 \
	    -nodes \
	    -days 398 \
	    -subj "$(certs-subj)/CN=RootCA" \
	    -key $< \
	    -out $@

# Generate a certificate signing request.
#
$(certs-d)/%/cert.csr: $(certs-d)/%/key.pem
	$(SILENT) openssl req \
	    -new \
	    -subj "$(certs-subj)/CN=$(cert-dns)" \
	    -key $< \
	    -out $@

# Copy CA cert and key.
#
.PRECIOUS: $(certs-d)/%/ca.pem
$(certs-d)/%/ca.pem: $(certs-d)/ca/root.pem
	@echo "+ $@"
	$(SILENT)ln -fs $(abspath $<) $@

.PRECIOUS: $(certs-d)/%/ca-key.pem
$(certs-d)/%/ca-key.pem: $(certs-d)/ca/key.pem
	@echo "+ $@"
	$(SILENT)ln -fs $(abspath $<) $@

# Generate a certificate.
#
.PRECIOUS: $(certs-d)/%/cert.pem
$(certs-d)/%/cert.pem: $(certs-d)/%/cert.csr $(certs-d)/%/ca.pem $(certs-d)/%/ca-key.pem
	@echo "+ $@"
	$(SILENT) openssl x509 \
	    -req \
	    -days 365 \
	    -in $< \
	    -CA $(@D)/ca.pem \
	    -CAkey $(@D)/ca-key.pem \
	    -CAcreateserial \
	    -sha256 \
	    -extfile <(echo "subjectAltName = DNS:$(cert-dns), DNS:localhost, DNS:127.0.0.1") \
	    -out $@

# Verify certs (loose sanity check).
#
$(certs-d)/%/.verified: $(certs-d)/%/cert.pem
	$(SILENT) openssl verify \
	    -CAfile $(@D)/ca.pem \
	    $<
	@echo >$@

#########
## E2E ##
#########

# E2E vars
# ========

SCANNER_E2E_IMAGE_TAG ?= $(TAG)
SCANNER_E2E_NAMESPACE ?= $(SCANNER_NAMESPACE)

e2e-d         := e2etests
e2e-chart     := $(e2e-d)/helmchart
e2e-namespace := $(SCANNER_E2E_NAMESPACE)
e2e-image-tag := $(SCANNER_E2E_IMAGE_TAG)
e2e-go-tag    := e2etests
e2e-timeout   := 20m

e2e-conf-files := db-postgresql.conf \
                  db-pg_hba.conf

e2e-certs := ca.pem \
             scanner-v4-key.pem \
             scanner-v4-cert.pem \
             scanner-v4-db-key.pem \
             scanner-v4-db-cert.pem

e2e-files-d      := $(e2e-chart)/files
e2e-conf-files-t := $(addprefix $(e2e-files-d)/,$(e2e-conf-files))
e2e-certs-t      := $(addprefix $(e2e-files-d)/,$(e2e-certs))

# E2E rules
# =========

.PHONY: e2e-deploy clean-e2e

ifeq ($(SILENT),@)
.SILENT: e2e-deploy $(e2e-conf-files-t)
endif

e2e-deploy: $(e2e-conf-files-t) $(e2e-certs-t)
	@echo "+ $@" $(e2e-image-tag)
	$(SILENT)-kubectl create namespace $(e2e-namespace)
	helm upgrade scanner-v4-e2e $(e2e-chart) \
	    --install \
	    --namespace $(e2e-namespace) \
	    --set image.tag="$(e2e-image-tag)"

.PHONY: e2e-run
e2e-run: deps
	@echo "+ $@"
	$(SILENT)$(GO_TEST_CMD) -tags $(e2e-go-tag) -count=1 -timeout=$(e2e-timeout) -v ./$(e2e-d)/...

.PHONY: e2e
e2e: e2e-deploy e2e-run

clean-e2e:
	@echo "+ $@"
	$(SILENT)rm -rf $(e2e-files-d)/*

# DB configuration files.
#
$(e2e-files-d)/db-%.conf: image/db/%.conf
	$(SILENT)mkdir -p $(@D)
	$(SILENT)cp $^ $@

# CA, cert and key from cert targets.
#
$(e2e-files-d)/%-key.pem: $(certs-d)/%/.verified
	$(SILENT)cp $(<D)/key.pem $@

$(e2e-files-d)/%-cert.pem: $(certs-d)/%/.verified
	$(SILENT)cp $(<D)/cert.pem $@

$(e2e-files-d)/ca.pem: $(certs-d)/ca/root.pem
	$(SILENT)cp $< $@

###########
## Clean ##
###########

.PHONY: clean
clean: clean-image clean-gobin clean-deps clean-e2e clean-certs clean-build
	@echo "+ $@"

.PHONY: clean-gobin
clean-gobin:
	@echo "+ $@"
	$(SILENT)rm -rf $(GOBIN)

.PHONY: clean-deps
clean-deps:
	@echo "+ $@"
	$(SILENT)rm -f deps
