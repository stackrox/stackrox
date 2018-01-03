ROX_PROJECT=pkg
TESTFLAGS=-race -p 4

API_SERVICES  = v1/agent_event_service
API_SERVICES += v1/alert_service
API_SERVICES += v1/benchmark_results_service
API_SERVICES += v1/benchmark_schedule_service
API_SERVICES += v1/benchmark_service
API_SERVICES += v1/benchmark_trigger_service
API_SERVICES += v1/cluster_service
API_SERVICES += v1/common
API_SERVICES += v1/configuration_policy
API_SERVICES += v1/deployment_service
API_SERVICES += v1/image_policy
API_SERVICES += v1/image_service
API_SERVICES += v1/notifier_service
API_SERVICES += v1/ping_service
API_SERVICES += v1/policy_service
API_SERVICES += v1/privilege_policy
API_SERVICES += v1/registry_service
API_SERVICES += v1/scanner_service

GENERATED_SRCS = $(GENERATED_API_SRCS) $(GENERATED_API_GW_SRCS)

include ../make/protogen.mk
include ../make/apollo.mk

.PHONY: all
all: style build-pkg test report

.PHONY: preclean
preclean:
	@echo "+ $@"
	@rm -rf api/generated

clean: clean-protos

SUBDIRS := $(foreach dir, $(wildcard */.), $(if $(shell find $(dir) -name "*.go" -not -path "api/*"),$(dir),))

.PHONY: build-pkg
build-pkg: preclean generated-srcs
	bazel run //:gazelle
	bazel build --cpu=k8 \
		//pkg/...

.PHONY: generated-srcs
generated-srcs: $(GENERATED_SRCS)

.PHONY: clean-generated-srcs
clean-generated-srcs: clean-generated

.PHONY: $(SUBDIRS)
$(SUBDIRS):
	@echo "+ build $@"
	@cd $@ && GOARCH=amd64 go build `glide novendor`
