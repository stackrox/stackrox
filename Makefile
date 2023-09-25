include $(CURDIR)/make/env.mk

PLATFORM ?= linux/amd64
ROX_PROJECT=apollo
TESTFLAGS=-race -p 4
BASE_DIR=$(CURDIR)

ifeq (,$(findstring podman,$(shell docker --version 2>/dev/null)))
# Podman DTRT by running processes unprivileged in containers,
# but it's UID mapping is more nuanced. Only set user for vanilla docker.
DOCKER_USER=--user "$(shell id -u)"
endif

# Set to empty string to echo some command lines which are hidden by default.
SILENT ?= @

# UNIT_TEST_IGNORE ignores a set of file patterns from the unit test make command.
# the pattern is passed to: grep -Ev
#  usage: "path/to/ignored|another/path"
# TODO: [ROX-19070] Update postgres store test generation to work for foreign keys
UNIT_TEST_IGNORE := "stackrox/rox/sensor/tests|stackrox/rox/operator/tests|stackrox/rox/central/reports/config/store/postgres"

ifeq ($(TAG),)
TAG=$(shell git describe --tags --abbrev=10 --dirty --long --exclude '*-nightly-*')
endif

# Set expiration on Quay.io for non-release tags.
ifeq ($(findstring x,$(TAG)),x)
QUAY_TAG_EXPIRATION=13w
else
QUAY_TAG_EXPIRATION=never
endif

ROX_PRODUCT_BRANDING ?= STACKROX_BRANDING

# ROX_IMAGE_FLAVOR is an ARG used in Dockerfiles that defines the default registries for main, scaner, and collector images.
# ROX_IMAGE_FLAVOR valid values are: development_build, stackrox.io, rhacs, opensource.
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
	BUILD_IMAGE = docker.io/library/golang:$(shell cat EXPECTED_GO_VERSION | cut -c 3-)
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

ifeq ($(UNAME_S),Darwin)
BIND_GOCACHE ?= 0
BIND_GOPATH ?= 0
else
BIND_GOCACHE ?= 1
BIND_GOPATH ?= 1
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

LOCAL_VOLUME_ARGS := -v$(CURDIR):/src:delegated -v $(GOCACHE_VOLUME_SRC):/linux-gocache:delegated -v $(GOPATH_VOLUME_SRC):/go:delegated
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

$(call go-tool, GOLANGCILINT_BIN, github.com/golangci/golangci-lint/cmd/golangci-lint, tools/linters)
$(call go-tool, EASYJSON_BIN, github.com/mailru/easyjson/easyjson)
$(call go-tool, CONTROLLER_GEN_BIN, sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0)
$(call go-tool, ROXVET_BIN, ./tools/roxvet)
$(call go-tool, STRINGER_BIN, golang.org/x/tools/cmd/stringer)
$(call go-tool, MOCKGEN_BIN, go.uber.org/mock/mockgen)
$(call go-tool, GO_JUNIT_REPORT_BIN, github.com/jstemmer/go-junit-report/v2, tools/test)
$(call go-tool, PROTOLOCK_BIN, github.com/nilslice/protolock/cmd/protolock, tools/linters)

###########
## Style ##
###########
.PHONY: style
style: golangci-lint style-slim

.PHONY: style-slim
style-slim: roxvet blanks newlines check-service-protos no-large-files storage-protos-compatible ui-lint qa-tests-style shell-style

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
	$(GOLANGCILINT_BIN) run $(GOLANGCILINT_FLAGS)
	@echo "Running with release tags..."
	@# We use --tests=false because some unit tests don't compile with release tags,
	@# since they use functions that we don't define in the release build. That's okay.
	$(GOLANGCILINT_BIN) run $(GOLANGCILINT_FLAGS) --build-tags "$(subst $(comma),$(space),$(RELEASE_GOTAGS))" --tests=false
else
	$(GOLANGCILINT_BIN) run $(GOLANGCILINT_FLAGS) --fix
	$(GOLANGCILINT_BIN) run $(GOLANGCILINT_FLAGS) --fix --build-tags "$(subst $(comma),$(space),$(RELEASE_GOTAGS))" --tests=false
endif

.PHONY: qa-tests-style
qa-tests-style:
	@echo "+ $@"
	make -C qa-tests-backend/ style

.PHONY: ui-lint
ui-lint:
	@echo "+ $@"
	make -C ui lint

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

.PHONY: fast-central
fast-central: deps
	@echo "+ $@"
	docker run $(DOCKER_USER) --rm $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make fast-central-build
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
	docker run $(DOCKER_USER) --rm $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make fast-migrator-build

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

.PHONY: storage-protos-compatible
storage-protos-compatible: $(PROTOLOCK_BIN)
	@echo "+ $@"
	$(SILENT)$(PROTOLOCK_BIN) status -lockdir=$(BASE_DIR)/proto/storage -protoroot=$(BASE_DIR)/proto/storage

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

PROTO_GENERATED_SRCS = $(GENERATED_PB_SRCS) $(GENERATED_API_GW_SRCS)

include make/protogen.mk

.PHONY: go-easyjson-srcs
go-easyjson-srcs: $(EASYJSON_BIN)
	@echo "+ $@"
	@# Files are ordered such that repeated runs of `make go-easyjson-srcs` don't create diffs.
	$(SILENT)$(EASYJSON_BIN) pkg/docker/types/image.go
	$(SILENT)$(EASYJSON_BIN) pkg/docker/types/container.go
	$(SILENT)$(EASYJSON_BIN) pkg/docker/types/types.go
	$(SILENT)$(EASYJSON_BIN) pkg/compliance/compress/compress.go

.PHONY: clean-easyjson-srcs
clean-easyjson-srcs:
	@echo "+ $@"
	$(SILENT)find . -name '*_easyjson.go' -exec rm {} \;

.PHONY: go-generated-srcs
go-generated-srcs: deps clean-easyjson-srcs go-easyjson-srcs $(MOCKGEN_BIN) $(STRINGER_BIN)
	@echo "+ $@"
	PATH="$(GOTOOLS_BIN):$(PATH):$(BASE_DIR)/tools/generate-helpers" MOCKGEN_BIN="$(MOCKGEN_BIN)" go generate -v -x ./...

proto-generated-srcs: $(PROTO_GENERATED_SRCS) $(GENERATED_API_SWAGGER_SPECS)
	@echo "+ $@"
	$(SILENT)touch proto-generated-srcs
	$(SILENT)$(MAKE) clean-obsolete-protos

clean-proto-generated-srcs:
	@echo "+ $@"
	git clean -xdf generated

.PHONY: generated-srcs
generated-srcs: go-generated-srcs

deps: $(BASE_DIR)/go.sum tools/linters/go.sum tools/test/go.sum
	@echo "+ $@"
	$(SILENT)touch deps

%/go.sum: %/go.mod
	$(SILENT)cd $*
	@echo "+ $@"
	$(SILENT)$(eval GOMOCK_REFLECT_DIRS=`find . -type d -name 'gomock_reflect_*'`)
	$(SILENT)test -z $(GOMOCK_REFLECT_DIRS) || { echo "Found leftover gomock directories. Please remove them and rerun make deps!"; echo $(GOMOCK_REFLECT_DIRS); exit 1; }
	$(SILENT)go mod tidy
ifdef CI
	$(SILENT)git diff --exit-code -- go.mod go.sum || { echo "go.mod/go.sum files were updated after running 'go mod tidy', run this command on your local machine and commit the results." ; exit 1 ; }
	go mod verify
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
ifneq ($(DOCKER_USER),)
	@echo "Restoring user's ownership of linux-gocache and go directories after previous runs which could set it to root..."
	$(SILENT)docker run --rm $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) chown -R "$(shell id -u)" /linux-gocache /go
endif

.PHONY: main-builder-image
main-builder-image: build-volumes
	@echo "+ $@"
	$(SILENT)# Ensure that the go version in the image matches the expected version
	# If the next line fails, you need to update the go version in rox-ci-image/images/stackrox-build.Dockerfile
	grep -q "$(shell head -n 1 EXPECTED_GO_VERSION)" <(docker run --rm "$(BUILD_IMAGE)" go version)

.PHONY: main-build
main-build: build-prep main-build-dockerized
	@echo "+ $@"

.PHONY: sensor-build-dockerized
sensor-build-dockerized: main-builder-image
	@echo "+ $@"
	docker run $(DOCKER_USER) --rm -e CI -e BUILD_TAG -e GOTAGS -e DEBUG_BUILD $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make sensor-build

.PHONY: sensor-kubernetes-build-dockerized
sensor-kubernetes-build-dockerized: main-builder-image
	@echo "+ $@"
	docker run $(DOCKER_USER) -e CI -e BUILD_TAG -e GOTAGS -e DEBUG_BUILD $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make sensor-kubernetes-build

.PHONY: sensor-build
sensor-build:
	$(GOBUILD) sensor/kubernetes sensor/admission-control
	CGO_ENABLED=0 $(GOBUILD) sensor/upgrader

.PHONY: sensor-kubernetes-build
sensor-kubernetes-build:
	$(GOBUILD) sensor/kubernetes

.PHONY: main-build-dockerized
main-build-dockerized: main-builder-image
	@echo "+ $@"
	docker run $(DOCKER_USER) -i -e RACE -e CI -e BUILD_TAG -e GOTAGS -e DEBUG_BUILD --rm $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make main-build-nodeps

.PHONY: main-build-nodeps
main-build-nodeps: central-build-nodeps migrator-build-nodeps
	$(GOBUILD) sensor/kubernetes sensor/admission-control compliance/collection
	$(GOBUILD) sensor/upgrader
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

.PHONY: mock-grpc-server-build
mock-grpc-server-build: build-prep
	for GOARCH in $$(echo $(PLATFORM) | sed -e 's/,/ /g') ; do \
		GOARCH="$${GOARCH##*/}" CGO_ENABLED=0 $(GOBUILD) integration-tests/mock-grpc-server; \
	done

.PHONY: gendocs
gendocs: $(GENERATED_API_DOCS)
	@echo "+ $@"

# We don't need to do anything here, because the $(MERGED_API_SWAGGER_SPEC) target already performs validation.
.PHONY: swagger-docs
swagger-docs: $(MERGED_API_SWAGGER_SPEC)
	@echo "+ $@"

UNIT_TEST_PACKAGES ?= ./...

.PHONY: test-prep
test-prep:
	@echo "+ $@"
	$(SILENT)mkdir -p test-output

.PHONY: go-unit-tests
go-unit-tests: build-prep test-prep
	set -o pipefail ; \
	CGO_ENABLED=1 GODEBUG=cgocheck=2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=$(GOTAGS),test scripts/go-test.sh -timeout 15m -race -cover -coverprofile test-output/coverage.out -v \
		$(shell git ls-files -- '*_test.go' | sed -e 's@^@./@g' | xargs -n 1 dirname | sort | uniq | xargs go list| grep -v '^github.com/stackrox/rox/tests$$' | grep -Ev $(UNIT_TEST_IGNORE)) \
		| tee $(GO_TEST_OUTPUT_PATH)
	# Exercise the logging package for all supported logging levels to make sure that initialization works properly
	for encoding in console json; do \
		for level in debug info warn error fatal panic; do \
			LOGENCODING=$$encoding LOGLEVEL=$$level CGO_ENABLED=1 GODEBUG=cgocheck=2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=$(GOTAGS),test scripts/go-test.sh -p 4 -race -v ./pkg/logging/... > /dev/null; \
		done; \
	done

.PHONE: sensor-integration-test
sensor-integration-test: build-prep test-prep
	set -eo pipefail ; \
	rm -rf  $(GO_TEST_OUTPUT_PATH); \
	for package in $(shell git ls-files ./sensor/tests | grep '_test.go' | xargs -n 1 dirname | uniq | sort | sed -e 's/sensor\/tests\///'); do \
		CGO_ENABLED=1 GODEBUG=cgocheck=2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=$(GOTAGS),test scripts/go-test.sh -p 4 -race -cover -coverprofile test-output/coverage.out -v ./sensor/tests/$$package \
		| tee -a $(GO_TEST_OUTPUT_PATH); \
	done \

sensor-pipeline-benchmark: build-prep test-prep
	LOGLEVEL="panic" go test -bench=. -run=^# -benchtime=30s -count=5 ./sensor/tests/pipeline | tee $(CURDIR)/test-output/pipeline.results.txt

.PHONY: go-postgres-unit-tests
go-postgres-unit-tests: build-prep test-prep
	@# The -p 1 passed to go test is required to ensure that tests of different packages are not run in parallel, so as to avoid conflicts when interacting with the DB.
	set -o pipefail ; \
	CGO_ENABLED=1 GODEBUG=cgocheck=2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 ROX_POSTGRES_DATASTORE=true GOTAGS=$(GOTAGS),test,sql_integration scripts/go-test.sh -p 1 -race -cover -coverprofile test-output/coverage.out -v \
		$(shell git grep -rl "//go:build sql_integration" central pkg migrator tools | sed -e 's@^@./@g' | xargs -n 1 dirname | sort | uniq | xargs go list -tags sql_integration | grep -v '^github.com/stackrox/rox/tests$$' | grep -Ev $(UNIT_TEST_IGNORE)) \
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
all-builds: cli main-build clean-image $(MERGED_API_SWAGGER_SPEC) ui-build

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
ifeq ("$(CLUSTER_TYPE)","kind")
	@echo "Loading image $(DEFAULT_IMAGE_REGISTRY)/main:$(TAG) into kind"
	kind load docker-image $(DEFAULT_IMAGE_REGISTRY)/main:$(TAG)
	@echo "Loading image stackrox/main:$(TAG) into kind"
	kind load docker-image stackrox/main:$(TAG)
endif

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
ifdef CI
	cp bin/linux_amd64/roxctl image/rhel/bin/roxctl-linux-amd64
	cp bin/linux_arm64/roxctl image/rhel/bin/roxctl-linux-arm64
	cp bin/linux_ppc64le/roxctl image/rhel/bin/roxctl-linux-ppc64le
	cp bin/linux_s390x/roxctl image/rhel/bin/roxctl-linux-s390x
	cp bin/darwin_amd64/roxctl image/rhel/bin/roxctl-darwin-amd64
	cp bin/windows_amd64/roxctl.exe image/rhel/bin/roxctl-windows-amd64.exe
else
ifneq ($(HOST_OS),linux)
	cp bin/linux_$(GOARCH)/roxctl image/rhel/bin/roxctl-linux-$(GOARCH)
endif
	cp bin/$(HOST_OS)_amd64/roxctl image/rhel/bin/roxctl-$(HOST_OS)-amd64
endif
	cp bin/linux_$(GOARCH)/migrator image/rhel/bin/migrator
	cp bin/linux_$(GOARCH)/kubernetes        image/rhel/bin/kubernetes-sensor
	cp bin/linux_$(GOARCH)/upgrader          image/rhel/bin/sensor-upgrader
	cp bin/linux_$(GOARCH)/admission-control image/rhel/bin/admission-control
	cp bin/linux_$(GOARCH)/collection        image/rhel/bin/compliance
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

.PHONY: mock-grpc-server-image
mock-grpc-server-image: mock-grpc-server-build clean-image
	cp -R bin integration-tests/mock-grpc-server/image
	docker buildx build --platform $(PLATFORM) --load \
		-t stackrox/grpc-server:$(TAG) \
		-t quay.io/rhacs-eng/grpc-server:$(TAG) \
		integration-tests/mock-grpc-server/image

.PHONY: mock-grpc-server-image-push
mock-grpc-server-image-push: mock-grpc-server-build
	cp -R bin integration-tests/mock-grpc-server/image
	docker buildx build --platform $(PLATFORM) --push \
		-t quay.io/rhacs-eng/grpc-server:$(TAG) \
		integration-tests/mock-grpc-server/image

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
	git clean -xf integration-tests/mock-grpc-server/image/bin/mock-grpc-server
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
	@cat COLLECTOR_VERSION

.PHONY: scanner-tag
scanner-tag:
	@cat SCANNER_VERSION

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
ifeq ($(UNAME_S),Darwin)
	@echo "Please manually install RocksDB if you haven't already. See README for details"
endif

.PHONY: roxvet
roxvet: $(ROXVET_BIN)
	@echo "+ $@"
	@# TODO(ROX-7574): Add options to ignore specific files or paths in roxvet
	$(SILENT)go vet -vettool "$(ROXVET_BIN)" $(shell go list -e ./... | grep -v 'operator/pkg/clientset')

##########
## Misc ##
##########
.PHONY: clean-offline-bundle
clean-offline-bundle:
	$(SILENT)find scripts/offline-bundle -name '*.img' -delete -o -name '*.tgz' -delete -o -name 'bin' -type d -exec rm -r "{}" \;

.PHONY: offline-bundle
offline-bundle: clean-offline-bundle
	$(SILENT)./scripts/offline-bundle/create.sh

.PHONY: ui-publish-packages
ui-publish-packages:
	make -C ui publish-packages

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
