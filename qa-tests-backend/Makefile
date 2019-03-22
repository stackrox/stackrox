.PHONY: all
all: style test

###########
## Style ##
###########
.PHONY: style
style:
	@echo "+ $@"
	@gradle --continue codenarcMain codenarcTest ; \
	STATUS=$$? ; \
	if [ $$STATUS -ne 0 ]; then \
	    cat build/reports/codenarc/main.txt build/reports/codenarc/test.txt >&2 2>/dev/null ; \
	fi ; \
	exit $$STATUS

.PHONY: style-fix
style-fix:
	@echo "+ $@"
	@scripts/fix_lint.py .

.PHONY: proto-generated-srcs
proto-generated-srcs:
	@echo "+ $@"
	-rm -r src/main/proto
	@scripts/migrate_protos.sh
	gradle generateProto generateTestProto

.PHONY: clean-generated-srcs
clean-generated-srcs:
	@echo "+ $@"
	git clean -xdf build/generated

.PHONY: test
test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=-Upgrade,-SensorBounce
	gradle test -Dgroups=SensorBounce

.PHONY: bat-test
bat-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=BAT
	gradle test -Dgroups=SensorBounce

.PHONY: smoke-test
smoke-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=SMOKE

.PHONY: runtime-test
runtime-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=RUNTIME

.PHONY: enforcement-test
enforcement-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=PolicyEnforcement

.PHONY: integration-test
integration-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=Integration

.PHONY: networkpolicy-simulator-test
networkpolicy-simulator-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=NetworkPolicySimulation

.PHONY: non-bat-test
non-bat-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=-BAT,-Upgrade,-SensorBounce

.PHONY: upgrade-test
upgrade-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=Upgrade

.PHONY: sensor-bounce-test
sensor-bounce-test: proto-generated-srcs
	@echo "+ $@"
	gradle test -Dgroups=SensorBounce