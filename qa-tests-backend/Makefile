.PHONY: all
all: style test

GRADLE := ./gradlew
GRADLE_TEST_ARGS =
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

src/main/proto: ../proto
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
	$(GRADLE) test $(GRADLE_TEST_ARGS) -Dgroups=-Upgrade,-SensorBounce,-SensorBounceNext

.PHONY: bat-test
bat-test: compile
	@echo "+ $@"
	$(GRADLE) test $(GRADLE_TEST_ARGS) -Dgroups=BAT

.PHONY: smoke-test
smoke-test: compile
	@echo "+ $@"
	$(GRADLE) test $(GRADLE_TEST_ARGS) -Dgroups=SMOKE

.PHONY: compatibility-test
compatibility-test: compile
	@echo "+ $@"
	$(GRADLE) test $(GRADLE_TEST_ARGS) -Dgroups=SMOKE


.PHONY: runtime-test
runtime-test: compile
	@echo "+ $@"
	$(GRADLE) test $(GRADLE_TEST_ARGS) -Dgroups=RUNTIME

.PHONY: enforcement-test
enforcement-test: compile
	@echo "+ $@"
	$(GRADLE) test $(GRADLE_TEST_ARGS) -Dgroups=PolicyEnforcement

.PHONY: integration-test
integration-test: compile
	@echo "+ $@"
	$(GRADLE) test $(GRADLE_TEST_ARGS) -Dgroups=Integration

.PHONY: networkpolicy-simulator-test
networkpolicy-simulator-test: compile
	@echo "+ $@"
	$(GRADLE) test $(GRADLE_TEST_ARGS) -Dgroups=NetworkPolicySimulation

.PHONY: non-bat-test
non-bat-test: compile
	@echo "+ $@"
	$(GRADLE) test $(GRADLE_TEST_ARGS) -Dgroups=-BAT,-Upgrade,-SensorBounce,-SensorBounceNext

.PHONY: upgrade-test
upgrade-test: compile
	@echo "+ $@"
	$(GRADLE) test $(GRADLE_TEST_ARGS) -Dgroups=Upgrade

.PHONY: graphql-test
graphql-test: compile
	@echo "+ $@"
	$(GRADLE) test $(GRADLE_TEST_ARGS) -Dgroups=GraphQL

.PHONY: sensor-bounce-test
sensor-bounce-test: compile
	@echo "+ $@"
	$(GRADLE) test $(GRADLE_TEST_ARGS) -Dgroups=SensorBounce
	$(GRADLE) test $(GRADLE_TEST_ARGS) -Dgroups=SensorBounceNext
