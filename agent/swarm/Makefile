ROX_PROJECT=agent
PROJECT_SUBDIR=agent/swarm
BASE_PATH=$(CURDIR)/../..
TESTFLAGS=-race -p 4

.PHONY: all
all: style build test report

.PHONY: preclean
preclean:
	@echo "+ $@"
	@rm -rf container/bin

include ../../build/apollo.mk
