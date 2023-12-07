UI_BASE_URL ?= https://localhost:8000
YARN_NETWORK_TIMEOUT ?= 60000

.PHONY: all
all: deps lint test build

SOURCES := $(shell find packages -type d \( -name node_modules \) -prune -o -print)
SOURCES += $(shell find apps -type d \( -name node_modules \) -prune -o -print)

# in monorepo yarn.lock might remain unchanged when new NPM packages are created,
# tracking package.json files of monorepo packages is the easiest way to discover such situations
PACKAGE_JSON_FILES := $(shell find . -type d \( -name node_modules \) -prune -false -o -name package.json)

export REACT_APP_ROX_PRODUCT_BRANDING := $(shell $(MAKE) --quiet --no-print-directory -C .. product-branding)

deps: yarn.lock $(PACKAGE_JSON_FILES)
	yarn install --frozen-lockfile --prefer-offline --network-timeout=$(YARN_NETWORK_TIMEOUT)
	@touch deps

.PHONY: printsrcs
printsrcs:
	@echo "+ $@"
	@echo "$(SOURCES)"

lint: deps $(SOURCES)
	@echo "+ $@"
	yarn lint
	@touch lint

build: deps $(SOURCES)
	@echo "+ $@"
	yarn build
	@touch build

.PHONY: start
start: deps
	@echo "+ $@"
	yarn start

.PHONY: test
test: deps $(SOURCES)
	@echo "+ $@"
ifdef CI
	cd apps/platform; yarn test-coverage
else
	cd apps/platform; yarn test
endif

.PHONY: test-e2e
test-e2e: deps $(SOURCES)
	@echo "+ $@"
	yarn test-e2e

.PHONY: clean
clean:
	@echo "+ $@"
	yarn clean
	rm -f deps
	rm -f lint
	rm -rf build
	rm -rf node_modules apps/*/node_modules packages/*/node_modules

.PHONY: publish-packages
publish-packages:
	yarn lerna:publish

######################
## Dev Dependencies ##
######################

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
		DEV_TARGET = linux
		UNAME_V := $(shell uname -v)
		ifneq (,$(findstring Ubuntu,$(UNAME_V)))
				DEV_TARGET = ubuntu
		endif
endif
ifeq ($(UNAME_S),Darwin)
		DEV_TARGET = osx
endif

.PHONY: dev
dev: dev-$(DEV_TARGET)
	@echo "+ $@"
	@echo "++ Updating npm..."
	@sudo npm install npm -g
	@echo "++ Installing serve..."
	@sudo npm install -g serve

.PHONY: dev-linux
dev-linux:
	@echo "+ $@"
	@echo 'Only OSX is supported in `make dev` right now.'
	@exit 1

.PHONY: dev-ubuntu
dev-ubuntu: nodejs-ubuntu
	sudo apt install libfontconfig -y
	@echo "+ $@"

.PHONY: dev-osx
dev-osx:
	@echo "+ $@"
	@echo '++ Installing node using `brew`'
	@brew install node
	@brew link --overwrite node
	@echo '++ Installing yarn using `brew`'
	@brew install yarn
	@brew link --overwrite yarn

.PHONY: nodejs-ubuntu
nodejs-ubuntu:
	@echo "++ Installing node.js/npm..."
	@curl -sL https://deb.nodesource.com/setup_6.x | sudo -E bash -
	@sudo apt-get install -y nodejs
	@sudo apt-get install -y build-essential
