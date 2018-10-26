ROX_PROJECT=sensor
PROJECT_SUBDIR=sensor/kubernetes
BASE_PATH=$(CURDIR)/../..
TESTFLAGS=-race -p 4

.PHONY: all
all: style build test report

include ../../make/stackrox.mk
