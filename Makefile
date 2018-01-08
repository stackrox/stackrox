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
	make -C apollo-ui lint

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
	@go vet $(shell go list -e ./... | grep -v generated | grep -v vendor)


##################
## Dependencies ##
##################
deps: proto-generated Gopkg.toml Gopkg.lock
	@echo "+ $@"
# `dep status` exits with a nonzero code if there is a toml->lock mismatch.
	dep status
	dep ensure
	@touch deps

.PHONY: clean-deps
clean-deps:
	@echo "+ $@"
	@rm -f deps

.PHONY: proto-generated
proto-generated:
	make -C pkg clean-generated-srcs generated-srcs


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
gazelle: deps proto-generated cleanup
	bazel run //:gazelle

.PHONY: build
build: gazelle
	bazel build $(BAZEL_FLAGS) //...

.PHONY: test
test: gazelle
# PURE is so that the test and image stages can share artifacts on Linux.
# action_env args are for running with remote Docker in CircleCI.
	bazel test $(PURE) \
	    --test_output=errors \
	    --action_env=CIRCLECI=$(CIRCLECI) \
	    --action_env=DOCKER_HOST=$(DOCKER_HOST) \
	    --action_env=DOCKER_CERT_PATH=$(DOCKER_CERT_PATH) \
	    --action_env=DOCKER_TLS_VERIFY=$(DOCKER_TLS_VERIFY) \
	    -- \
	    //... -vendor/... -benchmarks/...
# benchmark tests don't work in Bazel yet.
	make -C benchmarks test report
# neither do UI tests
	make -C apollo-ui test

###########
## Image ##
###########
image: gazelle clean-image
	@echo "+ $@"
	bazel build $(BAZEL_FLAGS) \
		//agent/kubernetes \
		//agent/swarm \
		//apollo \
		//benchmarks \
		//benchmark-bootstrap \

	make -C apollo-ui build

# TODO(cg): Replace with native bazel Docker build.
	cp -r apollo-ui/build image/ui/
	cp bazel-bin/agent/swarm/linux_amd64_pure_stripped/swarm image/bin/swarm-agent
	cp bazel-bin/agent/kubernetes/linux_amd64_pure_stripped/kubernetes image/bin/kubernetes-agent
	cp bazel-bin/apollo/linux_amd64_pure_stripped/apollo image/bin/apollo
	cp bazel-bin/benchmarks/linux_amd64_pure_stripped/benchmarks image/bin/benchmarks
	cp bazel-bin/benchmark-bootstrap/linux_amd64_pure_stripped/benchmark-bootstrap image/bin/benchmark-bootstrap
	chmod +w image/bin/*
	docker build -t stackrox/apollo:latest image/

###########
## Clean ##
###########
.PHONY: clean
clean: clean-image
	@echo "+ $@"
	make -C pkg clean

.PHONY: clean-image
clean-image:
	@echo "+ $@"
	git clean -xf image/bin
	git clean -xf image/ui

include make/protogen.mk
