ROX_PROJECT=apollo
TESTFLAGS=-race -p 4
BASE_DIR=$(CURDIR)

.PHONY: all
all: deps pkg apollo

deps: glide.yaml glide.lock
	@echo "+ $@"
	@glide --quiet install 2>&1 | tee glide.out
	@testerror="$$(grep 'Lock file may be out of date' glide.out | wc -l)" && test $$testerror -eq 0
	@touch deps

.PHONY: pkg
pkg:
	make -C pkg

.PHONY: build-pkg
build-pkg:
	make -C pkg build-pkg

.PHONY: apollo
apollo:
	make -C apollo

# TODO(c): These should just be bazel builds.
.PHONY: image/bin/apollo
image/bin/apollo:
	@echo "+ $@"
	make -C apollo static-binary
	cp apollo/container/bin/apollo image/bin/apollo

.PHONY: image/bin/swarm-agent
image/bin/swarm-agent:
	@echo "+ $@"
	make -C agent/swarm static-binary
	cp agent/swarm/container/bin/agent image/bin/swarm-agent

BINARIES  = image/bin/apollo
BINARIES += image/bin/swarm-agent

image: deps build-pkg $(BINARIES)
	@echo "+ $@"
	docker build -t stackrox/apollo:latest image/

clean:
	git clean -xf image/bin
	make -C pkg clean
	make -C apollo clean
	make -C agent/swarm clean

include build/protogen.mk
