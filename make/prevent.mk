# Set these variables only if not set by the including Makefile.
IMAGE ?= $(ROX_PROJECT)
PROJECT_SUBDIR ?= $(ROX_PROJECT)
BINARY ?= $(ROX_PROJECT)
BASE_PATH ?= $(CURDIR)/..
GO_BASE_PATH ?= /go/src/github.com/stackrox/rox

.DEFAULT_GOAL = all


###########################
## Developer Environment ##
###########################
.PHONY: dev
dev:
	@echo "+ $@"
	@go get -u github.com/golang/lint/golint
	@go get -u golang.org/x/tools/cmd/goimports
	@go get -u github.com/jstemmer/go-junit-report
	@go get -u github.com/golang/dep/cmd/dep

.PHONY: docs
docs: generated-srcs
	@echo "+ $@"
	@echo
	@echo 'Access your docs at http://localhost:6061/pkg/github.com/stackrox/rox/$(ROX_PROJECT)/'
	@echo 'Hit CTRL-C to quit.'
	@godoc -http=:6061


#######################
## Local Compilation ##
#######################

.PHONY: build
build:
	bazel run //:gazelle
	bazel build --cpu=k8 \
		//$(PROJECT_SUBDIR)


#############
## Testing ##
#############
.PHONY: test
test: pretest test-common posttest

.PHONY: pretest
pretest:

.PHONY: posttest
posttest:

.PHONY: test-common
test-common:
	@echo "+ $@"
	@go test -cover $(TESTFLAGS) -v $(shell go list -e ./... | grep -v generated | grep -v integration-tests | grep -v vendor) 2>&1 | tee test.log

.PHONY: test-integration
test-integration:
	@echo "+ $@"
	@go test -cover -tags integration -v $(shell go list -e ./... | grep -v generated | grep -v integration-tests | grep -v vendor) 2>&1 | tee test.log

.PHONY: test-all
test-all: test-integration

.PHONY: report
report:
	@echo "+ $@"
	@cat test.log | go-junit-report > report.xml
	@echo
	@echo "Test coverage summary:"
	@grep "^coverage: " -A1 test.log | grep -v -e '--' | paste -d " "  - -
	@echo
	@echo "Test pass/fail summary:"
	@grep failures report.xml
	@echo
	@echo "`grep 'FAIL	github.com/stackrox/rox' test.log | wc -l` package(s) detected with compilation or test failures."
	@-grep 'FAIL	github.com/stackrox/rox' test.log || true
	@echo
	@testerror="$$(grep -e 'can.t load package' -e '^# github.com/stackrox/rox/' -e 'FAIL	github.com/stackrox/rox' test.log | wc -l)" && test $$testerror -eq 0
