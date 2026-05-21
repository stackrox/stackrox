---
name: verify
description: >
  Build, deploy, and verify StackRox code changes against a live Kubernetes/OpenShift cluster.
  Handles cluster discovery, StackRox authentication, component building with crane, image pushing
  to ttl.sh, deployment patching, and test execution.
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

Verify these tools are available. Stop early with clear install instructions if any are missing.

```bash
command -v go      || echo "MISSING: go — install from https://go.dev/dl/"
command -v crane   || echo "MISSING: crane — run: go install github.com/google/go-containerregistry/cmd/crane@latest"
command -v kubectl || echo "MISSING: kubectl"
command -v curl    || echo "MISSING: curl"
command -v jq      || echo "MISSING: jq"
```

Also check `oc` availability — if present, prefer it over `kubectl` throughout.
Remember which command to use (`oc` or `kubectl`) and use it in all subsequent Bash calls.
Do NOT use shell variables like `oc` across separate Bash tool invocations — Claude Code
does not share shell state between calls. Instead, hardcode the chosen command in each call.

Test that `crane` can reach ttl.sh:
```bash
crane manifest ttl.sh/test:1h 2>&1 || true
```
This should return a valid JSON manifest. If it fails with TLS/x509 errors (e.g.,
`x509: OSStatus -26276`), this is caused by Claude Code's sandbox network proxy —
Go's `crypto/tls` cannot validate certificates through it. Retry with `--insecure`:
```bash
crane manifest --insecure ttl.sh/test:1h 2>&1 || true
```
If `--insecure` works, use `--insecure` on all subsequent `crane` commands throughout.

If crane cannot connect even with `--insecure`, check whether `docker` is available as a
fallback. If neither works, stop and inform the user.

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

Use the repo's deployment scripts. Prerequisites:
- `roxctl` must be in PATH (the deploy script calls `roxctl central generate` internally).
  If missing, build it: `go build -o "$TMPDIR/roxctl" ./roxctl && export PATH="$TMPDIR:$PATH"`
- The main image must be available at the specified registry/tag.

The key env vars to set:

```bash
export MAIN_IMAGE_TAG=$(make --quiet --no-print-directory tag)
export ROX_HTPASSWD_AUTH=true
export STORAGE=pvc
export LOAD_BALANCER=route   # on OpenShift; omit on plain k8s
```

Then run:
```bash
./deploy/deploy.sh
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
  `/usr/local/bin/scanner`, push to ttl.sh, and patch deployment/scanner.
- **Operator** changes are out of scope for this skill. Inform the user.

If no code changes are detected, inform the user and ask what they want to do.

## Phase 4: Build

Cross-compile each affected binary for the target architecture detected in Phase 1
(default `amd64`). Use the literal architecture value — do not rely on env vars:

```bash
GOOS=linux GOARCH=<arch> CGO_ENABLED=0 go build -ldflags="-s -w" -o "$TMPDIR/<binary-name>" ./<package-path>
```

The `-ldflags="-s -w"` strips debug info and reduces binary size. For version info in
`/v1/metadata`, add `-X` flags from `scripts/status.sh` if available, but this is optional.

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

## Phase 5: Push Image to ttl.sh

First, record the current image for **each** deployment being modified. Different deployments
may use different image references — always pull the base from the specific deployment:

```bash
oc -n stackrox get deployment/central -o jsonpath='{.spec.template.spec.containers[0].image}'
```
Record this value — you'll use it as the base image for `crane mutate`.

For scanner (uses a separate image):
```bash
oc -n stackrox get deployment/scanner -o jsonpath='{.spec.template.spec.containers[0].image}'
```

Generate a unique tag:
```bash
uuidgen | tr '[:upper:]' '[:lower:]'
```
Prepend `ttl.sh/` and append `:2h` to form the full tag (e.g., `ttl.sh/<uuid>:2h`).

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

The `--tag` flag pushes directly to ttl.sh. No separate `crane push` needed.

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

## Phase 7: Test

Execute the test plan based on `$context`. Common patterns:

### Bug reproduction
- Follow the reproduction steps described in `$context`
- Use `curl` against the API, `oc` commands, or roxctl as appropriate
- Report whether the bug reproduces or not, with evidence (command output, API responses)

### Fix verification
- First, check if the fix is already deployed (it should be from Phase 6)
- Follow the reproduction steps — the bug should NOT reproduce
- Report pass/fail with evidence

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

Capture ALL test output as evidence. Screenshots are not available in CLI, so
capture command output, API responses, and log snippets.

## Phase 8: Report

Summarize results concisely:

1. **What was built**: List components and binary sizes
2. **Where it was pushed**: Image reference (ttl.sh URL)
3. **What was deployed**: Which deployments were patched
4. **Original image**: The image reference before patching (so the user can restore with
   `oc -n <ns> set image deployment/<name> <container>=<original-image>`)
5. **Test results**: Pass/fail with evidence (command output, API responses)
6. **Issues found**: Any problems encountered during the process

This summary should be suitable for pasting into a PR description as proof of verification.
