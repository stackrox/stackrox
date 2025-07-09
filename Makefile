include $(CURDIR)/make/env.mk

PLATFORM ?= linux/amd64
ROX_PROJECT=apollo
TESTFLAGS=-race -p 4
BASE_DIR=$(CURDIR)
BENCHTIME ?= 1x
BENCHTIMEOUT ?= 20m
BENCHCOUNT ?= 1

SHELL = /bin/bash -o pipefail

podman =
# docker --version might not contain any traces of podman in the latest
# version, search for more output
ifneq (,$(findstring podman,$(shell docker --version 2>/dev/null)))
	podman = yes
endif
ifneq (,$(findstring Podman,$(shell docker version 2>/dev/null)))
	podman = yes
endif

ifdef podman
# Disable selinux for local podman builds.
DOCKER_OPTS=--security-opt label=disable
else
# Podman DTRT by running processes unprivileged in containers,
# but it's UID mapping is more nuanced. Only set user for vanilla docker.
DOCKER_OPTS=--user "$(shell id -u)"
endif

# Set to empty string to echo some command lines which are hidden by default.
SILENT ?= @

# UNIT_TEST_IGNORE ignores a set of file patterns from the unit test make command.
# the pattern is passed to: grep -Ev
#  usage: "path/to/ignored|another/path"
# TODO: [ROX-19070] Update postgres store test generation to work for foreign keys
UNIT_TEST_IGNORE := "stackrox/rox/sensor/tests|stackrox/rox/operator/tests|stackrox/rox/central/reports/config/store/postgres|stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres|stackrox/rox/central/auth/store/postgres|stackrox/rox/scanner/e2etests"

ifeq ($(TAG),)
TAG=$(shell git rev-parse --abbrev-ref HEAD)
endif

# Set expiration on Quay.io for non-release tags.
ifeq ($(findstring x,$(TAG)),x)
QUAY_TAG_EXPIRATION=13w
else
QUAY_TAG_EXPIRATION=never
endif

ROX_PRODUCT_BRANDING ?= STACKROX_BRANDING

# ROX_IMAGE_FLAVOR is an ARG used in Dockerfiles that defines the default registries for main, scanner, and collector images.
# ROX_IMAGE_FLAVOR valid values are: development_build, rhacs, opensource.
ROX_IMAGE_FLAVOR ?= $(shell \
	if [[ "$(ROX_PRODUCT_BRANDING)" == "STACKROX_BRANDING" ]]; then \
	  echo "opensource"; \
	else \
	  echo "development_build"; \
	fi)

DEFAULT_IMAGE_REGISTRY := quay.io/stackrox-io
ifeq ($(ROX_PRODUCT_BRANDING),RHACS_BRANDING)
	DEFAULT_IMAGE_REGISTRY := quay.io/rhacs-eng
endif

GOBUILD := $(CURDIR)/scripts/go-build.sh
DOCKERBUILD := $(CURDIR)/scripts/docker-build.sh
GO_TEST_OUTPUT_PATH=$(CURDIR)/test-output/test.log
GOPATH_VOLUME_NAME := stackrox-rox-gopath
GOCACHE_VOLUME_NAME := stackrox-rox-gocache
GENERATE_PATH ?= ./...

# If git branch name contains substring "-debug", a debug build will be made, unless overridden by environment variable.
ifneq (,$(findstring -debug,$(shell git rev-parse --abbrev-ref HEAD)))
	DEBUG_BUILD ?= yes
endif
DEBUG_BUILD ?= no

# Figure out whether to use standalone Docker volume for GOPATH/Go build cache, or bind
# mount one from the host filesystem.
# The latter is painfully slow on Mac OS X with Docker Desktop, so we default to using a
# standalone volume in that case, and to bind mounting otherwise.
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

BUILD_IMAGE := quay.io/stackrox-io/apollo-ci:$(shell sed 's/\s*\#.*//' BUILD_IMAGE_VERSION)
ifneq ($(UNAME_M),x86_64)
	GO_VERSION := $(shell sed -n 's/^go \([0-9]*\.[0-9]*\)\..*/\1/p' go.mod)
	BUILD_IMAGE = docker.io/library/golang:$(GO_VERSION)
endif

CENTRAL_DB_DOCKER_ARGS :=
ifeq ($(GOARCH),s390x)
	CENTRAL_DB_DOCKER_ARGS := \
		--build-arg="BASE_IMAGE=ubi9-minimal" \
		--build-arg="BASE_TAG=9.2" \
		--build-arg="RPMS_REGISTRY=quay.io" \
		--build-arg="RPMS_BASE_IMAGE=centos/centos" \
		--build-arg="RPMS_BASE_TAG=stream9"
endif

# By default, assume we are going to use a bind mount volume instead of a standalone one.
BIND_GOCACHE ?= 1
BIND_GOPATH ?= 1

# Only resort to local volumes on X86_64 Darwin, since on ARM Darwin the permissions of the
# standalone volume will be mapped to root instead of the local user, making the build fail.
# An alternative is to chown the directory of the standalone volume within the container.
ifeq ($(UNAME_S),Darwin)
ifneq ($(UNAME_M),arm64)
BIND_GOCACHE ?= 0
BIND_GOPATH ?= 0
endif
endif

ifeq ($(BIND_GOCACHE),1)
GOCACHE_VOLUME_SRC := $(CURDIR)/linux-gocache
else
GOCACHE_VOLUME_SRC := $(GOCACHE_VOLUME_NAME)
endif

ifeq ($(BIND_GOPATH),1)
GOPATH_VOLUME_SRC := $(GOPATH)
else
GOPATH_VOLUME_SRC := $(GOPATH_VOLUME_NAME)
endif

LOCAL_VOLUME_ARGS := -v $(CURDIR):/src:delegated -v $(GOCACHE_VOLUME_SRC):/linux-gocache:delegated -v $(GOPATH_VOLUME_SRC):/go:delegated
LOCAL_VOLUME_ARGS += $(EXTRA_LOCAL_VOLUME_ARGS)
GOPATH_WD_OVERRIDES := -w /src -e GOPATH=/go -e GOCACHE=/linux-gocache -e GIT_CONFIG_COUNT=1 -e GIT_CONFIG_KEY_0=safe.directory -e GIT_CONFIG_VALUE_0='/src'

null :=
space := $(null) $(null)
comma := ,

.PHONY: all
all: deps style test image

#####################################################################
###### Binaries we depend on (need to be defined on top) ############
#####################################################################

include make/gotools.mk

$(call go-tool, BUF_BIN, github.com/bufbuild/buf/cmd/buf, tools/proto)
$(call go-tool, GOLANGCILINT_BIN, github.com/golangci/golangci-lint/v2/cmd/golangci-lint, tools/linters)
$(call go-tool, EASYJSON_BIN, github.com/mailru/easyjson/easyjson)
$(call go-tool, ROXVET_BIN, ./tools/roxvet)
$(call go-tool, STRINGER_BIN, golang.org/x/tools/cmd/stringer)
$(call go-tool, MOCKGEN_BIN, go.uber.org/mock/mockgen)
$(call go-tool, GO_JUNIT_REPORT_BIN, github.com/jstemmer/go-junit-report/v2, tools/test)
$(call go-tool, PROTOLOCK_BIN, github.com/nilslice/protolock/cmd/protolock, tools/linters)
$(call go-tool, GOVULNCHECK_BIN, golang.org/x/vuln/cmd/govulncheck, tools/linters)
$(call go-tool, IMAGE_PREFETCHER_DEPLOY_BIN, github.com/stackrox/image-prefetcher/deploy, tools/test)
$(call go-tool, PROMETHEUS_METRIC_PARSER_BIN, github.com/stackrox/prometheus-metric-parser, tools/test)

###########
## Style ##
###########
.PHONY: style
style: golangci-lint style-slim

.PHONY: style-slim
style-slim: \
	blanks \
	check-service-protos \
	newlines \
	no-large-files \
	openshift-ci-style \
	proto-style \
	qa-tests-style \
	roxvet \
	shell-style \
	storage-protos-compatible \
	ui-lint

GOLANGCILINT_FLAGS := --verbose --print-resources-usage

.PHONY: golangci-lint-cache-status
golangci-lint-cache-status: $(GOLANGCILINT_BIN) deps
	@echo '+ $@'
	@echo "Checking golangci-lint cache status"
	$(GOLANGCILINT_BIN) cache status

.PHONY: golangci-lint
golangci-lint: $(GOLANGCILINT_BIN) deps
ifdef CI
	@echo '+ $@'
	@echo 'The environment indicates we are in CI; running linters in check mode.'
	@echo 'If this fails, run `make style`.'
	$(GOLANGCILINT_BIN) --version
	@echo "Running with no tags and no tests..."
	@# The first run is meant to have limited scope to warmup the cache.
	@# Adding it as first allowed to shorten the runtime of the following runs to about 5 min each
	$(GOLANGCILINT_BIN) run $(GOLANGCILINT_FLAGS) --tests=false
	@echo "Running with no tags..."
	@# We need to enable unused linter here as it will not work without tests or in release tag.
	$(GOLANGCILINT_BIN) run $(GOLANGCILINT_FLAGS) --enable=unused
	@echo "Running with release tags..."
	@# We use --tests=false because some unit tests don't compile with release tags,
	@# since they use functions that we don't define in the release build. That's okay.
	$(GOLANGCILINT_BIN) run $(GOLANGCILINT_FLAGS) --build-tags "$(subst $(comma),$(space),$(RELEASE_GOTAGS))" --tests=false
else
	$(GOLANGCILINT_BIN) run $(GOLANGCILINT_FLAGS) --fix --enable=unused
	$(GOLANGCILINT_BIN) run $(GOLANGCILINT_FLAGS) --fix --build-tags "$(subst $(comma),$(space),$(RELEASE_GOTAGS))" --tests=false
endif

.PHONY: proto-style
proto-style: $(BUF_BIN) deps
	@echo "+ $@"
	$(BUF_BIN) format --exit-code --diff -w

.PHONY: qa-tests-style
qa-tests-style:
	@echo "+ $@"
	make -C qa-tests-backend/ style

.PHONY: ui-lint
ui-lint:
	@echo "+ $@"
	make -C ui lint

.PHONY: openshift-ci-style
openshift-ci-style:
	@echo "+ $@"
	make -C .openshift-ci/ style

.PHONY: shell-style
shell-style:
	@echo "+ $@"
	$(SILENT)$(BASE_DIR)/scripts/style/shellcheck.sh

.PHONY: update-shellcheck-skip
update-shellcheck-skip:
	@echo "+ $@"
	$(SILENT)rm -f scripts/style/shellcheck_skip.txt
	$(SILENT)$(BASE_DIR)/scripts/style/shellcheck.sh update_failing_list

.PHONY: fast-central-build
fast-central-build: central-build-nodeps

.PHONY: central-build-nodeps
central-build-nodeps:
	@echo "+ $@"
	$(GOBUILD) central

.PHONY: config-controller-build-nodeps
config-controller-build-nodeps:
	@echo "+ $@"
	$(GOBUILD) config-controller

.PHONY: fast-central
fast-central: deps
	@echo "+ $@"
	docker run $(DOCKER_OPTS) -e CGO_ENABLED --rm $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make fast-central-build
	$(SILENT)$(BASE_DIR)/scripts/k8s/kill-pod.sh central

# fast is a dev mode options when using local dev
# it will automatically restart Central if there are any changes
.PHONY: fast
fast: fast-central

.PHONY: fast-sensor
fast-sensor: sensor-build-dockerized

.PHONY: fast-sensor-kubernetes
fast-sensor-kubernetes: sensor-kubernetes-build-dockerized
	$(SILENT)$(BASE_DIR)/scripts/k8s/kill-pod.sh sensor

.PHONY: fast-migrator
fast-migrator:
	@echo "+ $@"
	docker run $(DOCKER_OPTS) -e CGO_ENABLED --rm $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make fast-migrator-build

.PHONY: fast-migrator-build
fast-migrator-build: migrator-build-nodeps

.PHONY: migrator-build-nodeps
migrator-build-nodeps:
	@echo "+ $@"
	$(GOBUILD) migrator

.PHONY: check-service-protos
check-service-protos:
	@echo "+ $@"
	$(SILENT)$(BASE_DIR)/tools/check-service-protos/run.sh

.PHONY: no-large-files
no-large-files:
	$(BASE_DIR)/tools/detect-large-files.sh "$(BASE_DIR)/tools/allowed-large-files"

# adding the uptodate flag will inform us if the protos and lock are out of date if they pass the compatibility check.
.PHONY: storage-protos-compatible
storage-protos-compatible: $(PROTOLOCK_BIN)
	@echo "+ $@"
	$(SILENT)$(PROTOLOCK_BIN) status -lockdir=$(BASE_DIR)/proto/storage -protoroot=$(BASE_DIR)/proto/storage --uptodate true

.PHONY: update-storage-protolock
update-storage-protolock: $(PROTOLOCK_BIN)
	@echo "+ $@"
	$(SILENT)$(PROTOLOCK_BIN) commit -lockdir=$(BASE_DIR)/proto/storage -protoroot=$(BASE_DIR)/proto/storage

.PHONY: blanks
blanks:
	@echo "+ $@"
ifdef CI
	$(SILENT)git grep -L '^// Code generated by .* DO NOT EDIT\.' -- '*.go' | xargs -n 1000 $(BASE_DIR)/tools/import_validate.py
else
	$(SILENT)git grep -L '^// Code generated by .* DO NOT EDIT\.' -- '*.go' | xargs -n 1000 $(BASE_DIR)/tools/fix-blanks.sh
endif

.PHONY: newlines
newlines:
	@echo "+ $@"
ifdef CI
	$(SILENT)git grep --cached -Il '' | xargs $(BASE_DIR)/tools/check-newlines.sh
else
	$(SILENT)git grep --cached -Il '' | xargs $(BASE_DIR)/tools/check-newlines.sh --fix
endif

.PHONY: init-githooks
init-githooks:
	@echo "+ $@"
	./tools/githooks/install-hooks.sh tools/githooks/pre-commit

.PHONY: dev
dev: install-dev-tools
	@echo "+ $@"

#####################################
## Generated Code and Dependencies ##
#####################################

PROTO_GENERATED_SRCS = $(GENERATED_PB_SRCS) $(GENERATED_VT_SRCS) $(GENERATED_COMPAT_SRCS) $(GENERATED_API_SRCS) $(GENERATED_API_GW_SRCS)

include make/protogen.mk

.PHONY: go-easyjson-srcs
go-easyjson-srcs: $(EASYJSON_BIN)
	@echo "+ $@"
	@# Files are ordered such that repeated runs of `make go-easyjson-srcs` don't create diffs.
	$(SILENT)$(EASYJSON_BIN) pkg/compliance/compress/compress.go

.PHONY: clean-easyjson-srcs
clean-easyjson-srcs:
	@echo "+ $@"
	$(SILENT)find . -name '*_easyjson.go' -exec rm {} \;

.PHONY: go-generated-srcs
go-generated-srcs: deps clean-easyjson-srcs go-easyjson-srcs $(MOCKGEN_BIN) $(STRINGER_BIN)
	@echo "+ $@"
	PATH="$(GOTOOLS_BIN):$(PATH):$(BASE_DIR)/tools/generate-helpers" MOCKGEN_BIN="$(MOCKGEN_BIN)" go generate -v -x $(GENERATE_PATH)

proto-generated-srcs: $(PROTO_GENERATED_SRCS) $(GENERATED_API_SWAGGER_SPECS) $(GENERATED_API_SWAGGER_SPECS_V2) inject-proto-tags cleanup-swagger-json-gotags
	@echo "+ $@"
	$(SILENT)touch proto-generated-srcs
	$(SILENT)$(MAKE) clean-obsolete-protos

clean-proto-generated-srcs:
	@echo "+ $@"
	git clean -xdf generated

.PHONY: config-controller-gen
config-controller-gen:
	make -C config-controller/ manifests
	make -C config-controller/ generate
	cp config-controller/config/crd/bases/config.stackrox.io_securitypolicies.yaml image/templates/helm/stackrox-central/internal

.PHONY: generated-srcs
generated-srcs: go-generated-srcs config-controller-gen

deps: $(shell find $(BASE_DIR) -name "go.sum")
	@echo "+ $@"
	$(SILENT)touch deps

%/go.sum: %/go.mod
	$(SILENT)cd $*
	@echo "+ $@"
	$(SILENT)$(eval GOMOCK_REFLECT_DIRS=`find . -type d -name 'gomock_reflect_*'`)
	$(SILENT)test -z $(GOMOCK_REFLECT_DIRS) || { echo "Found leftover gomock directories. Please remove them and rerun make deps!"; echo $(GOMOCK_REFLECT_DIRS); exit 1; }
ifdef CI
	$(SILENT)GOTOOLCHAIN=local go mod tidy || { >&2 echo "Go toolchain does not match with installed Go version. This is a compatibility check that prevents breaking downstream builds. If you really need to update the toolchain version, ask in #forum-acs-golang" ; exit 1 ; }
	$(SILENT)git diff --exit-code -- go.mod go.sum || { echo "go.mod/go.sum files were updated after running 'go mod tidy', run this command on your local machine and commit the results." ; exit 1 ; }
else
	$(SILENT)go mod tidy
endif
	$(SILENT)touch $@

.PHONY: clean-deps
clean-deps:
	@echo "+ $@"
	$(SILENT)rm -f deps

.PHONY: clean-obsolete-protos
clean-obsolete-protos:
	@echo "+ $@"
	$(BASE_DIR)/tools/clean_autogen_protos.py --protos $(BASE_DIR)/proto --generated $(BASE_DIR)/generated

###########
## Build ##
###########

HOST_OS:=linux
ifeq ($(UNAME_S),Darwin)
    HOST_OS:=darwin
endif

.PHONY: build-prep
build-prep: deps
	mkdir -p bin/{darwin_amd64,darwin_arm64,linux_amd64,linux_arm64,linux_ppc64le,linux_s390x,windows_amd64}

.PHONY: cli-build
cli-build: cli-linux cli-darwin cli-windows

.PHONY: cli-install
cli-install:
	# Workaround a bug on MacOS
	rm -f $(GOPATH)/bin/roxctl
	# Copy the user's specific OS into gopath
	mkdir -p $(GOPATH)/bin
	cp bin/$(HOST_OS)_$(GOARCH)/roxctl $(GOPATH)/bin/roxctl
	chmod u+w $(GOPATH)/bin/roxctl

.PHONY: cli
cli: cli-build cli-install

cli-linux: cli_linux-amd64 cli_linux-arm64 cli_linux-ppc64le cli_linux-s390x
cli-darwin: cli_darwin-amd64 cli_darwin-arm64
cli-windows: cli_windows-amd64

cli_%: build-prep
	$(eval    w := $(subst -, ,$*))
	$(eval   os := $(firstword $(w)))
	$(eval arch := $(lastword  $(w)))
ifdef SKIP_CLI_BUILD
	test -f bin/$(os)_$(arch)/roxctl || RACE=0 CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) $(GOBUILD) ./roxctl
else
	RACE=0 CGO_ENABLED=0 GOOS=$(os) GOARCH=$(arch) $(GOBUILD) ./roxctl
endif

.PHONY: cli_host-arch
cli_host-arch: cli_$(HOST_OS)-$(GOARCH)

upgrader: bin/$(HOST_OS)_$(GOARCH)/upgrader

bin/$(HOST_OS)_$(GOARCH)/upgrader: build-prep
	GOOS=$(HOST_OS) GOARCH=$(GOARCH) $(GOBUILD) ./sensor/upgrader

bin/$(HOST_OS)_$(GOARCH)/admission-control: build-prep
	GOOS=$(HOST_OS) GOARCH=$(GOARCH) $(GOBUILD) ./sensor/admission-control

.PHONY: build-volumes
build-volumes:
	$(SILENT)mkdir -p $(CURDIR)/linux-gocache
	$(SILENT)docker volume inspect $(GOPATH_VOLUME_NAME) >/dev/null 2>&1 || docker volume create $(GOPATH_VOLUME_NAME)
	$(SILENT)docker volume inspect $(GOCACHE_VOLUME_NAME) >/dev/null 2>&1 || docker volume create $(GOCACHE_VOLUME_NAME)

.PHONY: main-build
main-build: build-prep main-build-dockerized
	@echo "+ $@"

.PHONY: sensor-build-dockerized
sensor-build-dockerized: build-volumes
	@echo "+ $@"
	docker run $(DOCKER_OPTS) --rm -e CI -e BUILD_TAG -e GOTAGS -e DEBUG_BUILD -e CGO_ENABLED $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make sensor-build

.PHONY: sensor-kubernetes-build-dockerized
sensor-kubernetes-build-dockerized: build-volumes
	@echo "+ $@"
	docker run $(DOCKER_OPTS) -e CI -e BUILD_TAG -e GOTAGS -e DEBUG_BUILD -e CGO_ENABLED $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make sensor-kubernetes-build

.PHONY: sensor-build
sensor-build:
	$(GOBUILD) sensor/kubernetes sensor/admission-control
	CGO_ENABLED=0 $(GOBUILD) sensor/upgrader

.PHONY: sensor-kubernetes-build
sensor-kubernetes-build:
	$(GOBUILD) sensor/kubernetes

.PHONY: main-build-dockerized
main-build-dockerized: build-volumes
	@echo "+ $@"
	docker run $(DOCKER_OPTS) -i -e RACE -e CI -e BUILD_TAG -e SHORTCOMMIT -e GOTAGS -e DEBUG_BUILD -e CGO_ENABLED --rm $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make main-build-nodeps

.PHONY: main-build-nodeps
main-build-nodeps:
	$(GOBUILD) \
		central \
		compliance/cmd/compliance \
		config-controller \
		migrator \
		sensor/admission-control \
		sensor/init-tls-certs \
		sensor/kubernetes \
		sensor/upgrader
ifndef CI
	CGO_ENABLED=0 $(GOBUILD) roxctl
endif

.PHONY: scale-build
scale-build: build-prep
	@echo "+ $@"
	CGO_ENABLED=0 $(GOBUILD) scale/profiler scale/chaos

.PHONY: webhookserver-build
webhookserver-build: build-prep
	@echo "+ $@"
	CGO_ENABLED=0 $(GOBUILD) webhookserver

.PHONY: syslog-build
syslog-build:build-prep
	@echo "+ $@"
	CGO_ENABLED=0 $(GOBUILD) qa-tests-backend/test-images/syslog

.PHONY: gendocs
gendocs: $(GENERATED_API_DOCS)
	@echo "+ $@"

# We don't need to do anything here, because the $(MERGED_API_SWAGGER_SPEC) and $(MERGED_API_SWAGGER_SPEC_V2) targets
# already perform validation.
.PHONY: swagger-docs
swagger-docs: $(MERGED_API_SWAGGER_SPEC) $(MERGED_API_SWAGGER_SPEC_V2) $(MERGED_API_OPENAPI_SPEC) $(MERGED_API_OPENAPI_SPEC_V2)
	@echo "+ $@"

UNIT_TEST_PACKAGES ?= ./...

.PHONY: test-prep
test-prep:
	@echo "+ $@"
	$(SILENT)mkdir -p test-output

.PHONY: go-unit-tests
go-unit-tests: build-prep test-prep
	set -o pipefail ; \
	CGO_ENABLED=1 GOEXPERIMENT=cgocheck2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=$(GOTAGS),test scripts/go-test.sh -timeout 15m -race -cover -coverprofile test-output/coverage.out -v \
		$(shell git ls-files -- '*_test.go' | sed -e 's@^@./@g' | xargs -n 1 dirname | sort | uniq | xargs go list| grep -v '^github.com/stackrox/rox/tests$$' | grep -Ev $(UNIT_TEST_IGNORE)) \
		| tee $(GO_TEST_OUTPUT_PATH)
	# Exercise the logging package for all supported logging levels to make sure that initialization works properly
	@echo "Run log tests"
	for encoding in console json; do \
		for level in debug info warn error fatal panic; do \
			LOGENCODING=$$encoding LOGLEVEL=$$level CGO_ENABLED=1 GOEXPERIMENT=cgocheck2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=$(GOTAGS),test scripts/go-test.sh -p 4 -race -v ./pkg/logging/... | grep -v "iteration"; \
		done; \
	done

.PHONY: sensor-integration-test
sensor-integration-test: build-prep test-prep
	set -eo pipefail ; \
	rm -rf  $(GO_TEST_OUTPUT_PATH); \
	for package in $(shell git ls-files ./sensor/tests | grep '_test.go' | xargs -n 1 dirname | uniq | sort | sed -e 's/sensor\/tests\///'); do \
		CGO_ENABLED=1 GOEXPERIMENT=cgocheck2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 LOGLEVEL=debug GOTAGS=$(GOTAGS),test scripts/go-test.sh -p 4 -race -cover -coverprofile test-output/coverage.out -v ./sensor/tests/$$package \
		| tee -a $(GO_TEST_OUTPUT_PATH); \
	done \

sensor-pipeline-benchmark: build-prep test-prep
	LOGLEVEL="panic" go test -bench=. -run=^# -benchtime=30s -count=5 ./sensor/tests/pipeline | tee $(CURDIR)/test-output/pipeline.results.txt

.PHONY: go-postgres-unit-tests
go-postgres-unit-tests: build-prep test-prep
	set -o pipefail ; \
	CGO_ENABLED=1 GOEXPERIMENT=cgocheck2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=$(GOTAGS),test,sql_integration scripts/go-test.sh -timeout 15m  -race -cover -coverprofile test-output/coverage.out -v \
		$(shell git grep -rl "//go:build sql_integration" central pkg tools | sed -e 's@^@./@g' | xargs -n 1 dirname | sort | uniq | xargs go list -tags sql_integration | grep -v '^github.com/stackrox/rox/tests$$' | grep -Ev $(UNIT_TEST_IGNORE)) \
		| tee $(GO_TEST_OUTPUT_PATH)
	@# The -p 1 passed to go test is required to ensure that tests of different packages are not run in parallel, so as to avoid conflicts when interacting with the DB.
	set -o pipefail ; \
	CGO_ENABLED=1 GOEXPERIMENT=cgocheck2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=$(GOTAGS),test,sql_integration scripts/go-test.sh -p 1 -race -cover -coverprofile test-output/migrator-coverage.out -v \
		$(shell git grep -rl "//go:build sql_integration" migrator | sed -e 's@^@./@g' | xargs -n 1 dirname | sort | uniq | xargs go list -tags sql_integration | grep -v '^github.com/stackrox/rox/tests$$' | grep -Ev $(UNIT_TEST_IGNORE)) \
		| tee -a $(GO_TEST_OUTPUT_PATH)

.PHONY: go-postgres-bench-tests
go-postgres-bench-tests: build-prep test-prep
	set -o pipefail ; \
	CGO_ENABLED=1 GOEXPERIMENT=cgocheck2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=$(GOTAGS),test,sql_integration scripts/go-test.sh -run=nonthing -bench=. -benchtime=$(BENCHTIME) -benchmem -timeout $(BENCHTIMEOUT) -count $(BENCHCOUNT) -v \
  $(shell git grep -rl "testing.B" central pkg migrator tools | sed -e 's@^@./@g' | xargs -n 1 dirname | sort | uniq | xargs go list -tags sql_integration | grep -v '^github.com/stackrox/rox/tests$$' | grep -Ev $(UNIT_TEST_IGNORE)) \
		| tee $(GO_TEST_OUTPUT_PATH)

.PHONY: shell-unit-tests
shell-unit-tests:
	@echo "+ $@"
	$(SILENT)mkdir -p shell-test-output
	bats --print-output-on-failure --verbose-run --recursive --report-formatter junit --output shell-test-output \
		scripts \
		tests/e2e/bats

.PHONY: ui-build
ui-build:
ifdef SKIP_UI_BUILD
	test -d ui/build || make -C ui build
else
	make -C ui build
endif

.PHONY: ui-test
ui-test:
	make -C ui test

.PHONY: ui-component-tests
ui-component-tests:
	make -C ui test-component

.PHONY: test
test: go-unit-tests ui-test shell-unit-tests

.PHONY: integration-unit-tests
integration-unit-tests: build-prep test-prep
	set -o pipefail ; \
	GOTAGS=$(GOTAGS),test,integration scripts/go-test.sh -count=1 -v \
		$(shell go list ./... | grep  "registries\|scanners\|notifiers") \
		| tee $(GO_TEST_OUTPUT_PATH)

.PHONY: generate-junit-reports
generate-junit-reports: junit-reports/report.xml

$(GO_TEST_OUTPUT_PATH):
	@echo "The test output log cannot be created via a direct Makefile rule. You must make the desired test targets"
	@echo "first to ensure the file's existence."
	@exit 1

junit-reports/report.xml: $(GO_TEST_OUTPUT_PATH) $(GO_JUNIT_REPORT_BIN)
	@mkdir -p junit-reports
	$(SILENT)$(GO_JUNIT_REPORT_BIN) <"$<" >"$@"

###########
## Image ##
###########

# image is an alias for main-image
.PHONY: image
image: main-image

.PHONY: all-builds
all-builds: cli main-build clean-image swagger-docs ui-build

.PHONY: main-image
main-image: all-builds
	make docker-build-main-image

.PHONY: docker-build-main-image
docker-build-main-image: copy-binaries-to-image-dir central-db-image
	$(DOCKERBUILD) \
		-t stackrox/main:$(TAG) \
		-t $(DEFAULT_IMAGE_REGISTRY)/main:$(TAG) \
		--build-arg DEBUG_BUILD="$(DEBUG_BUILD)" \
		--build-arg ROX_PRODUCT_BRANDING=$(ROX_PRODUCT_BRANDING) \
		--build-arg TARGET_ARCH=$(GOARCH) \
		--build-arg ROX_IMAGE_FLAVOR=$(ROX_IMAGE_FLAVOR) \
		--build-arg LABEL_VERSION=$(TAG) \
		--build-arg LABEL_RELEASE=$(TAG) \
		--build-arg QUAY_TAG_EXPIRATION=$(QUAY_TAG_EXPIRATION) \
		$(CENTRAL_DB_DOCKER_ARGS) \
		--file image/rhel/Dockerfile \
		image/rhel
	@echo "Built main image for RHEL with tag: $(TAG), image flavor: $(ROX_IMAGE_FLAVOR)"
	@echo "You may wish to:       export MAIN_IMAGE_TAG=$(TAG)"

.PHONY: docker-build-roxctl-image
docker-build-roxctl-image:
	cp -f bin/linux_$(GOARCH)/roxctl image/roxctl/roxctl-linux
	$(DOCKERBUILD) \
		-t stackrox/roxctl:$(TAG) \
		-t $(DEFAULT_IMAGE_REGISTRY)/roxctl:$(TAG) \
		-f image/roxctl/Dockerfile \
		--label quay.expires-after=$(QUAY_TAG_EXPIRATION) \
		image/roxctl

.PHONY: copy-go-binaries-to-image-dir
copy-go-binaries-to-image-dir:
	cp bin/linux_$(GOARCH)/central image/rhel/bin/central
	cp bin/linux_$(GOARCH)/config-controller image/rhel/bin/config-controller
ifdef CI
	cp bin/linux_amd64/roxctl image/rhel/bin/roxctl-linux-amd64
	cp bin/linux_arm64/roxctl image/rhel/bin/roxctl-linux-arm64
	cp bin/linux_ppc64le/roxctl image/rhel/bin/roxctl-linux-ppc64le
	cp bin/linux_s390x/roxctl image/rhel/bin/roxctl-linux-s390x
	cp bin/darwin_amd64/roxctl image/rhel/bin/roxctl-darwin-amd64
	cp bin/darwin_arm64/roxctl image/rhel/bin/roxctl-darwin-arm64
	cp bin/windows_amd64/roxctl.exe image/rhel/bin/roxctl-windows-amd64.exe
else
ifneq ($(HOST_OS),linux)
	cp bin/linux_$(GOARCH)/roxctl image/rhel/bin/roxctl-linux-$(GOARCH)
endif
	cp bin/$(HOST_OS)_amd64/roxctl image/rhel/bin/roxctl-$(HOST_OS)-amd64
endif
	cp bin/linux_$(GOARCH)/migrator image/rhel/bin/migrator
	cp bin/linux_$(GOARCH)/kubernetes        image/rhel/bin/kubernetes-sensor
	cp bin/linux_$(GOARCH)/init-tls-certs    image/rhel/bin/init-tls-certs
	cp bin/linux_$(GOARCH)/upgrader          image/rhel/bin/sensor-upgrader
	cp bin/linux_$(GOARCH)/admission-control image/rhel/bin/admission-control
	cp bin/linux_$(GOARCH)/compliance        image/rhel/bin/compliance
	# Workaround to bug in lima: https://github.com/lima-vm/lima/issues/602
	find image/rhel/bin -not -path "*/.*" -type f -exec chmod +x {} \;


.PHONY: copy-binaries-to-image-dir
copy-binaries-to-image-dir: copy-go-binaries-to-image-dir
	cp -r ui/build image/rhel/ui/
ifdef CI
	$(SILENT)[ -d image/rhel/THIRD_PARTY_NOTICES ] || { echo "image/rhel/THIRD_PARTY_NOTICES dir not found! It is required for CI-built images."; exit 1; }
else
	$(SILENT)[ -f image/rhel/THIRD_PARTY_NOTICES ] || mkdir -p image/rhel/THIRD_PARTY_NOTICES
endif
	$(SILENT)[ -d image/rhel/docs ] || { echo "Generated docs not found in image/rhel/docs. They are required for build."; exit 1; }

.PHONY: scale-image
scale-image: scale-build clean-image
	cp bin/linux_$(GOARCH)/profiler scale/image/rhel/bin/profiler
	cp bin/linux_$(GOARCH)/chaos scale/image/rhel/bin/chaos
	chmod +w scale/image/rhel/bin/*
	docker build \
		-t stackrox/scale:$(TAG) \
		-t quay.io/rhacs-eng/scale:$(TAG) \
		-f scale/image/Dockerfile scale

webhookserver-image: webhookserver-build
	-mkdir webhookserver/bin
	cp bin/linux_$(GOARCH)/webhookserver webhookserver/bin/webhookserver
	chmod +w webhookserver/bin/webhookserver
	docker build \
		-t stackrox/webhookserver:1.2 \
		-t quay.io/rhacs-eng/webhookserver:1.2 \
		-f webhookserver/Dockerfile webhookserver

syslog-image: syslog-build
	-mkdir qa-tests-backend/test-images/syslog/bin
	cp bin/linux_$(GOARCH)/syslog qa-tests-backend/test-images/syslog/bin/syslog
	chmod +w qa-tests-backend/test-images/syslog/bin/syslog
	docker build \
		-t stackrox/qa:syslog_server_1_0 \
		-t quay.io/rhacs-eng/qa:syslog_server_1_0 \
		-f qa-tests-backend/test-images/syslog/Dockerfile qa-tests-backend/test-images/syslog

.PHONY: central-db-image
central-db-image:
	$(DOCKERBUILD) \
		-t stackrox/central-db:$(TAG) \
		-t $(DEFAULT_IMAGE_REGISTRY)/central-db:$(TAG) \
		$(CENTRAL_DB_DOCKER_ARGS) \
		--file image/postgres/Dockerfile \
		image/postgres
	@echo "Built central-db image with tag $(TAG)"

###########
## Clean ##
###########
.PHONY: clean
clean: clean-image
	@echo "+ $@"

.PHONY: clean-image
clean-image:
	@echo "+ $@"
	git clean -xf image/bin image/rhel/bin
	git clean -xdf image/ui image/rhel/ui image/rhel/docs
	rm -f $(CURDIR)/image/rhel/bundle.tar.gz $(CURDIR)/image/postgres/bundle.tar.gz
	rm -rf $(CURDIR)/image/rhel/scripts

.PHONY: tag
tag:
	@echo $(TAG)

.PHONY: shortcommit
shortcommit:
ifdef SHORTCOMMIT
	@echo $(SHORTCOMMIT)
else
	@git rev-parse --short HEAD
endif

.PHONY: image-flavor
image-flavor:
	@echo $(ROX_IMAGE_FLAVOR)

.PHONY: default-image-registry
default-image-registry:
	@echo $(DEFAULT_IMAGE_REGISTRY)

.PHONY: product-branding
product-branding:
	@echo $(ROX_PRODUCT_BRANDING)

.PHONY: ossls-audit
ossls-audit: deps
	ossls version
	ossls audit

.PHONY: ossls-notice
ossls-notice: deps
	ossls version
	ossls audit --export image/rhel/THIRD_PARTY_NOTICES

.PHONY: collector-tag
collector-tag:
	@echo "$$(cat COLLECTOR_VERSION)"

.PHONY: scanner-tag
scanner-tag:
	@echo "$$(cat SCANNER_VERSION)"

.PHONY: clean-dev-tools
clean-dev-tools: gotools-clean
	@echo "+ $@"

.PHONY: reinstall-dev-tools
reinstall-dev-tools: clean-dev-tools
	@echo "+ $@"
	$(SILENT)$(MAKE) install-dev-tools

.PHONY: install-dev-tools
install-dev-tools: gotools-all
	@echo "+ $@"

.PHONY: roxvet
roxvet: skip-dirs := operator/pkg/clientset
roxvet: $(ROXVET_BIN)
	@echo "+ $@"
	@# TODO(ROX-7574): Add options to ignore specific files or paths in roxvet
	$(SILENT)go list -e ./... \
	    | $(foreach d,$(skip-dirs),grep -v '$(d)' |) \
	    xargs -n 1000 go vet -vettool "$(ROXVET_BIN)" -donotcompareproto -gogoprotofunctions -tags "sql_integration test_e2e test race destructive integration scanner_db_integration compliance externalbackups"
	$(SILENT)go list -e ./... \
	    | $(foreach d,$(skip-dirs),grep -v '$(d)' |) \
	    xargs -n 1000 go vet -vettool "$(ROXVET_BIN)"

##########
## Misc ##
##########
.PHONY: clean-offline-bundle
clean-offline-bundle:
	$(SILENT)find scripts/offline-bundle -name '*.img' -delete -o -name '*.tgz' -delete -o -name 'bin' -type d -exec rm -r "{}" \;

.PHONY: offline-bundle
offline-bundle: clean-offline-bundle
	$(SILENT)./scripts/offline-bundle/create.sh

.PHONY: check-debugger
check-debugger:
	/usr/bin/env DEBUG_BUILD="$(DEBUG_BUILD)" BUILD_TAG="$(BUILD_TAG)" TAG="$(TAG)" ./scripts/check-debugger.sh
ifeq ($(DEBUG_BUILD),yes)
	$(warning Warning: DEBUG_BUILD is enabled. Don not use this for production builds)
endif

.PHONY: policyutil
policyutil:
	@echo "+ $@"
	CGO_ENABLED=0 GOOS=$(HOST_OS) $(GOBUILD) ./tools/policyutil
	go install ./tools/policyutil

.PHONY: mitre
mitre:
	@echo "+ $@"
	CGO_ENABLED=0 GOOS=$(HOST_OS) $(GOBUILD) ./tools/mitre
	go install ./tools/mitre

.PHONY: bootstrap_migration
bootstrap_migration:
	$(SILENT)if [[ "x${DESCRIPTION}" == "x" ]]; then echo "Please set a description for your migration in the DESCRIPTION environment variable"; else go run tools/generate-helpers/bootstrap-migration/main.go --root . --description "${DESCRIPTION}" ;fi

.PHONY: image-prefetcher-deploy-bin
image-prefetcher-deploy-bin: $(IMAGE_PREFETCHER_DEPLOY_BIN) ## download and install

.PHONY: print-image-prefetcher-deploy-bin
print-image-prefetcher-deploy-bin:
	@echo $(IMAGE_PREFETCHER_DEPLOY_BIN)

.PHONY: prometheus-metric-parser
prometheus-metric-parser: $(PROMETHEUS_METRIC_PARSER_BIN)
	@echo $(PROMETHEUS_METRIC_PARSER_BIN)

DEV_VERSION = 4.8.x-nightly-20250307
DEV_LD_FLAGS = -buildvcs=false '-ldflags=-X "github.com/stackrox/rox/pkg/version/internal.MainVersion=$(DEV_VERSION)" -X "github.com/stackrox/rox/pkg/version/internal.CollectorVersion=$(DEV_VERSION)" -X "github.com/stackrox/rox/pkg/version/internal.ScannerVersion=$(DEV_VERSION)" -X "github.com/stackrox/rox/pkg/version/internal.GitShortSha=$(DEV_VERSION)"'

pkg := $(shell find pkg -name *.go)

bin/scanner:  $(shell find scanner -name *.go) ${pkg}
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./scanner/cmd/scanner

bin/kubernetes: $(shell find sensor/kubernetes/ sensor/common/ -name *.go) ${pkg}
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./sensor/kubernetes

bin/admission-control: $(shell find sensor/admission-control/ -name *.go) ${pkg}
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./sensor/admission-control

bin/compliance: $(shell find compliance/ -name *.go) ${pkg}
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./compliance/cmd/compliance

bin/upgrader: $(shell find sensor/upgrader/ -name *.go) ${pkg}
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./sensor/upgrader

bin/init-tls-certs: $(shell find sensor/init-tls-certs/ -name *.go) ${pkg}
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./sensor/init-tls-certs

bin/roxctl: $(shell find roxctl/ -name *.go) ${pkg}
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./roxctl

bin/central: $(shell find central/ -name *.go) ${pkg}
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./central

bin/config-controller: $(shell find config-controller/ -name *.go) ${pkg}
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./config-controller

bin/operator: $(shell find operator/ -name *.go) ${pkg}
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./operator/cmd

bin/migrator: $(shell find migrator/ -name *.go) ${pkg}
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./migrator

central: bin/central bin/config-controller bin/migrator bin/scanner-v4

secured-cluster: bin/kubernetes bin/admission-control bin/compliance bin/upgrader bin/init-tls-certs

bin/scanner-v4: $(shell find scanner/ -name *.go) ${pkg}
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./scanner/cmd/scanner

bin/scanner-v2: $(shell find scannerv2/ -name *.go)
	go build -C scannerv2 $(DEV_LD_FLAGS) -o ../$@ ./cmd/clair

bin/local-nodescanner-v2: $(shell find scannerv2/ -name *.go)
	go build -C scannerv2 $(DEV_LD_FLAGS) -o ../$@ ./tools/local-nodescanner

bin/installer: $(shell find installer/ -name *.go) config-controller/config/crd/bases/config.stackrox.io_securitypolicies.yaml
	mkdir -p installer/manifest/crds/
	cp config-controller/config/crd/bases/config.stackrox.io_securitypolicies.yaml installer/manifest/crds/
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./installer

bin/updater: $(shell find scannerv2/ -name *.go)
	go build -C ./scannerv2 $(DEV_LD_FLAGS) -o ../$@ ./cmd/updater

bin/agent: $(shell find agent/ -name *.go) ${pkg}
	CGO_ENABLED=0 go build $(DEV_LD_FLAGS) -o $@ ./agent

bin/collector: $(shell find collector/ -name *.go) $(shell find collector/ -name *.cpp)
	cmake --preset=vcpkg collector 
	cmake --build collector/cmake-build/vcpkg -j$(nproc)
	cp collector/cmake-build/vcpkg/collector/collector bin/collector
	cp collector/cmake-build/vcpkg/collector/self-checks bin/self-checks

bundle: scannerv2/image/scanner/dump/genesis_manifests.json
	mkdir -p /tmp/genesis-dump
	bin/updater generate-dump --out-file /tmp/genesis-dump/genesis-dump.zip
	ls -lrt /tmp/genesis-dump
	bin/updater print-stats /tmp/genesis-dump/genesis-dump.zip
	mkdir -p bundle/
	unzip -j /tmp/genesis-dump/genesis-dump.zip 'nvd/*.json' -d bundle/nvd_definitions
	unzip -j /tmp/genesis-dump/genesis-dump.zip 'k8s/*.yaml' -d bundle/k8s_definitions
	unzip -j /tmp/genesis-dump/genesis-dump.zip 'istio/*.yaml' -d bundle/istio_definitions
	unzip -j /tmp/genesis-dump/genesis-dump.zip 'rhelv2/repository-to-cpe.json' -d bundle/repo2cpe
	cp /tmp/genesis-dump/genesis-dump.zip bundle
	curl -L https://security.access.redhat.com/data/metrics/container-name-repos-map.json > bundle/repo2cpe/container-name-repos-map.json

ui/build: $(shell find ui -regex '.*.jsx?\|.*.tsx?\|.*.json\|.*.ico\|.*.html\|.*.css\|.*.svg' | grep -v 'build\|node_modules\|cypress')
	make -C ui build

.PHONY: scanner-v2
scanner-v2: bin/scanner-v2 bin/local-nodescanner-v2 bundle

.PHONY: all-binaries
all-binaries: secured-cluster central bin/installer scanner-v2 bin/collector

download: data
	rm -rf data
	mkdir data
	image/rhel/fetch-stackrox-data.sh data

.PHONY: build-combined-image
build-combined-image:
	podman build . | tee /tmp/stackrox-combined-image-tag

.PHONY: push-combined-image-local
push-combined-image-local: build-combined-image
	podman tag $(shell tail -n 1 /tmp/stackrox-combined-image-tag) localhost:5001/stackrox/stackrox:latest
	podman push --tls-verify=false localhost:5001/stackrox/stackrox:latest

.PHONY: combined-image
combined-image: $(GENERATED_API_DOCS) swagger-docs all-binaries download push-combined-image-local
