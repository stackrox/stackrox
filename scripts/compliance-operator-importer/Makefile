BINARY     := compliance-operator-importer
MODULE     := ./cmd/importer
IMAGE      ?= localhost/compliance-operator-importer
TAG        ?= latest
ARCHS      ?= amd64 arm64
GOARCH     ?= $(shell go env GOARCH)
CONTAINER  ?= $(shell command -v podman 2>/dev/null || echo docker)

.PHONY: build test image image-push image-multiarch clean \
        demo-seed demo-seed-down demo-seed-status demo help test-v

## ── Build ──────────────────────────────────────────────────────────────────

build: ## Build the importer binary
	CGO_ENABLED=0 go build -o $(BINARY) $(MODULE)

## ── Test ───────────────────────────────────────────────────────────────────

test: ## Run all unit tests
	go test ./...

test-v: ## Run all unit tests (verbose)
	go test -v ./...

## ── Container image ────────────────────────────────────────────────────────

image: ## Build container image for host arch
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -o $(BINARY) $(MODULE)
	$(CONTAINER) build -t $(IMAGE):$(TAG) .

image-multiarch: ## Build per-arch images (use before image-push)
	@for arch in $(ARCHS); do \
		echo "── Building $(IMAGE):$(TAG)-$$arch ──"; \
		CGO_ENABLED=0 GOOS=linux GOARCH=$$arch go build -o $(BINARY) $(MODULE) && \
		$(CONTAINER) build --platform linux/$$arch -t $(IMAGE):$(TAG)-$$arch . ; \
	done

image-push: image-multiarch ## Push multi-arch images and create manifest
	@for arch in $(ARCHS); do \
		$(CONTAINER) push $(IMAGE):$(TAG)-$$arch ; \
	done
	$(CONTAINER) manifest create $(IMAGE):$(TAG) \
		$(foreach arch,$(ARCHS),$(IMAGE):$(TAG)-$(arch))
	$(CONTAINER) manifest push $(IMAGE):$(TAG)
	@echo ""
	@echo "Pushed: $(IMAGE):$(TAG)"

## ── Demo ───────────────────────────────────────────────────────────────────

demo-seed: ## Seed demo fixtures (2 ACS scans + 3 SSBs, 1 conflicting)
	./hack/demo-seed.sh up

demo-seed-down: ## Remove demo fixtures
	./hack/demo-seed.sh down

demo-seed-status: ## Show demo fixture state
	./hack/demo-seed.sh status

demo: build ## Run interactive demo
	./hack/demo.sh

## ── Misc ───────────────────────────────────────────────────────────────────

clean: ## Remove build artifacts
	rm -f $(BINARY)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
