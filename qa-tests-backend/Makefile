.PHONY: all
all: style test

GRADLE := ./gradlew
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

build/generated: src/main/proto
	@echo "+ $@"
	$(GRADLE) generateProto generateTestProto

src/main/proto: $(shell find ../proto)
	@echo "+ migrate protos"
	-rm -r src/main/proto
	@scripts/migrate_protos.sh

.PHONY: clean-generated-srcs
clean-generated-srcs:
	@echo "+ $@"
	git clean -xdf build/generated

.PHONY: test
test: compile
	@echo "+ $@"
	$(GRADLE) testBegin testParallel testRest $(GRADLE_TEST_ARGS)

.PHONY: bat-test
bat-test: compile
	@echo "+ $@"
	$(GRADLE) testBegin testParallelBAT testBAT $(GRADLE_TEST_ARGS)

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