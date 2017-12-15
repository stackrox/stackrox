.PHONY: all
all: deps test image

.PHONY: image
image: build
	@echo "+ $@"
	docker build -t stackrox/apollo:ui-latest .

.PHONY: container
container:
	@echo "+ $@"
	docker run -p 3000:3000 --name apollo-ui -d stackrox/apollo:ui-latest

.PHONY: clean
clean:
	@echo "+ $@"
	rm -rf build
	rm -rf node_modules
	docker kill apollo-ui || true
	docker rm apollo-ui || true
	docker rmi stackrox/apollo:ui-latest || true

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
	

deps: package.json
	yarn install

.PHONY: build
build: deps
	@echo "+ $@"
	yarn build

.PHONY: start
start: deps
	@echo "+ $@"
	yarn start

.PHONY: test
test: deps
	@echo "+ $@"
	yarn test -- --coverage
