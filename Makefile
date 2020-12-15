include $(CURDIR)/make/env.mk

ROX_PROJECT=apollo
TESTFLAGS=-race -p 4
BASE_DIR=$(CURDIR)

ifeq ($(TAG),)
TAG=$(shell git describe --tags --abbrev=10 --dirty --long)
endif

ALPINE_MIRROR_BUILD_ARG := $(ALPINE_MIRROR:%=--build-arg ALPINE_MIRROR=%)

# Compute the tag of the build image based on the contents of the tracked files in
# build. This ensures that we build it if and only if necessary, pulling from DockerHub
# otherwise.
# `git ls-files -sm build` prints all files in build (including extra entries for locally
# modified files), along with the SHAs, and `git hash-object` just computes the SHA of that.
BUILD_DIR_HASH := $(shell git ls-files -sm build | git hash-object --stdin)
BUILD_IMAGE := stackrox/main:rocksdb-builder-$(BUILD_DIR_HASH)
RHEL_BUILD_IMAGE := stackrox/main:rocksdb-builder-rhel-$(BUILD_DIR_HASH)

GOBUILD := $(CURDIR)/scripts/go-build.sh

GOPATH_VOLUME_NAME := stackrox-rox-gopath
GOCACHE_VOLUME_NAME := stackrox-rox-gocache


# Figure out whether to use standalone Docker volume for GOPATH/Go build cache, or bind
# mount one from the host filesystem.
# The latter is painfully slow on Mac OS X with Docker Desktop, so we default to using a
# standalone volume in that case, and to bind mounting otherwise.
UNAME_S := $(shell uname -s)

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

SSH_AUTH_SOCK_MAGIC_PATH := /run/host-services/ssh-auth.sock
LOCAL_VOLUME_ARGS := -v$(CURDIR):/src:delegated -v $(SSH_AUTH_SOCK_MAGIC_PATH):$(SSH_AUTH_SOCK_MAGIC_PATH) -e SSH_AUTH_SOCK=$(SSH_AUTH_SOCK_MAGIC_PATH) -v $(GOCACHE_VOLUME_SRC):/linux-gocache:delegated -v $(GOPATH_VOLUME_SRC):/go:delegated -v $(HOME)/.ssh:/root/.ssh:ro -v $(HOME)/.gitconfig:/root/.gitconfig:ro
GOPATH_WD_OVERRIDES := -w /src -e GOPATH=/go

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
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

EASYJSON_BIN := $(GOBIN)/easyjson
$(EASYJSON_BIN): deps
	@echo "+ $@"
	go install github.com/mailru/easyjson/easyjson

STATICCHECK_BIN := $(GOBIN)/staticcheck
$(STATICCHECK_BIN): deps
	@echo "+ $@"
	@go install honnef.co/go/tools/cmd/staticcheck

GOVERALLS_BIN := $(GOBIN)/goveralls
$(GOVERALLS_BIN): deps
	@echo "+ $@"
	go install github.com/mattn/goveralls

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

PACKR_BIN := $(GOBIN)/packr
$(PACKR_BIN): deps
	@echo "+ $@"
	go install github.com/gobuffalo/packr/packr

GO_JUNIT_REPORT_BIN := $(GOBIN)/go-junit-report
$(GO_JUNIT_REPORT_BIN): deps
	@echo "+ $@"
	go install github.com/jstemmer/go-junit-report

PROTOLOCK_BIN := $(GOBIN)/protolock
$(PROTOLOCK_BIN): deps
	@echo "+ $@"
	@go install github.com/nilslice/protolock/cmd/protolock

###########
## Style ##
###########
.PHONY: style
style: golangci-lint roxvet blanks newlines validateimports no-large-files storage-protos-compatible ui-lint qa-tests-style

# staticcheck is useful, but extremely computationally intensive on some people's machines.
# Therefore, to allow people to continue running `make style`, staticcheck is not run along with
# the other style targets by default, when running locally.
# It is always run in CI.
# To run it locally along with the other style targets, you can `export RUN_STATIC_CHECK=true`.
# If you want to run just staticcheck, you can, of course, just `make staticcheck`.
ifdef CI
style: staticcheck
endif

ifdef RUN_STATIC_CHECK
style: staticcheck
endif

.PHONY: golangci-lint
golangci-lint: $(GOLANGCILINT_BIN) volatile-generated-srcs
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

.PHONY: staticcheck
staticcheck: $(STATICCHECK_BIN)
	@echo "+ $@"
	@$(BASE_DIR)/tools/staticcheck-wrap.sh ./...

.PHONY: fast-central-build
fast-central-build:
	@echo "+ $@"
	$(GOBUILD) central

.PHONY: fast-central
fast-central: deps
	@echo "+ $@"
	docker run --rm $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make fast-central-build
	@$(BASE_DIR)/scripts/k8s/kill-central.sh

# fast is a dev mode options when using local dev
# it will automatically restart Central if there are any changes
.PHONY: fast
fast: fast-central

.PHONY: fast-sensor
fast-sensor: sensor-build-dockerized

.PHONY: fast-sensor-kubernetes
fast-sensor-kubernetes: sensor-kubernetes-build-dockerized

.PHONY: service-init-build
service-init-build:
	@echo "+ $@"
	GOOS=linux CGO_ENABLED=0 $(GOBUILD) sensor/service-init
	GOOS=darwin CGO_ENABLED=0 $(GOBUILD) sensor/service-init

.PHONY: validateimports
validateimports:
	@echo "+ $@"
	@go run $(BASE_DIR)/tools/validateimports/verify.go $(shell go list -e ./...)

.PHONY: no-large-files
no-large-files:
	@echo "+ $@"
	@$(BASE_DIR)/tools/large-git-files/find.sh

.PHONY: keys
keys:
	@echo "+ $@"
	go generate github.com/stackrox/rox/central/ed

.PHONY: storage-protos-compatible
storage-protos-compatible: $(PROTOLOCK_BIN)
	@echo "+ $@"
	@protolock status -lockdir=$(BASE_DIR)/proto/storage -protoroot=$(BASE_DIR)/proto/storage

.PHONY: blanks
blanks:
	@echo "+ $@"
ifdef CI
	@git grep -L '^// Code generated by .* DO NOT EDIT\.' -- '*.go' | xargs -n 1000 $(BASE_DIR)/tools/import_validate.py
else
	@git grep -L '^// Code generated by .* DO NOT EDIT\.' -- '*.go' | xargs -n 1000 $(BASE_DIR)/tools/fix-blanks.sh
endif

.PHONY: newlines
newlines:
	@echo "+ $@"
ifdef CI
	@git grep --cached -Il '' | xargs $(BASE_DIR)/tools/check-newlines.sh
else
	@git grep --cached -Il '' | xargs $(BASE_DIR)/tools/check-newlines.sh --fix
endif

.PHONY: init-githooks
init-githooks:
	@echo "+ $@"
	./tools/githooks/install-hooks.sh

.PHONY: dev
dev: install-dev-tools
	@echo "+ $@"

#####################################
## Generated Code and Dependencies ##
#####################################

PROTO_GENERATED_SRCS = $(GENERATED_PB_SRCS) $(GENERATED_API_GW_SRCS)

include make/protogen.mk

.PHONY: go-packr-srcs
go-packr-srcs: $(PACKR_BIN)
	@echo "+ $@"
	@packr

# For some reasons, a `packr clean` is much slower than the `find`. It also does not work.
.PHONY: clean-packr-srcs
clean-packr-srcs:
	@echo "+ $@"
	@find . -name '*-packr.go' -exec rm {} \;

.PHONY: go-easyjson-srcs
go-easyjson-srcs: $(EASYJSON_BIN)
	@echo "+ $@"
	@easyjson -pkg pkg/docker/types/types.go
	@echo "//lint:file-ignore SA4006 This is a generated file" >> pkg/docker/types/types_easyjson.go
	@easyjson -pkg pkg/docker/types/container.go
	@echo "//lint:file-ignore SA4006 This is a generated file" >> pkg/docker/types/container_easyjson.go
	@easyjson -pkg pkg/docker/types/image.go
	@echo "//lint:file-ignore SA4006 This is a generated file" >> pkg/docker/types/image_easyjson.go
	@easyjson -pkg pkg/compliance/compress/compress.go
    @echo "//lint:file-ignore SA4006 This is a generated file" >> pkg/docker/types/compress_easyjson.go

.PHONY: clean-easyjson-srcs
clean-easyjson-srcs:
	@echo "+ $@"
	@find . -name '*_easyjson.go' -exec rm {} \;

.PHONY: go-generated-srcs
go-generated-srcs: deps go-easyjson-srcs $(MOCKGEN_BIN) $(STRINGER_BIN) $(GENNY_BIN)
	@echo "+ $@"
	PATH=$(PATH):$(BASE_DIR)/tools/generate-helpers go generate ./...

proto-generated-srcs: $(PROTO_GENERATED_SRCS)
	@echo "+ $@"
	@touch proto-generated-srcs

clean-proto-generated-srcs:
	@echo "+ $@"
	git clean -xdf generated

# volatile-generated-srcs are all generated sources that are NOT committed
.PHONY: volatile-generated-srcs
volatile-generated-srcs: proto-generated-srcs go-packr-srcs keys

.PHONY: generated-srcs
generated-srcs: volatile-generated-srcs go-generated-srcs

# clean-generated-srcs cleans ONLY volatile-generated-srcs.
.PHONY: clean-generated-srcs
clean-generated-srcs: clean-packr-srcs clean-proto-generated-srcs
	@echo "+ $@"

deps: go.mod proto-generated-srcs
	@echo "+ $@"
	@$(eval GOMOCK_REFLECT_DIRS=`find . -type d -name 'gomock_reflect_*'`)
	@test -z $(GOMOCK_REFLECT_DIRS) || { echo "Found leftover gomock directories. Please remove them and rerun make deps!"; echo $(GOMOCK_REFLECT_DIRS); exit 1; }
	@go mod tidy
	@$(MAKE) download-deps
ifdef CI
	@git diff --exit-code -- go.mod go.sum || { echo "go.mod/go.sum files were updated after running 'go mod tidy', run this command on your local machine and commit the results." ; exit 1 ; }
	go mod verify
endif
	@touch deps

.PHONY: download-deps
download-deps:
	@echo "+ $@"
	@go mod download

.PHONY: clean-deps
clean-deps:
	@echo "+ $@"
	@rm -f deps

###########
## Build ##
###########

HOST_OS:=linux
ifeq ($(UNAME_S),Darwin)
    HOST_OS:=darwin
endif

.PHONY: build-prep
build-prep: deps volatile-generated-srcs
	mkdir -p bin/{darwin,linux,windows}

cli: build-prep
	RACE=0 CGO_ENABLED=0 GOOS=darwin $(GOBUILD) ./roxctl
	RACE=0 CGO_ENABLED=0 GOOS=linux $(GOBUILD) ./roxctl
ifdef CI
	RACE=0 CGO_ENABLED=0 GOOS=windows $(GOBUILD) ./roxctl
endif
	# Copy the user's specific OS into gopath
	cp bin/$(HOST_OS)/roxctl $(GOPATH)/bin/roxctl
	chmod u+w $(GOPATH)/bin/roxctl

upgrader: bin/$(HOST_OS)/upgrader

bin/%/upgrader: build-prep
	GOOS=$* $(GOBUILD) ./sensor/upgrader

bin/%/admission-control: build-prep
	GOOS=$* $(GOBUILD) ./sensor/admission-control

.PHONY: build-volumes
build-volumes:
	@docker volume inspect $(GOPATH_VOLUME_NAME) >/dev/null 2>&1 || docker volume create $(GOPATH_VOLUME_NAME)
	@docker volume inspect $(GOCACHE_VOLUME_NAME) >/dev/null 2>&1 || docker volume create $(GOCACHE_VOLUME_NAME)

.PHONY: main-builder-image
main-builder-image: build-volumes
	@echo "+ $@"
	scripts/ensure_image.sh $(BUILD_IMAGE) build/Dockerfile build/
	@# Ensure that the go version in the image matches the expected version
	# If the next line fails, you need to update the go version in build/Dockerfile.
	grep -q "$(shell cat EXPECTED_GO_VERSION)" <(docker run --rm "$(BUILD_IMAGE)" go version)

.PHONY: main-builder-image-rhel
main-builder-image-rhel:
	@echo "+ $@"
	scripts/ensure_image.sh $(RHEL_BUILD_IMAGE) build/Dockerfile_rhel build/
	@# Ensure that the go version in the image matches the expected version
	# If the next line fails, you need to update the go version in build/Dockerfile_rhel.
	grep -q "$(shell cat EXPECTED_GO_VERSION)" <(docker run --rm "$(RHEL_BUILD_IMAGE)" go version)


.PHONY: main-build
main-build: build-prep main-build-dockerized
	@echo "+ $@"

.PHONY: main-rhel-build
main-rhel-build: build-prep main-rhel-build-dockerized
	@echo "+ $@"

.PHONY: main-build-dockerized
main-build-dockerized: main-builder-image
	@echo "+ $@"
	docker run --rm -e CI -e CIRCLE_TAG -e GOTAGS $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make main-build-nodeps

.PHONY: sensor-build-dockerized
sensor-build-dockerized: main-builder-image
	@echo "+ $@"
	docker run --rm -e CI -e CIRCLE_TAG -e GOTAGS $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make sensor-build

.PHONY: sensor-kubernetes-build-dockerized
sensor-kubernetes-build-dockerized: main-builder-image
	@echo "+ $@"
	docker run -e CI -e CIRCLE_TAG -e GOTAGS $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(BUILD_IMAGE) make sensor-kubernetes-build

.PHONY: sensor-build
sensor-build:
	$(GOBUILD) sensor/kubernetes sensor/admission-control
	CGO_ENABLED=0 $(GOBUILD) sensor/upgrader

.PHONY: sensor-kubernetes-build
sensor-kubernetes-build:
	$(GOBUILD) sensor/kubernetes

.PHONY: main-rhel-dockerized
main-rhel-build-dockerized: main-builder-image-rhel
	@echo "+ $@"
ifdef CI
	docker container create -e RACE -e CI -e CIRCLE_TAG -e GOTAGS --name builder $(RHEL_BUILD_IMAGE) make main-build-nodeps
	docker cp $(GOPATH) builder:/
	docker start -i builder
	docker cp builder:/go/src/github.com/stackrox/rox/bin/linux bin/
else
	docker run --rm $(GOPATH_WD_OVERRIDES) $(LOCAL_VOLUME_ARGS) $(RHEL_BUILD_IMAGE) make main-build-nodeps
endif

.PHONY: main-build-nodeps
main-build-nodeps:
	$(GOBUILD) central migrator sensor/kubernetes sensor/admission-control compliance/collection
	CGO_ENABLED=0 $(GOBUILD) sensor/upgrader sensor/service-init
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
	$(GOBUILD) integration-tests/mock-grpc-server

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
	@mkdir -p test-output

.PHONY: go-unit-tests
go-unit-tests: build-prep test-prep
	set -o pipefail ; \
	CGO_ENABLED=1 GODEBUG=cgocheck=2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=$(GOTAGS),test scripts/go-test.sh -p 4 -race -cover -coverprofile test-output/coverage.out -v \
		$(shell git ls-files -- '*_test.go' | sed -e 's@^@./@g' | xargs -n 1 dirname | sort | uniq | xargs go list| grep -v '^github.com/stackrox/rox/tests$$') \
		| tee test-output/test.log
	# Exercise the logging package for all supported logging levels to make sure that initialization works properly
	for encoding in console json; do \
		for level in debug info warn error fatal panic; do \
			LOGENCODING=$$encoding LOGLEVEL=$$level CGO_ENABLED=1 GODEBUG=cgocheck=2 MUTEX_WATCHDOG_TIMEOUT_SECS=30 GOTAGS=$(GOTAGS),test scripts/go-test.sh -p 4 -race -v ./pkg/logging/... > /dev/null; \
		done; \
	done

.PHONY: shell-unit-tests
shell-unit-tests:
	@echo "+ $@"
	@mkdir -p shell-test-output
	set -o pipefail ; \
	bats -t $(shell git ls-files -- '*_test.bats') | tee shell-test-output/test.log

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
integration-unit-tests: build-prep
	 GOTAGS=$(GOTAGS),test,integration scripts/go-test.sh -count=1 $(shell go list ./... | grep  "registries\|scanners\|notifiers")

upload-coverage: $(GOVERALLS_BIN)
	$(GOVERALLS_BIN) -coverprofile="test-output/coverage.out" -ignore 'central/graphql/resolvers/generated.go,generated/storage/*,generated/*/*/*' -service=circle-ci -repotoken="$$COVERALLS_REPO_TOKEN"

generate-junit-reports: $(GO_JUNIT_REPORT_BIN)
	$(BASE_DIR)/scripts/generate-junit-reports.sh

###########
## Image ##
###########

# image is an alias for main-image
.PHONY: image
image: main-image

monitoring/static-bin/%: image/static-bin/%
	mkdir -p "$(dir $@)"
	cp -fLp $< $@

.PHONY: monitoring-build-context
monitoring-build-context: monitoring/static-bin/save-dir-contents monitoring/static-bin/restore-all-dir-contents

.PHONY: monitoring-image
monitoring-image: monitoring-build-context
	scripts/ensure_image.sh stackrox/monitoring:$(shell cat MONITORING_VERSION) monitoring/Dockerfile monitoring/

.PHONY: all-builds
all-builds: cli main-build clean-image $(MERGED_API_SWAGGER_SPEC) ui-build

.PHONY: all-rhel-builds
all-rhel-builds: cli main-rhel-build clean-image $(MERGED_API_SWAGGER_SPEC) ui-build

.PHONY: main-image
main-image: all-builds
	make docker-build-main-image

.PHONY: main-image-rhel
main-image-rhel: all-rhel-builds
	make docker-build-main-image-rhel

.PHONY: deployer-image
deployer-image: build-prep
	$(GOBUILD) roxctl
	make docker-build-deployer-image

# The following targets copy compiled artifacts into the expected locations and
# runs the docker build.
# Please DO NOT invoke this target directly unless you know what you're doing;
# you probably want to run `make main-image`. This target is only in Make for convenience;
# it assumes the caller has taken care of the dependencies, and does not
# declare its dependencies explicitly.
.PHONY: docker-build-main-image
docker-build-main-image: copy-binaries-to-image-dir docker-build-data-image
	docker build -t stackrox/main:$(TAG) --build-arg BUILD_IMAGE=$(BUILD_IMAGE) --build-arg DATA_IMAGE_TAG=$(TAG) $(ALPINE_MIRROR_BUILD_ARG) image/
	@echo "Built main image with tag: $(TAG)"
	@echo "You may wish to:       export MAIN_IMAGE_TAG=$(TAG)"

$(CURDIR)/image/rhel/bundle.tar.gz:
	$(CURDIR)/image/rhel/create-bundle.sh $(CURDIR)/image stackrox-data:$(TAG) $(RHEL_BUILD_IMAGE) $(CURDIR)/image/rhel

.PHONY: docker-build-main-image-rhel
docker-build-main-image-rhel: copy-binaries-to-image-dir docker-build-data-image $(CURDIR)/image/rhel/bundle.tar.gz
	docker build -t stackrox/main-rhel:$(TAG) --file image/rhel/Dockerfile --label version=$(TAG) --label release=$(TAG) image/rhel
	@echo "Built main image for RHEL with tag: $(TAG)"
	@echo "You may wish to:       export MAIN_IMAGE_TAG=$(TAG)"

.PHONY: docker-build-data-image
docker-build-data-image:
	test -f $(CURDIR)/image/keys/data-key
	test -f $(CURDIR)/image/keys/data-iv
	docker build -t stackrox-data:$(TAG) \
		--build-arg DOCS_VERSION=$(shell cat DOCS_VERSION) \
		$(ALPINE_MIRROR_BUILD_ARG) \
		image/ \
		--file image/stackrox-data.Dockerfile

.PHONY: docker-build-deployer-image
docker-build-deployer-image:
	cp -f bin/linux/roxctl image/bin/roxctl-linux
	docker build -t stackrox/deployer:$(TAG) \
		--build-arg MAIN_IMAGE_TAG=$(TAG) \
		--build-arg SCANNER_IMAGE_TAG=$(shell cat SCANNER_VERSION) \
		image/ --file image/Dockerfile_gcp

.PHONY: docker-build-roxctl-image
docker-build-roxctl-image:
	cp -f bin/linux/roxctl image/bin/roxctl-linux
	docker build -t stackrox/roxctl:$(TAG) -f image/roxctl.Dockerfile image/


.PHONY: copy-binaries-to-image-dir
copy-binaries-to-image-dir:
	cp -r ui/build image/ui/
	cp bin/linux/central image/bin/central
ifdef CI
	cp bin/linux/roxctl image/bin/roxctl-linux
	cp bin/darwin/roxctl image/bin/roxctl-darwin
	cp bin/windows/roxctl.exe image/bin/roxctl-windows.exe
else
ifneq ($(HOST_OS),linux)
	cp bin/linux/roxctl image/bin/roxctl-linux
endif
	cp bin/$(HOST_OS)/roxctl image/bin/roxctl-$(HOST_OS)
endif
	cp bin/linux/migrator image/bin/migrator
	cp bin/linux/kubernetes        image/bin/kubernetes-sensor
	cp bin/linux/upgrader          image/bin/sensor-upgrader
	cp bin/linux/admission-control image/bin/admission-control
	cp bin/linux/collection        image/bin/compliance
	cp bin/linux/service-init      image/bin/service-init

ifdef CI
	@[ -d image/THIRD_PARTY_NOTICES ] || { echo "image/THIRD_PARTY_NOTICES dir not found! It is required for CI-built images."; exit 1; }
else
	@[ -f image/THIRD_PARTY_NOTICES ] || mkdir -p image/THIRD_PARTY_NOTICES
endif
	@[ -d image/docs ] || { echo "Generated docs not found in image/docs. They are required for build."; exit 1; }

.PHONY: scale-image
scale-image: scale-build clean-image
	cp bin/linux/mocksensor scale/image/bin/mocksensor
	cp bin/linux/mockcollector scale/image/bin/mockcollector
	cp bin/linux/profiler scale/image/bin/profiler
	cp bin/linux/chaos scale/image/bin/chaos
	chmod +w scale/image/bin/*
	docker build -t stackrox/scale:$(TAG) -f scale/image/Dockerfile scale

webhookserver-image: webhookserver-build
	-mkdir webhookserver/bin
	cp bin/linux/webhookserver webhookserver/bin/webhookserver
	chmod +w webhookserver/bin/webhookserver
	docker build -t stackrox/webhookserver:1.2 -f webhookserver/Dockerfile webhookserver

.PHONY: mock-grpc-server-image
mock-grpc-server-image: mock-grpc-server-build clean-image
	cp bin/linux/mock-grpc-server integration-tests/mock-grpc-server/image/bin/mock-grpc-server
	docker build -t stackrox/grpc-server:$(TAG) integration-tests/mock-grpc-server/image

###########
## Clean ##
###########
.PHONY: clean
clean: clean-image clean-generated-srcs
	@echo "+ $@"

.PHONY: clean-image
clean-image:
	@echo "+ $@"
	git clean -xf image/bin
	git clean -xdf image/ui image/docs
	git clean -xf integration-tests/mock-grpc-server/image/bin/mock-grpc-server
	rm -f $(CURDIR)/image/rhel/bundle.tar.gz
	rm -rf $(CURDIR)/image/rhel/scripts

.PHONY: tag
tag:
ifdef COMMIT
	@git describe $(COMMIT) --tags --abbrev=10 --long
else
	@echo $(TAG)
endif


.PHONY: render-helm-yamls
sensorChartDir="image/templates/helm/stackrox-secured-cluster"
collectorVersion=$(shell cat COLLECTOR_VERSION)
render-helm-yamls: proto-generated-srcs
	@rm -rf /tmp/$(TAG)
	@mkdir -p /tmp/$(TAG)
	@go run -tags "$(subst $(comma),$(space),$(GOTAGS))" $(BASE_DIR)/$(sensorChartDir)/main.go "$(TAG)" "$(collectorVersion)" /tmp/$(TAG)
	@cp $(BASE_DIR)/deploy/common/docker-auth.sh  /tmp/$(TAG)/scripts/

.PHONY: ossls-audit
ossls-audit: download-deps
	ossls version
	ossls audit

.PHONY: ossls-notice
ossls-notice: download-deps
	ossls version
	ossls audit --export image/THIRD_PARTY_NOTICES

.PHONY: collector-tag
collector-tag:
	@cat COLLECTOR_VERSION

.PHONY: docs-tag
docs-tag:
	@cat DOCS_VERSION

.PHONY: scanner-tag
scanner-tag:
	@cat SCANNER_VERSION

GET_DEVTOOLS_CMD := $(MAKE) -qp | sed -e '/^\# Not a target:$$/{ N; d; }' | egrep -v '^(\s*(\#.*)?$$|\s|%|\(|\.)' | egrep '^[^[:space:]:]*:' | cut -d: -f1 | sort | uniq | grep '^$(GOBIN)/'
.PHONY: clean-dev-tools
clean-dev-tools:
	@echo "+ $@"
	@$(GET_DEVTOOLS_CMD) | xargs rm -fv

.PHONY: reinstall-dev-tools
reinstall-dev-tools: clean-dev-tools
	@echo "+ $@"
	@$(MAKE) install-dev-tools

.PHONY: install-dev-tools
install-dev-tools:
	@echo "+ $@"
	@$(GET_DEVTOOLS_CMD) | xargs $(MAKE)
ifeq ($(UNAME_S),Darwin)
	@brew install rocksdb
endif

.PHONY: roxvet
roxvet: $(ROXVET_BIN)
	@echo "+ $@"
	@go vet -vettool "$(ROXVET_BIN)" $(shell go list -e ./...)

##########
## Misc ##
##########
.PHONY: clean-offline-bundle
clean-offline-bundle:
	@find scripts/offline-bundle -name '*.img' -delete -o -name '*.tgz' -delete -o -name 'bin' -type d -exec rm -r "{}" \;

.PHONY: offline-bundle
offline-bundle: clean-offline-bundle
	@./scripts/offline-bundle/create.sh

.PHONY: ui-publish-packages
ui-publish-packages:
	make -C ui publish-packages
