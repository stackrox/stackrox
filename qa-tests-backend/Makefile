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
	gradle generateProto generateTestProto

.PHONY: clean-generated-srcs
clean-generated-srcs:
	@echo "+ $@"
	git clean -xdf build/generated

.PHONY: test
test:
	@echo "+ $@"
	gradle test

.PHONY: bat-test
bat-test:
	@echo "+ $@"
	gradle test -Dgroups=bat

.PHONY: enforcement-test
enforcement-test:
	@echo "+ $@"
	gradle test -Dgroups=enforcement

.PHONY: integration-test
integration-test:
	@echo "+ $@"
	gradle test -Dgroups=it

.PHONY: networkpolicy-simulator-test
networkpolicy-simulator-test:
	@echo "+ $@"
	gradle test -Dgroups=networkpolicysimulation

.PHONY: non-bat-test
non-bat-test:
	@echo "+ $@"
	gradle test -Dgroups=-bat
