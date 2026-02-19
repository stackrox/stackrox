## Scale Testing with Fake Workloads

This guide covers two approaches for scale testing StackRox using fake workload simulators:

1. **Full stack deployment** (`scale/dev/` scripts) - Deploy Central + Sensor with fake workloads from scratch
2. **Add workload to existing deployment** (`launch_workload.sh`) - Patch an already-running sensor with fake workloads

Both approaches use the same workload definition files in `scale/workloads/`.

---

## Full Stack Scale Testing (Recommended for Migration Testing)

### Prerequisites

**Cluster Requirements:**
- Machine type: e2-standard-32 (32 vCPUs, 128GB RAM per node)
- Node count: 3 nodes
- Total: 96 vCPUs, 384GB RAM
- Create with: `./scale/dev/cluster.sh <cluster-name>`

**Why these specs?** The `scale/dev/launch_central.sh` script patches Central to request 8 CPU + 16Gi memory, and Central-DB to request 16 CPU + 32Gi memory (24 CPU total). Smaller instance types will cause scheduling failures.

**Environment Variables:**
```bash
export REGISTRY_USERNAME="your-quay-username"
export REGISTRY_PASSWORD="your-quay-password"
export USE_LOCAL_ROXCTL=true  # Required if docker not available
```

### Step 1: Deploy Central

```bash
# Optional: Set specific version for migration testing
# export MAIN_IMAGE_TAG=4.9.3

USE_LOCAL_ROXCTL=true scale/dev/launch_central.sh
```

This will:
- Deploy Central to `stackrox` namespace
- Set up port-forward to localhost:8000
- Save admin password to `deploy/k8s/central-deploy/password`

### Step 2: Deploy Sensor with Fake Workload

```bash
# Choose workload: small (200), scale-test (2.5K), or xlarge (15K)
USE_LOCAL_ROXCTL=true scale/dev/launch_sensor.sh xlarge
```

This will:
- Create configmap with workload definition
- Deploy sensor in "fake mode"
- Sensor will automatically restart once (configmap timing) - **this is normal**

### Step 3: Wait for Fake Mode Initialization

**Expected timing:**
- Small workloads (~100 RBAC): ~30 seconds
- Medium workloads (~5K RBAC): ~1-2 minutes
- Large workloads (50K RBAC like xlarge): ~5-7 minutes

**Monitor fake mode initialization:**
```bash
oc logs -n stackrox -l app=sensor -f | grep "fake"
```

You should see:
```
kubernetes/fake: namespaces: 0
kubernetes/fake: nodes: 0
...
kubernetes/fake: Created Workload manager for workload
```

**Note**: Sensor may appear "hung" at `kubernetes/fake: rolebindings: 0` for ~5 minutes with large RBAC counts - **this is normal** (slow pebble.db iteration).

### Step 4: Verify Resources Created

Query the database directly to verify deployment count:

```bash
# Get database password
PGPASSWORD=$(oc get secret -n stackrox central-db-password -o jsonpath='{.data.password}' | base64 -d)

# Count deployments
oc exec -n stackrox $(oc get pod -n stackrox -l app=central-db -o name) -- \
  env PGPASSWORD=$PGPASSWORD psql -U postgres -d central_active -c \
  "SELECT COUNT(*) FROM deployments;"

# Check schema
oc exec -n stackrox $(oc get pod -n stackrox -l app=central-db -o name) -- \
  env PGPASSWORD=$PGPASSWORD psql -U postgres -d central_active -c "\d deployments"
```

**Important**:
- Database name is `central_active`, not "central" or "stackrox"
- The Central API requires authentication even via port-forward: `-u admin:$(cat deploy/k8s/central-deploy/password)`

### Testing Migrations at Scale

#### 1. Verify Baseline Deployment Count
```bash
PGPASSWORD=$(oc get secret -n stackrox central-db-password -o jsonpath='{.data.password}' | base64 -d)
oc exec -n stackrox $(oc get pod -n stackrox -l app=central-db -o name) -- \
  env PGPASSWORD=$PGPASSWORD psql -U postgres -d central_active -c \
  "SELECT COUNT(*) FROM deployments;"
```

#### 2. Upgrade Central to Migration Branch
```bash
# Replace with your actual image tag from CI
oc set image deploy/central -n stackrox central=quay.io/stackrox-io/main:YOUR_TAG
```

#### 3. Retrieve Migration Logs

Get full logs after migration completes:
```bash
oc logs -n stackrox -l app=central --tail=100000 | grep -i "migration\|backfill"
```

Or tail in real-time:
```bash
oc logs -n stackrox -l app=central -f | tee /tmp/migration-logs.txt
```

#### 4. Verify Migration Success
```bash
PGPASSWORD=$(oc get secret -n stackrox central-db-password -o jsonpath='{.data.password}' | base64 -d)

# Verify column was added and populated
oc exec -n stackrox $(oc get pod -n stackrox -l app=central-db -o name) -- \
  env PGPASSWORD=$PGPASSWORD psql -U postgres -d central_active -c \
  "SELECT COUNT(*) FROM deployments WHERE your_new_column IS NOT NULL;"
```

#### 5. Verify Central Starts Successfully
```bash
oc get pods -n stackrox -l app=central
# Expected: Both pods 1/1 Running
```

#### 6. Optional: Test API Access
```bash
ADMIN_PASS=$(cat deploy/k8s/central-deploy/password)
curl -sk https://localhost:8000/v1/metadata -u admin:$ADMIN_PASS
```

### Cleanup

```bash
oc delete namespace stackrox
```

---

## Quick Scale Testing (Existing Deployment)

This quickstart guide is for adding fake workloads to an already-running StackRox deployment.

**Note: This is a destructive operation on your current active `kubectx` cluster that cannot be easily undone.**

```sh
cd scale
./launch_workload.sh <workload_name>
# <workload_name> is the name of a yaml file in the `workloads` directory, without file extension
# e.g. $ ./launch_workload.sh xlarge
```

### How It Works

Running `launch_workload.sh` does the following:
- Deletes the `admission-control` deployment
- Deletes the `collector` daemonset
- Creates a configmap from the yaml file specified in the command, and mounts it under `/var/scale/stackrox/workload.yaml` in the `sensor` container
- Sets some standard CPU/MEM resource limits on stackrox deployments (likely for reproducible results in actual automated tests)

When sensor restarts, it will be put into a "fake" mode when it detects the presense of the `/var/scale/stackrox/workload.yaml` file. This fake
mode will cause sensor to use a mocked k8s client instead of the real client, and it will start sending data to central based
on the values in the provided `workload.yaml` file. While in this fake mode, `sensor` no longer will listen for actual events happening in the cluster.

Each of the top level "workload" keys in this yaml represent a different resource that will be scaled up in some fashion. Some of the
properties in this file can be omitted, but it isn't currently documented which ones. A safer bet for things you don't care to
test is to just reduce the numbers so that the impact on your system is minimal.

Some of the items in the yaml, like `nodeWorkload: numNodes`, simply add a number of items to the database, while workloads
like `deploymentWorkload` have an effect that continues over time. Brief details of what each property does are noted in the commented `workloads/sample.yaml` file.

To tweak scale test values, modify your workload.yaml and re-run the `./launch_workload.sh` script. This will delete the old configmap
and recreate it with the new yaml. Then you can restart sensor with `kubectl -n stackrox rollout pause deploy sensor` which should cause the new config to take effect.

---

## Workload Files

Available in `scale/workloads/`:
- `small.yaml`: 200 deployments (quick tests)
- `scale-test.yaml`: 2,500 deployments (medium-scale testing)
- `xlarge.yaml`: 15,000 deployments (recommended for migration testing)

To create custom workloads, copy an existing file and adjust:
- `numDeployments`: Total deployment count
- `numNamespaces`: Namespace count (required field)
- `numBindings`, `numRoles`, `numServiceAccounts`: RBAC resource counts

See `workloads/sample.yaml` for detailed property documentation.

---

## Troubleshooting

### "No resources showing up in Central after 10+ minutes"

1. **Query database directly** to check actual deployment count (see database query commands above)
2. **Verify fake mode initialized**: `oc logs -n stackrox -l app=sensor | grep "fake"` - look for `kubernetes/fake:` logs

### "Sensor stuck at 'kubernetes/fake: rolebindings: 0' for several minutes"

This is **normal behavior** for large workloads with high RBAC counts (xlarge.yaml has 50K RBAC resources). Wait 5-7 minutes - it will progress. Not a hang, just slow pebble.db iteration.

### "Sensor keeps restarting once after deployment"

This is **expected behavior** when using the `scale/dev/launch_sensor.sh` script. The sensor restarts because the configmap mount takes a moment to become available. First start: runs in normal mode. After restart: picks up workload file and initializes fake mode.
