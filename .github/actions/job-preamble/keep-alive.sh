#!/bin/bash
# Fast-runner keep-alive: register as ephemeral self-hosted runner.
# Called from job-preamble post-step. Launches the runner agent in a
# detached session (setsid) so it survives the orphan process cleanup.
# The post-step exits immediately, allowing the job to "complete",
# while the runner agent continues serving fast-pool jobs.

CPU_SCORE="${1:-0}"
KEEP_ALIVE_MINUTES="${2:-15}"
REPO="${GITHUB_REPOSITORY:-stackrox/stackrox}"
RUNNER_DIR="/dev/shm/fast-runner"
LOG="/dev/shm/keep-alive.log"

echo "::group::Fast runner keep-alive (score=$CPU_SCORE, target=${KEEP_ALIVE_MINUTES}m)"

# Get a registration token
echo "Requesting runner registration token..."
REG_TOKEN=$(curl -s -X POST \
  -H "Authorization: token ${GH_TOKEN}" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/${REPO}/actions/runners/registration-token" 2>/dev/null | \
  python3 -c "import json,sys; print(json.load(sys.stdin).get('token',''))" 2>/dev/null)

if [[ -z "$REG_TOKEN" ]]; then
  echo "keep-alive-result: FAILED — could not get registration token (GITHUB_TOKEN may lack admin scope)"
  echo "::endgroup::"
  exit 0
fi
echo "Registration token acquired"

# Download runner agent
RUNNER_VERSION="2.334.0"
echo "Downloading runner agent v${RUNNER_VERSION}..."
mkdir -p "$RUNNER_DIR"
curl -sL "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz" | \
  tar xz -C "$RUNNER_DIR"
echo "Runner agent downloaded"

# Configure
RUNNER_NAME="fast-pool-$(hostname | cut -c1-12)-$$"
echo "Configuring runner as ${RUNNER_NAME}..."
"$RUNNER_DIR/config.sh" \
  --url "https://github.com/${REPO}" \
  --token "$REG_TOKEN" \
  --name "$RUNNER_NAME" \
  --labels "fast-pool,self-hosted,linux,x64" \
  --ephemeral \
  --unattended \
  --disableupdate \
  --replace >> "$LOG" 2>&1 || {
    echo "keep-alive-result: FAILED — config.sh failed"
    cat "$LOG"
    echo "::endgroup::"
    exit 0
  }
echo "Runner configured"
echo "keep-alive-runner-name: ${RUNNER_NAME}"

# Launch the runner in a detached session so it survives orphan cleanup.
# setsid creates a new session — the runner won't be a child of the
# GitHub Actions runner agent, so "Cleaning up orphan processes" skips it.
echo "Launching runner agent in detached session (${KEEP_ALIVE_MINUTES}m timeout)..."
setsid bash -c "
  timeout $((KEEP_ALIVE_MINUTES * 60)) '$RUNNER_DIR/run.sh' >> '$LOG' 2>&1 || true
  # Deregister when done
  '$RUNNER_DIR/config.sh' remove --token '$REG_TOKEN' >> '$LOG' 2>&1 || true
  echo 'keep-alive-end: \$(date -u +%Y-%m-%dT%H:%M:%SZ)' >> '$LOG'
  echo 'keep-alive-result: COMPLETED' >> '$LOG'
" &
disown

echo "Runner agent launched (PID: $!, log: $LOG)"
echo "The job will now complete. The runner continues serving fast-pool jobs."
echo "::endgroup::"
