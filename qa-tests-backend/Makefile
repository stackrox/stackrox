.PHONY: all
all: style test

GRADLE = TEST_TARGET="$@" ./gradlew
GRADLE_TEST_ARGS =
# Stop testing after the first test failure by defining FAIL_FAST. (Can be
# achieved on a PR with a ci-fail-fast label)
ifdef FAIL_FAST
GRADLE_TEST_ARGS = --fail-fast
endif

###########
## Style ##
###########
.PHONY: style
style: compile
	@echo "+ $@"
	@$(GRADLE) --continue codenarcMain codenarcTest ; \
	STATUS=$$? ; \
	if [ $$STATUS -ne 0 ]; then \
	    cat build/reports/codenarc/main.txt build/reports/codenarc/test.txt >&2 2>/dev/null ; \
	fi ; \
	exit $$STATUS

.PHONY: compile
compile: build/generated
	@echo "+ $@"
	@$(GRADLE) assemble testClasses

.PHONY: style-fix
style-fix:
	@echo "+ $@"
	@scripts/fix_lint.py .

.PHONY: proto-generated-srcs
proto-generated-srcs: build/generated

# CI runner pods suffer transient DNS flakes ("Name or service not known")
# that kill gradle's plugin/dependency resolution mid-step. Retry a couple of
# times, waiting for DNS to recover between attempts.
# Budget: 5 gradle attempts, up to 2 min of DNS wait + 30s sleep between attempts.
build/generated: src/main/proto/scanner
	@echo "+ $@"
	@status=1; \
	for attempt in 1 2 3 4 5; do \
	  if $(GRADLE) generateProto generateTestProto; then status=0; break; fi; \
	  [ "$$attempt" -eq 5 ] && break; \
	  echo "WARN: generateProto failed (attempt $$attempt), possibly a CI DNS flake; waiting for DNS, then retrying"; \
	  ../scripts/ci/wait-for-dns.sh 120 plugins.gradle.org plugins-artifacts.gradle.org services.gradle.org repo.maven.apache.org || true; \
	  sleep 30; \
	done; \
	exit $$status

src/main/proto/scanner: ../go.mod
	@echo "+ copy scanner protos"
	-rm -rf src/main/proto/scanner
	@scripts/copy_scanner_protos.sh

.PHONY: clean-generated-srcs
clean-generated-srcs:
	@echo "+ $@"
	git clean -xdf build/generated

.PHONY: test
test: compile
	@echo "+ $@"
	$(GRADLE) testParallel testRest $(GRADLE_TEST_ARGS)

.PHONY: bat-test
bat-test: compile
	@echo "+ $@"
	$(GRADLE) testParallelBAT testBAT $(GRADLE_TEST_ARGS)

.PHONY: smoke-test
smoke-test: compile
	@echo "+ $@"
	$(GRADLE) testSMOKE $(GRADLE_TEST_ARGS)

.PHONY: compatibility-test
compatibility-test: compile
	@echo "+ $@"
	$(GRADLE) testCOMPATIBILITY testCOMPATIBILITYSensorBounce $(GRADLE_TEST_ARGS)

.PHONY: runtime-test
runtime-test: compile
	@echo "+ $@"
	$(GRADLE) testRUNTIME $(GRADLE_TEST_ARGS)

.PHONY: enforcement-test
enforcement-test: compile
	@echo "+ $@"
	$(GRADLE) testPolicyEnforcement $(GRADLE_TEST_ARGS)

.PHONY: integration-test
integration-test: compile
	@echo "+ $@"
	$(GRADLE) testIntegration $(GRADLE_TEST_ARGS)

.PHONY: networkpolicy-simulator-test
networkpolicy-simulator-test: compile
	@echo "+ $@"
	$(GRADLE) testNetworkPolicySimulation $(GRADLE_TEST_ARGS)

.PHONY: upgrade-test
upgrade-test: compile
	@echo "+ $@"
	$(GRADLE) testUpgrade $(GRADLE_TEST_ARGS)

.PHONY: graphql-test
graphql-test: compile
	@echo "+ $@"
	$(GRADLE) testGraphQL $(GRADLE_TEST_ARGS)

.PHONY: sensor-bounce-test
sensor-bounce-test: compile
	@echo "+ $@"
	$(GRADLE) testSensorBounce testSensorBounceNext $(GRADLE_TEST_ARGS)

.PHONY: pz-test
pz-test: compile
	@echo "+ $@"
	$(GRADLE) testPZ $(GRADLE_TEST_ARGS)

.PHONY: pz-test-debug
pz-test-debug: compile
	@echo "+ $@"
	$(GRADLE) testPZDebug $(GRADLE_TEST_ARGS)

.PHONY: deployment-check-test
deployment-check-test: compile
	@echo "+ $@"
	$(GRADLE) testDeploymentCheck $(GRADLE_TEST_ARGS)
