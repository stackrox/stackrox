ROX_PROJECT=apollo
TESTFLAGS=-race -p 4
BASE_DIR=$(CURDIR)

.PHONY: all
all: deps image

deps: glide.yaml glide.lock
	@echo "+ $@"
	@glide --quiet install 2>&1 | tee glide.out
	@testerror="$$(grep 'Lock file may be out of date' glide.out | wc -l)" && test $$testerror -eq 0
	@testerror="$$(grep '[ERROR]' glide.out | wc -l)" && test $$testerror -eq 0
	@touch deps

.PHONY: proto-generated
proto-generated:
	make -C pkg clean generated-srcs

image: deps proto-generated
	@echo "+ $@"
	bazel run //:gazelle
	bazel build --features=pure --cpu=k8 \
		//agent/swarm \
		//apollo \
		//docker-bench \
		//docker-bench-bootstrap \

# '*' is present in these commands because there is sometimes '_pure' (on Mac)
# and sometimes not (on Linux).
# TODO(cg): Replace with native bazel Docker build.
	cp bazel-bin/agent/swarm/linux_amd64_pure_stripped/swarm image/bin/swarm-agent
	cp bazel-bin/apollo/linux_amd64_pure_stripped/apollo image/bin/apollo
	cp bazel-bin/docker-bench/linux_amd64_pure_stripped/docker-bench image/bin/docker-bench
	cp bazel-bin/docker-bench-bootstrap/linux_amd64_pure_stripped/docker-bench-bootstrap image/bin/docker-bench-bootstrap
	chmod +w image/bin/*
	docker build -t stackrox/apollo:latest image/

clean:
	git clean -xf image/bin
	make -C pkg clean
	make -C apollo clean
	make -C agent/swarm clean

include make/protogen.mk
