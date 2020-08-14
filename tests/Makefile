ROX_PROJECT=tests
TESTFLAGS=-race -p 1

.PHONY: all
all: test
	@echo "+ $@"
	@$(MAKE) report

.PHONY: destructive-tests
destructive-tests:
	@echo "+ $@"
	@go test -cover -tags destructive -run TestClusterDeletion 2>&1 | tee test.log
	@$(MAKE) report

.PHONY: external-backup-tests
external-backup-tests:
	@echo "+ $@"
	@go test -cover -tags externalbackups -run TestGCSExternalBackup 2>&1 | tee test.log
	@$(MAKE) report

include ../make/stackrox.mk
