ROX_PROJECT=pkg
TESTFLAGS=-race -p 4

include ../make/stackrox.mk

.PHONY: all
all: style build-pkg test report

.PHONY: preclean
preclean:
	@echo "+ $@"
	@rm -rf api/generated

SUBDIRS := $(foreach dir, $(wildcard */.), $(if $(shell find $(dir) -name "*.go"),$(dir),))

.PHONY: build-pkg
build-pkg: preclean
	bazel run //:gazelle
	bazel build --cpu=k8 \
		//pkg/...

.PHONY: $(SUBDIRS)
$(SUBDIRS):
	@echo "+ build $@"
	@cd $@ && GOARCH=amd64 go build `glide novendor`
