ROX_PROJECT=pkg
TESTFLAGS=-race -p 4

API_SERVICES = v1/image_rule_service
API_SERVICES += v1/benchmark_service
API_SERVICES += v1/scanner_service
API_SERVICES += v1/registry_service
API_SERVICES += v1/alert_service
API_SERVICES += v1/image_service
API_SERVICES += v1/common

GENERATED_SRCS = $(GENERATED_API_SRCS) $(GENERATED_API_GW_SRCS) $(GENERATED_API_VALIDATOR_SRCS)

include ../build/protogen.mk
include ../build/apollo.mk

.PHONY: all
all: style build-pkg test report

.PHONY: preclean
preclean:
	@echo "+ $@"
	@rm -rf serialization/generated
	@rm -rf api/generated

SUBDIRS := $(foreach dir, $(wildcard */.), $(if $(shell find $(dir) -name "*.go"),$(dir),))

.PHONY: build-pkg
build-pkg: preclean $(GENERATED_SRCS) $(SUBDIRS)

.PHONY: $(SUBDIRS)
$(SUBDIRS):
	@echo "+ build $@"
	@cd $@ && GOARCH=amd64 go build `glide novendor`

# Unfortunately this must come after protogen.mk and stackrox.mk are imported.
format-rules: $(GENERATED_SRCS)