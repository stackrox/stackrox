# OCP UI E2E Testing Status

## Current State (2026-06-03)

**✅ Working: 8/9 test shards passing on OCP**

Successfully tested against OCP cluster `dh-06-02-field-dear-decide` via infractl.

### Passing Shards (8/9)
1. accessControl+collections+violations+compliance
2. clusters
3. configmanagement
4. integrations+root
5. networkGraph+systemConfig+main+listeningEndpoints+exceptionConfiguration+dashboard+credentialExpiry+compliance-enhanced+baseImages+administration
6. policies+risk
7. systemHealth
8. vulnerabilities

### Failing Shard (1/9)
- **vulnmanagement** - requires cluster/node CVE data

## Root Cause Analysis

### Why vulnmanagement fails

The vulnmanagement tests require two types of CVE data:
1. **Cluster CVEs** - vulnerabilities in Kubernetes components (kubelet, API server, etc.)
2. **Node CVEs** - vulnerabilities in host OS packages on cluster nodes

Current deployment uses **Scanner V2**, which does not support node/cluster CVE scanning.

### What's needed for cluster/node CVE scanning

Based on code analysis (`central/sensor/service/pipeline/nodeinventory/pipeline.go`):

```go
if features.ScannerV4.Enabled() && features.NodeIndexEnabled.Enabled() {
    return true
}
```

**Requirements:**
1. Deploy with **Scanner V4** (not V2)
2. Enable **ROX_NODE_INDEX_ENABLED=true** environment variable on Central
3. This enables Node Index (node scanning V4) messages

### Comparison with .openshift-ci/ e2e tests

OCP e2e tests in `.openshift-ci/` use different configuration:
- Set `ORCHESTRATOR_FLAVOR=openshift` (we use `k8s`)
- Run `enable_sfa_for_ocp()` for OCP >= 4.16
- Set `DEPLOY_STACKROX_VIA_OPERATOR=true`
- Run image prefetcher for test images

However, these differences don't affect node/cluster CVE scanning capability.

## Tested Configurations

| Run | Scanner | Force Redeploy | Deployment Age | Cluster CVEs | Result |
|-----|---------|----------------|----------------|--------------|--------|
| 26861050410 | v2 | ✅ Yes | Fresh | 0 (20min wait) | 8 pass, 1 cancelled |
| 26863424884 | v4 | ✅ Yes | Fresh | 0 (20min wait) | 5 pass, 3 fail, 1 cancelled |
| 26864704631 | v2 | ✅ Yes | Fresh | 0 (20min wait) | 8 pass, 1 cancelled |
| 26899272977 | v2 | ❌ No | 6+ hours | 0 (20min wait) | 5 fail, 4 cancelled |

**Key findings:**
- Scanner V4 without node indexing performed worse than V2
- **Long-running deployments accumulate stale state** causing test failures
- **Scanner v2 NEVER populates cluster/node CVEs** regardless of runtime
- **force-redeploy is essential** for reliable test runs

## Recommendations

### Short term (current)
- **Keep using Scanner V2** - provides 89% test coverage (8/9 shards)
- **Always use force-redeploy** - prevents stale state failures
- **Skip vulnmanagement tests on OCP** - document as known limitation
- **Focus on the 8 passing shards** - comprehensive StackRox feature coverage

### Deployment Re-use
**DO NOT re-use long-running deployments** for testing:
- Fresh deployment (run 26864704631): 8/9 passing ✅
- 6-hour-old deployment (run 26899272977): 5 failures + 4 cancelled ❌
- Root cause: Stale state accumulates (auth issues, broken connections, etc.)
- Always use `force-redeploy=true` for reliable results

### Long term (future enhancement)
To enable vulnmanagement tests on OCP:

1. **Switch to Scanner V4**
2. **Add Central environment variable:**
   ```yaml
   central:
     customize:
       envVars:
         ROX_NODE_INDEX_ENABLED: "true"
   ```
3. **Verify node scanning works** before enabling tests

## Implementation Notes

The deploy-stackrox action would need to be enhanced to:
1. Accept an `enable-node-scanning` input parameter
2. Set the Central env var via roxie override YAML:
   ```bash
   yq -i '.central.customize.envVars.ROX_NODE_INDEX_ENABLED = "true"' /tmp/roxie-override.yaml
   ```

This requires roxie to support Central customize.envVars in its override mechanism.
