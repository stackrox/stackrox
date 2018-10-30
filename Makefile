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
imports: deps proto-generated-srcs
	@echo "+ $@"
ifdef CI
		@echo "The environment indicates we are in CI; checking goimports."
		@echo 'If this fails, run `make style`.'
		@$(eval IMPORTS=`find . -name vendor -prune -o -name generated -prune -o -name mocks -prune -o -name '*.go' -print | xargs goimports -l`)
		@echo "goimports problems in the following files, if any:"
		@echo $(IMPORTS)
		@test -z "$(IMPORTS)"
endif
	@find . -name vendor -prune -name generated -prune -o -name mocks -prune -o -name '*.go' -print | xargs goimports -w

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

.PHONY: dev
dev:
	@echo "+ $@"
	@go get -u golang.org/x/lint/golint
	@go get -u golang.org/x/tools/cmd/goimports
	@go get -u github.com/jstemmer/go-junit-report
	@go get -u github.com/golang/dep/cmd/dep


#####################################
## Generated Code and Dependencies ##
#####################################

GENERATED_SRCS = $(GENERATED_PB_SRCS) $(GENERATED_API_GW_SRCS)

include make/protogen.mk

MOCKERY_BIN := $(GOPATH)/bin/mockery
$(MOCKERY_BIN):
	@echo "+ $@"
	@go get github.com/vektra/mockery/...

STRINGER_BIN := $(GOPATH)/bin/stringer
$(STRINGER_BIN):
	@echo "+ $@"
	@$(BASE_PATH)/scripts/go-get-version.sh golang.org/x/tools/cmd/stringer a2dc47679d30b6c496245bafc6a166b46c5fe318

MOCKGEN_BIN := $(GOPATH)/bin/mockgen
$(MOCKGEN_BIN):
	@echo "+ $@"
	@$(BASE_PATH)/scripts/go-get-version.sh github.com/golang/mock/mockgen 8a44ef6e8be577e050008c7886f24fc705d709fb

GENNY_BIN := $(GOPATH)/bin/genny
$(GENNY_BIN):
	@echo "+ $@"
	@go get github.com/mauricelam/genny

.PHONY: go-generated-srcs
go-generated-srcs: $(MOCKERY_BIN) $(MOCKGEN_BIN) $(STRINGER_BIN) $(GENNY_BIN)
	@echo "+ $@"
	go generate ./...

.PHONY: proto-generated-srcs
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
	@# `dep ensure` can be flaky sometimes, so try rerunning it if it fails.
	dep ensure || (rm -rf .vendor-new && dep ensure)
	@touch deps

.PHONY: clean-deps
clean-deps:
	@echo "+ $@"
	@rm -f deps

###########
## Build ##
###########
PURE := --features=pure
RACE := --features=race
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
	bazel build $(BAZEL_FLAGS) -- //... -proto/... -qa-tests-backend/... -vendor/...

.PHONY: gendocs
gendocs: $(GENERATED_API_DOCS)
	@echo "+ $@"

# We don't need to do anything here, because the $(MERGED_API_SWAGGER_SPEC) target already performs validation.
.PHONY: testdocs
testdocs: $(MERGED_API_SWAGGER_SPEC)
	@echo "+ $@"

.PHONY: bazel-test
bazel-test: gazelle
	-rm vendor/github.com/coreos/pkg/BUILD
	-rm vendor/github.com/cloudflare/cfssl/script/BUILD
	-rm vendor/github.com/grpc-ecosystem/grpc-gateway/BUILD
	@# Be careful if you add action_env arguments; their values can invalidate cached
	@# test results. See https://github.com/bazelbuild/bazel/issues/2574#issuecomment-320006871.
	bazel coverage $(PURE) $(RACE) \
	    --test_output=errors \
	    -- \
	    //... -benchmarks/... -proto/... -qa-tests-backend/... -tests/... -vendor/...

.PHONY: benchmarks-test
benchmarks-test:
	@# Benchmark tests don't work in Bazel yet.
	make -C benchmarks test report

.PHONY: ui-test
ui-test:
	@# UI tests don't work in Bazel yet.
	make -C ui test

.PHONY: test
test: bazel-test benchmarks-test ui-test

upload-coverage:
	@# 'mode: set' is repeated in each coverage file, but Coveralls only wants it
	@# exactly once at the head of the file.
	@# We might be able to use Coveralls parallel builds to resolve this:
	@#     https://docs.coveralls.io/parallel-build-webhook

	@echo 'mode: set' > combined_coverage.dat
	@find ./bazel-testlogs/ -name 'coverage.dat' | xargs -I {} cat "{}" | grep -v 'mode: set' >> combined_coverage.dat
	goveralls -coverprofile="combined_coverage.dat" -service=circle-ci -repotoken="$$COVERALLS_REPO_TOKEN"

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
		//cmd/roxdetect \
		//cmd/deploy \
		//benchmarks \
		//benchmark-bootstrap \
		//sensor/kubernetes \
		//sensor/swarm \
		//integration-tests/mock-grpc-server \
		//scale/mocksensor

	make -C ui build

# TODO(cg): Replace with native bazel Docker build.
	cp -r ui/build image/ui/
	cp bazel-bin/central/linux_amd64_pure_stripped/central image/bin/central
	cp bazel-bin/cmd/deploy/linux_amd64_pure_stripped/deploy image/bin/deploy
	cp bazel-bin/benchmarks/linux_amd64_pure_stripped/benchmarks image/bin/benchmarks
	cp bazel-bin/benchmark-bootstrap/linux_amd64_pure_stripped/benchmark-bootstrap image/bin/benchmark-bootstrap
	cp bazel-bin/sensor/swarm/linux_amd64_pure_stripped/swarm image/bin/swarm-sensor
	cp bazel-bin/sensor/kubernetes/linux_amd64_pure_stripped/kubernetes image/bin/kubernetes-sensor
	cp bazel-bin/integration-tests/mock-grpc-server/linux_amd64_pure_stripped/mock-grpc-server integration-tests/mock-grpc-server/image/bin/mock-grpc-server
	cp bazel-bin/scale/mocksensor/linux_amd64_pure_stripped/mocksensor scale/image/bin/mocksensor
	echo "$(TAG)" > image/VERSION
	chmod +w image/bin/*
	chmod +w scale/image/bin/*
	docker build -t stackrox/main:$(TAG) image/
	docker build -t stackrox/monitoring:$(TAG) monitoring
	docker build -t stackrox/grpc-server:$(TAG) integration-tests/mock-grpc-server/image
	docker build -t stackrox/scale:$(TAG) scale/image
	@echo "Built images with tag: $(TAG)"
	@echo "You may wish to:       export MAIN_IMAGE_TAG=$(TAG)"


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
	git clean -xf integration-tests/mock-grpc-server/image/bin/mock-grpc-server

.PHONY: tag
tag:
	@echo $(TAG)
