MAKEFILE_DIR := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))
YARN_VERSION ?= 1.22.21

.DEFAULT_GOAL := prepare

.PHONY: clean
clean:
	rm --verbose --force "$(MAKEFILE_DIR)"/yarn-*.tgz

.PHONY: prepare
prepare: clean
	corepack prepare yarn@$(YARN_VERSION) --output="$(MAKEFILE_DIR)/yarn-$(YARN_VERSION).tgz"

.PHONY: install
install:
	corepack hydrate --activate "$(MAKEFILE_DIR)/yarn-$(YARN_VERSION).tgz"
	corepack enable yarn
	[[ "$$(yarn --version)" == "$(YARN_VERSION)" ]]
