# roxctl to Operator Migration Analysis

This directory contains a comprehensive analysis of `roxctl central generate` options and their impact on generated manifests, performed to enable users to migrate from roxctl-generated Central installations to operator-managed installations.

## 📋 Quick Start

**For developers building the migration tool:** Start with [DETECTION_COMMANDS.md](DETECTION_COMMANDS.md)

**For understanding the full analysis:** Read [SUMMARY.md](SUMMARY.md)

**For complete option catalog with test results:** Browse [MASTER_OPTIONS_LIST.md](MASTER_OPTIONS_LIST.md)

## 📁 Directory Structure

```
MIGRATION/
├── README.md                        # This file
├── PLAN.md                          # Original analysis plan
├── SUMMARY.md                       # Executive summary and recommendations
├── MASTER_OPTIONS_LIST.md           # Complete catalog of all 42 options with test results
├── DETECTION_COMMANDS.md            # kubectl commands for detecting options
├── test-option.sh                   # Helper script for testing options
├── help-outputs/                    # roxctl --help outputs for 4 modes
├── baselines/                       # Default manifests (no options)
├── baselines2/                      # Duplicate baselines (for randomness check)
├── test-outputs/                    # Test manifests with specific options
└── diffs/                           # Diff files comparing baseline vs options
```

## 🎯 Analysis Goals

1. **Identify all options** in `roxctl central generate` across 4 modes
2. **Test each option** to understand its impact on generated manifests
3. **Create detection methods** to identify which options were used in existing deployments
4. **Enable migration** by mapping roxctl options to Central CR fields

## 🔍 Methodology

### Phase 1: Discovery
- Captured `--help` output for all 4 modes:
  - `roxctl central generate openshift pvc`
  - `roxctl central generate k8s pvc`
  - `roxctl central generate openshift hostpath`
  - `roxctl central generate k8s hostpath`
- Compared outputs to identify mode-specific vs global options
- Created master list of 42 total options

### Phase 2: Baseline Establishment
- Generated default manifests for each mode with no options
- Documented the structure of generated bundles

### Phase 3: Randomness Identification
- Generated duplicate baselines
- Identified non-deterministic elements:
  - Admin password hashes
  - TLS certificates and private keys
  - JWT signing keys
- Created ignore patterns for diff analysis

### Phase 4: Impact Analysis
- Tested each manifest-affecting option across applicable modes
- Compared outputs against baselines
- Documented:
  - Which files change
  - What exactly changes (fields, values)
  - How to detect if option was used (kubectl commands)
  - Mapping to Central CR fields

## 📊 Key Findings

### Option Categories

| Category | Count | Examples |
|----------|-------|----------|
| Storage-specific (PVC) | 3 | --db-name, --db-size, --db-storage-class |
| Storage-specific (HostPath) | 3 | --db-hostpath, --db-node-selector-* |
| Platform-specific (OpenShift) | 2 | --openshift-monitoring, --openshift-version |
| Image configuration | 6 | --image-defaults, --main-image, --scanner-*-image |
| Exposure | 1 | --lb-type (with 4 values) |
| Security & policy | 1 | --enable-pod-security-policies |
| Operational | 3 | --offline, --enable-telemetry, --disable-admin-password |
| TLS/Certificates | 4 | --ca, --default-tls-cert, --default-tls-key, --backup-bundle |
| Declarative config | 2 | --declarative-config-secrets, --declarative-config-config-maps |
| Advanced networking | 2 | --istio-support, --plaintext-endpoints |
| Output control | 2 | --output-dir, --output-format |
| Client-side only | 11 | --endpoint, --insecure, --token-file, etc. |

### Critical Insights

1. **29 of 42 options affect manifests** - these must be detected and mapped to Central CR
2. **11 options are client-side only** - no impact on deployed resources
3. **Storage options differ by mode** - PVC has 3 options, HostPath has 3 different options
4. **Platform matters** - OpenShift has exclusive options like `--lb-type=route`
5. **Image defaults have broad impact** - affects all component deployments and scanners

## 🛠️ Detection Strategy

For each deployed Central installation, the migration tool should:

1. **Identify deployment mode** (PVC vs HostPath, OpenShift vs K8s)
2. **Query resources** using kubectl (see [DETECTION_COMMANDS.md](DETECTION_COMMANDS.md))
3. **Extract values** and compare against known defaults
4. **Infer roxctl options** that were likely used
5. **Map to Central CR** fields
6. **Generate CR** with equivalent configuration
7. **Validate** before applying

### Example Detection Flow

```bash
# 1. Check storage mode
if kubectl get pvc -n stackrox -l app=central-db &>/dev/null; then
    MODE="pvc"
    SIZE=$(kubectl get pvc -n stackrox -l app=central-db -o jsonpath='{.items[0].spec.resources.requests.storage}')
else
    MODE="hostpath"
    PATH=$(kubectl get sts -n stackrox central-db -o jsonpath='{.spec.template.spec.volumes[?(@.name=="disk")].hostPath.path}')
fi

# 2. Detect exposure
if kubectl get svc -n stackrox central-loadbalancer &>/dev/null; then
    EXPOSURE="loadbalancer"
elif kubectl get route -n stackrox central &>/dev/null; then
    EXPOSURE="route"
fi

# 3. Generate CR
cat <<EOF > central.yaml
apiVersion: platform.stackrox.io/v1alpha1
kind: Central
metadata:
  name: stackrox-central-services
  namespace: stackrox
spec:
  central:
    db:
      persistence:
        persistentVolumeClaim:
          size: $SIZE
    exposure:
      loadBalancer:
        enabled: $( [ "$EXPOSURE" = "loadbalancer" ] && echo true || echo false )
EOF
```

## 📈 Test Coverage

### Fully Tested Options (24)

Storage:
- ✅ --db-name
- ✅ --db-size
- ✅ --db-storage-class
- ✅ --db-hostpath
- ✅ --db-node-selector-key/value

Operational:
- ✅ --enable-telemetry
- ✅ --offline
- ✅ --disable-admin-password

Exposure:
- ✅ --lb-type (all values)

Images:
- ✅ --image-defaults
- ✅ --main-image

Security:
- ✅ --enable-pod-security-policies

Platform:
- ✅ --openshift-version
- ✅ --openshift-monitoring

Advanced:
- ✅ --istio-support
- ✅ --declarative-config-secrets
- ✅ --declarative-config-config-maps

### Partially Tested or Untested (5)

- ⚠️ --password (affects htpasswd secret but minimal impact)
- ⚠️ --ca, --default-tls-cert, --default-tls-key (certificate options)
- ⚠️ --backup-bundle (one-time import)
- ⚠️ --plaintext-endpoints
- ⚠️ Individual scanner image overrides (--scanner-db-image, etc.)

### Not Tested (Client-side only, 11)

These don't affect manifests:
- --endpoint, --insecure, --token-file, --server-name, etc.

## 🚀 Next Steps

### For Migration Tool Development

1. **Implement detection** using commands from [DETECTION_COMMANDS.md](DETECTION_COMMANDS.md)
2. **Build mapping logic** to convert detected values to Central CR fields
3. **Add validation** to warn about configuration changes
4. **Create dry-run mode** to show impact before applying
5. **Test on real deployments** with various option combinations

### For Documentation

1. **Create user guide** for migration process
2. **Document edge cases** and manual migration steps
3. **Provide examples** for common scenarios
4. **List limitations** and unsupported options

### For Testing

1. **Complete untested options** analysis
2. **Test option combinations** (not just individual options)
3. **Verify operator adoption** of existing resources
4. **Test upgrades** from various StackRox versions

## 🎓 Learning Resources

### Understanding roxctl

```bash
# View available modes
roxctl central generate --help

# Generate with specific mode
roxctl central generate openshift pvc --help

# Test an option
roxctl central generate k8s pvc --db-size=200 --output-dir /tmp/test
```

### Understanding Central CR

See the operator documentation and CRD definition:
- `operator/EXTENDING_CRDS.md`
- `operator/config/crd/bases/platform.stackrox.io_centrals.yaml`

### kubectl Queries

All detection commands use kubectl with JSONPath:
```bash
# Get a specific field
kubectl get <resource> -o jsonpath='{.path.to.field}'

# Filter with JSONPath
kubectl get <resource> -o jsonpath='{.items[?(@.condition)]}'

# Format output
kubectl get <resource> -o jsonpath='{.items[*].name}' | tr ' ' '\n'
```

## 📞 Questions?

This analysis was performed to support the roxctl-to-operator migration feature. For questions or clarifications, refer to:

- Full analysis plan: [PLAN.md](PLAN.md)
- Summary and recommendations: [SUMMARY.md](SUMMARY.md)
- Complete option catalog with findings: [MASTER_OPTIONS_LIST.md](MASTER_OPTIONS_LIST.md)

## 🔧 Tools Used

- **roxctl**: StackRox CLI for generating manifests
- **diff**: For comparing manifest outputs
- **kubectl**: For querying deployed resources
- **bash**: For automation scripts
- **grep/sed/awk**: For text processing

## 📝 Notes

- Analysis performed on: 2026-04-16
- roxctl version: From PATH (latest)
- Kubernetes version: 1.24+
- Test modes: All 4 combinations (openshift/k8s × pvc/hostpath)
