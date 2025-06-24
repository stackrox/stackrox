UI_BASE_URL ?= https://localhost:8000
UI_PKG_INSTALL_NETWORK_TIMEOUT ?= 60000

.PHONY: all
all: deps lint test test-component build

SOURCES := $(shell find apps -type d \( -name node_modules \) -prune -o -print)

ROX_PRODUCT_BRANDING ?= $(shell $(MAKE) --quiet --no-print-directory -C .. product-branding)
export ROX_PRODUCT_BRANDING := $(ROX_PRODUCT_BRANDING)

deps: apps/platform/package.json apps/platform/package-lock.json
	cd apps/platform && npm ci --prefer-offline --fetch-timeout=$(UI_PKG_INSTALL_NETWORK_TIMEOUT) --no-audit --no-fund $(UI_PKG_INSTALL_EXTRA_ARGS) && npm audit --omit=dev || true
	@touch deps

.PHONY: printsrcs
printsrcs:
	@echo "+ $@"
	@echo "$(SOURCES)"

lint: deps $(SOURCES)
	@echo "+ $@"
	npm run lint
	@touch lint

print-npm-logs:
	@bash -c 'shopt -s nullglob; for f in /opt/app-root/src/.npm/_logs/*; do cat "$$f"; done'

build: deps print-npm-logs $(SOURCES)
	@echo "+ $@"
	npm run build
	@touch build

.PHONY: start
start: deps
	@echo "+ $@"
	npm run start

.PHONY: test
test: deps $(SOURCES)
	@echo "+ $@"
	npm run test

.PHONY: test-component
test-component: deps $(SOURCES)
	@echo "+ $@"
	npm run test-component

.PHONY: test-e2e
test-e2e: deps $(SOURCES)
	@echo "+ $@"
	npm run test-e2e

.PHONY: clean
clean:
	@echo "+ $@"
	npm run clean
	rm -f deps
	rm -f lint
	rm -rf build
	rm -rf node_modules apps/*/node_modules

