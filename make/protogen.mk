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
GENERATED_API_SWAGGER_SPECS = $(API_SERVICE_PROTOS:%.proto=$(GENERATED_DOC_PATH)/%.swagger.json)

##############
## Protobuf ##
##############
# Set some platform variables for protoc.
PROTOC_VERSION := 3.6.1
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
PROTOC_ARCH = linux
endif
ifeq ($(UNAME_S),Darwin)
PROTOC_ARCH = osx
endif

PROTOC_ZIP := protoc-$(PROTOC_VERSION)-$(PROTOC_ARCH)-x86_64.zip
PROTOC_FILE := $(BASE_PATH)/$(PROTOC_ZIP)

PROTOC_TMP := $(BASE_PATH)/protoc-tmp

PROTOC := $(PROTOC_TMP)/bin/protoc

PROTOC_INCLUDES := $(PROTOC_TMP)/include/google

PROTOC_GEN_GO := $(GOPATH)/src/github.com/golang/protobuf/protoc-gen-go

PROTOC_GEN_GO_BIN := $(GOPATH)/bin/protoc-gen-gofast

$(GOPATH)/src/github.com/gogo/protobuf/types:
	@echo "+ $@"
	@$(BASE_PATH)/scripts/go-get-version.sh github.com/gogo/protobuf/types v1.2.0

$(PROTOC_GEN_GO_BIN): $(GOPATH)/src/github.com/gogo/protobuf/types
	@echo "+ $@"
	@$(BASE_PATH)/scripts/go-get-version.sh github.com/gogo/protobuf/protoc-gen-gofast v1.2.0

GOGO_M_STR := Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types

# The --go_out=M... argument specifies the go package to use for an imported proto file. Here, we instruct protoc-gen-go
# to import the go source for proto file $(BASE_PATH)/<path>/*.proto to
# "github.com/stackrox/rox/generated/<path>".
M_ARGS = $(foreach proto,$(ALL_PROTOS_REL),M$(proto)=github.com/stackrox/rox/generated/$(patsubst %/,%,$(dir $(proto))))
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

$(GOPATH)/src/github.com/golang/protobuf/protoc-gen-go:
	@echo "+ $@"
# This pins protoc-gen-go to v1.0.0, which is the same version of golang/protobuf that we vendor.
	@$(BASE_PATH)/scripts/go-get-version.sh github.com/golang/protobuf/protoc-gen-go v1.1.0

$(PROTOC_FILE):
	@echo "+ $@"
	@wget -q https://github.com/google/protobuf/releases/download/v$(PROTOC_VERSION)/$(PROTOC_ZIP) -O $(PROTOC_FILE)

$(PROTOC_INCLUDES): $(PROTOC_TMP)
	@echo "+ $@"
	@touch $@

$(PROTOC): $(PROTOC_TMP)
	@echo "+ $@"
	@touch $@

$(PROTOC_TMP): $(PROTOC_FILE)
	@echo "+ $@"
	@mkdir $(PROTOC_TMP)
	@unzip -q -d $(PROTOC_TMP) $(PROTOC_FILE)

.PHONY: proto-fmt
proto-fmt:
	@go get github.com/ckaznocha/protoc-gen-lint
	@echo "Checking for proto style errors"
	@$(PROTOC) \
		-I$(PROTOC_INCLUDES) \
		-I$(GOPATH)/src \
		-I$(GOPATH)/src/github.com/gogo/protobuf/protobuf \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--lint_out=. \
		--proto_path=$(PROTO_BASE_PATH) \
		$(ALL_PROTOS)

PROTO_DEPS=$(PROTOC_GEN_GO) $(PROTOC) $(PROTOC_INCLUDES)

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

PROTOC_GEN_GRPC_GATEWAY := $(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway

$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway:
	@echo "+ $@"
	@$(BASE_PATH)/scripts/go-get-version.sh google.golang.org/genproto/googleapis 7bb2a897381c9c5ab2aeb8614f758d7766af68ff --skip-install
	@$(BASE_PATH)/scripts/go-get-version.sh github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/... c2b051dd2f71ce445909aab7b28479fd84d00086
	@$(BASE_PATH)/scripts/go-get-version.sh github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/... c2b051dd2f71ce445909aab7b28479fd84d00086

$(GENERATED_DOC_PATH):
	@echo "+ $@"
	@mkdir -p $(GENERATED_DOC_PATH)

# Generate all of the proto messages and gRPC services with one invocation of
# protoc when any of the .pb.go sources don't exist or when any of the .proto
# files change.
$(GENERATED_BASE_PATH)/%.pb.go: $(PROTO_BASE_PATH)/%.proto $(PROTO_DEPS) $(PROTOC_GEN_GO) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_GO_BIN) $(ALL_PROTOS)
	@echo "+ $@"
	@mkdir -p $(dir $@)
	@$(PROTOC) \
		-I=$(GOPATH)/src/github.com/gogo \
		-I$(PROTOC_INCLUDES) \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--proto_path=$(PROTO_BASE_PATH) \
		--gofast_out=$(GOGO_M_STR),$(M_ARGS_STR),plugins=grpc:$(GENERATED_BASE_PATH) \
		$(dir $<)/*.proto

# Generate all of the reverse-proxies (gRPC-Gateways) with one invocation of
# protoc when any of the .pb.gw.go sources don't exist or when any of the
# .proto files change.
$(GENERATED_BASE_PATH)/%_service.pb.gw.go: $(PROTO_BASE_PATH)/%_service.proto $(GENERATED_BASE_PATH)/%_service.pb.go $(ALL_PROTOS)
	@echo "+ $@"
	@mkdir -p $(dir $@)
	@$(PROTOC) \
		-I$(PROTOC_INCLUDES) \
		-I=$(GOPATH)/src/github.com/gogo \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--proto_path=$(PROTO_BASE_PATH) \
		--grpc-gateway_out=$(GATEWAY_M_ARGS_STR),logtostderr=true:$(GENERATED_BASE_PATH) \
		$(dir $<)/*.proto

# Generate all of the swagger specifications with one invocation of protoc
# when any of the .swagger.json sources don't exist or when any of the
# .proto files change.
$(GENERATED_DOC_PATH)/%.swagger.json: $(PROTO_BASE_PATH)/%.proto $(PROTO_DEPS) $(PROTOC_GEN_GRPC_GATEWAY) $(GENERATED_DOC_PATH) $(ALL_PROTOS)
	@echo "+ $@"
	@$(PROTOC) \
		-I=$(GOPATH)/src/github.com/gogo \
		-I$(PROTOC_INCLUDES) \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--proto_path=$(PROTO_BASE_PATH) \
		--swagger_out=logtostderr=true:$(GENERATED_DOC_PATH) \
		$(dir $<)/*.proto

# Generate the docs from the merged swagger specs.
$(MERGED_API_SWAGGER_SPEC): $(BASE_PATH)/scripts/mergeswag.sh $(GENERATED_API_SWAGGER_SPECS)
	@echo "+ $@"
	$(BASE_PATH)/scripts/mergeswag.sh $(GENERATED_DOC_PATH)/api/v1

# Generate the docs from the merged swagger specs.
$(GENERATED_API_DOCS): $(MERGED_API_SWAGGER_SPEC) $(PROTOC_GEN_GRPC_GATEWAY)
	@echo "+ $@"
	docker run --user $(shell id -u) --rm -v $(CURDIR)/$(GENERATED_DOC_PATH):/tmp/$(GENERATED_DOC_PATH) swaggerapi/swagger-codegen-cli generate -l html2 -i /tmp/$< -o /tmp/$@

.PHONY: clean-protos
clean-protos: clean-generated
	@rm -rf $(GOPATH)/src/github.com/gogo
	@rm -rf $(GOPATH)/src/github.com/grpc-ecosystem
	@rm -rf $(GOPATH)/src/github.com/golang/protobuf
	@rm -rf $(GOPATH)/src/golang.google.org/genproto/googleapis
	@rm -f $(GOPATH)/bin/protoc-gen-grpc-gateway
	@rm -f $(GOPATH)/bin/protoc-gen-go
	@rm -rf $(PROTOC_TMP)
	@rm -f $(PROTOC_FILE)

.PHONY: clean-generated
clean-generated:
	@find $(GENERATED_BASE_PATH) \( -name '*.pb.go' -o -name '*.pb.*.go' \) -exec rm {} \;
	@find $(GENERATED_BASE_PATH) -type d -empty -delete
