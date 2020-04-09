ROX_PROJECT=tests
TESTFLAGS=-race -p 1

.PHONY: all
all: test report

destructive-tests:
	@echo "+ $@"
	@go test -cover -tags destructive -run TestClusterDeletion 2>&1 | tee test.log

include ../make/stackrox.mk
