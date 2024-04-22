ifeq ($(TAG),)
TAG=$(shell git describe --tags --abbrev=10 --dirty --long --exclude '*-nightly-*')
endif

define tag-impl
echo $(TAG)
endef

.PHONY: tag
tag:
	$(call tag-impl)
