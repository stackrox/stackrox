.PHONY: all
all: style test

###########
## Style ##
###########
.PHONY: style
style:
	@echo "+ $@"
	@gradle --continue codenarcMain codenarcTest ; \
	STATUS=$$? ; \
	if [ $$STATUS -ne 0 ]; then \
	    cat build/reports/codenarc/main.txt build/reports/codenarc/test.txt >&2 2>/dev/null ; \
	fi ; \
	exit $$STATUS

.PHONY: style-fix
style-fix:
	@echo "+ $@"
	@scripts/fix_lint.py .

.PHONY: proto-generated-srcs
proto-generated-srcs:
	@echo "+ $@"
	-rm -r src/main/proto
	@scripts/migrate_protos.sh
	gradle generateProto generateTestProto

.PHONY: clean-generated-srcs
clean-generated-srcs:
	@echo "+ $@"
	git clean -xdf build/generated

.PHONY: test
test: proto-generated-srcs
	@echo "+ $@"
	gradle test

.PHONY: bat-test
bat-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=bat

.PHONY: smoke-test
smoke-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=smoke

.PHONY: enforcement-test
enforcement-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=enforcement

.PHONY: integration-test
integration-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=it

.PHONY: networkpolicy-simulator-test
networkpolicy-simulator-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=networkpolicysimulation

.PHONY: non-bat-test
non-bat-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=-bat
