# Quick Reference: Detection Commands for roxctl Options

This is a quick reference for developers building the migration tool. Each command detects whether a specific roxctl option was used.

**Assumptions:**
- Namespace: `stackrox` (adjust as needed)
- kubectl is configured and has access to the cluster

## Storage Options (PVC mode)

```bash
# Detect --db-name
kubectl get pvc -n stackrox -l app=central-db -o jsonpath='{.items[0].metadata.name}'
# Default: central-db

# Detect --db-size
kubectl get pvc -n stackrox -l app=central-db -o jsonpath='{.items[0].spec.resources.requests.storage}'
# Default: 100Gi

# Detect --db-storage-class
kubectl get pvc -n stackrox -l app=central-db -o jsonpath='{.items[0].spec.storageClassName}'
# Default: (empty or cluster default)
```

## Storage Options (HostPath mode)

```bash
# Detect --db-hostpath
kubectl get sts -n stackrox central-db -o jsonpath='{.spec.template.spec.volumes[?(@.name=="disk")].hostPath.path}'
# Default: /var/lib/stackrox-central

# Detect --db-node-selector-key/value
kubectl get sts -n stackrox central-db -o jsonpath='{.spec.template.spec.nodeSelector}'
# Default: {}
# If set: {"key":"value"}
```

## Operational Options

```bash
# Detect --offline mode
kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="ROX_OFFLINE_MODE")].value}'
# Default: false
# If --offline=true: true

# Detect --enable-telemetry (inverse of offline mode)
# Same as above - if ROX_OFFLINE_MODE=true, telemetry is disabled
```

## Exposure Options

```bash
# Detect --lb-type=lb
kubectl get svc -n stackrox central-loadbalancer -o jsonpath='{.spec.type}' 2>/dev/null
# If exists: LoadBalancer
# If not exists: option not used

# Detect --lb-type=route (OpenShift only)
kubectl get route -n stackrox central 2>/dev/null && echo "route" || echo "none"

# Detect --lb-type=np
kubectl get svc -n stackrox central -o jsonpath='{.spec.type}'
# If NodePort: --lb-type=np was used
# If ClusterIP: default (none)
```

## Image Options

```bash
# Detect --image-defaults
kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.containers[0].image}'
# If contains "rhacs-eng": --image-defaults=rhacs (default)
# If contains "stackrox-io": --image-defaults=opensource

# Detect --main-image
kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.containers[?(@.name=="central")].image}'
# Compare against default for current --image-defaults
# If different: custom --main-image was used

# Detect --central-db-image
kubectl get sts -n stackrox central-db -o jsonpath='{.spec.template.spec.containers[0].image}'

# Detect --scanner-image
kubectl get deploy -n stackrox scanner -o jsonpath='{.spec.template.spec.containers[0].image}'

# Detect --scanner-db-image
kubectl get deploy -n stackrox scanner-db -o jsonpath='{.spec.template.spec.containers[0].image}'

# Detect --scanner-v4-image (matcher and indexer)
kubectl get deploy -n stackrox scanner-v4-matcher -o jsonpath='{.spec.template.spec.containers[0].image}'
kubectl get deploy -n stackrox scanner-v4-indexer -o jsonpath='{.spec.template.spec.containers[0].image}'

# Detect --scanner-v4-db-image
kubectl get deploy -n stackrox scanner-v4-db -o jsonpath='{.spec.template.spec.containers[0].image}'
```

## Security & Policy Options

```bash
# Detect --enable-pod-security-policies
kubectl get psp 2>/dev/null | grep stackrox && echo "enabled" || echo "disabled"
# If PSPs exist: enabled
# If not: disabled (default)
```

## Platform-Specific Options (OpenShift)

```bash
# Detect --openshift-monitoring
kubectl get servicemonitor -n stackrox central 2>/dev/null && echo "enabled" || echo "auto/disabled"
# If ServiceMonitor exists: monitoring enabled
# If not: auto (default) or disabled

# Detect --openshift-version
# Not easily detectable from deployed resources
# May be visible in Helm values or annotations
```

## Advanced Networking

```bash
# Detect --istio-support
kubectl get svc -n stackrox central -o jsonpath='{.metadata.annotations}' | grep -o 'traffic\.sidecar\.istio\.io' && echo "enabled" || echo "disabled"
# If Istio annotations present: enabled
# If not: disabled (default)
```

## Declarative Configuration

```bash
# Detect --declarative-config-secrets
kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.volumes[?(@.secret)].secret.secretName}' | tr ' ' '\n' | grep -v 'htpasswd\|tls\|monitoring' || echo "none"
# Lists secret names mounted in declarative config path
# Filter out standard StackRox secrets

# Detect --declarative-config-config-maps
kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.volumes[?(@.configMap)].configMap.name}' | tr ' ' '\n' | grep -v 'additional-ca' || echo "none"
# Lists ConfigMap names mounted in declarative config path
# Filter out standard StackRox ConfigMaps

# Detailed check for declarative config mounts
kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.containers[0].volumeMounts[*].mountPath}' | tr ' ' '\n' | grep 'declarative-configuration'
# Lists mount paths under /run/stackrox.io/declarative-configuration/
```

## Complete Detection Script

```bash
#!/bin/bash
NS="${1:-stackrox}"

echo "=== Storage Configuration ==="
echo "Storage type: $(kubectl get pvc -n $NS -l app=central-db &>/dev/null && echo 'PVC' || echo 'HostPath')"

if kubectl get pvc -n $NS -l app=central-db &>/dev/null; then
    echo "PVC name: $(kubectl get pvc -n $NS -l app=central-db -o jsonpath='{.items[0].metadata.name}')"
    echo "PVC size: $(kubectl get pvc -n $NS -l app=central-db -o jsonpath='{.items[0].spec.resources.requests.storage}')"
    echo "Storage class: $(kubectl get pvc -n $NS -l app=central-db -o jsonpath='{.items[0].spec.storageClassName}')"
else
    echo "HostPath: $(kubectl get sts -n $NS central-db -o jsonpath='{.spec.template.spec.volumes[?(@.name=="disk")].hostPath.path}')"
    echo "Node selector: $(kubectl get sts -n $NS central-db -o jsonpath='{.spec.template.spec.nodeSelector}')"
fi

echo ""
echo "=== Operational Settings ==="
echo "Offline mode: $(kubectl get deploy -n $NS central -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="ROX_OFFLINE_MODE")].value}')"

echo ""
echo "=== Exposure ==="
kubectl get svc -n $NS central-loadbalancer &>/dev/null && echo "LoadBalancer: enabled" || echo "LoadBalancer: disabled"
kubectl get route -n $NS central &>/dev/null && echo "Route: enabled" || echo "Route: disabled"
SVC_TYPE=$(kubectl get svc -n $NS central -o jsonpath='{.spec.type}')
echo "Service type: $SVC_TYPE"

echo ""
echo "=== Images ==="
echo "Central: $(kubectl get deploy -n $NS central -o jsonpath='{.spec.template.spec.containers[?(@.name=="central")].image}')"
echo "Central DB: $(kubectl get sts -n $NS central-db -o jsonpath='{.spec.template.spec.containers[0].image}')"

echo ""
echo "=== Advanced Features ==="
kubectl get psp 2>/dev/null | grep -q stackrox && echo "PSPs: enabled" || echo "PSPs: disabled"
kubectl get servicemonitor -n $NS central &>/dev/null && echo "OpenShift monitoring: enabled" || echo "OpenShift monitoring: disabled"
kubectl get svc -n $NS central -o jsonpath='{.metadata.annotations}' | grep -q 'istio' && echo "Istio support: enabled" || echo "Istio support: disabled"

echo ""
echo "=== Declarative Config ==="
echo "Secrets: $(kubectl get deploy -n $NS central -o jsonpath='{.spec.template.spec.containers[0].volumeMounts[*].mountPath}' | tr ' ' '\n' | grep 'declarative-configuration' | sed 's|.*/||' || echo 'none')"
```

## Usage Examples

### Detect if --db-size was customized
```bash
SIZE=$(kubectl get pvc -n stackrox -l app=central-db -o jsonpath='{.items[0].spec.resources.requests.storage}')
if [ "$SIZE" != "100Gi" ]; then
    echo "Custom db-size detected: $SIZE"
    echo "Add to Central CR: spec.central.db.resources.requests.storage: $SIZE"
fi
```

### Detect exposure method
```bash
if kubectl get svc -n stackrox central-loadbalancer &>/dev/null; then
    echo "LoadBalancer exposure detected"
    echo "Add to Central CR: spec.central.exposure.loadBalancer.enabled: true"
elif kubectl get route -n stackrox central &>/dev/null; then
    echo "Route exposure detected (OpenShift)"
    echo "Add to Central CR: spec.central.exposure.route.enabled: true"
fi
```

### Detect custom images
```bash
CENTRAL_IMAGE=$(kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.containers[?(@.name=="central")].image}')
if echo "$CENTRAL_IMAGE" | grep -q "stackrox-io"; then
    echo "Opensource images detected"
    echo "Add to Central CR: spec.image.registry: quay.io/stackrox-io"
elif ! echo "$CENTRAL_IMAGE" | grep -q "rhacs-eng"; then
    echo "Custom image detected: $CENTRAL_IMAGE"
    echo "Add to Central CR: spec.central.image: $CENTRAL_IMAGE"
fi
```

## Notes

1. **Error Handling**: All commands should handle missing resources gracefully (use `2>/dev/null` or `|| true`)
2. **Multiple Namespaces**: Replace hardcoded `stackrox` with variable `$NS`
3. **RBAC**: Ensure service account has `get` permissions for all resource types
4. **Version Compatibility**: Commands tested against Kubernetes 1.24+
5. **JSONPath**: Some complex JSONPath queries may need adjustment based on K8s version

## Next Steps for Implementation

1. Wrap each detection command in a function
2. Build a detection result structure (JSON/YAML)
3. Map detected values to Central CR fields
4. Generate Central CR YAML from detection results
5. Add validation logic to warn about unsupported options
