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

.PHONY: test
test:
	@echo "+ $@"
	gradle test

bat-test:
	@echo "+ $@"
	gradle test -Dgroups=bat

integration-test:
	@echo "+ $@"
	gradle test -Dgroups=it
