include $(CURDIR)/../make/env.mk

# Store tooling in a location that does not affect the system.
GOBIN := $(CURDIR)/.gobin
PATH := $(GOBIN):"$(PATH)"
export PATH

# Set to empty string to echo some command lines which are hidden by default.
SILENT ?= @

ifeq ($(TAG),)
	TAG := $(shell $(MAKE) -C ../ --quiet --no-print-directory tag)
endif

# Set expiration on Quay.io for non-release tags.
ifeq ($(findstring x,$(TAG)),x)
QUAY_TAG_EXPIRATION=13w
else
QUAY_TAG_EXPIRATION=never
endif

# CGO is disabled for Scanner V4 builds because:
#
# 1. Local builds aren't dockerized. And, upon packaging the locally-built scanner in a container when e.g. host
# OS's glibc does not match version in target container, scanner refuses to start with errors looking like this:
#    $ /usr/local/bin/scanner --help
#    /usr/local/bin/scanner: /lib64/libc.so.6: version `GLIBC_2.32' not found (required by /usr/local/bin/scanner)
#    /usr/local/bin/scanner: /lib64/libc.so.6: version `GLIBC_2.34' not found (required by /usr/local/bin/scanner)
#
# 2. GitHub builds for arm64, ppc64le and s390x need support for cross-compilation which, when unavailable, makes
# builds fail with errors like
#    gcc_arm64.S:30: Error: no such instruction: `stp x29,x30,[sp,'
#    gcc_arm64.S:34: Error: too many memory references for `mov'
#
# This flag must be overridden in commercial/downstream builds in arguments to `make`.
#    make CGO_ENABLED=1 blah          # Correct - flips it to 1.
#    CGO_ENABLED=1 make blah          # Wrong! This has no effect. Don't do this.
#    export CGO_ENABLED=1; make blah  # Wrong! This has no effect. Don't do this.
# Similarly, don't do the following in Dockerfile
#    ENV CGO_ENABLED=1                # Wrong!..
#    RUN make blah                    # because it has no effect. Don't do this.
# If you need to separate the declaration and the usage in Dockerfile, do this instead
#    ARG CGO_ENABLED=1
#    RUN make CGO_ENABLED=$CGO_ENABLED blah
CGO_ENABLED = 0
RACE_FLAG =
ifeq ($(RACE),true)
	# cgo is required for data race detection.
	CGO_ENABLED = 1
	RACE_FLAG = -race
endif

HOST_OS := linux
ifeq ($(shell uname -s),Darwin)
	HOST_OS := darwin
endif
GOOS := $(HOST_OS)

GO_BUILD_FLAGS = CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH)
GO_BUILD_CMD   = $(GO_BUILD_FLAGS) go build \
                   -trimpath \
                   -ldflags="-X github.com/stackrox/rox/scanner/internal/version.Version=$(TAG) -X github.com/stackrox/rox/scanner/internal/version.VulnerabilityVersion=$(SCANNER_VULNERABILITY_VERSION)" \
                   -tags=$(GOTAGS) \
                   $(RACE_FLAG)
GO_TEST_CMD    = $(GO_BUILD_FLAGS) go test

DOCKERBUILD := $(CURDIR)/../scripts/docker-build.sh

DOCKERBUILD_ARGS = --build-arg LABEL_VERSION=$(TAG) \
                   --build-arg LABEL_RELEASE=$(TAG) \
                   --build-arg QUAY_TAG_EXPIRATION=$(QUAY_TAG_EXPIRATION)

DB_DOCKERBUILD_ARGS =
ifeq ($(GOARCH),s390x)
	DB_DOCKERBUILD_ARGS = \
		--build-arg="RPMS_REGISTRY=quay.io" \
		--build-arg="RPMS_BASE_IMAGE=centos/centos" \
		--build-arg="RPMS_BASE_TAG=stream9" \
		--build-arg="BASE_IMAGE=ubi9-minimal"
endif

# The namespace to use for targets that require it (E2E, certs, etc).
SCANNER_NAMESPACE ?= stackrox

# The init bundle version to use when building the db image.
SCANNER_DB_INIT_BUNDLE_VERSION ?= dev

# If true enable the DB init bundle when building the db image.
SCANNER_DB_INIT_BUNDLE_ENABLED ?= false

SCANNER_VULNERABILITY_VERSION = $(shell cat VULNERABILITY_VERSION)

.PHONY: all
all: images

#######################
## Build Information ##
#######################

.PHONY: tag
tag:
	@echo $(TAG)

.PHONY: vulnerability-version
vulnerability-version: VULNERABILITY_VERSION
	@echo $(SCANNER_VULNERABILITY_VERSION)


###############
## Go Builds ##
###############

build-cmd  = $(GO_BUILD_CMD)
build-d   := bin

build-t      := $(addprefix $(build-d)/,$(notdir $(wildcard cmd/*)))
build-deps-t := deps

.PHONY: build
build: $(build-t)

.PHONY: clean-build
clean-build:
	@echo "+ $@"
	$(SILENT)rm -rf bin/
	$(SILENT)rm -f $(build-deps-t)

.PHONY: $(build-t)
$(build-t): $(build-deps-t)
	@echo "+ $@"
	$(build-cmd) -o $@ ./cmd/$(@F)

# Fetch and validate build dependencies unless NODEPS is defined.
ifdef NODEPS
.PHONY: $(build-deps-t)
else
$(build-deps-t): $(shell find $(CURDIR)/.. -name "go.sum")
	@echo "+ $@"
	$(SILENT)touch $@
endif

%/go.sum: %/go.mod
	@echo "+ $@"
	$(SILENT)go mod tidy
ifdef CI
	$(SILENT)GOTOOLCHAIN=local go mod tidy || { >&2 echo "Go toolchain does not match with installed Go version. This is a compatibility check that prevents breaking downstream builds. If you really need to update the toolchain version, ask in #forum-acs-golang" ; exit 1 ; }
	$(SILENT)git diff --exit-code -- ../go.mod ../go.sum || { echo "go.mod/go.sum files were updated after running 'go mod tidy', run this command on your local machine and commit the results." ; exit 1 ; }
endif
	$(SILENT)go mod verify
	$(SILENT)touch $@

############
## Images ##
############

image-prefix := scanner-v4

image-scripts        := restore-all-dir-contents \
                        import-additional-cas \
                        save-dir-contents
image-scripts-prefix := image/scanner/scripts
image-scripts-t      := $(addprefix $(image-scripts-prefix)/,$(image-scripts))

image-db-init-enabled := $(SCANNER_DB_INIT_BUNDLE_ENABLED)
image-db-init-ver     := $(SCANNER_DB_INIT_BUNDLE_VERSION)
image-db-init-d       := image/db/init-bundles
image-db-init-t       := $(image-db-init-d)/db-init.dump.zst

.PHONY: images
images: image-scanner image-db

image/scanner/bin/scanner: GOOS=$(DEFAULT_GOOS)
image/scanner/bin/scanner: bin/scanner
	@echo "+ $@"
	cp $< $@

$(image-scripts-prefix)/%: ../image/rhel/static-bin/%
	@echo "+ $@"
	cp $< $@

# Used by downstream builds.
.PHONY: copy-scripts
copy-scripts: $(image-scripts-t)
	@echo "+ $@"

.PHONY: image-scanner
image-scanner: $(if $(CI),,image/scanner/bin/scanner) \
               $(image-scripts-t) \
               ossls-notice
	@echo "+ $@"
	$(DOCKERBUILD) \
        -t stackrox/$(image-prefix):$(TAG) \
        $(DOCKERBUILD_ARGS) \
        -f image/scanner/Dockerfile image/scanner

$(image-db-init-d):
	mkdir -p $@

$(image-db-init-d)/db-init-$(image-db-init-ver).dump.zst: | $(image-db-init-d)
	@echo "+ $@"
	$(SILENT)curl \
	    --retry 3 --silent --show-error --fail --output $@ \
	    https://storage.googleapis.com/scanner-v4-test/database-init-bundles/$(@F)

.PHONY: $(image-db-init-t)

ifeq ($(image-db-init-enabled),true)
$(error Scanner DB init bundles are not updated anymore, and this flag is disabled (use 'force' to force it))
endif

ifeq ($(image-db-init-enabled),force)
$(warning Scanner DB init bundles are not updated anymore, but enabling it anyway (force requested))
$(image-db-init-t): $(image-db-init-d)/db-init-$(image-db-init-ver).dump.zst
	@echo "+ $@"
	$(SILENT)cp $^ $@
else
$(image-db-init-t): | $(image-db-init-d)
	@echo "+ $@: disabled (empty)"
	$(SILENT)>$@
endif

.PHONY: image-db
image-db: $(image-db-init-t)
	@echo "+ $@"
	$(DOCKERBUILD) \
        -t stackrox/$(image-prefix)-db:$(TAG) \
        $(DOCKERBUILD_ARGS) \
        $(DB_DOCKERBUILD_ARGS) \
        -f image/db/Dockerfile image/db

###########
## Tools ##
###########

.PHONY: ossls-notice
ossls-notice: $(build-deps-t)
	@echo "+ $@"
ifdef CI
	$(SILENT)ossls version
	$(SILENT)ossls audit --export image/scanner/THIRD_PARTY_NOTICES
else
	$(SILENT)mkdir -p image/scanner/THIRD_PARTY_NOTICES
endif

#################
## Integration ##
#################

db-integration-timeout := 20m
db-integration-d       := datastore/postgres
db-integration-go-tag  := scanner_db_integration

.PHONY: db-integration-test
db-integration-test: $(build-deps-t)
	@echo "+ $@"
	$(SILENT)$(GO_TEST_CMD) -tags $(db-integration-go-tag) -count=1 -timeout=$(db-integration-timeout) -v ./$(db-integration-d)/...

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
certs-subj := /C=US/ST=North Carolina/L=Raleigh/O=Red Hat, Inc./OU=SCANNER_V4_SERVICE

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
	    -subj "$(certs-subj)/CN=StackRox Certificate Authority" \
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
	$(SILENT)cp $(abspath $<) $@

.PRECIOUS: $(certs-d)/%/ca-key.pem
$(certs-d)/%/ca-key.pem: $(certs-d)/ca/key.pem
	@echo "+ $@"
	$(SILENT)cp $(abspath $<) $@

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

# E2E vars to config images
# -------------------------

SCANNER_E2E_IMAGE_REGISTRY ?= quay.io/stackrox-io
SCANNER_E2E_IMAGE_NAME     ?=
SCANNER_E2E_IMAGE_TAG      ?= $(TAG)

SCANNER_E2E_DB_IMAGE_REGISTRY ?= $(SCANNER_E2E_IMAGE_REGISTRY)
SCANNER_E2E_DB_IMAGE_NAME     ?= $(or $(SCANNER_E2E_IMAGE_NAME),scanner-v4-db)
SCANNER_E2E_DB_IMAGE_TAG      ?= $(SCANNER_E2E_IMAGE_TAG)

SCANNER_E2E_SCANNER_IMAGE_REGISTRY ?= $(SCANNER_E2E_IMAGE_REGISTRY)
SCANNER_E2E_SCANNER_IMAGE_NAME     ?= $(or $(SCANNER_E2E_IMAGE_NAME),scanner-v4)
SCANNER_E2E_SCANNER_IMAGE_TAG      ?= $(SCANNER_E2E_IMAGE_TAG)

# E2E vars to config credentials
# ------------------------------

SCANNER_E2E_QUAY_USERNAME ?=
SCANNER_E2E_QUAY_PASSWORD ?=

SCANNER_E2E_REDHAT_USERNAME ?=
SCANNER_E2E_REDHAT_PASSWORD ?=

SCANNER_E2E_DOCKER_USERNAME ?=
SCANNER_E2E_DOCKER_PASSWORD ?=

# E2E vars other
# --------------

SCANNER_E2E_DEBUG ?=

# E2E rules
# =========

e2e-conf-files := db-postgresql.conf \
                  db-pg_hba.conf

e2e-certs := ca.pem \
             scanner-v4-key.pem \
             scanner-v4-cert.pem \
             scanner-v4-db-key.pem \
             scanner-v4-db-cert.pem

e2e-d            := e2etests
e2e-chart-d      := $(e2e-d)/helmchart
e2e-files-d      := $(e2e-chart-d)/files
e2e-conf-files-t := $(addprefix $(e2e-files-d)/,$(e2e-conf-files))
e2e-certs-t      := $(addprefix $(e2e-files-d)/,$(e2e-certs))

# E2E General rules
# -----------------

.PHONY: e2e
e2e: e2e-deploy e2e-run

.PHONY: clean-e2e
clean-e2e:
	@echo "+ $@"
	$(SILENT)rm -rf $(e2e-files-d)/*

# E2E Deploy Rules
# ----------------

.PHONY: e2e-deploy

e2e-deploy: namespace := $(SCANNER_NAMESPACE)
e2e-deploy: release   := scanner-v4-e2e

e2e-deploy: registry :=
e2e-deploy: name     :=
e2e-deploy: tag      :=

e2e-deploy: db-registry := $(or $(registry),$(SCANNER_E2E_DB_IMAGE_REGISTRY))
e2e-deploy: db-name     := $(or $(name),$(SCANNER_E2E_DB_IMAGE_NAME))
e2e-deploy: db-tag      := $(or $(tag),$(SCANNER_E2E_DB_IMAGE_TAG))

e2e-deploy: scanner-registry := $(or $(registry),$(SCANNER_E2E_SCANNER_IMAGE_REGISTRY))
e2e-deploy: scanner-name     := $(or $(name),$(SCANNER_E2E_SCANNER_IMAGE_NAME))
e2e-deploy: scanner-tag      := $(or $(tag),$(SCANNER_E2E_SCANNER_IMAGE_TAG))

# If set will set DB's volume under the specified host path.
e2e-deploy: db-host-path :=

# If set will debug the helm templates.
e2e-deploy: debug-templates :=

# Wait timeout for the E2E deployment to be successful.
e2e-deploy: timeout := 30m

e2e-deploy: $(e2e-chart-d) $(e2e-conf-files-t) $(e2e-certs-t)
	@echo "+ $@" db=$(db-tag) scanner=$(scanner-tag)
	helm $(if $(debug-templates),template,upgrade --install) --debug $(release) $< \
	    --namespace $(namespace) \
	    --create-namespace \
	    $(if $(CI),,--atomic) \
	    --timeout $(timeout) \
	    $(and $(db-host-path),--set app.db.persistence.hostPath=$(db-host-path)) \
	    --set app.db.image.registry=$(db-registry) \
	    --set app.db.image.name=$(db-name) \
	    --set app.db.image.tag=$(db-tag) \
	    --set app.scanner.image.registry=$(scanner-registry) \
	    --set app.scanner.image.name=$(scanner-name) \
	    --set app.scanner.image.tag=$(scanner-tag)

# E2E Run Rules
# -------------

.PHONY: e2e-run

e2e-run: matcher-addr := :8443
e2e-run: indexer-addr := :8443

e2e-run: go-tag  := scanner_e2e_tests
e2e-run: timeout := 30m
e2e-run: re :=

e2e-run: export SCANNER_E2E_MATCHER_ADDRESS=$(matcher-addr)
e2e-run: export SCANNER_E2E_INDEXER_ADDRESS=$(indexer-addr)
e2e-run: export QUAY_RHACS_ENG_RO_USERNAME=$(SCANNER_E2E_QUAY_USERNAME)
e2e-run: export QUAY_RHACS_ENG_RO_PASSWORD=$(SCANNER_E2E_QUAY_PASSWORD)
e2e-run: export REDHAT_USERNAME=$(SCANNER_E2E_REDHAT_USERNAME)
e2e-run: export REDHAT_PASSWORD=$(SCANNER_E2E_REDHAT_PASSWORD)
e2e-run: export DOCKER_USERNAME=$(SCANNER_E2E_DOCKER_USERNAME)
e2e-run: export DOCKER_PASSWORD=$(SCANNER_E2E_DOCKER_PASSWORD)

e2e-run: $(build-deps-t)
	@echo "+ $@"
	$(GO_TEST_CMD) -tags $(go-tag) -count=1 -timeout=$(timeout) $(and $(re),--run $(re)) --v ./$(e2e-d)/...

$(e2e-files-d)/db-%.conf: ../image/templates/helm/shared/config-templates/scanner-v4-db/%.conf.default
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
clean: clean-image clean-gobin clean-e2e clean-certs clean-build
	@echo "+ $@"

.PHONY: clean-image
clean-image:
	@echo "+ $@"
	$(SILENT)git clean -xdf image/scanner/bin

.PHONY: clean-gobin
clean-gobin:
	@echo "+ $@"
	$(SILENT)rm -rf $(GOBIN)
