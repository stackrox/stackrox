#!/bin/bash
# Fast-runner keep-alive: register as ephemeral self-hosted runner
# to serve additional jobs from the fast-pool queue.
# Called from job-preamble post-step when CPU score exceeds threshold.

set -euo pipefail

CPU_SCORE="${1:-0}"
KEEP_ALIVE_MINUTES="${2:-15}"
REPO="${GITHUB_REPOSITORY:-stackrox/stackrox}"
RUNNER_DIR="/dev/shm/fast-runner"

ka_start=$(date -u +%Y-%m-%dT%H:%M:%SZ)
ka_start_epoch=${EPOCHSECONDS:-$(date +%s)}
echo "::group::Fast runner keep-alive (score=$CPU_SCORE, target=${KEEP_ALIVE_MINUTES}m)"
echo "keep-alive-start: ${ka_start}"

# Get a registration token
echo "Requesting runner registration token..."
REG_TOKEN=$(curl -s -X POST \
  -H "Authorization: token ${GH_TOKEN}" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/${REPO}/actions/runners/registration-token" | \
  python3 -c "import json,sys; print(json.load(sys.stdin)['token'])" 2>/dev/null)

if [[ -z "$REG_TOKEN" || "$REG_TOKEN" == "None" ]]; then
  echo "keep-alive-result: FAILED — could not get registration token"
  echo "::endgroup::"
  exit 0
fi
echo "Registration token acquired"

# Download runner agent
RUNNER_VERSION="2.334.0"
RUNNER_URL="https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz"
mkdir -p "$RUNNER_DIR"
echo "Downloading runner agent v${RUNNER_VERSION}..."
curl -sL "$RUNNER_URL" | tar xz -C "$RUNNER_DIR"
echo "Runner agent downloaded"

# Configure as ephemeral runner with fast-pool label
RUNNER_NAME="fast-pool-$(hostname)-$$"
echo "Configuring runner as ${RUNNER_NAME}..."
"$RUNNER_DIR/config.sh" \
  --url "https://github.com/${REPO}" \
  --token "$REG_TOKEN" \
  --name "$RUNNER_NAME" \
  --labels "fast-pool,self-hosted,linux,x64" \
  --ephemeral \
  --unattended \
  --disableupdate \
  --replace 2>&1 || {
    echo "keep-alive-result: FAILED — config.sh failed"
    echo "::endgroup::"
    exit 0
  }
echo "Runner configured"

# Start the runner agent with a timeout
# The runner will pick up one job (ephemeral) or exit when time is up
echo "Starting runner agent (timeout ${KEEP_ALIVE_MINUTES}m)..."
echo "keep-alive-runner-name: ${RUNNER_NAME}"
echo "keep-alive-labels: fast-pool,self-hosted,linux,x64"

timeout $((KEEP_ALIVE_MINUTES * 60)) "$RUNNER_DIR/run.sh" 2>&1 &
RUNNER_PID=$!

# Monitor while runner is alive
tick=0
while kill -0 $RUNNER_PID 2>/dev/null; do
  tick=$((tick + 1))
  elapsed=$(( ${EPOCHSECONDS:-$(date +%s)} - ka_start_epoch ))
  echo "keep-alive tick=$tick elapsed=${elapsed}s $(date -u +%H:%M:%SZ) mem=$(free -m | awk '/^Mem:/{printf "%d/%dMB", $3, $2}') disk=$(df -BGB --output=avail / | tail -1 | tr -d ' ')"
  sleep 60
done

wait $RUNNER_PID 2>/dev/null || true

# Cleanup: remove the runner registration
echo "Removing runner registration..."
"$RUNNER_DIR/config.sh" remove --token "$REG_TOKEN" 2>&1 || true

ka_end=$(date -u +%Y-%m-%dT%H:%M:%SZ)
ka_total=$(( ${EPOCHSECONDS:-$(date +%s)} - ka_start_epoch ))
echo "keep-alive-end: ${ka_end}"
echo "keep-alive-duration: ${ka_total}s (target was $((KEEP_ALIVE_MINUTES * 60))s)"

if [[ $ka_total -ge $((KEEP_ALIVE_MINUTES * 60 - 5)) ]]; then
  echo "keep-alive-result: SUCCESS — runner served fast-pool for full ${KEEP_ALIVE_MINUTES}m"
else
  echo "keep-alive-result: COMPLETED — runner picked up a job and finished in ${ka_total}s"
fi
echo "::endgroup::"
