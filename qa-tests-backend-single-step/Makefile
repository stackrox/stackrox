#ROX_CI_IMAGE       := quay.io/cgorman1/apollo-ci:0.3.18
ROX_CI_IMAGE       := rox-ci-image:latest
USER_SCRIPTS_DEST  := ${HOME}/bin/port-forward
LAUNCH_AGENTS_DEST := ${HOME}/Library/LaunchAgents

lint:
	shellcheck --external-sources --source-path=./scripts ./scripts/*.sh
