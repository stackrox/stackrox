BASE_PATH ?= $(CURDIR)

# GENERATED_API_XXX and PROTO_API_XXX variables contain standard paths used to
# generate gRPC proto messages, services, and gateways for the API.
GENERATED_BASE_PATH = $(BASE_PATH)/generated
GENERATED_API_PATH = $(GENERATED_BASE_PATH)/api/v1
GENERATED_DOC_PATH = docs
MERGED_API_SWAGGER_SPEC = $(GENERATED_DOC_PATH)/api/v1/swagger.json
GENERATED_API_DOCS = $(GENERATED_DOC_PATH)/api/v1/reference
GENERATED_PB_SRCS = $(API_SERVICES:%=$(GENERATED_API_PATH)/%.pb.go) $(PB_COMMON_FILES:%=$(GENERATED_API_PATH)/%.pb.go)
GENERATED_API_GW_SRCS = $(API_SERVICES:%=$(GENERATED_API_PATH)/%.pb.gw.go)
GENERATED_API_VALIDATOR_SRCS = $(API_SERVICES:%=$(GENERATED_API_PATH)/%.validator.pb.go)
GENERATED_API_SWAGGER_SPECS = $(API_SERVICES:%=$(GENERATED_DOC_PATH)/%.swagger.json)

PROTO_API_PATH = $(BASE_PATH)/api/v1
PROTO_API_PROTOS = $(API_SERVICES:%=$(PROTO_API_PATH)/%.proto) $(PB_COMMON_FILES:%=$(PROTO_API_PATH)/%.proto)

##############
## Protobuf ##
##############
# Set some platform variables for protoc.
PROTOC_VERSION := 3.5.0
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
	@$(BASE_PATH)/scripts/go-get-version.sh github.com/gogo/protobuf/types v1.0.0

$(PROTOC_GEN_GO_BIN): $(GOPATH)/src/github.com/gogo/protobuf/types
	@echo "+ $@"
	@$(BASE_PATH)/scripts/go-get-version.sh github.com/gogo/protobuf/protoc-gen-gofast v1.0.0

GOGO_M_STR := Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/empty.proto=github.com/gogo/protobuf/types


$(GOPATH)/src/github.com/golang/protobuf/protoc-gen-go:
	@echo "+ $@"
# This pins protoc-gen-go to v1.0.0, which is the same version of golang/protobuf that we vendor.
	@$(BASE_PATH)/scripts/go-get-version.sh github.com/golang/protobuf/protoc-gen-go v1.1.0

$(PROTOC_FILE):
	@echo "+ $@"
	@wget -q https://github.com/google/protobuf/releases/download/v$(PROTOC_VERSION)/$(PROTOC_ZIP) -O $(PROTOC_FILE)

$(PROTOC_INCLUDES): $(PROTOC_TMP)
	@echo "+ $@"

$(PROTOC): $(PROTOC_TMP)
	@echo "+ $@"

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
		--proto_path=$(BASE_PATH) \
		$(PROTO_API_PROTOS)

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

PROTOC_GEN_GOVALIDATORS := $(GOPATH)/src/github.com/mwitkow/go-proto-validators

$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway:
	@echo "+ $@"
	@$(BASE_PATH)/scripts/go-get-version.sh google.golang.org/genproto/googleapis 7bb2a897381c9c5ab2aeb8614f758d7766af68ff --skip-install
	@$(BASE_PATH)/scripts/go-get-version.sh github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/... c2b051dd2f71ce445909aab7b28479fd84d00086
	@$(BASE_PATH)/scripts/go-get-version.sh github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/... c2b051dd2f71ce445909aab7b28479fd84d00086

$(GOPATH)/src/github.com/mwitkow/go-proto-validators:
	@echo "+ $@"
	@go get -u github.com/mwitkow/go-proto-validators/protoc-gen-govalidators

$(GENERATED_DOC_PATH):
	@echo "+ $@"
	@mkdir -p $(GENERATED_DOC_PATH)

# Generate all of the proto messages and gRPC services with one invocation of
# protoc when any of the .pb.go sources don't exist or when any of the .proto
# files change.
$(GENERATED_API_PATH)/%.pb.go: $(PROTO_DEPS) $(PROTOC_GEN_GO) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_GOVALIDATORS) $(PROTO_API_PROTOS) $(PROTOC_GEN_GO_BIN)
	@echo "+ $@"
	@mkdir -p $(GENERATED_API_PATH)
	@$(PROTOC) \
		-I$(PROTOC_INCLUDES) \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--proto_path=$(BASE_PATH) \
		--gofast_out=$(GOGO_M_STR),plugins=grpc:$(GENERATED_BASE_PATH) \
		$(PROTO_API_PROTOS)

# Generate all of the reverse-proxies (gRPC-Gateways) with one invocation of
# protoc when any of the .pb.gw.go sources don't exist or when any of the
# .proto files change.
$(GENERATED_API_PATH)/%.pb.gw.go: $(PROTO_DEPS) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_GOVALIDATORS) $(PROTO_API_PROTOS)
	@echo "+ $@"
	@mkdir -p $(GENERATED_API_PATH)
	@$(PROTOC) \
		-I$(PROTOC_INCLUDES) \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--proto_path=$(BASE_PATH) \
		--grpc-gateway_out=logtostderr=true:$(GENERATED_BASE_PATH) \
		$(PROTO_API_PROTOS)

# Generate all of the validator sources with one invocation of protoc
# when any of the .validator.pb.go sources don't exist or when any of the
# .proto files change.
$(GENERATED_API_VALIDATOR_SRCS) : $(PROTO_DEPS) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_GOVALIDATORS) $(GENERATED_DOC_PATH) $(PROTO_API_PROTOS)
	@echo "+ $@"
	@$(PROTOC) \
		-I$(PROTOC_INCLUDES) \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--proto_path=$(BASE_PATH) \
		--govalidators_out=$(GENERATED_BASE_PATH) \
		$(PROTO_API_PROTOS)

# Generate all of the swagger specifications with one invocation of protoc
# when any of the .swagger.json sources don't exist or when any of the
# .proto files change.
$(GENERATED_DOC_PATH)/%.swagger.json: $(PROTO_DEPS) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_GOVALIDATORS) $(GENERATED_DOC_PATH) $(PROTO_API_PROTOS)
	@echo "+ $@"
	@$(PROTOC) \
		-I$(PROTOC_INCLUDES) \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--proto_path=$(BASE_PATH) \
		--swagger_out=logtostderr=true:$(GENERATED_DOC_PATH) \
		$(PROTO_API_PROTOS)

# Generate the docs from the merged swagger specs.
$(MERGED_API_SWAGGER_SPEC): $(BASE_PATH)/scripts/mergeswag.sh $(GENERATED_API_SWAGGER_SPECS)
	@echo "+ $@"
	$(BASE_PATH)/scripts/mergeswag.sh $(GENERATED_DOC_PATH)/api/v1

# Generate the docs from the merged swagger specs.
$(GENERATED_API_DOCS): $(MERGED_API_SWAGGER_SPEC) $(PROTOC_GEN_GRPC_GATEWAY)
	@echo "+ $@"
	docker run --user $(shell id -u) --rm -v $(CURDIR)/docs:/tmp/docs swaggerapi/swagger-codegen-cli generate -l html2 -i /tmp/$< -o /tmp/$@

.PHONY: clean-protos
clean-protos:
	@rm -rf $(GOPATH)/src/github.com/grpc-ecosystem
	@rm -rf $(GOPATH)/src/github.com/golang/protobuf
	@rm -rf $(GOPATH)/src/golang.google.org/genproto/googleapis
	@rm -f $(GOPATH)/bin/protoc-gen-grpc-gateway
	@rm -f $(GOPATH)/bin/protoc-gen-go
	@rm -rf $(PROTOC_TMP)
	@rm -f $(PROTOC_FILE)
	@test -n "$(GENERATED_API_PATH)" && rm -rf "$(GENERATED_API_PATH)" || true

.PHONY: clean-generated
clean-generated:
	@rm -rf "$(GENERATED_API_PATH)"
