.PHONY: all
all: style test

GRADLE := ./gradlew

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
	$(GRADLE) test -Dgroups=-Upgrade,-SensorBounce,-SensorBounceNext

.PHONY: bat-test
bat-test: compile
	@echo "+ $@"
	$(GRADLE) test -Dgroups=BAT

.PHONY: smoke-test
smoke-test: compile
	@echo "+ $@"
	$(GRADLE) test -Dgroups=SMOKE

.PHONY: runtime-test
runtime-test: compile
	@echo "+ $@"
	$(GRADLE) test -Dgroups=RUNTIME

.PHONY: enforcement-test
enforcement-test: compile
	@echo "+ $@"
	$(GRADLE) test -Dgroups=PolicyEnforcement

.PHONY: integration-test
integration-test: compile
	@echo "+ $@"
	$(GRADLE) test -Dgroups=Integration

.PHONY: networkpolicy-simulator-test
networkpolicy-simulator-test: compile
	@echo "+ $@"
	$(GRADLE) test -Dgroups=NetworkPolicySimulation

.PHONY: non-bat-test
non-bat-test: compile
	@echo "+ $@"
	$(GRADLE) test -Dgroups=-BAT,-Upgrade,-SensorBounce,-SensorBounceNext

.PHONY: upgrade-test
upgrade-test: compile
	@echo "+ $@"
	$(GRADLE) test -Dgroups=Upgrade

.PHONY: graphql-test
graphql-test: compile
	@echo "+ $@"
	$(GRADLE) test -Dgroups=GraphQL

.PHONY: sensor-bounce-test
sensor-bounce-test: compile
	@echo "+ $@"
	$(GRADLE) test -Dgroups=SensorBounce
	$(GRADLE) test -Dgroups=SensorBounceNext
