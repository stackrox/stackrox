BASE_PATH ?= $(CURDIR)
# Set to empty string to echo some command lines which are hidden by default.
SILENT ?= @

# GENERATED_API_XXX and PROTO_API_XXX variables contain standard paths used to
# generate gRPC proto messages, services, and gateways for the API.
PROTO_BASE_PATH = $(CURDIR)/proto
ALL_PROTOS = $(shell find $(PROTO_BASE_PATH) -name '*.proto')
SERVICE_PROTOS = $(filter %_service.proto,$(ALL_PROTOS))

ALL_PROTOS_REL = $(ALL_PROTOS:$(PROTO_BASE_PATH)/%=%)
SERVICE_PROTOS_REL = $(SERVICE_PROTOS:$(PROTO_BASE_PATH)/%=%)

API_SERVICE_PROTOS = $(filter api/v1/%, $(SERVICE_PROTOS_REL))

# Space separated list of v2 service proto files to include in API docs
V2_SERVICES_TO_INCLUDE_IN_DOCS = report_service.proto

V2_SERVICE_PROTOS_REL = $(V2_SERVICES_TO_INCLUDE_IN_DOCS:%=api/v2/%)
API_SERVICE_PROTOS_V2 = $(filter $(V2_SERVICE_PROTOS_REL), $(SERVICE_PROTOS_REL))

STORAGE_PROTOS = $(filter storage/%, $(ALL_PROTOS_REL))

GENERATED_BASE_PATH = $(BASE_PATH)/generated
GENERATED_DOC_PATH = image/rhel/docs
MERGED_API_SWAGGER_SPEC = $(GENERATED_DOC_PATH)/api/v1/swagger.json
MERGED_API_SWAGGER_SPEC_V2 = $(GENERATED_DOC_PATH)/api/v2/swagger.json
GENERATED_API_DOCS = $(GENERATED_DOC_PATH)/api/v1/reference
GENERATED_PB_SRCS = $(ALL_PROTOS_REL:%.proto=$(GENERATED_BASE_PATH)/%.pb.go)
GENERATED_API_GW_SRCS = $(SERVICE_PROTOS_REL:%.proto=$(GENERATED_BASE_PATH)/%.pb.gw.go)
GENERATED_API_SWAGGER_SPECS = $(API_SERVICE_PROTOS:%.proto=$(GENERATED_BASE_PATH)/%.swagger.json)
GENERATED_API_SWAGGER_SPECS_V2 = $(API_SERVICE_PROTOS_V2:%.proto=$(GENERATED_BASE_PATH)/%.swagger.json)

SCANNER_DIR = $(shell go list -f '{{.Dir}}' -m github.com/stackrox/scanner)
ifneq ($(SCANNER_DIR),)
SCANNER_PROTO_BASE_PATH = $(SCANNER_DIR)/proto
ALL_SCANNER_PROTOS = $(shell find $(SCANNER_PROTO_BASE_PATH) -name '*.proto')
ALL_SCANNER_PROTOS_REL = $(ALL_SCANNER_PROTOS:$(SCANNER_PROTO_BASE_PATH)/%=%)
endif

$(call go-tool, PROTOC_GEN_GO_BIN, github.com/gogo/protobuf/)
$(call go-tool, PROTOC_GEN_SWAGGER, github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger, tools/proto)
$(call go-tool, PROTOC_GEN_GRPC_GATEWAY, github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway, tools/proto)


##############
## Protobuf ##
##############
# Set some platform variables for protoc.
# If the proto version is changed, be sure it is also changed in qa-tests-backend/build.gradle.
PROTOC_VERSION := 22.0
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
PROTOC_OS = linux
endif
ifeq ($(UNAME_S),Darwin)
PROTOC_OS = osx
endif
PROTOC_ARCH=$(shell case $$(uname -m) in (arm64) echo aarch_64 ;; (s390x) echo s390_64 ;; (*) uname -m ;; esac)

PROTO_PRIVATE_DIR := $(BASE_PATH)/.proto

PROTOC_DIR := $(PROTO_PRIVATE_DIR)/protoc-$(PROTOC_OS)-$(PROTOC_ARCH)-$(PROTOC_VERSION)

PROTOC := $(PROTOC_DIR)/bin/protoc

PROTOC_DOWNLOADS_DIR := $(PROTO_PRIVATE_DIR)/.downloads

PROTO_GOBIN := $(PROTO_PRIVATE_DIR)/bin

$(PROTOC_DOWNLOADS_DIR):
	@echo "+ $@"
	$(SILENT)mkdir -p "$@"

$(PROTO_GOBIN):
	@echo "+ $@"
	$(SILENT)mkdir -p "$@"

PROTOC_ZIP := protoc-$(PROTOC_VERSION)-$(PROTOC_OS)-$(PROTOC_ARCH).zip
PROTOC_FILE := $(PROTOC_DOWNLOADS_DIR)/$(PROTOC_ZIP)

include $(BASE_PATH)/make/github.mk

$(PROTOC_FILE): $(PROTOC_DOWNLOADS_DIR)
	@$(GET_GITHUB_RELEASE_FN); \
	get_github_release "$@" "https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/$(PROTOC_ZIP)"

.PRECIOUS: $(PROTOC_FILE)

$(PROTOC):
	@echo "+ $@"
	$(SILENT)$(MAKE) "$(PROTOC_FILE)"
	$(SILENT)mkdir -p "$(PROTOC_DIR)"
	$(SILENT)unzip -q -o -d "$(PROTOC_DIR)" "$(PROTOC_FILE)"
	$(SILENT)test -x "$@"


PROTOC_INCLUDES := $(PROTOC_DIR)/include/google

PROTOC_GEN_GO_BIN := $(PROTO_GOBIN)/protoc-gen-gofast

MODFILE_DIR := $(PROTO_PRIVATE_DIR)/modules

$(MODFILE_DIR)/%/UPDATE_CHECK: go.sum
	@echo "+ Checking if $* is up-to-date"
	$(SILENT)mkdir -p $(dir $@)
	$(SILENT)go list -m -json $* | jq '.Dir' >"$@.tmp"
	$(SILENT)(cmp -s "$@.tmp" "$@" && rm "$@.tmp") || mv "$@.tmp" "$@"

GOGO_M_STR := Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types

# The --go_out=M... argument specifies the go package to use for an imported proto file.
# Here, we instruct protoc-gen-go to import the go source for proto file $(BASE_PATH)/<path>/*.proto to
# "github.com/stackrox/rox/generated/<path>".
ROX_M_ARGS = $(foreach proto,$(ALL_PROTOS_REL),M$(proto)=github.com/stackrox/rox/generated/$(patsubst %/,%,$(dir $(proto))))
# Here, we instruct protoc-gen-go to import the go source for proto file github.com/stackrox/scanner/proto/<path>/*.proto to
# "github.com/stackrox/scanner/generated/<path>".
SCANNER_M_ARGS = $(foreach proto,$(ALL_SCANNER_PROTOS_REL),M$(proto)=github.com/stackrox/scanner/generated/$(patsubst %/,%,$(dir $(proto))))
# Combine the *_M_ARGS.
M_ARGS = $(ROX_M_ARGS) $(SCANNER_M_ARGS)
# This is the M_ARGS used for the grpc-gateway invocation. We only map the storage protos, because
# - the gateway code produces no output (possibly because of a bug) if we pass M_ARGS_STR to it.
# - the gateway code doesn't need access to anything outside api/v1 except storage. In particular, it should NOT import internalapi protos.
GATEWAY_M_ARGS = $(foreach proto,$(STORAGE_PROTOS),M$(proto)=github.com/stackrox/rox/generated/$(patsubst %/,%,$(dir $(proto))))

# Hack: there's no straightforward way to escape a comma in a $(subst ...) command, so we have to resort to this little
# trick.
null :=
space := $(null) $(null)
comma := ,

M_ARGS_STR := $(subst $(space),$(comma),$(strip $(M_ARGS)))
GATEWAY_M_ARGS_STR := $(subst $(space),$(comma),$(strip $(GATEWAY_M_ARGS)))


$(PROTOC_INCLUDES): $(PROTOC)

GOGO_DIR = $(shell go list -f '{{.Dir}}' -m github.com/gogo/protobuf)
GRPC_GATEWAY_DIR = $(shell go list -f '{{.Dir}}' -m github.com/grpc-ecosystem/grpc-gateway)

PROTO_DEPS=$(PROTOC) $(PROTOC_INCLUDES)

###############
## Utilities ##
###############

.PHONY: printdocs
printdocs:
	@echo $(GENERATED_API_DOCS)

.PHONY: printswaggers
printswaggers:
	@echo $(GENERATED_API_SWAGGER_SPECS)

.PHONY: printswaggersv2
printswaggersv2:
	@echo $(GENERATED_API_SWAGGER_SPECS_V2)

.PHONY: printsrcs
printsrcs:
	@echo $(GENERATED_SRCS)

.PHONY: printapisrcs
printapisrcs:
	@echo $(GENERATED_PB_SRCS)

.PHONY: printgwsrcs
printgwsrcs:
	@echo $(GENERATED_API_GW_SRCS)

.PHONY: printvalidatorsrcs
printvalidatorsrcs:
	@echo $(GENERATED_API_VALIDATOR_SRCS)

.PHONY: printprotos
printprotos:
	@echo $(PROTO_API_PROTOS)

#######################################################################
## Generate gRPC proto messages, services, and gateways for the API. ##
#######################################################################

$(GENERATED_DOC_PATH):
	@echo "+ $@"
	$(SILENT)mkdir -p $(GENERATED_DOC_PATH)

# Generate all of the proto messages and gRPC services with one invocation of
# protoc when any of the .pb.go sources don't exist or when any of the .proto
# files change.
$(GENERATED_BASE_PATH)/%.pb.go: $(PROTO_BASE_PATH)/%.proto $(PROTO_DEPS) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_GO_BIN) $(ALL_PROTOS)
	@echo "+ $@"
ifeq ($(SCANNER_DIR),)
	$(error Cached directory of scanner dependency not found, run 'go mod tidy')
endif
	$(SILENT)mkdir -p $(dir $@)
	$(SILENT)PATH=$(PROTO_GOBIN) $(PROTOC) \
		-I$(GOGO_DIR) \
		-I$(PROTOC_INCLUDES) \
		-I$(GRPC_GATEWAY_DIR)/third_party/googleapis \
		-I$(SCANNER_PROTO_BASE_PATH) \
		--proto_path=$(PROTO_BASE_PATH) \
		--gofast_out=$(GOGO_M_STR:%=%,)$(M_ARGS_STR:%=%,)plugins=grpc:$(GENERATED_BASE_PATH) \
		$(dir $<)/*.proto

# Generate all of the reverse-proxies (gRPC-Gateways) with one invocation of
# protoc when any of the .pb.gw.go sources don't exist or when any of the
# .proto files change.
$(GENERATED_BASE_PATH)/%_service.pb.gw.go: $(PROTO_BASE_PATH)/%_service.proto $(GENERATED_BASE_PATH)/%_service.pb.go $(ALL_PROTOS)
	@echo "+ $@"
ifeq ($(SCANNER_DIR),)
	$(error Cached directory of scanner dependency not found, run 'go mod tidy')
endif
	$(SILENT)mkdir -p $(dir $@)
	$(SILENT)PATH=$(PROTO_GOBIN) $(PROTOC) \
		-I$(PROTOC_INCLUDES) \
		-I$(GOGO_DIR) \
		-I$(GRPC_GATEWAY_DIR)/third_party/googleapis \
		-I$(SCANNER_PROTO_BASE_PATH) \
		--proto_path=$(PROTO_BASE_PATH) \
		--grpc-gateway_out=$(GATEWAY_M_ARGS_STR:%=%,)allow_colon_final_segments=true,logtostderr=true:$(GENERATED_BASE_PATH) \
		$(dir $<)/*.proto

# Generate all of the swagger specifications with one invocation of protoc
# when any of the .swagger.json sources don't exist or when any of the
# .proto files change.
$(GENERATED_BASE_PATH)/api/v1/%.swagger.json: $(PROTO_BASE_PATH)/api/v1/%.proto $(PROTO_DEPS) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_SWAGGER) $(ALL_PROTOS)
	@echo "+ $@"
ifeq ($(SCANNER_DIR),)
	$(error Cached directory of scanner dependency not found, run 'go mod tidy')
endif
	$(SILENT)PATH=$(PROTO_GOBIN) $(PROTOC) \
		-I$(GOGO_DIR) \
		-I$(PROTOC_INCLUDES) \
		-I$(GRPC_GATEWAY_DIR)/third_party/googleapis \
		-I$(SCANNER_PROTO_BASE_PATH) \
		--proto_path=$(PROTO_BASE_PATH) \
		--swagger_out=logtostderr=true,json_names_for_fields=true:$(GENERATED_BASE_PATH) \
		$(dir $<)/*.proto


$(GENERATED_BASE_PATH)/api/v2/%.swagger.json: $(PROTO_BASE_PATH)/api/v2/%.proto $(PROTO_DEPS) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_SWAGGER) $(ALL_PROTOS)
	@echo "+ $@"
ifeq ($(SCANNER_DIR),)
	$(error Cached directory of scanner dependency not found, run 'go mod tidy')
endif
	$(SILENT)PATH=$(PROTO_GOBIN) $(PROTOC) \
		-I$(GOGO_DIR) \
		-I$(PROTOC_INCLUDES) \
		-I$(GRPC_GATEWAY_DIR)/third_party/googleapis \
		-I$(SCANNER_PROTO_BASE_PATH) \
		--proto_path=$(PROTO_BASE_PATH) \
		--swagger_out=logtostderr=true,json_names_for_fields=true:$(GENERATED_BASE_PATH) \
		$<

# Generate the docs from the merged swagger specs.
$(MERGED_API_SWAGGER_SPEC): $(BASE_PATH)/scripts/mergeswag.sh $(GENERATED_API_SWAGGER_SPECS) $(GENERATED_API_SWAGGER_SPECS_V2)
	@echo "+ $@"
	$(SILENT)mkdir -p "$(dir $@)"
	$(BASE_PATH)/scripts/mergeswag.sh "$(GENERATED_BASE_PATH)/api/v1" "1" >"$@"

$(MERGED_API_SWAGGER_SPEC_V2): $(BASE_PATH)/scripts/mergeswag.sh $(GENERATED_API_SWAGGER_SPECS_V2)
	@echo "+ $@"
	$(SILENT)mkdir -p "$(dir $@)"
	$(BASE_PATH)/scripts/mergeswag.sh "$(GENERATED_BASE_PATH)/api/v2" "2" >"$@"

# Generate the docs from the merged swagger specs.
$(GENERATED_API_DOCS): $(MERGED_API_SWAGGER_SPEC) $(PROTOC_GEN_GRPC_GATEWAY)
	@echo "+ $@"
	docker run $(DOCKER_USER) --rm -v $(CURDIR)/$(GENERATED_DOC_PATH):/tmp/$(GENERATED_DOC_PATH) swaggerapi/swagger-codegen-cli generate -l html2 -i /tmp/$< -o /tmp/$@

# Nukes pretty much everything that goes into building protos.
# You should not have to run this day-to-day, but it occasionally is useful
# to get out of a bad state after a version update.
.PHONY: clean-proto-deps
clean-proto-deps:
	@echo "+ $@"
	rm -rf "$(PROTO_PRIVATE_DIR)"
