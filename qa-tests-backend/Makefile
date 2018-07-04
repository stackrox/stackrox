.PHONY: all
all: style

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
