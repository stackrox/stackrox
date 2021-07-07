# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
# The version reported by the root Makefile is converted to be compatible with SemVer. Specifically, inner-zero is
# dropped (e.g. 3.0.61.1 -> 3.61.1) and development version ".x" is changed to ".0" (e.g. 3.0.61.x-123 -> 3.0.61.0-123).
VERSION ?= $(shell $(MAKE) --quiet -C .. tag | sed -E 's@^(([[:digit:]]+\.)+)x(-)?@\10\3@g' | sed -E 's@^3.0.([[:digit:]]+\.[[:digit:]]+)(-)?@3.\1\2@g')

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=preview,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="preview,fast,stable")
CHANNELS ?= latest
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
DEFAULT_CHANNEL ?= latest
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
IMAGE_TAG_BASE ?= docker.io/stackrox/stackrox-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

ALL_BUNDLE_IMGS ?= $(BUNDLE_IMG)

# INDEX_IMG defines the image:tag used for the index (a.k.a. catalog) image, containing the single operator bundle image.
INDEX_IMG ?= $(IMAGE_TAG_BASE)-index:v$(VERSION)

# Image URL to use all building/pushing image targets
IMG ?= $(IMAGE_TAG_BASE):$(VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Lowercase Operating System name, needed for downloading GitHub releases.
OS=$(shell uname | tr A-Z a-z)

PROJECT_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash
.SHELLFLAGS = -o pipefail -euc

# This Makefile is not created for and likely would not benefit from concurrent jobs execution.
.NOTPARALLEL:

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

tag: ## Print the correct operator version (== image tag).
	@echo $(VERSION)

# `everything` and `all` are separate targets.
# `everything` pushes images. `all` is the default target that's executed when you simply run `make` without arguments.
# If `all` would push images, parameter-less `make` execution would have side-effect of pushing images to registry
# which might not be what the user intended to do (even though it could be safe in many cases).
# `everything` is a target that needs to be explicitly called for this reason.
.PHONY: everything
everything: build bundle ## Build everything (local binary, operator image, bundle image, index image) and push all images.
	$(MAKE) docker-build bundle-build
	$(MAKE)	docker-push docker-push-bundle
	$(MAKE) index-build
	$(MAKE) docker-push-index

##@ Dependencies download

CONTROLLER_GEN = $(PROJECT_DIR)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(PROJECT_DIR)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# TODO(ROX-7417): Migrate from 1.5.2 to the newest version and have only one operator-sdk target in the file.
OPERATOR_SDK_VERSION = 1.5.2
OPERATOR_SDK = $(PROJECT_DIR)/bin/operator-sdk-$(OPERATOR_SDK_VERSION)
.PHONY: operator-sdk
operator-sdk: ## Download operator-sdk necessary for scaffolding and bundling.
# See https://sdk.operatorframework.io/docs/installation/#install-from-github-release
	@ARCH=$$(case $$(uname -m) in (x86_64) echo -n amd64 ;; (aarch64) echo -n arm64 ;; (*) echo -n $$(uname -m) ;; esac) ;\
	OPERATOR_SDK_URL=https://github.com/operator-framework/operator-sdk/releases/download/v$(OPERATOR_SDK_VERSION)/operator-sdk_$(OS)_$${ARCH} ;\
	$(GET_GITHUB_RELEASE_FN); \
	get_github_release $(OPERATOR_SDK) $${OPERATOR_SDK_URL}

.PHONY: operator-sdk-for-testing
operator-sdk-for-testing: ## Download newer operator-sdk version necessary for testing bundle.
	@OPERATOR_SDK_VERSION=1.9.0 ;\
	OPERATOR_SDK=$(PROJECT_DIR)/bin/operator-sdk-$${OPERATOR_SDK_VERSION} ;\
	ARCH=$$(case $$(uname -m) in (x86_64) echo -n amd64 ;; (aarch64) echo -n arm64 ;; (*) echo -n $$(uname -m) ;; esac) ;\
	OPERATOR_SDK_URL=https://github.com/operator-framework/operator-sdk/releases/download/v$${OPERATOR_SDK_VERSION}/operator-sdk_$(OS)_$${ARCH} ;\
	$(GET_GITHUB_RELEASE_FN); \
	get_github_release $${OPERATOR_SDK} $${OPERATOR_SDK_URL}

# TODO(porridge): change back to kudobuilder release once https://github.com/kudobuilder/kuttl/issues/298 is fixed
KUTTL_VERSION = 0.10.0-gke-events2
KUTTL_UPSTREAM = porridge
KUTTL ?= $(PROJECT_DIR)/bin/kubectl-kuttl-$(KUTTL_VERSION)
.PHONY: kuttl
kuttl: ## Download kuttl.
	@ARCH=$$(case $$(uname -m) in (aarch64) echo -n armv6 ;; (*) echo -n $$(uname -m) ;; esac) ;\
	KUTTL_URL=https://github.com/$(KUTTL_UPSTREAM)/kuttl/releases/download/v$(KUTTL_VERSION)/kubectl-kuttl_$(KUTTL_VERSION)_$(OS)_$${ARCH} ;\
	$(GET_GITHUB_RELEASE_FN); \
	get_github_release $(KUTTL) $${KUTTL_URL}

OPM_VERSION = 1.17.3
OPM = $(PROJECT_DIR)/bin/opm-$(OPM_VERSION)
.PHONY: opm
opm: ## Download opm.
	@OPM_URL=https://github.com/operator-framework/operator-registry/releases/download/v$(OPM_VERSION)/$(OS)-$$(go env GOARCH)-opm ;\
	$(GET_GITHUB_RELEASE_FN); \
	get_github_release $(OPM) $${OPM_URL}

GET_GITHUB_RELEASE_FN = get_github_release() { \
	[ -f $${1} ] || { \
		set -euxo pipefail ;\
		mkdir -p bin ;\
		curl --fail --location --output $${1} $${2} ;\
		chmod +x $${1} ;\
		[[ "$$(uname -s)" != "Darwin" ]] || xattr -c $${1} ;\
	} \
}

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

##@ Development

parent-proto-generate: ## Make sure ../generated directory has up-to-date content that this operator (transitively) depends upon.
	$(MAKE) -C .. proto-generated-srcs

manifests: parent-proto-generate controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: parent-proto-generate controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

test-e2e: build install kuttl  ## Run e2e tests.
	mkdir -p $(PROJECT_DIR)/build/kuttl-test-artifacts
	$(KUTTL) test --namespace stackrox

stackrox-image-pull-secret: ## Create default image pull secret for StackRox images on DockerHub. Used by Helm chart.
# Create stackrox namespace if not exists.
	echo '{ "apiVersion": "v1", "kind": "Namespace", "metadata": { "name": "stackrox" } }' | kubectl apply -f -
# Create stackrox image pull secret in stackrox namespace.
	$(PROJECT_DIR)/../deploy/common/pull-secret.sh stackrox docker.io | kubectl -n stackrox apply -f -

run: manifests generate fmt vet ## Run operator from your host without deploying it to a cluster.
	../scripts/go-run.sh ./main.go

##@ Build

build: generate fmt vet ## Build operator local binary.
	../scripts/go-build-file.sh ./main.go bin/manager

docker-build: test ## Build docker image with the operator.
	@( \
		`# status.sh file is used by scripts/go-build.sh which we try to run in the docker container.` \
		`# status.sh needs git repo, make and Makefile and who knows what else but its actual output is simple.` \
		`# Here we grab the output and create a new build/status.sh from it. That file will be copied into the` \
		`# container and used there without having to bring all dependencies.` \
		set -euxo pipefail ;\
		smuggled_status_sh="$(PROJECT_DIR)/build/status.sh" ;\
		mkdir -p "$(PROJECT_DIR)/build" ;\
		cd "$(PROJECT_DIR)/.." ;\
		echo "#!/bin/sh" > "$${smuggled_status_sh}" ;\
		./status.sh | awk '{print "echo \"" $$0 "\""}' >> "$${smuggled_status_sh}" ;\
		chmod +x "$${smuggled_status_sh}" ;\
		`# Verify that the resulting status.sh is actually runnable` \
		"$${smuggled_status_sh}" ;\
	)
	DOCKER_BUILDKIT=1 BUILDKIT_PROGRESS=plain docker build -t ${IMG} --ssh default=${SSH_AUTH_SOCK} -f Dockerfile ..

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy operator image to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: kustomize ## Undeploy operator image from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

##@ Bundle and Index build

.PHONY: bundle
bundle: manifests kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
# Version is hardcoded to 0.0.1 here because otherwise git-versioned files are changed which we want to avoid.
# The correct version is updated later.
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMAGE_TAG_BASE):0.0.1
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle -q --overwrite --version 0.0.1 $(BUNDLE_METADATA_OPTS)
# operator-sdk 1.5.x has a bug in that it doesn't sort some metadata labels, and thus produces non-deterministic output.
	./bundle_helpers/normalize-metadata.py 'LABEL operators.operatorframework.io.metrics.' <bundle.Dockerfile >bundle.Dockerfile.tmp
	mv bundle.Dockerfile.tmp bundle.Dockerfile
	./bundle_helpers/normalize-metadata.py '  operators.operatorframework.io.metrics.' <bundle/metadata/annotations.yaml >bundle/metadata/annotations.yaml.tmp
	mv bundle/metadata/annotations.yaml.tmp bundle/metadata/annotations.yaml
# Delete lines that copy the original bundle files in bundle.Dockerfile (we can't just use `rm` in scratch image).
# Post-processed files will be copied instead, as configured in bundle.Dockerfile.extra.
	sed -i'.bak' -e '/# Copy files to locations specified by labels./d' bundle.Dockerfile
	sed -i'.bak' -E -e '/^COPY .* \/(manifests|metadata|tests\/scorecard)\/$$/d' bundle.Dockerfile
	rm -f bundle.Dockerfile.bak
# We cannot add changes to bundle.Dockerfile directly since `operator-sdk generate bundle` would wipe them.
# Yet we want most of the contents autogenerated from the Makefile variables as a single source of truth.
# Therefore we append ".extra" file to the end of bundle's dockerfile.
	cat bundle.Dockerfile.extra >> bundle.Dockerfile
# Run a python script to fix the orders in the specDescriptors (children must not appear before their parents).
	set -euo pipefail ;\
	python3 -m venv bundle_helpers/.venv ;\
	. bundle_helpers/.venv/bin/activate ;\
	pip3 install --upgrade pip setuptools ;\
	pip3 install -r bundle_helpers/requirements.txt ;\
	bundle_helpers/fix-spec-descriptor-order.py \
	  <bundle/manifests/rhacs-operator.clusterserviceversion.yaml \
	  >bundle/manifests/rhacs-operator.clusterserviceversion.yaml.fixed
	mv bundle/manifests/rhacs-operator.clusterserviceversion.yaml.fixed \
       bundle/manifests/rhacs-operator.clusterserviceversion.yaml
	$(OPERATOR_SDK) bundle validate ./bundle --select-optional suite=operatorframework

.PHONY: bundle-post-process
bundle-post-process: operator-sdk ## Post-process CSV file to include correct operator versions, etc.
# Copy the original bundle to build/ directory.
	mkdir -p build/
	rm -rf build/bundle
	cp -a bundle build/
# Run Python script to post-process necessary files in the build/bundle/. Configure Python dependencies if not done already.
	set -euo pipefail ;\
	python3 -m venv bundle_helpers/.venv ;\
	. bundle_helpers/.venv/bin/activate ;\
	pip3 install --upgrade pip setuptools ;\
	pip3 install -r bundle_helpers/requirements.txt ;\
	./bundle_helpers/patch-csv.py \
		--use-version $(VERSION) \
		--first-version 3.62.0 \
		--operator-image $(IMG) \
		--no-related-images \
		< bundle/manifests/rhacs-operator.clusterserviceversion.yaml > build/bundle/manifests/rhacs-operator.clusterserviceversion.yaml
# Check that the resulting bundle still passes validations.
	$(OPERATOR_SDK) bundle validate ./build/bundle --select-optional suite=operatorframework

.PHONY: bundle-build
bundle-build: bundle.Dockerfile bundle-post-process ## Build the bundle image.
	docker build -f $< -t $(BUNDLE_IMG) .

.PHONY: bundle-test
bundle-test: operator-sdk bundle-post-process ## Run scorecard tests against bundle files.
	$(OPERATOR_SDK) scorecard build/bundle/

.PHONY: bundle-test-image
bundle-test-image: operator-sdk ## Run scorecard tests against bundle image in the registry. Bundle image must be pushed beforehand.
	$(OPERATOR_SDK) scorecard $(BUNDLE_IMG)

.PHONY: index-build
index-build: opm ## Build Index (a.k.a. Catalog) image with the bundle. Bundle image must be pushed beforehand.
	DOCKER_BUILDKIT=1 BUILDKIT_PROGRESS=plain $(OPM) index add --bundles $(ALL_BUNDLE_IMGS) --tag $(INDEX_IMG) --build-tool docker

##@ Push images

.PHONY: docker-push
docker-push: ## Push docker image with the operator.
	docker push ${IMG}

.PHONY: docker-push-bundle
docker-push-bundle: ## Push docker image with the bundle.
	docker push ${BUNDLE_IMG}

.PHONY: docker-push-index
docker-push-index: ## Push docker image with the index.
	docker push ${INDEX_IMG}
