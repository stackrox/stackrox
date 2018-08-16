ROX_PROJECT=apollo
TESTFLAGS=-race -p 4
BASE_DIR=$(CURDIR)
TAG=$(shell git describe --tags --abbrev=10 --dirty)

.PHONY: all
all: deps style test image

###########
## Style ##
###########
.PHONY: style
style: fmt imports lint vet blanks crosspkgimports ui-lint qa-tests-style

.PHONY: qa-tests-style
qa-tests-style:
	@echo "+ $@"
	make -C qa-tests-backend/ style

.PHONY: ui-lint
ui-lint:
	@echo "+ $@"
	make -C ui lint

.PHONY: fmt
fmt:
	@echo "+ $@"
ifdef CI
		@echo "The environment indicates we are in CI; checking gofmt."
		@echo 'If this fails, run `make style`.'
		@$(eval FMT=`find . -name vendor -prune -o -name generated -prune -o -name '*.go' -print | xargs gofmt -s -l`)
		@echo "gofmt problems in the following files, if any:"
		@echo $(FMT)
		@test -z "$(FMT)"
endif
	@find . -name vendor -prune -o -name generated -prune -o -name '*.go' -print | xargs gofmt -s -l -w

.PHONY: imports
imports:
	@echo "+ $@"
ifdef CI
		@echo "The environment indicates we are in CI; checking goimports."
		@echo 'If this fails, run `make style`.'
		@$(eval IMPORTS=`find . -name vendor -prune -o -name generated -prune -o -name '*.go' -print | xargs goimports -l`)
		@echo "goimports problems in the following files, if any:"
		@echo $(IMPORTS)
		@test -z "$(IMPORTS)"
endif
	@find . -name vendor -prune -name generated -prune -o -name '*.go' -print | xargs goimports -w

.PHONY: crosspkgimports
crosspkgimports:
	@echo "+ $@"
	@go run $(BASE_DIR)/tools/crosspkgimports/verify.go $(shell go list -e ./... | grep -v generated | grep -v vendor)

.PHONY: lint
lint:
	@echo "+ $@"
	@set -e; for pkg in $(shell go list -e ./... | grep -v generated | grep -v vendor); do golint -set_exit_status $$pkg; done

.PHONY: vet
vet:
	@echo "+ $@"
	@$(BASE_DIR)/tools/go-vet.sh $(shell go list -e ./... | grep -v generated | grep -v vendor)

.PHONY: blanks
blanks:
	@echo "+ $@"
	@find . \( \( -name vendor -o -name generated \) -type d -prune \) -o \( -name \*.go -print0 \) | xargs -0 $(BASE_PATH)/tools/import_validate.py

#####################################
## Generated Code and Dependencies ##
#####################################
API_SERVICES  = alert_service
API_SERVICES += api_token_service
API_SERVICES += auth_service
API_SERVICES += authprovider_service
API_SERVICES += benchmark_results_service
API_SERVICES += benchmark_scan_service
API_SERVICES += benchmark_schedule_service
API_SERVICES += benchmark_service
API_SERVICES += benchmark_trigger_service
API_SERVICES += cluster_service
API_SERVICES += deployment_service
API_SERVICES += detection_service
API_SERVICES += dnr_integration_service
API_SERVICES += enrichment_service
API_SERVICES += image_integration_service
API_SERVICES += image_service
API_SERVICES += metadata_service
API_SERVICES += network_policy_service
API_SERVICES += notifier_service
API_SERVICES += ping_service
API_SERVICES += policy_service
API_SERVICES += role_service
API_SERVICES += search_service
API_SERVICES += secret_service
API_SERVICES += sensor_event_service
API_SERVICES += service_identity_service
API_SERVICES += signal_service
API_SERVICES += summary_service

# These .proto files do not contain gRPC methods and thus don't need gateway files.
PB_COMMON_FILES  = common

GENERATED_SRCS = $(GENERATED_PB_SRCS) $(GENERATED_API_GW_SRCS)

include make/protogen.mk

MOCKERY_BIN := $(GOPATH)/bin/mockery
STRINGER_BIN := $(GOPATH)/bin/stringer

$(MOCKERY_BIN):
	@echo "+ $@"
	@go get github.com/vektra/mockery/.../

$(STRINGER_BIN):
	@echo "+ $@"
	@go get golang.org/x/tools/cmd/stringer

.PHONY: go-generated-srcs
go-generated-srcs: $(MOCKERY_BIN) $(STRINGER_BIN)
	@echo "+ $@"
	go generate ./...

.PHONY: generated-srcs
proto-generated-srcs: $(GENERATED_SRCS)

.PHONY: generated-srcs
generated-srcs: $(GENERATED_SRCS) go-generated-srcs

.PHONY: clean-generated-srcs
clean-generated-srcs:
	@echo "+ $@"
	git clean -xdf generated

deps: $(GENERATED_SRCS) Gopkg.toml Gopkg.lock
	@echo "+ $@"
	@# `dep check` exits with a nonzero code if there is a toml->lock mismatch.
	dep check -skip-vendor
	dep ensure
	@touch deps

.PHONY: clean-deps
clean-deps:
	@echo "+ $@"
	@rm -f deps

###########
## Build ##
###########
PURE := --features=pure
LINUX_AMD64 := --cpu=k8
BAZEL_FLAGS := $(PURE) $(LINUX_AMD64)
cleanup:
	@echo "Total BUILD.bazel files deleted: "
	@find . -mindepth 2 -name BUILD.bazel -print | grep -v "^./image" | xargs rm -v | wc -l

.PHONY: gazelle
gazelle: deps $(GENERATED_SRCS) cleanup
	bazel run //:gazelle

.PHONY: build
build: gazelle
	bazel build $(BAZEL_FLAGS) -- //... -vendor/... -qa-tests-backend/...

.PHONY: gendocs
gendocs: $(GENERATED_API_DOCS)
	@echo "+ $@"

# We don't need to do anything here, because the $(MERGED_API_SWAGGER_SPEC) target already performs validation.
.PHONY: testdocs
testdocs: $(MERGED_API_SWAGGER_SPEC)
	@echo "+ $@"

.PHONY: test
test: gazelle
# PURE is so that the test and image stages can share artifacts on Linux.
# action_env args are for running with remote Docker in CircleCI.
	-rm vendor/github.com/coreos/pkg/BUILD
	-rm vendor/github.com/cloudflare/cfssl/script/BUILD
	-rm vendor/github.com/grpc-ecosystem/grpc-gateway/BUILD
	bazel test \
	    --test_output=errors \
	    --action_env=CIRCLECI=$(CIRCLECI) \
	    --action_env=DOCKER_HOST=$(DOCKER_HOST) \
	    --action_env=DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) \
	    --action_env=DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) \
	    -- \
	    //... -vendor/... -benchmarks/... -tests/... -qa-tests-backend/...
# benchmark tests don't work in Bazel yet.
	make -C benchmarks test report
# neither do UI tests
	make -C ui test

.PHONY: coverage
coverage:
	@echo "+ $@"
	@go test -cover -coverprofile coverage.out $(shell go list -e ./... | grep -v /tests)
	@go tool cover -html=coverage.out -o coverage.html

###########
## Image ##
###########
image: gazelle clean-image
	@echo "+ $@"
	bazel build $(BAZEL_FLAGS) \
		//central \
		//cmd/base64 \
		//cmd/deploy \
		//benchmarks \
		//benchmark-bootstrap \
		//sensor/kubernetes \
		//sensor/swarm \

	make -C ui build

# TODO(cg): Replace with native bazel Docker build.
	cp -r ui/build image/ui/
	cp bazel-bin/cmd/base64/linux_amd64_pure_stripped/base64 image/bin/base64
	cp bazel-bin/central/linux_amd64_pure_stripped/central image/bin/central
	cp bazel-bin/cmd/deploy/linux_amd64_pure_stripped/deploy image/bin/deploy
	cp bazel-bin/benchmarks/linux_amd64_pure_stripped/benchmarks image/bin/benchmarks
	cp bazel-bin/benchmark-bootstrap/linux_amd64_pure_stripped/benchmark-bootstrap image/bin/benchmark-bootstrap
	cp bazel-bin/sensor/swarm/linux_amd64_pure_stripped/swarm image/bin/swarm-sensor
	cp bazel-bin/sensor/kubernetes/linux_amd64_pure_stripped/kubernetes image/bin/kubernetes-sensor
	echo "$(TAG)" > image/VERSION
	chmod +w image/bin/*
	docker build -t stackrox/prevent:$(TAG) image/
	docker build -t stackrox/prevent-health:$(TAG) prometheus/container
	@echo "Built images with tag: $(TAG)"
	@echo "You may wish to:       export PREVENT_IMAGE_TAG=$(TAG)"


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
	git clean -xdf image/ui

.PHONY: tag
tag:
	@echo $(TAG)
