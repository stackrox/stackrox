#!/bin/bash
# Register as a self-hosted runner and serve jobs from a warm pool.
# Usage: keep-alive.sh <pool-label> <keep-alive-minutes>
#
# Requires GH_TOKEN env var with admin scope for runner registration.
# The runner serves ephemeral jobs — after each job it re-registers
# and waits for the next one, until the keep-alive window expires.

set -euo pipefail

POOL_LABEL="${1:?Usage: keep-alive.sh <pool-label> <minutes>}"
KEEP_ALIVE_MINUTES="${2:-120}"
REPO="${GITHUB_REPOSITORY:-stackrox/stackrox}"
RUNNER_DIR="/tmp/runner-agent"
RUNNER_VERSION="2.334.0"

echo "::group::Warm runner pool: ${POOL_LABEL} (${KEEP_ALIVE_MINUTES}m)"
ka_start=$(date +%s)
ka_end=$((ka_start + KEEP_ALIVE_MINUTES * 60))

# Get registration token — try org-level first (needs Self-hosted runners permission),
# fall back to repo-level (needs Administration permission).
ORG="${REPO%%/*}"
echo "Requesting runner registration token (org: ${ORG})..."
REG_RESPONSE=$(curl -s -X POST \
  -H "Authorization: token ${GH_TOKEN}" \
  -H "Accept: application/vnd.github+json" \
  "https://api.github.com/orgs/${ORG}/actions/runners/registration-token" 2>/dev/null)
REG_TOKEN=$(echo "$REG_RESPONSE" | python3 -c "import json,sys; print(json.load(sys.stdin).get('token',''))" 2>/dev/null)

if [[ -z "$REG_TOKEN" || "$REG_TOKEN" == "None" ]]; then
  echo "Org-level token failed, trying repo-level..."
  REG_RESPONSE=$(curl -s -X POST \
    -H "Authorization: token ${GH_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/${REPO}/actions/runners/registration-token" 2>/dev/null)
  REG_TOKEN=$(echo "$REG_RESPONSE" | python3 -c "import json,sys; print(json.load(sys.stdin).get('token',''))" 2>/dev/null)
fi

if [[ -z "$REG_TOKEN" || "$REG_TOKEN" == "None" ]]; then
  echo "::error::Failed to get registration token. Response: ${REG_RESPONSE}"
  echo "::endgroup::"
  exit 1
fi

# Download runner agent (reuse if already present)
if [[ ! -f "$RUNNER_DIR/run.sh" ]]; then
  echo "Downloading runner agent v${RUNNER_VERSION}..."
  mkdir -p "$RUNNER_DIR"
  curl -sL "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz" | \
    tar xz -C "$RUNNER_DIR"
fi

RUNNER_NAME="${POOL_LABEL}-$(hostname | cut -c1-8)-$$"
echo "Runner: ${RUNNER_NAME}"
echo "Labels: ${POOL_LABEL},self-hosted,linux,x64"
echo "Keep-alive until: $(date -u -d @${ka_end} +%H:%M:%SZ 2>/dev/null || date -u +%H:%M:%SZ)"

# GHA has a 6-hour job timeout. Stop accepting new jobs at 5h30m
# to ensure any picked-up job has time to complete.
GHA_HARD_LIMIT=$((ka_start + 330 * 60))  # 5h30m from job start

# Serve jobs in a loop until the keep-alive window expires
iteration=0
while [[ $(date +%s) -lt $ka_end && $(date +%s) -lt $GHA_HARD_LIMIT ]]; do
  iteration=$((iteration + 1))
  remaining=$(( (ka_end - $(date +%s)) / 60 ))
  gha_remaining=$(( (GHA_HARD_LIMIT - $(date +%s)) / 60 ))
  if [[ $gha_remaining -lt $remaining ]]; then
    remaining=$gha_remaining
    echo "Note: GHA 6h limit approaching, ${gha_remaining}m left"
  fi

  # Register as ephemeral runner
  "$RUNNER_DIR/config.sh" \
    --url "https://github.com/${ORG}" \
    --token "$REG_TOKEN" \
    --name "$RUNNER_NAME" \
    --labels "${POOL_LABEL},self-hosted,linux,x64" \
    --ephemeral \
    --unattended \
    --disableupdate \
    --replace > /dev/null 2>&1 || {
      echo "config.sh failed on iteration ${iteration}"
      sleep 30
      # Refresh registration token (they expire after 1 hour)
      REG_TOKEN=$(curl -s -X POST \
        -H "Authorization: token ${GH_TOKEN}" \
        -H "Accept: application/vnd.github+json" \
        "https://api.github.com/orgs/${ORG}/actions/runners/registration-token" | \
        python3 -c "import json,sys; print(json.load(sys.stdin).get('token',''))" 2>/dev/null)
      continue
    }

  # Run the agent — blocks until it picks up and completes one job.
  # No timeout: let picked-up jobs finish regardless of keep-alive window.
  "$RUNNER_DIR/run.sh" > /dev/null 2>&1 || true

  echo "Job completed. Checking if keep-alive window still open..."
done

# Window expired — don't re-register, but any running job already finished
# (the loop only checks the window between jobs, never mid-job).
echo "Keep-alive window expired. Deregistering..."
"$RUNNER_DIR/config.sh" remove --token "$REG_TOKEN" 2>&1 || true

echo "Total iterations: ${iteration}"
echo "::endgroup::"
