ROX_PROJECT=docker-bench
TESTFLAGS=-race -p 1

GENERATED_SRCS = $(GENERATED_API_SRCS)

.PHONY: all
all: style build test report image

.PHONY: noimage
noimage: style build test report

.PHONY: preclean
preclean:
	@echo "+ $@"
	@rm -rf container/bin

include ../build/protogen.mk
include ../build/apollo.mk
