.PHONY: all
all: style test

###########
## Style ##
###########
.PHONY: style
style:
	@echo "+ $@"
	gradle --continue codenarcMain codenarcTest

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
	git clean -xdf src/main/java

.PHONY: test
test:
	@echo "+ $@"
	gradle test

.PHONY: bat-test
bat-test:
	@echo "+ $@"
	gradle test -Dgroups=bat

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
