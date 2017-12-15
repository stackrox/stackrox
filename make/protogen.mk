BASE_PATH ?= $(CURDIR)/..

# GENERATED_API_XXX and PROTO_API_XXX variables contain standard paths used to
# generate gRPC proto messages, services, and gateways for the API.
GENERATED_API_PATH = api/generated/api
GENERATED_DOC_PATH = docs/generated/api
MERGED_API_SWAGGER_SPEC = $(GENERATED_DOC_PATH)/v1/swagger.json
GENERATED_API_DOCS = $(GENERATED_DOC_PATH)/v1/reference
GENERATED_API_SRCS = $(API_SERVICES:%=$(GENERATED_API_PATH)/%.pb.go)
GENERATED_API_GW_SRCS = $(API_SERVICES:%=$(GENERATED_API_PATH)/%.pb.gw.go)
GENERATED_API_VALIDATOR_SRCS = $(API_SERVICES:%=$(GENERATED_API_PATH)/%.validator.pb.go)
GENERATED_API_SWAGGER_SPECS = $(API_SERVICES:%=$(GENERATED_DOC_PATH)/%.swagger.json)
PROTO_API_PATH = $(BASE_PATH)/proto/api
PROTO_API_PROTOS = $(API_SERVICES:%=$(PROTO_API_PATH)/%.proto)

##############
## Protobuf ##
##############
# Set some platform variables for protoc.
PROTOC_VERSION := 3.4.0
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

$(GOPATH)/src/github.com/golang/protobuf/protoc-gen-go:
	@echo "+ $@"
	@go get -u github.com/golang/protobuf/protoc-gen-go

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
		--proto_path=../proto/data/ \
		--lint_out=. \
		../proto/data/*.proto
	@$(PROTOC) \
		-I$(PROTOC_INCLUDES) \
		-I$(GOPATH)/src \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--lint_out=. \
		--proto_path=$(PROTO_API_PATH) \
		$(PROTO_API_PROTOS)

PROTO_DEPS=$(PROTOC_GEN_GO) $(PROTOC) $(PROTOC_INCLUDES)

.PHONY: proto-generated-files
proto-generated-files: $(PROTO_DEPS) $(BASE_PATH)/proto/data
	@echo "+ $@"
	@echo '++ Seeing errors in this step? Make sure you have run `make dev`.'
	@mkdir -p $(BASE_PATH)/pkg/serialization/generated/data/
	@$(PROTOC) --proto_path=$(BASE_PATH)/proto/data/ --go_out=$(BASE_PATH)/pkg/serialization/generated/data/ $(BASE_PATH)/proto/data/*.proto

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
	@echo $(GENERATED_API_SRCS)

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
	@go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/...
	@go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/...

$(GOPATH)/src/github.com/mwitkow/go-proto-validators:
	@echo "+ $@"
	@go get -u github.com/mwitkow/go-proto-validators/protoc-gen-govalidators

$(GENERATED_API_PATH):
	@echo "+ $@"
	@mkdir -p $(GENERATED_API_PATH)

$(GENERATED_DOC_PATH):
	@echo "+ $@"
	@mkdir -p $(GENERATED_DOC_PATH)

# Generate all of the proto messages and gRPC services with one invocation of
# protoc when any of the .pb.go sources don't exist or when any of the .proto
# files change.
$(GENERATED_API_PATH)/%.pb.go: $(PROTO_DEPS) $(PROTOC_GEN_GO) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_GOVALIDATORS) $(GENERATED_API_PATH) $(PROTO_API_PROTOS)
	@echo "+ $@"
	@$(PROTOC) \
		-I$(PROTOC_INCLUDES) \
		-I$(GOPATH)/src \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--proto_path=$(PROTO_API_PATH) \
		--go_out=plugins=grpc:$(GENERATED_API_PATH) \
		$(PROTO_API_PROTOS)

# Generate all of the reverse-proxies (gRPC-Gateways) with one invocation of
# protoc when any of the .pb.gw.go sources don't exist or when any of the
# .proto files change.
$(GENERATED_API_PATH)/%.pb.gw.go: $(PROTO_DEPS) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_GOVALIDATORS) $(GENERATED_API_PATH) $(PROTO_API_PROTOS)
	@echo "+ $@"
	@$(PROTOC) \
		-I$(PROTOC_INCLUDES) \
		-I$(GOPATH)/src \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--proto_path=$(PROTO_API_PATH) \
		--grpc-gateway_out=logtostderr=true:$(GENERATED_API_PATH) \
		$(PROTO_API_PROTOS)

# Generate all of the validator sources with one invocation of protoc
# when any of the .validator.pb.go sources don't exist or when any of the
# .proto files change.
$(GENERATED_API_VALIDATOR_SRCS) : $(PROTO_DEPS) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_GOVALIDATORS) $(GENERATED_DOC_PATH) $(PROTO_API_PROTOS)
	@echo "+ $@"
	@$(PROTOC) \
		-I$(PROTOC_INCLUDES) \
		-I$(GOPATH)/src \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--proto_path=$(PROTO_API_PATH) \
		--govalidators_out=$(GENERATED_API_PATH) \
		$(PROTO_API_PROTOS)

# Generate all of the swagger specifications with one invocation of protoc
# when any of the .swagger.json sources don't exist or when any of the
# .proto files change.
$(GENERATED_DOC_PATH)/%.swagger.json: $(PROTO_DEPS) $(PROTOC_GEN_GRPC_GATEWAY) $(PROTOC_GEN_GOVALIDATORS) $(GENERATED_DOC_PATH) $(PROTO_API_PROTOS)
	@echo "+ $@"
	@$(PROTOC) \
		-I$(PROTOC_INCLUDES) \
		-I$(GOPATH)/src \
		-I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--proto_path=$(PROTO_API_PATH) \
		--swagger_out=logtostderr=true:$(GENERATED_DOC_PATH) \
		$(PROTO_API_PROTOS)

# Generate the docs from the merged swagger specs.
$(MERGED_API_SWAGGER_SPEC): $(BASE_PATH)/scripts/mergeswag.sh $(GENERATED_API_SWAGGER_SPECS)
	@echo "+ $@"
	$(BASE_PATH)/scripts/mergeswag.sh $(GENERATED_DOC_PATH)/v1

# Generate the docs from the merged swagger specs.
$(GENERATED_API_DOCS): $(MERGED_API_SWAGGER_SPEC) $(PROTOC_GEN_GRPC_GATEWAY)
	@echo "+ $@"
	docker run --user $(shell id -u) --rm -v $(CURDIR)/docs:/tmp/docs swaggerapi/swagger-codegen-cli generate -l html2 -i /tmp/$< -o /tmp/$@

.PHONY: clean-protos
clean-protos:
	@rm -rf $(GOPATH)/src/github.com/grpc-ecosystem
	@rm -rf $(GOPATH)/src/github.com/golang/protobuf
	@rm -f $(GOPATH)/bin/protoc-gen-grpc-gateway
	@rm -f $(GOPATH)/bin/protoc-gen-go
	@rm -rf $(PROTOC_TMP)
	@rm -f $(PROTOC_FILE)
	@test -n "$(GENERATED_API_PATH)" && rm -rf "$(GENERATED_API_PATH)" || true
