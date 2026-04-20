# roxctl Central Generate Migration Analysis - Summary

## Executive Summary

This analysis examined all 42 options available in `roxctl central generate` across 4 deployment modes to understand their impact on generated manifests. The goal is to help users migrate from roxctl-generated installations to operator-managed Central installations.

## Key Findings

### Option Categories

1. **11 Client-side options** - No impact on manifests (e.g., --endpoint, --insecure)
2. **2 Output control options** - Affect output location/format, not content
3. **29 Manifest-affecting options** - Actually change deployed resources

### Critical Insight

**Users must match 29 manifest-affecting options** when creating a Central CR, otherwise the operator will apply default values that may differ from their original roxctl-generated deployment.

## Options Requiring Migration Attention

### Tier 1: High-Impact Options (Commonly Used)

These options are frequently used and have significant impact:

| Option | Default | Impact | Operator CR Field (Estimated) |
|--------|---------|--------|-------------------------------|
| `--db-size` | 100 | PVC size | `spec.central.db.persistence.size` |
| `--db-storage-class` | (cluster default) | PVC storage class | `spec.central.db.persistence.storageClass` |
| `--db-hostpath` | `/var/lib/stackrox-central` | HostPath volume path | `spec.central.db.persistence.hostPath` |
| `--lb-type` | none | Service exposure method | `spec.central.exposure.type` |
| `--image-defaults` | rhacs | Image registry/branding | `spec.customize.envVars` or image overrides |
| `--offline` | false | Offline mode | `spec.central.offlineMode` |
| `--enable-telemetry` | true | Telemetry collection | `spec.central.telemetry.enabled` |

### Tier 2: Platform-Specific Options

| Option | Platform | Impact | Detection Method |
|--------|----------|--------|------------------|
| `--openshift-monitoring` | OpenShift | ServiceMonitor creation | `kubectl get servicemonitor -n stackrox` |
| `--lb-type=route` | OpenShift | Route creation | `kubectl get route -n stackrox central` |

### Tier 3: Advanced Options

| Option | Use Case | Impact |
|--------|----------|--------|
| `--enable-pod-security-policies` | Pre-v1.25 K8s | Creates PSP resources |
| `--istio-support` | Istio mesh | Service annotations |
| `--declarative-config-secrets` | GitOps | Secret volume mounts |
| `--db-node-selector-key/value` | HostPath pinning | Node affinity |
| `--main-image` | Custom builds | Image override |
| `--scanner-*-image` | Custom scanner | Scanner image overrides |

## Detection Strategy

For each deployed Central installation, the migration tool should:

### 1. Query Deployed Resources

```bash
# Namespace (typically stackrox)
NS="stackrox"

# Storage detection
kubectl get pvc -n $NS -l app=central-db -o json > central-pvc.json

# Deployment config
kubectl get deploy -n $NS central -o json > central-deployment.json
kubectl get sts -n $NS central-db -o json > central-db.json

# Exposure detection
kubectl get svc -n $NS -o json > services.json
kubectl get route -n $NS -o json > routes.json 2>/dev/null || true

# Additional resources
kubectl get psp 2>/dev/null | grep stackrox || true
kubectl get servicemonitor -n $NS 2>/dev/null || true
```

### 2. Extract Configuration Values

From the JSON outputs, extract:

**Storage (PVC mode):**
- PVC name: `.items[].metadata.name` where labels match central-db
- PVC size: `.spec.resources.requests.storage`
- Storage class: `.spec.storageClassName`

**Storage (HostPath mode):**
- Host path: `.spec.template.spec.volumes[?(@.name=="disk")].hostPath.path`
- Node selector: `.spec.template.spec.nodeSelector`

**Operational:**
- Offline mode: env var `ROX_OFFLINE_MODE` in central deployment
- Telemetry: Check if `ROX_OFFLINE_MODE` is true (telemetry disabled) or false (enabled)

**Exposure:**
- Check for `central-loadbalancer` Service (type: LoadBalancer)
- Check for `central` Route (OpenShift)
- Check for NodePort Service

**Images:**
- Central image: `.spec.template.spec.containers[?(@.name=="central")].image`
- Parse image to detect registry (quay.io/rhacs-eng vs quay.io/stackrox-io)
- Detect custom images by comparing against known defaults

**Advanced:**
- PSPs: Check if PSP resources exist
- Declarative config: Check volume mounts in `/run/stackrox.io/declarative-configuration/`
- Istio: Check service annotations for `traffic.sidecar.istio.io/*`

### 3. Map to Central CR

Create a mapping table:

| Detected Value | Inferred roxctl Option | Central CR Field |
|----------------|------------------------|------------------|
| PVC size = 200Gi | --db-size=200 | `spec.central.db.resources.requests.storage: 200Gi` |
| StorageClass = fast-ssd | --db-storage-class=fast-ssd | `spec.central.db.persistence.persistentVolumeClaim.storageClassName: fast-ssd` |
| Service type = LoadBalancer | --lb-type=lb | `spec.central.exposure.loadBalancer.enabled: true` |
| ROX_OFFLINE_MODE = true | --offline=true | `spec.central.exposure.internetDisabled: true` |
| ServiceMonitor exists | --openshift-monitoring=true | (automatically created by operator on OpenShift) |

### 4. Generate Central CR

```yaml
apiVersion: platform.stackrox.io/v1alpha1
kind: Central
metadata:
  name: stackrox-central-services
  namespace: stackrox
spec:
  central:
    db:
      isEnabled: Default
      persistence:
        persistentVolumeClaim:
          claimName: central-db  # if --db-name was used
          size: 100Gi            # from --db-size
          storageClassName: fast-ssd  # from --db-storage-class
        # OR for hostpath:
        # hostPath:
        #   path: /custom/path   # from --db-hostpath
        #   nodeSelectionAttributes:
        #     - key: kubernetes.io/hostname  # from --db-node-selector-key
        #       value: node1                  # from --db-node-selector-value
    exposure:
      loadBalancer:
        enabled: true            # if --lb-type=lb
      route:
        enabled: true            # if --lb-type=route (OpenShift)
      nodePort:
        enabled: true            # if --lb-type=np
    # If custom images detected:
    # image:
    #   fullRef: quay.io/stackrox-io/main:custom-tag  # from --main-image
```

## Recommendations for Migration Tool

### Required Features

1. **Auto-detection mode**
   - Query deployed resources
   - Extract configuration
   - Generate matching Central CR

2. **Validation mode**
   - Compare provided CR against detected config
   - Warn about discrepancies
   - Highlight values that will change

3. **Dry-run mode**
   - Show what would change without creating CR
   - Display diff between current and operator-managed state

### User Workflow

```bash
# Step 1: Analyze existing deployment
migration-tool analyze --namespace stackrox > detected-config.yaml

# Step 2: Review detected configuration
cat detected-config.yaml

# Step 3: Generate Central CR
migration-tool generate-cr --namespace stackrox --output central.yaml

# Step 4: Validate before applying
migration-tool validate --cr central.yaml --namespace stackrox

# Step 5: Apply CR
kubectl apply -f central.yaml
```

### Critical Warnings

The tool should warn about:

1. **Options that will change defaults:**
   - "Detected db-size=200Gi but Central CR will use 100Gi if not specified"
   - "Detected offline mode enabled, ensure spec.central.exposure.internetDisabled=true"

2. **Options with no CR equivalent:**
   - "Pod Security Policies detected but operator does not support PSPs"
   - "Custom certificate bundle detected - manual migration required"

3. **Platform migrations:**
   - "OpenShift Route detected but deploying to vanilla K8s - use LoadBalancer instead"

## Options Requiring Special Handling

### Cannot Be Directly Migrated

1. **--backup-bundle** - One-time import, not ongoing configuration
2. **--ca, --default-tls-cert, --default-tls-key** - Custom CA/certs must be provided as separate secrets
3. **--password** - Admin password is auto-generated by operator
4. **--enable-pod-security-policies** - Deprecated in modern Kubernetes

### Requires Manual Intervention

1. **Custom certificates** - User must create secrets before creating CR
2. **Declarative configuration** - ConfigMaps/Secrets must exist before CR creation
3. **Image pull secrets** - Must be configured in operator namespace

## Testing Recommendations

1. **Test each tier of options** in realistic combinations
2. **Verify operator adopts resources** without recreation
3. **Test upgrade paths** from old roxctl versions
4. **Validate on both** OpenShift and vanilla K8s

## File Organization

```
MIGRATION/
├── PLAN.md                          # Original plan
├── MASTER_OPTIONS_LIST.md           # Complete option catalog with test results
├── SUMMARY.md                       # This file
├── DETECTION_COMMANDS.md            # kubectl detection commands
├── help-outputs/                    # roxctl --help outputs
│   ├── help-openshift-pvc.txt
│   ├── help-k8s-pvc.txt
│   ├── help-openshift-hostpath.txt
│   └── help-k8s-hostpath.txt
├── baselines/                       # Default manifests
│   ├── openshift-pvc/
│   ├── k8s-pvc/
│   ├── openshift-hostpath/
│   └── k8s-hostpath/
├── test-outputs/                    # Test manifests with options
│   ├── db-size-openshift-pvc/
│   ├── lb-type-openshift-pvc/
│   └── ... (one per option per mode)
└── diffs/                           # Diff outputs
    ├── db-size-openshift-pvc.diff
    └── ...
```

## Next Steps

1. **Complete testing** of remaining untested options (--password, --plaintext-endpoints, scanner image overrides)
2. **Create mapping table** from detected values to Central CR fields
3. **Implement detection tool** using kubectl queries
4. **Build CR generator** with validation logic
5. **Test migration** on real deployments
6. **Document edge cases** and limitations

## Conclusion

The migration from roxctl-generated to operator-managed Central is feasible with automated detection. The key is querying deployed resources to infer the original roxctl options, then mapping those to Central CR fields.

**Success criteria:**
- Users create Central CR in same namespace
- Operator adopts existing resources without disruption
- No configuration drift from original deployment
- Clear warnings about any changes
