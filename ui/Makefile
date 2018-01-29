UI_BASE_URL ?= https://localhost:8000

.PHONY: all
all: deps lint test build

deps: package.json
	yarn install
	@touch deps

SOURCES := $(shell ls *.json) tailwind.js yarn.lock
SOURCES += $(shell find prod)
SOURCES += $(shell find public)
SOURCES += $(shell find src -name '*.js' -print)
SOURCES += $(shell find src -name '*.css' -print)
SOURCES += $(shell find src -name '*.html' -print)

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

.PHONY: start
start: deps
	@echo "+ $@"
	yarn start

.PHONY: test
test: deps $(SOURCES)
	@echo "+ $@"
	yarn test

.PHONY: test-watch
test-watch: deps
	@echo "+ $@"
	yarn test-watch

.PHONY: test-e2e-local
test-e2e-local: deps $(SOURCES)
	@echo "+ $@"
	yarn test-e2e-local

.PHONY: test-e2e-ci
test-e2e-ci: deps $(SOURCES)
	@echo "+ $@"
	yarn test-e2e-ci --config baseUrl=$(UI_BASE_URL)

.PHONY: clean
clean:
	@echo "+ $@"
	rm -f deps
	rm -f lint
	rm -rf build
	rm -rf node_modules

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
