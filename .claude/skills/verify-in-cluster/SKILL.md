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
command -v crane   && echo "OK: crane"   || echo "MISSING: crane"
command -v docker  && echo "OK: docker"  || echo "MISSING: docker"
command -v oc      && echo "OK: oc"      || echo "MISSING: oc"
command -v kubectl && echo "OK: kubectl" || echo "MISSING: kubectl"
command -v curl    && echo "OK: curl"    || echo "MISSING: curl"
command -v jq      && echo "OK: jq"      || echo "MISSING: jq"
command -v freeze  && echo "OK: freeze"  || echo "MISSING: freeze (optional)"
```

**Required**: `go`, `curl`, `jq` — stop with install instructions if any are missing.

**Cluster access**: need at least one of `oc` or `kubectl`. If both are available, prefer
`oc`. Stop if neither is available. Use the chosen command literally in every subsequent
Bash call — do not use shell variables across calls (Claude Code does not share shell state).

**Visual proof**: `freeze` (charmbracelet/freeze) is optional. If available, use it to
render command outputs as PNG images for PR attachment. Install: `brew install charmbracelet/tap/freeze`.

**Image push**: need at least one of `crane` or `docker`. Stop if neither is available.

**Registry selection** (quay.io preferred, ttl.sh fallback):

If `crane` is available, check for quay.io push credentials first:
```bash
crane auth get quay.io 2>&1 | jq -r '{authenticated: (has("Username") and has("Secret")), username: .Username}'
```
This prints only the username and whether auth is configured — never the secret/token.
If `authenticated` is `true`, quay.io is ready. Determine the push repository using the
convention `quay.io/<user>/stackrox/main` where `<user>` is the authenticated username
from the crane output above. Verify the repo exists **and is public** (the cluster must
pull without credentials):
```bash
curl -s "https://quay.io/api/v1/repository/<user>/stackrox/main" | jq '{is_public, name}'
```
If `is_public` is `true`, quay.io is ready. If the repo doesn't exist (`null` / 404) or
`is_public` is `false`, fall back to ttl.sh — do NOT attempt to create or change visibility
of repos via the Quay API (it requires OAuth tokens, not Docker credentials).

If quay.io is not available, test connectivity to ttl.sh:
```bash
crane manifest ttl.sh/test:1h 2>&1 || true
```
If it fails with TLS/x509 errors (e.g., `x509: OSStatus -26276`), this is caused by
Claude Code's sandbox network proxy. Retry with `--insecure`:
```bash
crane manifest --insecure ttl.sh/test:1h 2>&1 || true
```
If `--insecure` works, use `--insecure` on all subsequent `crane` commands throughout.
If crane cannot connect even with `--insecure`, fall back to `docker`.

Remember which registry was selected (quay.io or ttl.sh) — you'll use it in Phase 5.
When using quay.io, tags don't need a TTL suffix. When using ttl.sh, append `:2h`.

**Note on sandbox**: Both `crane` and Docker commands may require the user to approve
sandbox override prompts or may need `--insecure` flags due to the sandbox proxy.

## Phase 1: Discover Cluster

Find a usable Kubernetes/OpenShift cluster. Try in order:

1. `KUBECONFIG` environment variable — check with `echo $KUBECONFIG` in a Bash call
2. Argument passed to this skill (if `$context` contains a path to a kubeconfig)
3. Uploaded artifacts — if running in an environment like Ambient where files may be
   uploaded, check for kubeconfig files in the workspace or uploads directory
4. Default kubectl context (`kubectl config current-context`)

**KUBECONFIG persistence**: Like all env vars, `KUBECONFIG` does not persist between
Bash tool calls. If `KUBECONFIG` is set to a non-default path, you must prepend
`export KUBECONFIG=<path> &&` to every `oc`/`kubectl` command, or hardcode the
`--kubeconfig=<path>` flag.

Verify connectivity (`oc` if available, otherwise `kubectl` — use the chosen
command literally in every Bash call throughout):
```bash
oc cluster-info        # or: kubectl cluster-info
```

Detect the cluster's architecture for cross-compilation later:
```bash
oc get nodes -o jsonpath='{.items[0].status.nodeInfo.architecture}'
```
Remember the result (e.g., `amd64` or `arm64`) — you'll use it as the literal
`GOARCH` value in Phase 4 and `--platform linux/<arch>` in Phase 5.
If the result is `arm64`, use `GOARCH=arm64` instead of `GOARCH=amd64` in Phase 4.
Default to `amd64` if detection fails.

Print the cluster name/context and API server URL. Unless in YOLO mode, ask the user:
> "I found cluster [name] at [url]. Is it OK to modify deployments in the StackRox namespace?"

## Phase 2: Find StackRox and Authenticate

### 2a: Check if StackRox is deployed

Check the `stackrox` namespace first (used by deploy scripts), then `rhacs-operator`
(used by the operator):
```bash
oc -n stackrox get deployment central --no-headers 2>/dev/null
```
If not found, try `rhacs-operator`:
```bash
oc -n rhacs-operator get deployment central --no-headers 2>/dev/null
```
Use whichever namespace has Central throughout. If neither has it, go to
**Phase 2c: Deploy StackRox**.

### 2b: Authenticate to Central

Determine the Central endpoint. Try in order:

1. **OpenShift route** (preferred — survives pod restarts):
   ```bash
   oc -n <ns> get route central -o jsonpath='{.status.ingress[0].host}' 2>/dev/null
   ```
   If found, the endpoint is `<route-host>:443`.

2. **LoadBalancer service**:
   ```bash
   oc -n <ns> get svc central-loadbalancer -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null
   ```
   If found, the endpoint is `<lb-ip>:443`.

3. **Port-forward** (fallback — use `run_in_background` for the Bash tool call):
   ```bash
   oc -n <ns> port-forward svc/central 8000:443
   ```
   Set `API_ENDPOINT="localhost:8000"`
   Note: port-forward breaks on pod restart. Prefer routes/LB when available.
   If port 8000 is already in use (common in Ambient/cloud workspaces), use a different
   local port, e.g., `18443:443`, and set `API_ENDPOINT="localhost:18443"`.

Now authenticate. Try credentials in this order:

1. **`ROX_ADMIN_PASSWORD` env var** — if already set, use it
2. **Well-known password files** from previous `deploy/deploy.sh` runs:
   - `deploy/k8s/central-deploy/password`
   - `deploy/openshift/central-deploy/password`
   Read whichever exists and use that value.
3. **`admin` / `admin`** — last resort, only works when `ROX_HTPASSWD_AUTH=true` was set
   explicitly during deployment

Test credentials:
```bash
curl -sk -u "admin:<password>" "https://<endpoint>/v1/ping"
```

If none work, ask the user for credentials. If in YOLO mode and no credentials work,
fail with a clear error — do not silently skip auth.

When credentials are successfully discovered, remember them:
- If the password came from a non-obvious source (not env var, not well-known file),
  save a memory noting how auth works for this cluster so future sessions don't re-discover.

Remember the password and endpoint values you discovered — substitute them literally
into every subsequent command. Shell exports do not persist across Bash tool calls.

### 2c: Deploy StackRox (only if not found)

Use the repo's deployment scripts.

**Prerequisites for `deploy/deploy.sh`:**
- `oc` or `kubectl` — the script uses `oc` by default. If only `kubectl` is available,
  symlink it: `ln -s "$(which kubectl)" /usr/local/bin/oc`
- `roxctl` must be in PATH (the deploy script calls `roxctl central generate` internally).
  If missing, build it with **version ldflags** — roxctl panics on startup without them.
  **Critical**: build for the **host platform** (not `GOOS=linux`) since roxctl runs locally
  to generate deployment configs. Also, `ScannerVersion` is **required** — the embedded Helm
  chart templates use `required ""  .ScannerImageTag` which is populated from this value.
  Without it, `roxctl central generate` fails with a template error.

  **Important**: Use the **Quay `MAIN_IMAGE_TAG`** (fetched below) as `MainVersion`, NOT
  `make tag`. The `MainVersion` baked into roxctl determines the image tag for ALL components
  (central, sensor, scanner-v4, scanner-v4-db, etc.) via `roxctl central generate`. If you
  use a local `make tag` (which produces a `-dirty` suffix), the deploy will try to pull
  images that don't exist on Quay. Fetch the Quay tag first (see "Finding a base image"
  below), then build roxctl:
  ```bash
  # MAIN_IMAGE_TAG must be set first — see "Finding a base image from CI" below
  SCANNER_VERSION=$(cat SCANNER_VERSION)
  COLLECTOR_VERSION=$(cat COLLECTOR_VERSION)
  CGO_ENABLED=0 go build \
    -ldflags="-s -w \
      -X github.com/stackrox/rox/pkg/version/internal.MainVersion=$MAIN_IMAGE_TAG \
      -X github.com/stackrox/rox/pkg/version/internal.ScannerVersion=$SCANNER_VERSION \
      -X github.com/stackrox/rox/pkg/version/internal.CollectorVersion=$COLLECTOR_VERSION" \
    -o "$TMPDIR/roxctl" ./roxctl
  export PATH="$TMPDIR:$PATH"
  ```
  The `MainVersion` and `ScannerVersion` ldflags are **required** — `MainVersion` prevents
  a hard-panic, and `ScannerVersion` prevents the Helm chart template `required` error.
  `CollectorVersion` is optional but recommended.
- `helm` — needed for the monitoring stack sub-step. If missing, the deploy script may
  fail at that step but Central/Scanner YAML generation usually succeeds. You can set
  `MONITORING_SUPPORT=false` to skip it.
- The main image must be available at the specified registry/tag.

**Finding a base image from CI (Quay):**
Local `make tag` produces a tag that only exists if you've pushed to your own registry.
For from-scratch deployments, use CI-built images from `quay.io/stackrox-io/`. Fetch the
latest tag whose embedded commit is on `origin/master`:

```bash
git fetch origin master --quiet 2>/dev/null || true

# Fetch recent multi-arch tags from Quay (exclude arch-specific suffixes)
TAGS=$(curl -s "https://quay.io/api/v1/repository/stackrox-io/main/tag/?limit=100&onlyActiveTags=true" \
  | jq -r '.tags[].name | select(test("^[0-9]+[.][0-9]+[.]x-")) | select(test("-(arm64|amd64|s390x|ppc64le)$") | not)' )

# Find the first tag whose commit is an ancestor of origin/master
for tag in $TAGS; do
  hash="${tag##*-g}"
  if git merge-base --is-ancestor "$hash" origin/master 2>/dev/null; then
    echo "Found master-based tag: $tag"
    MAIN_IMAGE_TAG="$tag"
    break
  fi
done
```

If no master-based tag is found, fall back to the most recent tag:
```bash
MAIN_IMAGE_TAG=$(curl -s "https://quay.io/api/v1/repository/stackrox-io/main/tag/?limit=5&onlyActiveTags=true" \
  | jq -r '.tags[].name | select(test("^[0-9]+[.][0-9]+[.]x-")) | select(test("-(arm64|amd64|s390x|ppc64le)$") | not)' \
  | head -1)
```

Also fetch tags for the supporting images (these are built separately and have their own
version cadence):
```bash
# Scanner V4 (separate build pipeline)
SCANNERV4_TAG=$(curl -s "https://quay.io/api/v1/repository/stackrox-io/scanner-v4/tag/?limit=5&onlyActiveTags=true" \
  | jq -r '.tags[].name | select(test("^[0-9]+[.][0-9]+[.]x-")) | select(test("-(arm64|amd64|s390x|ppc64le)$") | not)' \
  | head -1)

# Central DB
DBS_TAG=$(curl -s "https://quay.io/api/v1/repository/stackrox-io/central-db/tag/?limit=5&onlyActiveTags=true" \
  | jq -r '.tags[].name | select(test("^[0-9]+[.][0-9]+[.]x-")) | select(test("-(arm64|amd64|s390x|ppc64le)$") | not)' \
  | head -1)

# Collector
COLLECTOR_TAG=$(curl -s "https://quay.io/api/v1/repository/stackrox-io/collector/tag/?limit=5&onlyActiveTags=true" \
  | jq -r '.tags[].name | select(test("^[0-9]+[.][0-9]+[.]x-")) | select(test("-(arm64|amd64|s390x|ppc64le)$") | not)' \
  | head -1)
```

The key env vars to set:

```bash
export MAIN_IMAGE_TAG="<tag-from-above>"    # CI tag from Quay, NOT make tag
export MAIN_IMAGE_REPO=quay.io/stackrox-io/main
export CENTRAL_DB_IMAGE_REPO=quay.io/stackrox-io/central-db
export SCANNERV4_IMAGE_REPO=quay.io/stackrox-io/scanner-v4
export SCANNERV4_DB_IMAGE_REPO=quay.io/stackrox-io/scanner-v4-db
export COLLECTOR_IMAGE_REPO=quay.io/stackrox-io/collector
export SCANNER_IMAGE_REPO=quay.io/stackrox-io/scanner
export SCANNER_DB_IMAGE_REPO=quay.io/stackrox-io/scanner-db
export DBS_TAG="<central-db-tag>"
export SCANNERV4_TAG="<scanner-v4-tag>"
export COLLECTOR_TAG="<collector-tag>"
export ROX_HTPASSWD_AUTH=true
export STORAGE=pvc
export LOAD_BALANCER=route   # on OpenShift; omit on plain k8s
export MONITORING_SUPPORT=false  # skip if helm is unavailable
```

Then run:
```bash
./deploy/deploy.sh
```

**Post-deploy image fix for scanner-v4-db**: The deploy script uses `MAIN_IMAGE_TAG` for
scanner-v4 and scanner-v4-db images. Since scanner-v4 is built on a separate pipeline with
its own tags, the `MAIN_IMAGE_TAG` may not exist in the scanner-v4 repos. If scanner-v4-db
pods show `ImagePullBackOff`, patch them with the correct scanner-v4 tag:
```bash
# Use the SCANNERV4_TAG fetched earlier from quay.io/stackrox-io/scanner-v4
oc -n stackrox set image deployment/scanner-v4-db \
  db="quay.io/stackrox-io/scanner-v4-db:$SCANNERV4_TAG" \
  init-db="quay.io/stackrox-io/scanner-v4-db:$SCANNERV4_TAG"
oc -n stackrox set image deployment/scanner-v4-indexer \
  indexer="quay.io/stackrox-io/scanner-v4:$SCANNERV4_TAG"
oc -n stackrox set image deployment/scanner-v4-matcher \
  matcher="quay.io/stackrox-io/scanner-v4:$SCANNERV4_TAG"
```
Similarly, if central-db has a pull error, patch it:
```bash
oc -n stackrox set image deployment/central-db \
  central-db="quay.io/stackrox-io/central-db:$DBS_TAG"
```

The deploy script will generate credentials and store them in
`deploy/k8s/central-deploy/password` (or the openshift equivalent).
Read the password from there after deployment completes.

Wait for Central to become ready before continuing.

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

The `-ldflags="-s -w"` strips debug info and reduces binary size. For version info in
`/v1/metadata`, add `-X` flags from `scripts/go-build.sh` if available, but this is optional
for central/migrator/sensor — they run fine without version ldflags.

**Exception: roxctl** — if you need to build roxctl (e.g., for Phase 2c deployment), you
**must** include at least `-X github.com/stackrox/rox/pkg/version/internal.MainVersion=<tag>`.
Without it, roxctl hard-panics on startup. See Phase 2c for the full build command.

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

First, record the current image for **each** deployment being modified. Different deployments
may use different image references — always pull the base from the specific deployment:

```bash
oc -n stackrox get deployment/central -o jsonpath='{.spec.template.spec.containers[0].image}'
```
Record this value — you'll use it as the base image for `crane mutate`.

**Choosing the right base image:** The base image you append layers to MUST have a DB
migration sequence compatible with your source tree. If the deployed image is from a CI
nightly and your branch has newer migrations, central will crash on startup (see Phase 6
"DB migration version mismatch"). To avoid this, use a base image from the same commit
range as your source. If Phase 2c deployed StackRox using a master-based Quay tag, the
deployed image is already correct. If you're patching a pre-existing deployment whose
image version differs significantly from your branch, consider redeploying with a
compatible base first.

For scanner (uses a separate image):
```bash
oc -n stackrox get deployment/scanner -o jsonpath='{.spec.template.spec.containers[0].image}'
```

Generate a unique tag using the registry selected in Phase 0:

- **quay.io**: `quay.io/<user>/stackrox/main:verify-<short-uuid>`
- **ttl.sh**: `ttl.sh/<uuid>:2h`

```bash
UUID=$(uuidgen | tr '[:upper:]' '[:lower:]' | cut -c1-8)
```

### Method A: crane (preferred)

Create tar layers with binaries at their correct **absolute** container paths.
The tar MUST use absolute paths (e.g., the file entry must be `stackrox/central`,
which maps to `/stackrox/central` in the container):

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

Push by appending layers to the current image. Use `--platform linux/<arch>` matching
the cluster architecture detected in Phase 1 (e.g., `linux/amd64` or `linux/arm64`):
```bash
crane mutate "<current-central-image>" \
  --platform linux/<arch> \
  --append central-layer.tar \
  --append migrator-layer.tar \
  --tag "<tag>"
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

The `--tag` flag pushes directly to the registry. No separate `crane push` needed.

If the push to quay.io fails with `UNAUTHORIZED` or `DENIED`, fall back to ttl.sh
and retry with a `ttl.sh/<uuid>:2h` tag.

### Method B: docker (fallback if crane fails)

Use this ONLY if crane failed in Phase 0 (TLS/proxy issues). Create a Dockerfile:

```dockerfile
FROM <current-central-image>
COPY central /stackrox/central
COPY migrator /stackrox/bin/migrator
```

Build with explicit `--platform` matching the cluster architecture:

```bash
docker buildx build --platform linux/<arch> --push -t "<tag>" -f Dockerfile .
```

Print the image reference for the user.

## Phase 6: Deploy

Patch the deployment to use the new image. Use the concrete command (`oc` or `kubectl`)
you determined in Phase 0 and the namespace discovered in Phase 2a — do not use shell
variables across Bash calls.

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

If using port-forward (not route/LB), restart it after deployment completes:
```bash
pkill -f "port-forward.*svc/central" 2>/dev/null || true
sleep 2
```
Then start a new port-forward using `run_in_background` for the Bash tool call:
```bash
oc -n <ns> port-forward svc/central 8000:443
```

### Post-deployment health check

```bash
# Verify pod is running
oc -n <ns> get pods -l app=central

# Verify API is responding and check version/build info
curl -sk -u "admin:<password>" "https://<endpoint>/v1/metadata" | jq .
```

Substitute the actual password and endpoint values you discovered in Phase 2.

If pods crash-loop after deployment, capture logs and report them:
```bash
oc -n <ns> logs deployment/central --previous --tail=50
```
Do NOT automatically roll back. Report the crash to the parent context — it may want to
inspect the failure, fix the code, and re-run this skill.

**DB migration version mismatch**: If central crashes with a message about database version
or migration sequence numbers (e.g., "current DB version seq X but expected Y"), this means
the source code has newer migrations than the base nightly image's database. The appended
central binary expects a higher migration sequence than what the nightly's DB was initialized
with. To fix: use a base image whose version matches the source tree, or temporarily adjust
the `CurrentDBVersionSeqNum` constant in `pkg/migrations/seq.go` to match the deployed DB.

## Phase 7: Test

Execute the test plan based on `$context`. Common patterns:

### Bug reproduction
- Follow the reproduction steps described in `$context`
- Use `curl` against the API, `oc` commands, or roxctl as appropriate
- Report whether the bug reproduces or not, with evidence (command output, API responses)

### Fix verification
When verifying changes to **existing logic** (bug fixes, behavior changes), produce
a before/after comparison proving the change works. Both sides must be tested on the
real cluster — not made up.

**Step 1 — Identify the fix:**
Figure out what constitutes "the fix" from context: `$context`, branch name,
git log, and the relationship between the current branch and the base branch.
The fix could be uncommitted changes, a single commit, multiple commits on a
feature branch, etc. Determine how to temporarily revert it (e.g., `git stash`,
`git revert`, checking out the base branch). If you cannot determine this
confidently, ask the user.

**Step 2 — Test BEFORE (without the fix):**
- Temporarily remove the fix using the approach from Step 1
- Rebuild the affected component(s) and redeploy (Phase 4-6)
- Run the verification steps and capture output
- Write results to `$TMPDIR/verify-before.log` — include only the verification
  commands and their output, with a header like:
  ```
  # ── BEFORE: <title> (base branch / without fix) ──────
  ```

**Step 3 — Test AFTER (with the fix):**
- Restore the fix (e.g., `git stash pop`, `git checkout <branch>`)
- Rebuild and redeploy the fixed version (Phase 4-6)
- Run the same verification steps and capture output
- Write results to `$TMPDIR/verify-after.log` with a header like:
  ```
  # ── AFTER: <title> (with fix applied) ────────────────
  ```

**Step 4 — Render proof:**
If `freeze` is available, render both logs as images:
```bash
freeze --language bash --window --margin 16 \
  --output "$TMPDIR/verify-before.png" < "$TMPDIR/verify-before.log"
freeze --language bash --window --margin 16 \
  --output "$TMPDIR/verify-after.png" < "$TMPDIR/verify-after.log"
```

Report paths to both images so the caller can attach them side by side to the PR.

For **new features** (no previous behavior to compare against), skip the BEFORE step
and only produce the AFTER proof.

### API testing
- Use curl with the authenticated endpoint:
  ```bash
  curl -sk -u "admin:<password>" "https://<endpoint>/..." | jq .
  ```

### E2E test execution
- If `$context` names a specific Go test:
  ```bash
  go test -v -timeout 10m -count=1 -tags e2e ./<test-package> -run <test-name>
  ```
  Set `ROX_ENDPOINT` and `ROX_API_TOKEN` env vars as needed. To generate an API token:
  ```bash
  curl -sk -u "admin:<password>" "https://<endpoint>/v1/apitokens/generate" \
    -d '{"name":"e2e-verify","roles":["Admin"]}' | jq -r .token
  ```

### Manual verification
- If `$context` describes manual steps, execute them and report results

Capture ALL test output as evidence.

**Proof logs**: For each test pattern, accumulate proof in log files under `$TMPDIR/`.
Include only verification commands and their output — not build, deploy, or cluster
discovery steps. Use bash comment headers and strip verbose noise. Redact passwords
with `***`. The fix verification pattern above produces `verify-before.log` and
`verify-after.log`; other patterns produce a single `verify-session.log`.

If `freeze` is available, render each log as a PNG image at the end of Phase 7.
When freeze is not available, the plain-text logs are the proof.

## Phase 8: Report

Summarize results concisely:

1. **What was built**: List components and binary sizes
2. **Where it was pushed**: Image reference (quay.io or ttl.sh URL)
3. **What was deployed**: Which deployments were patched
4. **Original image**: The image reference before patching (so the user can restore with
   `oc -n <ns> set image deployment/<name> <container>=<original-image>`)
5. **Test results**: Pass/fail with evidence (command output, API responses)
6. **Proof**: Paths to proof files — for fix verification: `verify-before.{log,png}`
   and `verify-after.{log,png}`; for other patterns: `verify-session.{log,png}`
7. **Issues found**: Any problems encountered during the process

This summary should be suitable for pasting into a PR description as proof of verification.
Mention the proof file paths so the caller can attach them to the PR.
