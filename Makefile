ROX_PROJECT=apollo
TESTFLAGS=-race -p 4
BASE_DIR=$(CURDIR)

.PHONY: all
all: deps style test image

###########
## Style ##
###########
.PHONY: style
style: fmt imports lint vet ui-lint

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
		@$(eval FMT=`find . -name vendor -prune -name generated -prune -o -name '*.go' -print | xargs gofmt -s -l`)
		@echo "gofmt problems in the following files, if any:"
		@echo $(FMT)
		@test -z "$(FMT)"
endif
	@find . -name vendor -prune -name generated -prune -o -name '*.go' -print | xargs gofmt -s -l -w

.PHONY: imports
imports:
	@echo "+ $@"
ifdef CI
		@echo "The environment indicates we are in CI; checking goimports."
		@echo 'If this fails, run `make style`.'
		@$(eval IMPORTS=`find . -name vendor -prune -name generated -prune -o -name '*.go' -print | xargs goimports -l`)
		@echo "goimports problems in the following files, if any:"
		@echo $(IMPORTS)
		@test -z "$(IMPORTS)"
endif
	@find . -name vendor -prune -name generated -prune -o -name '*.go' -print | xargs goimports -w

.PHONY: lint
lint:
	@echo "+ $@"
	@set -e; for pkg in $(shell go list -e ./... | grep -v generated | grep -v vendor); do golint -set_exit_status $$pkg; done

.PHONY: vet
vet:
	@echo "+ $@"
	@$(BASE_DIR)/tools/go-vet.sh $(shell go list -e ./... | grep -v generated | grep -v vendor)

#####################################
## Generated Code and Dependencies ##
#####################################
API_SERVICES  = alert_service
API_SERVICES += auth_service
API_SERVICES += authprovider_service
API_SERVICES += benchmark_results_service
API_SERVICES += benchmark_scan_service
API_SERVICES += benchmark_schedule_service
API_SERVICES += benchmark_service
API_SERVICES += benchmark_trigger_service
API_SERVICES += cluster_service
API_SERVICES += deployment_service
API_SERVICES += image_service
API_SERVICES += notifier_service
API_SERVICES += ping_service
API_SERVICES += policy_service
API_SERVICES += registry_service
API_SERVICES += scanner_service
API_SERVICES += search_service
API_SERVICES += sensor_event_service
API_SERVICES += service_identity_service
API_SERVICES += summary_service

# These .proto files do not contain gRPC methods and thus don't need gateway files.
PB_COMMON_FILES  = common
PB_COMMON_FILES += configuration_policy
PB_COMMON_FILES += image_policy
PB_COMMON_FILES += privilege_policy

GENERATED_SRCS = $(GENERATED_PB_SRCS) $(GENERATED_API_GW_SRCS)

include make/protogen.mk

.PHONY: generated-srcs
generated-srcs: $(GENERATED_SRCS)

.PHONY: clean-generated-srcs
clean-generated-srcs:
	@echo "+ $@"
	git clean -xdf generated

deps: $(GENERATED_SRCS) Gopkg.toml Gopkg.lock
	@echo "+ $@"
# `dep status` exits with a nonzero code if there is a toml->lock mismatch.
	dep status
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
	@find . -mindepth 2 -name BUILD.bazel -print | grep -v "^./image" | xargs rm | wc -l | xargs echo

.PHONY: gazelle
gazelle: deps $(GENERATED_SRCS) cleanup
	bazel run //:gazelle

.PHONY: build
build: gazelle
	bazel build $(BAZEL_FLAGS) -- //... -vendor/...

.PHONY: test
test: gazelle
# PURE is so that the test and image stages can share artifacts on Linux.
# action_env args are for running with remote Docker in CircleCI.
	bazel test \
	    --test_output=errors \
	    --action_env=CIRCLECI=$(CIRCLECI) \
	    --action_env=DOCKER_HOST=$(DOCKER_HOST) \
	    --action_env=DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) \
	    --action_env=DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) \
	    -- \
	    //... -vendor/... -benchmarks/... -tests/...
# benchmark tests don't work in Bazel yet.
	make -C benchmarks test report
# neither do UI tests
	make -C ui test

###########
## Image ##
###########
image: gazelle clean-image
	@echo "+ $@"
	bazel build $(BAZEL_FLAGS) \
		//central \
		//cmd/clair \
		//cmd/deploy \
		//benchmarks \
		//benchmark-bootstrap \
		//sensor/kubernetes \
		//sensor/swarm \

	make -C ui build

# TODO(cg): Replace with native bazel Docker build.
	cp -r ui/build image/ui/
	cp bazel-bin/central/linux_amd64_pure_stripped/central image/bin/central
	cp bazel-bin/cmd/clair/linux_amd64_pure_stripped/clair image/bin/clair
	cp bazel-bin/cmd/deploy/linux_amd64_pure_stripped/deploy image/bin/deploy
	cp bazel-bin/benchmarks/linux_amd64_pure_stripped/benchmarks image/bin/benchmarks
	cp bazel-bin/benchmark-bootstrap/linux_amd64_pure_stripped/benchmark-bootstrap image/bin/benchmark-bootstrap
	cp bazel-bin/sensor/swarm/linux_amd64_pure_stripped/swarm image/bin/swarm-sensor
	cp bazel-bin/sensor/kubernetes/linux_amd64_pure_stripped/kubernetes image/bin/kubernetes-sensor
	chmod +w image/bin/*
	docker build -t stackrox/prevent:latest image/

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
