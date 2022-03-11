BASE_PATH ?= $(CURDIR)

# GENERATED_API_XXX and PROTO_API_XXX variables contain standard paths used to
# generate gRPC proto messages, services, and gateways for the API.
PROTO_BASE_PATH = $(CURDIR)/proto
ALL_PROTOS = $(shell find $(PROTO_BASE_PATH) -name '*.proto')
SERVICE_PROTOS = $(filter %_service.proto,$(ALL_PROTOS))

ALL_PROTOS_REL = $(ALL_PROTOS:$(PROTO_BASE_PATH)/%=%)
SERVICE_PROTOS_REL = $(SERVICE_PROTOS:$(PROTO_BASE_PATH)/%=%)

API_SERVICE_PROTOS = $(filter api/v1/%, $(SERVICE_PROTOS_REL))
STORAGE_PROTOS = $(filter storage/%, $(ALL_PROTOS_REL))

GENERATED_BASE_PATH = $(BASE_PATH)/generated
GENERATED_DOC_PATH = image/docs
MERGED_API_SWAGGER_SPEC = $(GENERATED_DOC_PATH)/api/v1/swagger.json
GENERATED_API_DOCS = $(GENERATED_DOC_PATH)/api/v1/reference
GENERATED_PB_SRCS = $(ALL_PROTOS_REL:%.proto=$(GENERATED_BASE_PATH)/%.pb.go)
GENERATED_API_GW_SRCS = $(SERVICE_PROTOS_REL:%.proto=$(GENERATED_BASE_PATH)/%.pb.gw.go)
GENERATED_API_SWAGGER_SPECS = $(API_SERVICE_PROTOS:%.proto=$(GENERATED_BASE_PATH)/%.swagger.json)

SCANNER_DIR = $(shell go list -f '{{.Dir}}' -m github.com/stackrox/scanner)
ifneq ($(SCANNER_DIR),)
SCANNER_PROTO_BASE_PATH = $(SCANNER_DIR)/proto
ALL_SCANNER_PROTOS = $(shell find $(SCANNER_PROTO_BASE_PATH) -name '*.proto')
ALL_SCANNER_PROTOS_REL = $(ALL_SCANNER_PROTOS:$(SCANNER_PROTO_BASE_PATH)/%=%)
endif

##############
## Protobuf ##
##############
# Set some platform variables for protoc.
# If the proto version is changed, be sure it is also changed in qa-tests-backend/build.gradle.
PROTOC_VERSION := 3.19.4
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
PROTOC_ARCH = linux
endif
ifeq ($(UNAME_S),Darwin)
PROTOC_ARCH = osx
endif

PROTO_PRIVATE_DIR := $(BASE_PATH)/.proto

PROTOC_DIR := $(PROTO_PRIVATE_DIR)/protoc-$(PROTOC_ARCH)-$(PROTOC_VERSION)

PROTOC := $(PROTOC_DIR)/bin/protoc

PROTOC_DOWNLOADS_DIR := $(PROTO_PRIVATE_DIR)/.downloads

PROTO_GOBIN := $(PROTO_PRIVATE_DIR)/bin

$(PROTOC_DOWNLOADS_DIR):
	@echo "+ $@"
	@mkdir -p "$@"

$(PROTO_GOBIN):
	@echo "+ $@"
	@mkdir -p "$@"

PROTOC_ZIP := protoc-$(PROTOC_VERSION)-$(PROTOC_ARCH)-x86_64.zip
PROTOC_FILE := $(PROTOC_DOWNLOADS_DIR)/$(PROTOC_ZIP)

$(PROTOC_FILE): $(PROTOC_DOWNLOADS_DIR)
	@echo "+ $@"
	@wget -q "https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VERSION)/$(PROTOC_ZIP)" -O "$@"

.PRECIOUS: $(PROTOC_FILE)

$(PROTOC):
	@echo "+ $@"
	@$(MAKE) "$(PROTOC_FILE)"
	@mkdir -p "$(PROTOC_DIR)"
	@unzip -q -o -d "$(PROTOC_DIR)" "$(PROTOC_FILE)"
	@test -x "$@"


PROTOC_INCLUDES := $(PROTOC_DIR)/include/google

PROTOC_GEN_GO_BIN := $(PROTO_GOBIN)/protoc-gen-gofast

MODFILE_DIR := $(PROTO_PRIVATE_DIR)/modules

$(MODFILE_DIR)/%/UPDATE_CHECK: go.sum
	@echo "+ Checking if $* is up-to-date"
	@mkdir -p $(dir $@)
	@go list -m -json $* | jq '.Dir' >"$@.tmp"
	@(cmp -s "$@.tmp" "$@" && rm "$@.tmp") || mv "$@.tmp" "$@"

$(PROTOC_GEN_GO_BIN): $(MODFILE_DIR)/github.com/gogo/protobuf/UPDATE_CHECK $(PROTO_GOBIN)
	@echo "+ $@"
	@GOBIN=$(PROTO_GOBIN) go install github.com/gogo/protobuf/$(notdir $@)

PROTOC_GEN_LINT := $(PROTO_GOBIN)/protoc-gen-lint
$(PROTOC_GEN_LINT): $(MODFILE_DIR)/github.com/ckaznocha/protoc-gen-lint/UPDATE_CHECK $(PROTO_GOBIN)
	@echo "+ $@"
	@GOBIN=$(PROTO_GOBIN) go install github.com/ckaznocha/protoc-gen-lint

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

.PHONY: proto-fmt
proto-fmt: $(PROTOC_GEN_LINT)
	@echo "Checking for proto style errors"
	@PATH=$(PROTO_GOBIN) $(PROTOC) \
		-I$(PROTOC_INCLUDES) \
		-I$(GOGO_DIR)/protobuf \
		-I$(GRPC_GATEWAY_DIR)/third_party/googleapis \
		-I$(SCANNER_PROTO_BASE_PATH) \
		--lint_out=. \
		--proto_path=$(PROTO_BASE_PATH) \
		$(ALL_PROTOS)

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

PROTOC_GEN_GRPC_GATEWAY := $(PROTO_GOBIN)/protoc-gen-grpc-gateway

$(PROTOC_GEN_GRPC_GATEWAY): $(MODFILE_DIR)/github.com/grpc-ecosystem/grpc-gateway/UPDATE_CHECK $(PROTO_GOBIN)
	@echo "+ $@"
	@GOBIN=$(PROTO_GOBIN) go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway

PROTOC_GEN_SWAGGER := $(PROTO_GOBIN)/protoc-gen-swagger

$(PROTOC_GEN_SWAGGER): $(MODFILE_DIR)/github.com/grpc-ecosystem/grpc-gateway/UPDATE_CHECK $(PROTO_GOBIN)
	@echo "+ $@"
	@GOBIN=$(PROTO_GOBIN) go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

$(GENERATED_DOC_PATH):
	@echo "+ $@"
	@mkdir -p $(GENERATED_DOC_PATH)

# Generate all of the proto messages and gRPC services with one invocation of
# protoc when any of the .pb.go sources don't exist or when any of the .proto
# files change.
$(GENERATED_BASE_PATH)/%.pb.go: $(PROTO_BASE_PATH)/%.proto $(PROTO_DEPS) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_GO_BIN) $(ALL_PROTOS)
	@echo "+ $@"
	@mkdir -p $(dir $@)
	@PATH=$(PROTO_GOBIN) $(PROTOC) \
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
	@mkdir -p $(dir $@)
	@PATH=$(PROTO_GOBIN) $(PROTOC) \
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
$(GENERATED_BASE_PATH)/%.swagger.json: $(PROTO_BASE_PATH)/%.proto $(PROTO_DEPS) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_SWAGGER) $(ALL_PROTOS)
	@echo "+ $@"
	@PATH=$(PROTO_GOBIN) $(PROTOC) \
		-I$(GOGO_DIR) \
		-I$(PROTOC_INCLUDES) \
		-I$(GRPC_GATEWAY_DIR)/third_party/googleapis \
		-I$(SCANNER_PROTO_BASE_PATH) \
		--proto_path=$(PROTO_BASE_PATH) \
		--swagger_out=logtostderr=true,json_names_for_fields=true:$(GENERATED_BASE_PATH) \
		$(dir $<)/*.proto

# Generate the docs from the merged swagger specs.
$(MERGED_API_SWAGGER_SPEC): $(BASE_PATH)/scripts/mergeswag.sh $(GENERATED_API_SWAGGER_SPECS)
	@echo "+ $@"
	@mkdir -p "$(dir $@)"
	$(BASE_PATH)/scripts/mergeswag.sh "$(GENERATED_BASE_PATH)/api/v1" >"$@"

# Generate the docs from the merged swagger specs.
$(GENERATED_API_DOCS): $(MERGED_API_SWAGGER_SPEC) $(PROTOC_GEN_GRPC_GATEWAY)
	@echo "+ $@"
	docker run --user $(shell id -u) --rm -v $(CURDIR)/$(GENERATED_DOC_PATH):/tmp/$(GENERATED_DOC_PATH) swaggerapi/swagger-codegen-cli generate -l html2 -i /tmp/$< -o /tmp/$@

# Nukes pretty much everything that goes into building protos.
# You should not have to run this day-to-day, but it occasionally is useful
# to get out of a bad state after a version update.
.PHONY: clean-proto-deps
clean-proto-deps:
	@echo "+ $@"
	rm -f $(PROTOC_FILE)
	rm -rf $(PROTOC_DIR)
	rm -f $(PROTO_GOBIN)
