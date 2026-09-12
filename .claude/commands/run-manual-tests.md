---
name: run-manual-tests
description: Deploy StackRox (or use an existing deployment), run manual tests (Cypress UI tests with video, API/GraphQL calls, deployment upgrades/restarts), and capture all outputs. Supports interleaved UI and non-UI testing steps.
argument-hint: [cluster-name] [image-tag] [test-steps...]
---

You are an expert at deploying StackRox and running manual tests against it. This includes UI testing with Cypress (with video recording), REST API calls, GraphQL calls, deployment upgrades, restarts, and any other manual testing steps. Follow this workflow precisely, pausing for user approval at checkpoints.

Throughout this workflow, use the correct CLI tool based on the cluster flavor:
- **GKE (`gke-default`)**: use `kubectl`
- **OpenShift (`openshift-4`)**: use `oc`
- **Other flavors**: determine the appropriate CLI based on the flavor type; ask the user if unclear

All examples below use `<kube-cli>` as a placeholder. Substitute with `kubectl` or `oc` as appropriate.

## User Input
${1:User provided: $ARGUMENTS}
${1:else:No arguments provided - you will need to gather requirements interactively.}

---

## Phase 1: Environment Verification

### Step 1: Determine Cluster Name, Flavor, and Deployment State

Identify from the user's prompt:
- **Cluster name** and **flavor**
- **Whether StackRox is already deployed** — if the user says the cluster is already deployed, skip Phase 2 entirely but still perform KUBECONFIG verification below

If any of this is unclear, ask the user.

The cluster name is needed to set up the artifacts directory at `tmp/<cluster-name>/`.

### Step 2: KUBECONFIG Verification

Perform a **three-part verification** to ensure you are targeting the correct cluster:

**a) Verify `$KUBECONFIG` is set and points to the correct artifacts directory:**
```bash
echo "$KUBECONFIG"
# Must point to a kubeconfig file (e.g., tmp/<cluster-name>/kubeconfig)
```

**b) Verify the cluster name inside the kubeconfig matches the expected cluster:**
```bash
<kube-cli> config get-clusters
# The cluster name listed must match the expected infra cluster
```

**c) Verify connectivity to the correct cluster:**
```bash
<kube-cli> config current-context
<kube-cli> cluster-info
```

Show all three verification outputs to the user.

**CHECKPOINT: Do NOT proceed until the user confirms the context is pointing to the correct cluster.** This is critical to avoid deploying to or modifying someone else's cluster.

### Step 3: Set Up Artifacts Directory

Create a temporary directory for all test artifacts (videos, outputs, logs):
```bash
mkdir -p tmp/<cluster-name>/test-artifacts
```

All test outputs, Cypress videos, API call results, and logs will be stored here.

---

## Phase 2: Deploy StackRox

**Skip this entire phase if the user indicated StackRox is already deployed.** If skipping, still verify pods are running (Step 7) and set up port-forward (Step 8) if not already active.

### Step 4: Set Image Tag

Check if the user specified an **image tag**. If not, ask the user.

```bash
export MAIN_IMAGE_TAG=<image-tag>
```

### Step 5: Set Feature Flags and Environment Variables

Check if the user specified any feature flags or environment variables to enable. Set them before deploying.

### Step 6: Deploy

Run the deploy script. Use `./deploy/deploy.sh` by default. Only use `./deploy/deploy-local.sh` if the user explicitly requested a local deployment:
```bash
./deploy/deploy.sh
```

### Step 7: Verify Pod Readiness

Check if pods in the `stackrox` namespace are ready. If the user specified specific pods, check those. Otherwise check these common pods:
- `admission-control-*`
- `central-*`
- `central-db-*`
- `collector-*`
- `sensor-*`
- `scanner-v4-*`

```bash
<kube-cli> get pods -n stackrox
```

Poll until all expected pods show `Running` or `Completed`. Use `<kube-cli> wait` or check periodically.

### Step 8: Port-Forward and Get Password

**GKE (`gke-default`)**:
```bash
./deploy/k8s/central-deploy/central/scripts/port-forward.sh 8000
```
Admin password:
```bash
cat deploy/k8s/central-deploy/password
```

**OpenShift (`openshift-4`)**:
```bash
./deploy/openshift/central-deploy/central/scripts/port-forward.sh 8000
```
Admin password:
```bash
cat deploy/openshift/central-deploy/password
```

Store the admin password for use in subsequent steps. Central is now accessible at `https://localhost:8000`.

### Step 9: Obtain API Token

For Cypress tests, the `cypress.sh` script handles auth automatically — no manual token needed.

For direct API or GraphQL calls, obtain an API token:
```bash
API_TOKEN=$(curl -sk -u admin:<password> https://localhost:8000/v1/apitokens/generate \
  -d '{"name":"manual-test-token","roles":["Admin"]}' | jq -r '.token')
```

Store this token for use in API/GraphQL calls throughout the test session.

---

## Phase 3: Execute Test Steps

The user may request any combination of the following test actions, potentially interleaved. Execute them in the order specified by the user.

### UI Testing (Cypress)

#### Running Existing Cypress Tests

For existing tests in `ui/apps/platform/cypress/integration/`:
```bash
cd ui/apps/platform
NODE_TLS_REJECT_UNAUTHORIZED=0 \
  UI_BASE_URL=https://localhost:8000 \
  TEST_RESULTS_OUTPUT_DIR=../../../tmp/<cluster-name>/test-artifacts \
  ROX_USERNAME=admin \
  ROX_ADMIN_PASSWORD=$(cat ../../../deploy/k8s/central-deploy/password) \
  npm run cypress-spec -- "<spec-filename>.test.js"
```

The `cypress.sh` script auto-prefixes `cypress/integration/` to the spec path, handles auth token generation, feature flags, and captures videos/screenshots to `TEST_RESULTS_OUTPUT_DIR`.

#### Writing New Cypress Tests

If the user describes UI steps to test, write the spec to `ui/apps/platform/cypress/integration/`. This is required because Cypress resolves the support file chain and helper imports using relative paths from the spec location — specs placed outside the project (e.g., in `tmp/`) will fail with module resolution errors.

**File naming convention:** Include the cluster name in the filename so the user can identify and clean up test files later:
- `<cluster-name>-<descriptive-name>.test.js`
- Example: `cs-03-25-1-deferAndApproveCves.test.js`

When writing Cypress tests:
- **Tests must navigate the UI** using `cy.visit()`, `cy.get()`, `cy.click()`, etc. Do NOT use `cy.request()` for the main test flow — Cypress is for testing the UI end-to-end. If the user wants API-only testing, use `curl` directly instead
- Use the existing project conventions (check `ui/apps/platform/cypress.config.js` for config)
- The spec **must** use `.test.js` or `.test.ts` extension to match existing conventions in `cypress/integration/`
- **Reuse existing helpers** from `cypress/helpers/` and `cypress/integration/vulnerabilities/` — check for helpers that already implement the desired flow before writing custom code. Key helpers:
  - `cypress/helpers/basicAuth.js` — `withAuth()` sets up auth via localStorage for each test
  - `cypress/helpers/visit.js` — `visit()` navigates with route interception
  - `cypress/helpers/request.js` — `interactAndWaitForResponses()` for intercepting API calls during UI interactions
  - `cypress/integration/vulnerabilities/workloadCves/WorkloadCves.helpers.js` — CVE page navigation, selection, deferral form
  - `cypress/integration/vulnerabilities/exceptionManagement/ExceptionManagement.helpers.ts` — deferral/approval/denial flows
- **Import paths:** Since specs are at the root of `cypress/integration/`, use `../helpers/` for helpers and `./vulnerabilities/` for vulnerability helpers (NOT `../../helpers/` which is for specs in subdirectories)
- Video recording is enabled by default (`video: true` in cypress.config.js)

Run the spec using the project's `cypress.sh` script via `npm run cypress-spec`. This script automatically handles auth token generation, feature flag export, specPattern configuration, and base URL setup:
```bash
cd ui/apps/platform
NODE_TLS_REJECT_UNAUTHORIZED=0 \
  UI_BASE_URL=https://localhost:8000 \
  TEST_RESULTS_OUTPUT_DIR=../../../tmp/<cluster-name>/test-artifacts \
  ROX_USERNAME=admin \
  ROX_ADMIN_PASSWORD=$(cat ../../../deploy/k8s/central-deploy/password) \
  npm run cypress-spec -- "<cluster-name>-<test-name>.test.js"
```

The script auto-prefixes `cypress/integration/` to the spec path, so pass just the filename. Set `UI_BASE_URL` if Central is not on the default port. Set `TEST_RESULTS_OUTPUT_DIR` to capture videos and screenshots in the cluster's artifacts directory. `NODE_TLS_REJECT_UNAUTHORIZED=0` is needed for self-signed certs. `ROX_USERNAME` and `ROX_ADMIN_PASSWORD` are used by the script to generate auth tokens and fetch feature flags.

#### Opening Interactive Cypress Runner

If the user wants to watch tests live:
```bash
cd ui/apps/platform
npm run cypress-open
```

### API Calls (REST)

Execute REST API calls against Central and capture output:
```bash
curl -sk -H "Authorization: Bearer $API_TOKEN" \
  https://localhost:8000/v1/<endpoint> \
  | tee tmp/<cluster-name>/test-artifacts/api-<endpoint-name>-$(date +%s).json
```

Show the output and a brief summary to the user.

### GraphQL Calls

Execute GraphQL queries against Central and capture output:
```bash
curl -sk -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"<graphql-query>"}' \
  https://localhost:8000/api/graphql \
  | tee tmp/<cluster-name>/test-artifacts/graphql-<query-name>-$(date +%s).json
```

Show the output and a brief summary to the user.

### Deployment Upgrades

When the user wants to upgrade a deployment to a different image:
1. Confirm the deployment name and new image with the user
2. Perform the upgrade:
   ```bash
   <kube-cli> -n stackrox set image deployment/<deployment-name> <container-name>=<new-image>
   ```
3. Wait for the rollout to complete:
   ```bash
   <kube-cli> -n stackrox rollout status deployment/<deployment-name> --timeout=300s
   ```
4. Verify pod readiness (same as Step 7)
5. **If the upgraded deployment is `central`**: the port-forward will be broken. Re-establish it (same as Step 8). Wait for the new central pod to be ready before port-forwarding.
6. Log the upgrade to artifacts:
   ```bash
   echo "$(date): Upgraded <deployment-name> to <new-image>" >> tmp/<cluster-name>/test-artifacts/upgrade-log.txt
   ```

### Deployment Restarts

When the user wants to restart a deployment:
1. Restart the deployment:
   ```bash
   <kube-cli> -n stackrox rollout restart deployment/<deployment-name>
   ```
2. Wait for the rollout to complete:
   ```bash
   <kube-cli> -n stackrox rollout status deployment/<deployment-name> --timeout=300s
   ```
3. Verify pod readiness (same as Step 7)
4. **If the restarted deployment is `central`**: the port-forward will be broken. Re-establish it (same as Step 8). Wait for the new central pod to be ready before port-forwarding.
5. Log the restart to artifacts:
   ```bash
   echo "$(date): Restarted <deployment-name>" >> tmp/<cluster-name>/test-artifacts/restart-log.txt
   ```

### Environment Variable, Secret, or ConfigMap Changes

When the user modifies environment variables, secrets, or declarative configs on a deployment:
1. Apply the requested change
2. If the affected deployment is `central`:
   - The central pod will likely restart automatically
   - Wait for the new pod to be ready
   - **Re-establish the port-forward** (same as Step 8) since the old pod is gone
   - Verify Central is accessible before proceeding
3. For other deployments, wait for rollout and verify pod readiness

**Rule: Any time the `central` deployment is restarted, upgraded, or has its config/env/secrets changed, always check if port-forward needs to be re-established and do so.**

### Pod Readiness Checks

After any upgrade, restart, or when requested by the user, check pod readiness:
```bash
<kube-cli> get pods -n stackrox
```

Wait for all expected pods to be `Running` or `Completed` before proceeding to the next test step. If specific pods are failing, show the logs to the user:
```bash
<kube-cli> logs -n stackrox <pod-name> --tail=50
```

### Other Commands

The user may request arbitrary `<kube-cli>` commands, shell commands, or other operations. Execute them and capture output to the artifacts directory:
```bash
<command> | tee tmp/<cluster-name>/test-artifacts/cmd-$(date +%s).txt
```

Always show the output and a brief summary to the user.

---

## Phase 4: Summary

After all test steps are completed (or when the user indicates they are done):

1. List all artifacts captured:
   ```bash
   find tmp/<cluster-name>/test-artifacts/ -type f
   ```
2. Provide a summary of:
   - Test steps executed
   - Cypress test results (pass/fail) and video locations
   - API/GraphQL call results
   - Any upgrades or restarts performed
   - Any failures or issues encountered
3. Remind the user where artifacts are stored: `tmp/<cluster-name>/test-artifacts/`

## Important Notes

- **Never delete** anything in the `stackrox` namespace or the infra cluster without explicit user instructions
- **Always ask for approval** before destructive actions (upgrades, restarts, config changes)
- **Capture everything** — all outputs go to `tmp/<cluster-name>/test-artifacts/`
- **Show outputs** — always display command output and a brief summary to the user
- **Wait for readiness** — after any deployment change, wait for pods to be ready before proceeding
- **Port-forward after central changes** — any time central is restarted, upgraded, or has config/env/secret changes, re-establish the port-forward
- When in doubt, ask the user