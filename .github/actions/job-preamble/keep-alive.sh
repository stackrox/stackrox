#!/bin/bash
# Fast-runner keep-alive: register as ephemeral self-hosted runner.
# Runs INLINE in the post-step — blocks until timeout. This keeps the
# job "in_progress" and the VM alive for the full keep-alive window.
# No setsid/nohup needed — the post-step IS the keep-alive.

CPU_SCORE="${1:-0}"
KEEP_ALIVE_MINUTES="${2:-15}"
REPO="${GITHUB_REPOSITORY:-stackrox/stackrox}"
RUNNER_DIR="/dev/shm/fast-runner"
LOG="/dev/shm/keep-alive.log"

echo "::group::Fast runner keep-alive (score=$CPU_SCORE, target=${KEEP_ALIVE_MINUTES}m)"
ka_start=$(date -u +%Y-%m-%dT%H:%M:%SZ)
ka_start_epoch=${EPOCHSECONDS:-$(date +%s)}
echo "keep-alive-start: ${ka_start}"

# Get a registration token
echo "Requesting runner registration token..."
REG_RESPONSE=$(curl -s -X POST \
  -H "Authorization: token ${GH_TOKEN}" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/repos/${REPO}/actions/runners/registration-token" 2>/dev/null)
REG_TOKEN=$(echo "$REG_RESPONSE" | python3 -c "import json,sys; print(json.load(sys.stdin).get('token',''))" 2>/dev/null)

if [[ -z "$REG_TOKEN" ]]; then
  echo "keep-alive-result: FAILED — could not get registration token"
  echo "API response: $REG_RESPONSE"
  # Fall back to simple sleep so we can at least verify the VM stays alive
  echo "Falling back to simple keep-alive (sleep) to verify VM survival..."
  for i in $(seq 1 "$KEEP_ALIVE_MINUTES"); do
    elapsed=$(( ${EPOCHSECONDS:-$(date +%s)} - ka_start_epoch ))
    echo "keep-alive tick=$i/${KEEP_ALIVE_MINUTES} elapsed=${elapsed}s $(date -u +%H:%M:%SZ) mem=$(free -m | awk '/^Mem:/{printf "%d/%dMB", $3, $2}') disk=$(df -BGB --output=avail / | tail -1 | tr -d ' ')"
    sleep 60
  done
  ka_end=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  ka_total=$(( ${EPOCHSECONDS:-$(date +%s)} - ka_start_epoch ))
  echo "keep-alive-end: ${ka_end}"
  echo "keep-alive-duration: ${ka_total}s (target was $((KEEP_ALIVE_MINUTES * 60))s)"
  if [[ $ka_total -ge $((KEEP_ALIVE_MINUTES * 60 - 5)) ]]; then
    echo "keep-alive-result: SUCCESS — VM stayed alive for full ${KEEP_ALIVE_MINUTES}m (no runner registration)"
  else
    echo "keep-alive-result: PARTIAL — VM only survived ${ka_total}s of $((KEEP_ALIVE_MINUTES * 60))s"
  fi
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

# Configure as ephemeral runner with fast-pool label
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
echo "Runner configured: ${RUNNER_NAME}"
echo "keep-alive-runner-name: ${RUNNER_NAME}"
echo "keep-alive-labels: fast-pool,self-hosted,linux,x64"

# Run the agent INLINE — this blocks the post-step, keeping the VM alive.
# The job stays "in_progress" while the runner serves fast-pool jobs.
echo "Starting runner agent inline (blocks for up to ${KEEP_ALIVE_MINUTES}m)..."
timeout $((KEEP_ALIVE_MINUTES * 60)) "$RUNNER_DIR/run.sh" 2>&1 || true

# Cleanup
echo "Runner agent exited. Deregistering..."
"$RUNNER_DIR/config.sh" remove --token "$REG_TOKEN" >> "$LOG" 2>&1 || true

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
