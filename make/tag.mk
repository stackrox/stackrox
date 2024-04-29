SUFFIX=

ifeq ($(TAG),)
ifeq ($(KONFLUX_CI),true)
SUFFIX=-fast
endif
TAG=$(shell git describe --tags --abbrev=10 --dirty --long --exclude '*-nightly-*')$(SUFFIX)
endif

.PHONY: tag
tag:
	@echo $(TAG)
