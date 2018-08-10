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

.PHONY: test
test:
	@echo "+ $@"
	gradle test
