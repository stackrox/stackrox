---
name: verify-in-cluster
description: >
  Build, deploy, and verify StackRox code changes against a live Kubernetes/OpenShift cluster.
  Handles cluster discovery, StackRox authentication, component building with crane, image pushing
  to a container registry (quay.io preferred, ttl.sh fallback), deployment patching, and test execution.
argument-hint: "<what-to-verify>"
arguments: [context]
allowed-tools:
  - Bash
  - Read
  - Agent
  - AskUserQuestion
  - Write
  - Edit
effort: xhigh
disable-model-invocation: true
---

# Verify StackRox Changes Against a Real Cluster

You are verifying code changes by building, deploying, and testing them on a live cluster.
The `$context` argument describes WHAT to verify (bug repro, fix verification, manual API test,
e2e test name, etc.). If empty, infer from the current diff.

**YOLO mode**: Check whether `$context` contains the word "YOLO". If so, skip all user
confirmations throughout (cluster modification consent, credential confirmation, etc.).
Store this decision in a variable you reference later — do NOT rely on shell environment
variables persisting between Bash tool calls (they don't in Claude Code).

**Cleanup**: After verification completes (pass or fail), record the original image reference
so the user can restore it if desired. Do NOT auto-restore — the user or parent agent may
want to keep the modified deployment running for further inspection.

## Phase 0: Prerequisites Check

Check which tools are available:

```bash
command -v go      && echo "OK: go"      || echo "MISSING: go"
command -v roxie   && echo "OK: roxie"   || echo "MISSING: roxie"
command -v crane   && echo "OK: crane"   || echo "MISSING: crane"
command -v docker  && echo "OK: docker"  || echo "MISSING: docker"
command -v oc      && echo "OK: oc"      || echo "MISSING: oc"
command -v kubectl && echo "OK: kubectl" || echo "MISSING: kubectl"
command -v roxctl  && echo "OK: roxctl"  || echo "MISSING: roxctl"
command -v curl    && echo "OK: curl"    || echo "MISSING: curl"
command -v jq      && echo "OK: jq"      || echo "MISSING: jq"
command -v gh      && echo "OK: gh"      || echo "MISSING: gh (optional)"
command -v freeze  && echo "OK: freeze"  || echo "MISSING: freeze (optional)"
```

**Required**: `go`, `curl`, `jq` — stop with install instructions if any are missing.

**Cluster access**: need at least one of `oc` or `kubectl`. Prefer `oc` if both are
available. Use the chosen command literally in every subsequent Bash call — do not use
shell variables across calls (Claude Code does not share shell state).

**Roxie** (`github.com/stackrox/roxie`): strongly preferred for cluster setup and
deployment (Phases 1, 2). If missing, install it:
```bash
GOBIN=/usr/local/bin go install github.com/stackrox/roxie/cmd@latest && mv /usr/local/bin/cmd /usr/local/bin/roxie
```
If installation fails, fall back to manual cluster discovery and `deploy/deploy.sh`.

**roxctl**: needed by Roxie for deployment. If missing, build it with version ldflags
(roxctl panics without `MainVersion`):
```bash
MAIN_TAG=$(make --quiet --no-print-directory tag)
SCANNER_VERSION=$(cat SCANNER_VERSION)
CGO_ENABLED=0 go build \
  -ldflags="-s -w \
    -X github.com/stackrox/rox/pkg/version/internal.MainVersion=$MAIN_TAG \
    -X github.com/stackrox/rox/pkg/version/internal.ScannerVersion=$SCANNER_VERSION" \
  -o /usr/local/bin/roxctl ./roxctl
```

**Visual proof**: `freeze` (charmbracelet/freeze) is optional. If available, use it to
render command outputs as PNG images for PR attachment.

**Image push**: need at least one of `crane` or `docker`. Stop if neither is available.
If `crane` is available, test connectivity to ttl.sh:
```bash
crane manifest ttl.sh/test:1h 2>&1 || true
```
If it fails with TLS/x509 errors, retry with `--insecure`. If that works, use
`--insecure` on all subsequent `crane` commands. If crane cannot connect even with
`--insecure`, fall back to `docker`.

## Phase 1: Discover Cluster

### With Roxie (preferred)

Find the kubeconfig. Try in order:
1. `KUBECONFIG` environment variable
2. `$context` argument (if it contains a kubeconfig path)
3. Uploaded files — check `/workspace/file-uploads/` for kubeconfig files
4. Default `~/.kube/config`

**KUBECONFIG persistence**: env vars do not persist between Bash tool calls. If
`KUBECONFIG` is set to a non-default path, prepend `export KUBECONFIG=<path> &&`
to every Bash call that uses `oc`, `kubectl`, or `roxie`.

Run Roxie's environment detection:
```bash
export KUBECONFIG=<path> && roxie env
```
This prints the cluster type (OpenShift4, GKE, Kind, etc.), context name, and
kubeconfig path. Roxie auto-detects architecture and cluster capabilities.

Detect node architecture for cross-compilation:
```bash
export KUBECONFIG=<path> && oc get nodes -o jsonpath='{.items[0].status.nodeInfo.architecture}'
```
Default to `amd64` if detection fails.

Unless in YOLO mode, ask the user:
> "I found cluster [name] at [url]. Is it OK to modify deployments in the StackRox namespace?"

### Without Roxie (fallback)

Verify connectivity manually:
```bash
export KUBECONFIG=<path> && oc cluster-info
```

## Phase 2: Find StackRox and Authenticate

### 2a: Check if StackRox is deployed

Check common namespaces where Central may be running:
```bash
export KUBECONFIG=<path> &&   for ns in stackrox acs-central rhacs-operator; do
    if oc -n "$ns" get deployment central --no-headers 2>/dev/null; then
      echo "FOUND in namespace: $ns"
      break
    fi
  done
```
- `stackrox` — used by `deploy/deploy.sh` and Roxie with `--single-namespace`
- `acs-central` — Roxie's default namespace (when not using `--single-namespace`)
- `rhacs-operator` — used by the operator

Use whichever namespace has Central throughout. If none has it, go to **Phase 2c**.

### 2b: Authenticate to Central

#### With Roxie (preferred — if Roxie deployed this cluster)

Use `roxie shell` to retrieve endpoint and credentials from the saved manifest:
```bash
export KUBECONFIG=<path> && roxie shell -- bash -c 'echo "ENDPOINT=$ROX_ENDPOINT PASSWORD=$ROX_ADMIN_PASSWORD"'
```
If this succeeds, you have the endpoint and password. Verify with:
```bash
curl -sk -u "admin:<password>" "https://<endpoint>/v1/ping"
```

#### Manual fallback (if roxie shell fails or deployment wasn't done by Roxie)

Determine the Central endpoint. Try in order:
1. **LoadBalancer**: `oc -n <ns> get svc central-loadbalancer -o jsonpath='{.status.loadBalancer.ingress[0].ip}'`
2. **OpenShift route**: `oc -n <ns> get route central -o jsonpath='{.status.ingress[0].host}'`
3. **Port-forward** (use `run_in_background`): `oc -n <ns> port-forward svc/central 8000:443`

Try credentials: `ROX_ADMIN_PASSWORD` env var, then password files in `deploy/*/central-deploy/password`.
Test with `curl -sk -u "admin:<password>" "https://<endpoint>/v1/ping"`.

If none work, ask the user. In YOLO mode, fail with a clear error.

Remember the password and endpoint — substitute them literally into every subsequent command.

### 2c: Deploy StackRox (only if not found)

#### With Roxie (preferred)

Roxie handles cluster detection, image pull secrets, operator deployment, Central + SecuredCluster
CRs, readiness waiting, and credential generation — all in one command.

**CRITICAL: You MUST pass `--tag` to Roxie.** Without it, Roxie defaults to an old hardcoded
version (e.g., `4.9.2`) that will NOT match your source tree — causing DB migration mismatches
and wasted time. Fetch the latest master-based tag from Quay first:
```bash
git fetch origin master --quiet 2>/dev/null || true
TAGS=$(curl -s "https://quay.io/api/v1/repository/stackrox-io/main/tag/?limit=100&onlyActiveTags=true" \
  | jq -r '.tags[].name | select(test("^[0-9]+[.][0-9]+[.]x-")) | select(test("-(arm64|amd64|s390x|ppc64le)$") | not)')
for tag in $TAGS; do
  hash="${tag##*-g}"
  if git merge-base --is-ancestor "$hash" origin/master 2>/dev/null; then
    MAIN_IMAGE_TAG="$tag"
    echo "Using master-based tag: $MAIN_IMAGE_TAG"
    break
  fi
done
```
If no master-based tag is found, take the most recent one:
```bash
MAIN_IMAGE_TAG=$(echo "$TAGS" | head -1)
```

Now deploy with `--tag` and `--envrc` (Claude Code cannot use interactive subshells):
```bash
export KUBECONFIG=<path> && roxie deploy both \
  --tag "$MAIN_IMAGE_TAG" \
  --envrc /tmp/roxie-env.sh \
  --exposure loadbalancer \
  --resources auto
```
The `--tag` flag is mandatory — do NOT omit it or let Roxie use its default tag.

After deployment, read the credentials from the envrc file:
```bash
cat /tmp/roxie-env.sh
```
This contains `ROX_ENDPOINT`, `ROX_ADMIN_PASSWORD`, `ROX_BASE_URL`, etc.
Extract and remember these values for use in subsequent commands.

If Roxie's deploy fails (e.g., operator issues, image pull errors), check logs:
```bash
export KUBECONFIG=<path> && roxie logs operator 2>&1 | tail -30
```

#### Without Roxie (fallback)

If Roxie is not available, use `deploy/deploy.sh` with the appropriate env vars.
See the repo's `deploy/` directory for details. Key env vars:
```bash
export MAIN_IMAGE_TAG="<tag>"
export ROX_HTPASSWD_AUTH=true
export STORAGE=pvc
export LOAD_BALANCER=route   # on OpenShift
export MONITORING_SUPPORT=false
```
Then run `./deploy/deploy.sh`. Read the password from `deploy/openshift/central-deploy/password`
(or the k8s equivalent).

## Phase 3: Analyze Changes

Determine what source code has changed and which components need rebuilding.

```bash
git diff --name-only HEAD              # unstaged changes
git diff --name-only --cached HEAD     # staged changes
git ls-files --others --exclude-standard  # untracked new files
```

### Fast-path: Use CI-built image when no local changes exist

If there are **no local changes** (all three commands above produce empty output), check
whether a CI-built image already exists for the current HEAD commit. StackRox CI builds
images for every PR push and posts the tag as a GitHub comment by `github-actions[bot]`.

To check, find the PR number for the current branch and query its comments:
```bash
# Get current branch and find its PR
BRANCH=$(git branch --show-current)
PR_NUMBER=$(gh pr list --head "$BRANCH" --json number --jq '.[0].number' 2>/dev/null)

# Look for the CI bot comment with the image tag
if [[ -n "$PR_NUMBER" ]]; then
  CI_TAG=$(gh api "repos/stackrox/stackrox/issues/$PR_NUMBER/comments" \
    --jq '[.[] | select(.user.login == "github-actions[bot]") | select(.body | test("Build Images Ready"))] | last | .body' 2>/dev/null \
    | grep -oP 'MAIN_IMAGE_TAG=\K[^\s`]+' || true)
fi
```

If a CI tag is found, verify it matches the current HEAD:
```bash
HEAD_SHORT=$(git rev-parse --short=12 HEAD)
if [[ -n "$CI_TAG" && "$CI_TAG" == *"$HEAD_SHORT"* ]]; then
  echo "CI image matches HEAD: $CI_TAG"
fi
```

**When the CI tag matches HEAD and there are no local changes**, skip Phases 4-5 entirely.
Instead, go directly to Phase 6 and patch the deployment with the CI image:
```bash
oc -n <ns> set image deployment/central central=quay.io/stackrox-io/main:$CI_TAG
```
This is significantly faster (no build, no crane push) and uses a properly built image
with correct ldflags, UI assets, and matching DB migrations.

**When to NOT use this fast-path:**
- There are local uncommitted/staged changes — the CI image doesn't include them
- The CI tag's commit doesn't match HEAD — the image is stale
- No PR exists for the branch (e.g., working directly on a local branch)
- The `gh` CLI is not available

In all these cases, fall through to the normal build path (Phases 4-5).

### Component mapping

Map changed files to components using this table. Components sharing the **main image**
(central, sensor, admission-control, migrator, compliance, config-controller, roxctl) are
all built into the same container image. When using `crane mutate --append`, you can append
multiple binary layers to that single base image.

| Source directory prefix         | Component          | Go package path                    | Binary name           | Container binary path              | K8s resource                     | Container name        | Image            |
|---------------------------------|--------------------|------------------------------------|----------------------|------------------------------------|---------------------------------|----------------------|------------------|
| `central/`                      | central            | `./central`                        | `central`            | `/stackrox/central`                | deployment/central               | `central`            | main             |
| `migrator/`                     | migrator           | `./migrator`                       | `migrator`           | `/stackrox/bin/migrator`           | (init container in central pod)  | —                    | main             |
| `sensor/`                       | sensor             | `./sensor/kubernetes`              | `kubernetes-sensor`  | `/stackrox/bin/kubernetes-sensor`  | deployment/sensor                | `sensor`             | main             |
| `sensor/admission-control/`     | admission-control  | `./sensor/admission-control`       | `admission-control`  | `/stackrox/bin/admission-control`  | deployment/admission-control     | `admission-control`  | main             |
| `compliance/`                   | compliance         | `./compliance/cmd/compliance`      | `compliance`         | `/stackrox/bin/compliance`         | (see note below)                 | `compliance`         | main             |
| `scanner/`                      | scanner            | `./scanner/cmd/scanner`            | `scanner`            | `/usr/local/bin/scanner`           | deployment/scanner               | `scanner`            | **scanner** (separate) |
| `roxctl/`                       | roxctl             | `./roxctl`                         | `roxctl`             | `/stackrox/roxctl`                 | (CLI, not deployed)              | —                    | main             |
| `config-controller/`            | config-controller  | `./config-controller`              | `config-controller`  | `/stackrox/bin/config-controller`  | deployment/config-controller     | `manager`            | main             |
| `sensor/upgrader/`              | upgrader           | `./sensor/upgrader`                | `upgrader`           | `/stackrox/bin/sensor-upgrader`    | (used during upgrades)           | —                    | main             |
| `pkg/`, `generated/`, `proto/`  | **shared**         | —                                  | —                    | —                                  | (rebuild all affected)           | —                    | —                |
| `operator/`                     | operator           | —                                  | —                    | —                                  | (out of scope — has own image)   | —                    | operator (separate) |

**Important rules:**
- If `central/` is changed, ALWAYS also rebuild `migrator` — Central runs migrator on startup.
- If `pkg/` or `generated/` or `proto/` files changed, determine which components import
  the changed packages and rebuild those. Use `go list` or trace imports to decide.
- If only `roxctl/` changed, build it but don't patch any deployment — just make the binary
  available locally.
- **Sensor binary rename**: `go build ./sensor/kubernetes` produces a binary named `kubernetes`.
  You MUST rename it to `kubernetes-sensor` when creating the tar layer (see Phase 5 example).
- **Compliance** runs as a container in the `collector` DaemonSet, not a standalone
  Deployment. To patch it: `oc -n <ns> set image daemonset/collector compliance=<tag>`
- **Scanner uses a separate image** — you cannot append scanner binaries to the main image.
  To patch scanner, pull the current scanner image separately, append the scanner binary to
  `/usr/local/bin/scanner`, push to the selected registry, and patch deployment/scanner.
- **Operator** changes are out of scope for this skill. Inform the user.

If no code changes are detected, inform the user that there is nothing to build or deploy.
In YOLO mode, exit with a clear message: "No code changes detected. Nothing to verify."
Do not proceed to Phase 4.

## Phase 4: Build

Cross-compile each affected binary for the target architecture detected in Phase 1
(default `amd64`). Use the literal architecture value — do not rely on env vars:

```bash
GOOS=linux GOARCH=<arch> CGO_ENABLED=0 go build -ldflags="-s -w" -o "$TMPDIR/<binary-name>" ./<package-path>
```

Example for central + migrator (on an amd64 cluster):
```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$TMPDIR/central" ./central
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$TMPDIR/migrator" ./migrator
```

Example for sensor (note: binary must be renamed for container path):
```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$TMPDIR/kubernetes-sensor" ./sensor/kubernetes
```

Build binaries in parallel where possible (they're independent).

If the build fails, report the error and stop. Do NOT proceed with a broken build.

## Phase 5: Push Image

First, record the current image for **each** deployment being modified:

```bash
oc -n stackrox get deployment/central -o jsonpath='{.spec.template.spec.containers[0].image}'
```
Record this value — you'll use it as the base image for `crane mutate`.

**Choosing the right base image:** The base image must have a DB migration sequence
compatible with your source tree. If the deployed image was set up by Roxie with a
master-based Quay tag, it's already correct. If patching a pre-existing deployment whose
image version differs significantly from your branch, consider redeploying first.

**Private registry note:** Roxie deploys from `quay.io/rhacs-eng/` which requires
authentication. If `crane` cannot pull the base image (auth error), use the equivalent
public image from `quay.io/stackrox-io/` with the same tag. For example, replace
`quay.io/rhacs-eng/main:<tag>` with `quay.io/stackrox-io/main:<tag>`.

Generate a unique tag:
```bash
UUID=$(uuidgen | tr '[:upper:]' '[:lower:]' | cut -c1-8)
TAG="ttl.sh/$UUID:2h"
```

### Method A: crane (preferred)

Create tar layers with binaries at their correct **absolute** container paths:

```bash
cd "$TMPDIR"

# Central binary → /stackrox/central
mkdir -p stackrox && cp central stackrox/central && chmod +x stackrox/central
tar cf central-layer.tar stackrox/central && rm -rf stackrox

# Migrator binary → /stackrox/bin/migrator
mkdir -p stackrox/bin && cp migrator stackrox/bin/migrator && chmod +x stackrox/bin/migrator
tar cf migrator-layer.tar stackrox/bin/migrator && rm -rf stackrox

# Sensor binary → /stackrox/bin/kubernetes-sensor (note the rename!)
mkdir -p stackrox/bin && cp kubernetes-sensor stackrox/bin/kubernetes-sensor && chmod +x stackrox/bin/kubernetes-sensor
tar cf sensor-layer.tar stackrox/bin/kubernetes-sensor && rm -rf stackrox
```

Push by appending layers to the current image:
```bash
crane mutate "<current-central-image>" \
  --platform linux/<arch> \
  --append central-layer.tar \
  --append migrator-layer.tar \
  --tag "$TAG"
```

For scanner (separate image, different binary path):
```bash
mkdir -p usr/local/bin && cp scanner usr/local/bin/scanner && chmod +x usr/local/bin/scanner
tar cf scanner-layer.tar usr/local/bin/scanner && rm -rf usr

crane mutate "<current-scanner-image>" \
  --platform linux/<arch> \
  --append scanner-layer.tar \
  --tag "<scanner-tag>"
```

### Method B: docker (fallback if crane fails)

```dockerfile
FROM <current-central-image>
COPY central /stackrox/central
COPY migrator /stackrox/bin/migrator
```

```bash
docker buildx build --platform linux/<arch> --push -t "$TAG" -f Dockerfile .
```

## Phase 6: Deploy

Patch the deployment to use the new image:

```bash
oc -n <ns> set image deployment/central central=<tag>
```

Deployment/DaemonSet to container name mapping:
- deployment/`central` → container `central`
- deployment/`sensor` → container `sensor`
- deployment/`admission-control` → container `admission-control`
- deployment/`scanner` → container `scanner`
- deployment/`config-controller` → container `manager`
- daemonset/`collector` → container `compliance` (for compliance changes)

Wait for rollout:
```bash
oc -n <ns> rollout status deployment/central --timeout=300s
```

If using port-forward, restart it after deployment (use `run_in_background`):
```bash
pkill -f "port-forward.*svc/central" 2>/dev/null || true
sleep 2
oc -n <ns> port-forward svc/central 8000:443
```

### Post-deployment health check

```bash
oc -n <ns> get pods -l app=central
curl -sk -u "admin:<password>" "https://<endpoint>/v1/metadata" | jq .
```

If pods crash-loop, capture logs:
```bash
oc -n <ns> logs deployment/central --previous --tail=50
```
Do NOT automatically roll back. Report the crash — the caller may want to inspect it.

**DB migration version mismatch**: If central crashes with a message about migration
sequence numbers, the source code has newer migrations than the base image's database.
To fix: use a base image matching the source tree, or temporarily adjust
`CurrentDBVersionSeqNum` in `pkg/migrations/seq.go` to match the deployed DB.

## Phase 7: Test

Execute the test plan based on `$context`. Common patterns:

### Bug reproduction
- Follow the reproduction steps described in `$context`
- Use `curl` against the API, `oc` commands, or roxctl as appropriate
- Report whether the bug reproduces or not, with evidence

### Fix verification
When verifying changes to **existing logic** (bug fixes, behavior changes), produce
a before/after comparison proving the change works. Both sides must be tested on the
real cluster.

**Step 1 — Identify the fix:**
Figure out what constitutes "the fix" from context: `$context`, branch name,
git log, and the relationship between the current branch and the base branch.
Determine how to temporarily revert it (e.g., `git stash`, `git revert`).

**Step 2 — Test BEFORE (without the fix):**
- Temporarily remove the fix
- Rebuild and redeploy (Phase 4-6)
- Capture output to `$TMPDIR/verify-before.log`

**Step 3 — Test AFTER (with the fix):**
- Restore the fix
- Rebuild and redeploy
- Capture output to `$TMPDIR/verify-after.log`

**Step 4 — Render proof:**
If `freeze` is available:
```bash
freeze --language bash --window --margin 16 \
  --output "$TMPDIR/verify-before.png" < "$TMPDIR/verify-before.log"
freeze --language bash --window --margin 16 \
  --output "$TMPDIR/verify-after.png" < "$TMPDIR/verify-after.log"
```

For **new features**, skip the BEFORE step and only produce AFTER proof.

### API testing
```bash
curl -sk -u "admin:<password>" "https://<endpoint>/..." | jq .
```

### E2E test execution
```bash
go test -v -timeout 10m -count=1 -tags e2e ./<test-package> -run <test-name>
```

### Manual verification
- Execute steps from `$context` and report results

Capture ALL test output as evidence. Redact passwords with `***`.

## Phase 8: Report

Summarize results concisely:

1. **What was built**: List components and binary sizes
2. **Where it was pushed**: Image reference
3. **What was deployed**: Which deployments were patched
4. **Original image**: The image reference before patching (for restore)
5. **Test results**: Pass/fail with evidence
6. **Proof**: Paths to proof files
7. **Issues found**: Any problems encountered
