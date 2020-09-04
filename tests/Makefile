ROX_PROJECT=tests
TESTFLAGS=-race -p 1

TOPLEVEL=$(CURDIR)/..

.PHONY: all
all: test
	@echo "+ $@"
	@$(MAKE) report

.PHONY: destructive-tests
destructive-tests:
	@echo "+ $@"
	@GOTAGS=$(GOTAGS),test,destructive ../scripts/go-test.sh -cover -run TestClusterDeletion 2>&1 | tee test.log
	@$(MAKE) report

.PHONY: external-backup-tests
external-backup-tests:
	@echo "+ $@"
	@GOTAGS=$(GOTAGS),test,externalbackups ../scripts/go-test.sh -cover -run TestGCSExternalBackup 2>&1 | tee test.log
	@$(MAKE) report

include ../make/stackrox.mk
