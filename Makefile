include $(CURDIR)/make/env.mk

ROX_PROJECT=apollo
TESTFLAGS=-race -p 4
BASE_DIR=$(CURDIR)

# Set to empty string to echo some command lines which are hidden by default.
SILENT ?= @

# UNIT_TEST_IGNORE ignores a set of file patterns from the unit test make command.
# the pattern is passed to: grep -Ev
#  usage: "path/to/ignored|another/path"
UNIT_TEST_IGNORE := "stackrox/rox/sensor/tests"

ifeq ($(TAG),)
ifeq (,$(wildcard CI_TAG))
TAG=$(shell git describe --tags --abbrev=10 --dirty --long --exclude '*-nightly-*')
else
TAG=$(shell cat CI_TAG)
endif
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
	elif [[ "$(GOTAGS)" == *"$(RELEASE_GOTAGS)"* ]]; then \
	  echo "stackrox.io"; \
	else \
	  echo "development_build"; \
	fi)

DEFAULT_IMAGE_REGISTRY := quay.io/stackrox-io
ifeq ($(ROX_PRODUCT_BRANDING),RHACS_BRANDING)
	DEFAULT_IMAGE_REGISTRY := quay.io/rhacs-eng
endif

DOCS_IMAGE = $(DEFAULT_IMAGE_REGISTRY)/docs:$(shell make --quiet --no-print-directory docs-tag)

GOBUILD := $(CURDIR)/scripts/go-build.sh

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
ifeq ($(UNAME_S),Darwin)
ifeq ($(UNAME_M),arm64)
	# TODO(ROX-12064) build these images in the CI pipeline
	BUILD_IMAGE = quay.io/rhacs-eng/sandbox:apollo-ci-stackrox-build-0.3.44-arm64
endif
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
GOPATH_WD_OVERRIDES := -w /src -e GOPATH=/go -e GOCACHE=/linux-gocache

null :=
space := $(null) $(null)
comma := ,

.PHONY: all
all: deps style test image

#####################################################################
###### Binaries we depend on (need to be defined on top) ############
#####################################################################

GOLANGCILINT_BIN := $(GOBIN)/golangci-lint
$(GOLANGCILINT_BIN): deps
	@echo "+ $@"
	@cd tools/linters/ && go install github.com/golangci/golangci-lint/cmd/golangci-lint

EASYJSON_BIN := $(GOBIN)/easyjson
$(EASYJSON_BIN): deps
	$(SILENT)echo "+ $@"
	go install github.com/mailru/easyjson/easyjson

GOVERALLS_BIN := $(GOBIN)/goveralls
$(GOVERALLS_BIN): deps
	@echo "+ $@"
	$(SILENT)cd tools/test/ && go install github.com/mattn/goveralls

ROXVET_BIN := $(GOBIN)/roxvet
.PHONY: $(ROXVET_BIN)
$(ROXVET_BIN): deps
	@echo "+ $@"
	go install ./tools/roxvet

STRINGER_BIN := $(GOBIN)/stringer
$(STRINGER_BIN): deps
	@echo "+ $@"
	go install golang.org/x/tools/cmd/stringer

MOCKGEN_BIN := $(GOBIN)/mockgen
$(MOCKGEN_BIN): deps
	@echo "+ $@"
	go install github.com/golang/mock/mockgen

GENNY_BIN := $(GOBIN)/genny
$(GENNY_BIN): deps
	@echo "+ $@"
	go install github.com/mauricelam/genny

GO_JUNIT_REPORT_BIN := $(GOBIN)/go-junit-report
$(GO_JUNIT_REPORT_BIN): deps
	@echo "+ $@"
	$(SILENT)cd tools/test/ && go install github.com/jstemmer/go-junit-report/v2

PROTOLOCK_BIN := $(GOBIN)/protolock
$(PROTOLOCK_BIN): deps
	@echo "+ $@"
	$(SILENT)cd tools/linters/ && go install github.com/nilslice/protolock/cmd/protolock

###########
## Style ##
###########
.PHONY: style
style: golangci-lint roxvet blanks newlines check-service-protos no-large-files storage-protos-compatible ui-lint qa-tests-style shell-style

.PHONY: golangci-lint
golangci-lint: $(GOLANGCILINT_BIN)
ifdef CI
	@echo '+ $@'
	@echo 'The environment indicates we are in CI; running linters in check mode.'
	@echo 'If this fails, run `make style`.'
	@echo "Running with no tags..."
	golangci-lint run
	@echo "Running with release tags..."
	@# We use --tests=false because some unit tests don't compile with release tags,
	@# since they use functions that we don't define in the release build. That's okay.
	golangci-lint run --build-tags "$(subst $(comma),$(space),$(RELEASE_GOTAGS))" --tests=false
else
	golangci-lint run --fix
	golangci-lint run --fix --build-tags "$(subst $(comma),$(space),$(RELEASE_GOTAGS))" --tests=false
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

.PHONY: ci-config-validate
ci-config-validate:
	@echo "+ $@"
	$(SILENT)circleci diagnostic > /dev/null 2>&1 || (echo "Must first set CIRCLECI_CLI_TOKEN or run circleci setup"; exit 1)
	circleci config validate --org-slug gh/stackrox

.PHONY: fast-central-build
fast-central-build:
	@echo "+ $@"
	$(GOBUILD) central

.PHONY: fast-central
fast-central: deps
	@echo "+ $@"
	docker run --rm $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make fast-central-build
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
	docker run --rm $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make fast-migrator-build

.PHONY: fast-migrator-build
fast-migrator-build:
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
	$(SILENT)protolock status -lockdir=$(BASE_DIR)/proto/storage -protoroot=$(BASE_DIR)/proto/storage

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
	$(SILENT)easyjson -pkg pkg/docker/types/types.go
	$(SILENT)easyjson -pkg pkg/docker/types/container.go
	$(SILENT)easyjson -pkg pkg/docker/types/image.go
	$(SILENT)easyjson -pkg pkg/compliance/compress/compress.go

.PHONY: clean-easyjson-srcs
clean-easyjson-srcs:
	@echo "+ $@"
	$(SILENT)find . -name '*_easyjson.go' -exec rm {} \;

.PHONY: go-generated-srcs
go-generated-srcs: deps clean-easyjson-srcs go-easyjson-srcs $(MOCKGEN_BIN) $(STRINGER_BIN) $(GENNY_BIN)
	@echo "+ $@"
	PATH="$(PATH):$(BASE_DIR)/tools/generate-helpers" go generate -v -x ./...

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
	mkdir -p bin/{darwin_amd64,linux_amd64,linux_ppc64le,linux_s390x,windows_amd64}

.PHONY: cli-build
cli-build: build-prep
	RACE=0 CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) ./roxctl
	RACE=0 CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) ./roxctl
	RACE=0 CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le $(GOBUILD) ./roxctl
	RACE=0 CGO_ENABLED=0 GOOS=linux GOARCH=s390x $(GOBUILD) ./roxctl
ifdef CI
	RACE=0 CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) ./roxctl
endif

.PHONY: cli
cli: cli-build
	# Copy the user's specific OS into gopath
	cp bin/$(HOST_OS)_$(GOARCH)/roxctl $(GOPATH)/bin/roxctl
	chmod u+w $(GOPATH)/bin/roxctl

cli-linux: build-prep
	RACE=0 CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) ./roxctl
	RACE=0 CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le $(GOBUILD) ./roxctl
	RACE=0 CGO_ENABLED=0 GOOS=linux GOARCH=s390x $(GOBUILD) ./roxctl

cli-darwin: build-prep
	RACE=0 CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) ./roxctl

upgrader: bin/$(HOST_OS)_$(GOARCH)/upgrader

bin/$(HOST_OS)_$(GOARCH)/upgrader: build-prep
	GOOS=$(HOST_OS) GOARCH=$(GOARCH) $(GOBUILD) ./sensor/upgrader

bin/$(HOST_OS)_$(GOARCH)/admission-control: build-prep
	GOOS=$(HOST_OS) GOARCH=$(GOARCH) $(GOBUILD) ./sensor/admission-control

.PHONY: build-volumes
build-volumes:
	$(SILENT)docker volume inspect $(GOPATH_VOLUME_NAME) >/dev/null 2>&1 || docker volume create $(GOPATH_VOLUME_NAME)
	$(SILENT)docker volume inspect $(GOCACHE_VOLUME_NAME) >/dev/null 2>&1 || docker volume create $(GOCACHE_VOLUME_NAME)

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
	docker run --rm -e CI -e CIRCLE_TAG -e GOTAGS -e DEBUG_BUILD $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make sensor-build

.PHONY: sensor-kubernetes-build-dockerized
sensor-kubernetes-build-dockerized: main-builder-image
	@echo "+ $@"
	docker run -e CI -e CIRCLE_TAG -e GOTAGS -e DEBUG_BUILD $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make sensor-kubernetes-build

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
ifeq ($(CIRCLE_JOB),build-race-condition-debug-image)
	docker container create -e RACE -e CI -e CIRCLE_TAG -e GOTAGS -e DEBUG_BUILD --name builder $(BUILD_IMAGE) make main-build-nodeps
	docker cp $(GOPATH) builder:/
	docker start -i builder
	docker cp builder:/go/src/github.com/stackrox/rox/bin/linux bin/
else
	docker run -i -e RACE -e CI -e CIRCLE_TAG -e GOTAGS -e DEBUG_BUILD --rm $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make main-build-nodeps
endif

.PHONY: main-build-nodeps
main-build-nodeps:
	$(GOBUILD) central migrator sensor/kubernetes sensor/admission-control compliance/collection
	CGO_ENABLED=0 $(GOBUILD) sensor/upgrader
ifndef CI
    CGO_ENABLED=0 $(GOBUILD) roxctl
endif

.PHONY: scale-build
scale-build: build-prep
	@echo "+ $@"
	CGO_ENABLED=0 $(GOBUILD) scale/mocksensor scale/mockcollector scale/profiler scale/chaos

.PHONY: webhookserver-build
webhookserver-build: build-prep
	@echo "+ $@"
	CGO_ENABLED=0 $(GOBUILD) webhookserver

.PHONY: mock-grpc-server-build
mock-grpc-server-build: build-prep
	@echo "+ $@"
	CGO_ENABLED=0 $(GOBUILD) integration-tests/mock-grpc-server

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
	CGO_ENABLED=1 GODEBUG=cgocheck=2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=$(GOTAGS),test scripts/go-test.sh -p 4 -race -cover -coverprofile test-output/coverage.out -v \
		$(shell git ls-files -- '*_test.go' | sed -e 's@^@./@g' | xargs -n 1 dirname | sort | uniq | xargs go list| grep -v '^github.com/stackrox/rox/tests$$' | grep -Ev $(UNIT_TEST_IGNORE)) \
		| tee test-output/test.log
	# Exercise the logging package for all supported logging levels to make sure that initialization works properly
	for encoding in console json; do \
		for level in debug info warn error fatal panic; do \
			LOGENCODING=$$encoding LOGLEVEL=$$level CGO_ENABLED=1 GODEBUG=cgocheck=2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=$(GOTAGS),test scripts/go-test.sh -p 4 -race -v ./pkg/logging/... > /dev/null; \
		done; \
	done

.PHONE: sensor-integration-test
sensor-integration-test: build-prep test-prep
	set -eo pipefail ; \
	for package in $(shell git ls-files ./sensor/tests | grep '_test.go' | xargs -n 1 dirname | uniq | sort | sed -e 's/sensor\/tests\///'); do \
		CGO_ENABLED=1 GODEBUG=cgocheck=2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=$(GOTAGS),test scripts/go-test.sh -p 4 -race -cover -coverprofile test-output/coverage.out -v ./sensor/tests/$$package \
		| tee test-output/$$(echo $$package | sed -e 's/\//\_/').integration.log; \
	done \

.PHONY: go-postgres-unit-tests
go-postgres-unit-tests: build-prep test-prep
	@# The -p 1 passed to go test is required to ensure that tests of different packages are not run in parallel, so as to avoid conflicts when interacting with the DB.
	set -o pipefail ; \
	CGO_ENABLED=1 GODEBUG=cgocheck=2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 ROX_POSTGRES_DATASTORE=true GOTAGS=$(GOTAGS),test,sql_integration scripts/go-test.sh -p 1 -race -cover -coverprofile test-output/coverage.out -v \
		$(shell git ls-files -- '*postgres/*_test.go' '*postgres_test.go' '*datastore_sac_test.go' | sed -e 's@^@./@g' | xargs -n 1 dirname | sort | uniq | xargs go list| grep -v '^github.com/stackrox/rox/tests$$' | grep -Ev $(UNIT_TEST_IGNORE)) \
		| tee test-output/test.log

.PHONY: shell-unit-tests
shell-unit-tests:
	@echo "+ $@"
	$(SILENT)mkdir -p shell-test-output
	bats --print-output-on-failure --verbose-run --recursive --report-formatter junit --output shell-test-output scripts

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
		| tee test-output/test.log

generate-junit-reports: $(GO_JUNIT_REPORT_BIN)
	$(BASE_DIR)/scripts/generate-junit-reports.sh

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

$(CURDIR)/image/rhel/bundle.tar.gz:
	/usr/bin/env DEBUG_BUILD="$(DEBUG_BUILD)" $(CURDIR)/image/rhel/create-bundle.sh $(CURDIR)/image stackrox-data:$(TAG) $(BUILD_IMAGE) $(CURDIR)/image/rhel

.PHONY: $(CURDIR)/image/rhel/Dockerfile.gen
$(CURDIR)/image/rhel/Dockerfile.gen:
	ROX_IMAGE_FLAVOR=$(ROX_IMAGE_FLAVOR) \
	LABEL_VERSION=$(TAG) \
	LABEL_RELEASE=$(TAG) \
	QUAY_TAG_EXPIRATION=$(QUAY_TAG_EXPIRATION) \
	envsubst '$${ROX_IMAGE_FLAVOR} $${LABEL_VERSION} $${LABEL_RELEASE} $${QUAY_TAG_EXPIRATION}' \
	< $(CURDIR)/image/rhel/Dockerfile.envsubst > $(CURDIR)/image/rhel/Dockerfile.gen

.PHONY: docker-build-main-image
docker-build-main-image: copy-binaries-to-image-dir docker-build-data-image central-db-image \
                         $(CURDIR)/image/rhel/bundle.tar.gz $(CURDIR)/image/rhel/Dockerfile.gen
	docker build \
		-t stackrox/main:$(TAG) \
		-t $(DEFAULT_IMAGE_REGISTRY)/main:$(TAG) \
		--build-arg ROX_PRODUCT_BRANDING=$(ROX_PRODUCT_BRANDING) \
		--file image/rhel/Dockerfile.gen \
		image/rhel
	@echo "Built main image for RHEL with tag: $(TAG), image flavor: $(ROX_IMAGE_FLAVOR)"
	@echo "You may wish to:       export MAIN_IMAGE_TAG=$(TAG)"

.PHONY: docs-image
docs-image:
	scripts/ensure_image.sh $(DOCS_IMAGE) docs/Dockerfile docs/

.PHONY: docker-build-data-image
docker-build-data-image: docs-image
	docker build -t stackrox-data:$(TAG) \
	    --build-arg DOCS_IMAGE=$(DOCS_IMAGE) \
		--label quay.expires-after=$(QUAY_TAG_EXPIRATION) \
		image/ \
		--file image/stackrox-data.Dockerfile

.PHONY: docker-build-roxctl-image
docker-build-roxctl-image:
	cp -f bin/linux_$(GOARCH)/roxctl image/roxctl/roxctl-linux
	docker build \
		-t stackrox/roxctl:$(TAG) \
		-t $(DEFAULT_IMAGE_REGISTRY)/roxctl:$(TAG) \
		-f image/roxctl/Dockerfile \
		--label quay.expires-after=$(QUAY_TAG_EXPIRATION) \
		image/roxctl

.PHONY: copy-go-binaries-to-image-dir
copy-go-binaries-to-image-dir:
	cp bin/linux_$(GOARCH)/central image/bin/central
ifdef CI
	cp bin/linux_amd64/roxctl image/bin/roxctl-linux-amd64
	cp bin/linux_ppc64le/roxctl image/bin/roxctl-linux-ppc64le
	cp bin/linux_s390x/roxctl image/bin/roxctl-linux-s390x
	cp bin/darwin_amd64/roxctl image/bin/roxctl-darwin-amd64
	cp bin/windows_amd64/roxctl.exe image/bin/roxctl-windows-amd64.exe
else
ifneq ($(HOST_OS),linux)
	cp bin/linux_$(GOARCH)/roxctl image/bin/roxctl-linux-$(GOARCH)
endif
	cp bin/$(HOST_OS)_amd64/roxctl image/bin/roxctl-$(HOST_OS)-amd64
endif
	cp bin/linux_$(GOARCH)/migrator image/bin/migrator
	cp bin/linux_$(GOARCH)/kubernetes        image/bin/kubernetes-sensor
	cp bin/linux_$(GOARCH)/upgrader          image/bin/sensor-upgrader
	cp bin/linux_$(GOARCH)/admission-control image/bin/admission-control
	cp bin/linux_$(GOARCH)/collection        image/bin/compliance
	# Workaround to bug in lima: https://github.com/lima-vm/lima/issues/602
	find image/bin -not -path "*/.*" -type f -exec chmod +x {} \;


.PHONY: copy-binaries-to-image-dir
copy-binaries-to-image-dir: copy-go-binaries-to-image-dir
	cp -r ui/build image/ui/
ifdef CI
	$(SILENT)[ -d image/THIRD_PARTY_NOTICES ] || { echo "image/THIRD_PARTY_NOTICES dir not found! It is required for CI-built images."; exit 1; }
else
	$(SILENT)[ -f image/THIRD_PARTY_NOTICES ] || mkdir -p image/THIRD_PARTY_NOTICES
endif
	$(SILENT)[ -d image/docs ] || { echo "Generated docs not found in image/docs. They are required for build."; exit 1; }

.PHONY: scale-image
scale-image: scale-build clean-image
	cp bin/linux_$(GOARCH)/mocksensor scale/image/bin/mocksensor
	cp bin/linux_$(GOARCH)/mockcollector scale/image/bin/mockcollector
	cp bin/linux_$(GOARCH)/profiler scale/image/bin/profiler
	cp bin/linux_$(GOARCH)/chaos scale/image/bin/chaos
	chmod +w scale/image/bin/*
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

.PHONY: mock-grpc-server-image
mock-grpc-server-image: mock-grpc-server-build clean-image
	cp bin/linux_$(GOARCH)/mock-grpc-server integration-tests/mock-grpc-server/image/bin/mock-grpc-server
	docker build \
		-t stackrox/grpc-server:$(TAG) \
		-t quay.io/rhacs-eng/grpc-server:$(TAG) \
		integration-tests/mock-grpc-server/image

$(CURDIR)/image/postgres/bundle.tar.gz:
	/usr/bin/env DEBUG_BUILD="$(DEBUG_BUILD)" $(CURDIR)/image/postgres/create-bundle.sh $(CURDIR)/image/postgres $(CURDIR)/image/postgres

.PHONY: $(CURDIR)/image/postgres/Dockerfile.gen
$(CURDIR)/image/postgres/Dockerfile.gen:
	ROX_IMAGE_FLAVOR=$(ROX_IMAGE_FLAVOR) \
	envsubst '$${ROX_IMAGE_FLAVOR}' \
	< $(CURDIR)/image/postgres/Dockerfile.envsubst > $(CURDIR)/image/postgres/Dockerfile.gen

.PHONY: central-db-image
central-db-image: $(CURDIR)/image/postgres/bundle.tar.gz $(CURDIR)/image/postgres/Dockerfile.gen
	docker build \
		-t stackrox/central-db:$(TAG) \
		-t $(DEFAULT_IMAGE_REGISTRY)/central-db:$(TAG) \
		--build-arg ROX_IMAGE_FLAVOR=$(ROX_IMAGE_FLAVOR) \
		--file image/postgres/Dockerfile.gen \
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
	git clean -xf image/bin
	git clean -xdf image/ui image/docs
	git clean -xf integration-tests/mock-grpc-server/image/bin/mock-grpc-server
	rm -f $(CURDIR)/image/rhel/bundle.tar.gz $(CURDIR)/image/postgres/bundle.tar.gz
	rm -rf $(CURDIR)/image/rhel/scripts

.PHONY: tag
tag:
ifneq (,$(wildcard CI_TAG))
	@cat CI_TAG
else
ifdef COMMIT
	@git describe $(COMMIT) --tags --abbrev=10 --long --exclude '*-nightly-*'
else
	@echo $(TAG)
endif
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
	ossls audit --export image/THIRD_PARTY_NOTICES

.PHONY: collector-tag
collector-tag:
	@cat COLLECTOR_VERSION

.PHONY: docs-tag
docs-tag:
	@echo $$(git ls-tree -d --abbrev=8 HEAD docs | awk '{ print $$3; }')-$$(git submodule status -- docs/content | cut -c 2-9)-$$(git submodule status -- docs/tools | cut -c 2-9)

.PHONY: scanner-tag
scanner-tag:
	@cat SCANNER_VERSION

GET_DEVTOOLS_CMD := $(MAKE) -qp | sed -e '/^\# Not a target:$$/{ N; d; }' | egrep -v '^(\s*(\#.*)?$$|\s|%|\(|\.)' | egrep '^[^[:space:]:]*:' | cut -d: -f1 | sort | uniq | grep '^$(GOBIN)/'
.PHONY: clean-dev-tools
clean-dev-tools:
	@echo "+ $@"
	$(SILENT)$(GET_DEVTOOLS_CMD) | xargs rm -fv

.PHONY: reinstall-dev-tools
reinstall-dev-tools: clean-dev-tools
	@echo "+ $@"
	$(SILENT)$(MAKE) install-dev-tools

.PHONY: install-dev-tools
install-dev-tools:
	@echo "+ $@"
	$(SILENT)test -n "$(GOPATH)" || { echo "Set GOPATH before installing dev tools"; exit 1; }
	$(SILENT)$(GET_DEVTOOLS_CMD) | xargs $(MAKE)
ifeq ($(UNAME_S),Darwin)
	@echo "Please manually install RocksDB if you haven't already. See README for details"
endif

.PHONY: roxvet
roxvet: $(ROXVET_BIN)
	@echo "+ $@"
	# TODO(ROX-7574): Add options to ignore specific files or paths in roxvet
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
	/usr/bin/env DEBUG_BUILD="$(DEBUG_BUILD)" CIRCLE_TAG="$(CIRCLE_TAG)" TAG="$(TAG)" ./scripts/check-debugger.sh
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
