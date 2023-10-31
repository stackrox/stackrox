# Set these variables only if not set by the including Makefile.
IMAGE ?= $(ROX_PROJECT)
PROJECT_SUBDIR ?= $(ROX_PROJECT)
BINARY ?= $(ROX_PROJECT)
# Set to empty string to echo some command lines which are hidden by default.
SILENT ?= @

GO111MODULE := on
export GO111MODULE

.DEFAULT_GOAL = all

###########################
## Developer Environment ##
###########################
.PHONY: docs
docs: generated-srcs
	@echo "+ $@"
	@echo
	@echo 'Access your docs at http://localhost:6061/pkg/github.com/stackrox/rox/$(ROX_PROJECT)/'
	@echo 'Hit CTRL-C to quit.'
	$(SILENT)godoc -http=:6061


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
	$(SILENT)$(TOPLEVEL)/scripts/go-test.sh -cover $(TESTFLAGS) -v $(shell go list -e ./... | grep -v generated | grep -v vendor) 2>&1 | tee test.log

.PHONY: test-integration
test-integration:
	@echo "+ $@"
	$(SILENT)GOTAGS=$(GOTAGS),test,integration $(TOPLEVEL)/scripts/go-test.sh -cover -v $(shell go list -e ./... | grep -v generated | grep -v vendor) 2>&1 | tee test.log

.PHONY: test-all
test-all: test-integration

GO_JUNIT_REPORT_BIN := $(GOBIN)/go-junit-report
$(GO_JUNIT_REPORT_BIN):
	@echo "+ $@"
	$(SILENT)cd $(TOPLEVEL)/tools/test/ && go install github.com/jstemmer/go-junit-report/v2

.PHONY: report
report: $(GO_JUNIT_REPORT_BIN)
	@echo "+ $@"
	@cat test.log | go-junit-report > report.xml
	@mkdir -p $(JUNIT_OUT)
	@cp test.log report.xml $(JUNIT_OUT)
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
