# roxctl CLI to Operator Migration Guide

This guide helps users migrate from installing StackRox Central using `roxctl central generate` to using the Operator with the `Central` Custom Resource (CR).

## Table of Contents

- [Overview](#overview)
- [Pre-Migration Checklist](#pre-migration-checklist)
- [Flag-by-Flag Migration Guide](#flag-by-flag-migration-guide)
  - [Storage Configuration](#storage-configuration)
  - [Exposure and Networking](#exposure-and-networking)
  - [Images and Repositories](#images-and-repositories)
  - [Security and Authentication](#security-and-authentication)
  - [Persistence and Database](#persistence-and-database)
  - [Platform-Specific Options](#platform-specific-options)
  - [Advanced Configuration](#advanced-configuration)

## Overview

The `roxctl central generate` command generates Kubernetes manifests (kubectl YAML, Helm charts, or Helm values) for deploying StackRox Central. The Operator-based deployment uses a `Central` Custom Resource (CR) that is managed by the StackRox Operator.

**Key Differences:**
- **CLI approach**: Generate static manifests → apply them manually
- **Operator approach**: Define desired state in CR → Operator continuously reconciles

## Pre-Migration Checklist

Before migrating, gather information about your current deployment:

1. **Check your original roxctl command**:
   ```bash
   # Look in your deployment scripts or command history
   history | grep "roxctl central generate"
   ```

2. **Inspect your current deployment**:
   ```bash
   # Check ConfigMaps and Secrets for hints about configuration
   kubectl get configmap,secret -n stackrox
   
   # Check Central service type
   kubectl get svc central -n stackrox
   
   # Check PVC configuration
   kubectl get pvc -n stackrox
   ```

3. **Identify storage type**:
   ```bash
   # Check if using PVC
   kubectl get pvc central-db -n stackrox
   
   # Check if using hostPath (look at central-db deployment)
   kubectl get deployment central-db -n stackrox -o yaml | grep -A 5 hostPath
   ```

## Flag-by-Flag Migration Guide

### Storage Configuration

#### PVC Storage (Recommended)

**CLI Usage:**
```bash
roxctl central generate k8s pvc \
  --db-name=central-db \
  --db-size=100 \
  --db-storage-class=standard
```

**How to check if used:**
```bash
kubectl get pvc -n stackrox
# Look for: central-db PVC
```

**Operator Equivalent:**
```yaml
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
          claimName: central-db           # Maps to --db-name (default: "central-db")
          size: 100Gi                      # Maps to --db-size (in Gi)
          storageClassName: standard       # Maps to --db-storage-class
```

**Notes:**
- Default PVC name is `central-db` if not specified
- Size must include unit (e.g., `100Gi` not just `100`)
- If `storageClassName` is omitted, the cluster's default storage class is used

---

#### HostPath Storage (Not Recommended)

**CLI Usage:**
```bash
roxctl central generate k8s hostpath \
  --db-hostpath=/var/lib/stackrox-central \
  --db-node-selector-key=kubernetes.io/hostname \
  --db-node-selector-value=node-1
```

**How to check if used:**
```bash
kubectl get deployment central-db -n stackrox -o yaml | grep -A 10 hostPath
# Look for: hostPath volume configuration
```

**Operator Equivalent:**
```yaml
apiVersion: platform.stackrox.io/v1alpha1
kind: Central
metadata:
  name: stackrox-central-services
  namespace: stackrox
spec:
  central:
    db:
      persistence:
        hostPath:
          path: /var/lib/stackrox-central  # Maps to --db-hostpath
      nodeSelector:                        # Maps to --db-node-selector-key/value
        kubernetes.io/hostname: node-1
      tolerations: []                      # Add if node has taints
```

**Notes:**
- HostPath requires node selector to ensure pod always runs on the same node
- Not recommended for production; data loss risk if node fails
- The operator sets `nodeSelector` at the DB component level, not in the hostPath configuration itself

---

### Exposure and Networking

#### Load Balancer

**CLI Usage:**
```bash
# Kubernetes
roxctl central generate k8s pvc --lb-type=lb

# OpenShift
roxctl central generate openshift pvc --lb-type=lb
```

**How to check if used:**
```bash
kubectl get svc central-loadbalancer -n stackrox
# Look for: TYPE=LoadBalancer
```

**Operator Equivalent:**
```yaml
spec:
  central:
    exposure:
      loadBalancer:
        enabled: true
        port: 443                    # Optional, default: 443
        ip: ""                       # Optional, for static IP
```

**Notes:**
- Port range: 1-65535
- `ip` field is for reserved static IP addresses

---

#### Node Port

**CLI Usage:**
```bash
roxctl central generate k8s pvc --lb-type=np
```

**How to check if used:**
```bash
kubectl get svc central -n stackrox
# Look for: TYPE=NodePort
```

**Operator Equivalent:**
```yaml
spec:
  central:
    exposure:
      nodePort:
        enabled: true
        port: 30443                  # Optional, specific node port
```

**Notes:**
- Port range: 1-65535
- If `port` is omitted, Kubernetes assigns one automatically from the NodePort range

---

#### OpenShift Route (Passthrough)

**CLI Usage:**
```bash
roxctl central generate openshift pvc --lb-type=route
```

**How to check if used:**
```bash
oc get route -n stackrox
# Look for: route/central
```

**Operator Equivalent:**
```yaml
spec:
  central:
    exposure:
      route:
        enabled: true
        host: ""                     # Optional, custom hostname
```

**Notes:**
- Only available on OpenShift clusters
- Default creates a passthrough route (TLS terminated at Central)
- If `host` is empty, OpenShift generates one automatically

---

#### No Exposure

**CLI Usage:**
```bash
roxctl central generate k8s pvc --lb-type=none
```

**How to check if used:**
```bash
kubectl get svc -n stackrox
# Only see: central (ClusterIP type)
```

**Operator Equivalent:**
```yaml
spec:
  central:
    exposure:
      loadBalancer:
        enabled: false
      nodePort:
        enabled: false
      route:
        enabled: false
```

**Notes:**
- This is the default if no exposure is configured
- Central is only accessible from within the cluster via ClusterIP service

---

### Images and Repositories

**Important Note on Image Overrides:**

The Operator manages all component images automatically based on the operator version. Image references **cannot be overridden via the Central CR**. However, for disconnected/air-gapped environments where you need to mirror images to a local registry, you can override images by setting `RELATED_IMAGE_*` environment variables when installing or configuring the operator.

#### Overriding Images for Disconnected Clusters

**For disconnected/air-gapped clusters** where images must be mirrored to a local registry, configure the operator at installation time:

**Method 1: Helm Chart Installation**

If installing the operator using Helm:

```bash
# 1. Mirror images to your local registry
skopeo copy docker://registry.redhat.io/rhacs/main:4.11.0 \
            docker://internal.registry.example.com/rhacs/main:4.11.0

# 2. Install operator with custom values
helm install rhacs-operator operator/dist/chart \
  --namespace rhacs-operator \
  --create-namespace \
  --set manager.envOverrides.RELATED_IMAGE_MAIN=internal.registry.example.com/rhacs/main:4.11.0 \
  --set manager.envOverrides.RELATED_IMAGE_CENTRAL_DB=internal.registry.example.com/rhacs/central-db:4.11.0 \
  --set manager.envOverrides.RELATED_IMAGE_SCANNER=internal.registry.example.com/rhacs/scanner:4.11.0 \
  --set manager.envOverrides.RELATED_IMAGE_SCANNER_DB=internal.registry.example.com/rhacs/scanner-db:4.11.0 \
  --set manager.envOverrides.RELATED_IMAGE_SCANNER_V4=internal.registry.example.com/rhacs/scanner-v4:4.11.0 \
  --set manager.envOverrides.RELATED_IMAGE_SCANNER_V4_DB=internal.registry.example.com/rhacs/scanner-v4-db:4.11.0
```

Or create a values file:

```yaml
# custom-values.yaml
manager:
  envOverrides:
    RELATED_IMAGE_MAIN: internal.registry.example.com/rhacs/main:4.11.0
    RELATED_IMAGE_CENTRAL_DB: internal.registry.example.com/rhacs/central-db:4.11.0
    RELATED_IMAGE_SCANNER: internal.registry.example.com/rhacs/scanner:4.11.0
    RELATED_IMAGE_SCANNER_DB: internal.registry.example.com/rhacs/scanner-db:4.11.0
    RELATED_IMAGE_SCANNER_V4: internal.registry.example.com/rhacs/scanner-v4:4.11.0
    RELATED_IMAGE_SCANNER_V4_DB: internal.registry.example.com/rhacs/scanner-v4-db:4.11.0
```

Then install:
```bash
helm install rhacs-operator operator/dist/chart \
  --namespace rhacs-operator \
  --create-namespace \
  -f custom-values.yaml
```

**Method 2: OLM Subscription (OpenShift/OLM)**

If installing via Operator Lifecycle Manager (OLM):

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: rhacs-operator
  namespace: rhacs-operator
spec:
  channel: stable
  name: rhacs-operator
  source: redhat-operators
  sourceNamespace: openshift-marketplace
  config:
    env:
      - name: RELATED_IMAGE_MAIN
        value: internal.registry.example.com/rhacs/main:4.11.0
      - name: RELATED_IMAGE_CENTRAL_DB
        value: internal.registry.example.com/rhacs/central-db:4.11.0
      - name: RELATED_IMAGE_SCANNER
        value: internal.registry.example.com/rhacs/scanner:4.11.0
      - name: RELATED_IMAGE_SCANNER_DB
        value: internal.registry.example.com/rhacs/scanner-db:4.11.0
      - name: RELATED_IMAGE_SCANNER_V4
        value: internal.registry.example.com/rhacs/scanner-v4:4.11.0
      - name: RELATED_IMAGE_SCANNER_V4_DB
        value: internal.registry.example.com/rhacs/scanner-v4-db:4.11.0
```

**Available RELATED_IMAGE_* environment variables:**

| Environment Variable | Component |
|---------------------|-----------|
| `RELATED_IMAGE_MAIN` | Central (main) |
| `RELATED_IMAGE_CENTRAL_DB` | Central DB (PostgreSQL) |
| `RELATED_IMAGE_SCANNER` | Scanner (v2) |
| `RELATED_IMAGE_SCANNER_SLIM` | Scanner Slim |
| `RELATED_IMAGE_SCANNER_DB` | Scanner DB |
| `RELATED_IMAGE_SCANNER_DB_SLIM` | Scanner DB Slim |
| `RELATED_IMAGE_SCANNER_V4` | Scanner V4 (indexer/matcher) |
| `RELATED_IMAGE_SCANNER_V4_DB` | Scanner V4 DB |
| `RELATED_IMAGE_COLLECTOR` | Collector (for Sensor) |
| `RELATED_IMAGE_FACT` | FACT component |

---

#### Main Central Image

**CLI Usage:**
```bash
roxctl central generate k8s pvc \
  --main-image=quay.io/stackrox-io/main:4.11.0 \
  --image-defaults=rhacs
```

**How to check if used:**
```bash
kubectl get deployment central -n stackrox -o yaml | grep "image:"
```

**Operator Equivalent:**
```yaml
# Images are managed by the operator - no Central CR configuration
# For disconnected clusters: set RELATED_IMAGE_MAIN when installing operator
# See "Overriding Images for Disconnected Clusters" section above
```

**Notes:**
- **Cannot be set in Central CR**: Image references are managed by the operator
- `--image-defaults` (rhacs/opensource) is determined by which operator bundle you install
- For disconnected clusters: use `RELATED_IMAGE_MAIN` environment variable during operator installation
- Default images are tied to the operator version

---

#### Central DB Image

**CLI Usage:**
```bash
roxctl central generate k8s pvc \
  --central-db-image=registry.redhat.io/rhacs/central-db:4.11.0
```

**How to check if used:**
```bash
kubectl get deployment central-db -n stackrox -o yaml | grep "image:"
```

**Operator Equivalent:**
```yaml
# Images are managed by the operator - no Central CR configuration
# For disconnected clusters: set RELATED_IMAGE_CENTRAL_DB when installing operator
```

**Notes:**
- Use `RELATED_IMAGE_CENTRAL_DB` environment variable for disconnected clusters
- Operator automatically determines the correct DB image version

---

#### Scanner Images

**CLI Usage:**
```bash
roxctl central generate k8s pvc \
  --scanner-image=registry.redhat.io/rhacs/scanner:4.11.0 \
  --scanner-db-image=registry.redhat.io/rhacs/scanner-db:4.11.0
```

**How to check if used:**
```bash
kubectl get deployment scanner -n stackrox -o yaml | grep "image:"
```

**Operator Equivalent:**
```yaml
# Images are managed by the operator
# Scanner can be disabled entirely if not needed:
spec:
  scanner:
    scannerComponent: Disabled
```

**Notes:**
- Use `RELATED_IMAGE_SCANNER` and `RELATED_IMAGE_SCANNER_DB` for disconnected clusters
- Scanner can be disabled: `spec.scanner.scannerComponent: Disabled`

---

#### Scanner V4 Images

**CLI Usage:**
```bash
roxctl central generate k8s pvc \
  --scanner-v4-image=registry.redhat.io/rhacs/scanner-v4:4.11.0 \
  --scanner-v4-db-image=registry.redhat.io/rhacs/scanner-v4-db:4.11.0
```

**How to check if used:**
```bash
kubectl get deployment scanner-v4-indexer -n stackrox -o yaml | grep "image:"
```

**Operator Equivalent:**
```yaml
spec:
  scannerV4:
    scannerComponent: Enabled        # Explicitly enable Scanner V4
    # Images managed by operator via RELATED_IMAGE_SCANNER_V4 and RELATED_IMAGE_SCANNER_V4_DB
```

**Notes:**
- Scanner V4 enablement is controlled by `scannerComponent` field
- Use `RELATED_IMAGE_SCANNER_V4` and `RELATED_IMAGE_SCANNER_V4_DB` for disconnected clusters

---

### Security and Authentication

#### Administrator Password

**CLI Usage:**
```bash
# Custom password
roxctl central generate k8s pvc --password='MySecretPassword123'

# Auto-generated (default)
roxctl central generate k8s pvc
```

**How to check if used:**
```bash
kubectl get secret central-htpasswd -n stackrox -o yaml
# Check if secret exists with 'htpasswd' data field
```

**Operator Equivalent:**

**Custom password:**
```yaml
# 1. Create a secret first
apiVersion: v1
kind: Secret
metadata:
  name: custom-admin-password
  namespace: stackrox
type: Opaque
stringData:
  password: MySecretPassword123

---
# 2. Reference it in Central CR
apiVersion: platform.stackrox.io/v1alpha1
kind: Central
metadata:
  name: stackrox-central-services
  namespace: stackrox
spec:
  central:
    adminPasswordSecret:
      name: custom-admin-password
```

**Auto-generated (default):**
```yaml
spec:
  central:
    # No adminPasswordSecret specified
    # Operator auto-generates and stores in central-htpasswd secret
```

**Notes:**
- Default behavior: operator generates password and stores in `central-htpasswd` secret
- Secret data key must be `password`
- Password can be retrieved: `kubectl get secret central-htpasswd -n stackrox -o jsonpath='{.data.password}' | base64 -d`

---

#### Disable Admin Password

**CLI Usage:**
```bash
roxctl central generate k8s pvc --disable-admin-password
```

**How to check if used:**
```bash
kubectl get secret central-htpasswd -n stackrox
# Error: NotFound means it was disabled
```

**Operator Equivalent:**
```yaml
spec:
  central:
    adminPasswordGenerationDisabled: true
```

**Notes:**
- **Dangerous**: Only use if you've already configured an Identity Provider (IdP)
- Without an admin password or IdP, you cannot access Central

---

#### Default TLS Certificate

**CLI Usage:**
```bash
roxctl central generate k8s pvc \
  --default-tls-cert=/path/to/cert.pem \
  --default-tls-key=/path/to/key.pem
```

**How to check if used:**
```bash
kubectl get secret central-default-tls-cert -n stackrox
# Look for: default-tls.crt and default-tls.key
```

**Operator Equivalent:**
```yaml
# 1. Create a secret with your TLS certificate
apiVersion: v1
kind: Secret
metadata:
  name: my-tls-cert
  namespace: stackrox
type: kubernetes.io/tls
data:
  tls.crt: <base64-encoded-cert>
  tls.key: <base64-encoded-key>

---
# 2. Reference it in Central CR
spec:
  central:
    defaultTLSSecret:
      name: my-tls-cert
```

**Notes:**
- This is for user-facing TLS (HTTPS access to Central)
- If not specified, Central serves an internal self-signed certificate
- You must handle TLS termination at ingress/load balancer level if not set
- Secret must be type `kubernetes.io/tls` with keys `tls.crt` and `tls.key`

---

#### Custom CA Certificate

**CLI Usage:**
```bash
roxctl central generate k8s pvc --ca=/path/to/ca.pem
```

**How to check if used:**
```bash
kubectl get configmap additional-cas -n stackrox
# Or check central deployment for CA volume mounts
```

**Operator Equivalent:**
```yaml
spec:
  tls:
    additionalCAs:
      - name: my-ca                        # File basename
        content: |                         # PEM format
          -----BEGIN CERTIFICATE-----
          MIIDXTCCAkWgAwIBAgIJAKJ...
          -----END CERTIFICATE-----
```

**Notes:**
- Used for trusting custom/internal CAs
- `name` must be a valid filename
- `content` must be PEM-formatted certificate
- Multiple CAs can be added as list items

---

### Persistence and Database

#### Central DB Password

**CLI Usage:**
```bash
# Auto-generated in all cases when using roxctl
```

**How to check current:**
```bash
kubectl get secret central-db-password -n stackrox -o yaml
```

**Operator Equivalent:**

**Auto-generated (default):**
```yaml
# No configuration needed - operator generates automatically
# Password stored in central-db-password secret
```

**Custom password:**
```yaml
# 1. Create secret
apiVersion: v1
kind: Secret
metadata:
  name: my-db-password
  namespace: stackrox
type: Opaque
stringData:
  password: MyDatabasePassword123

---
# 2. Reference in CR
spec:
  central:
    db:
      passwordSecret:
        name: my-db-password
```

**Notes:**
- Default: operator auto-generates and stores in `central-db-password`
- Secret key must be named `password`

---

#### External Database (Connection String)

**CLI Usage:**
```bash
# Not supported via roxctl - requires manual editing of generated manifests
```

**How to check if used:**
```bash
kubectl get deployment central -n stackrox -o yaml | grep DB_CONNECTION
# Look for external database connection string
```

**Operator Equivalent:**
```yaml
spec:
  central:
    db:
      connectionString: "host=my-postgres.example.com port=5432 user=central database=stackrox sslmode=require"
      passwordSecret:
        name: external-db-password     # Required with external DB
```

**Notes:**
- When `connectionString` is set, operator does not manage Central DB deployment
- You must provide the password secret separately
- External DB must be PostgreSQL-compatible

---

#### Database Connection Pool Size

**CLI Usage:**
```bash
# Not configurable via roxctl
```

**Operator Equivalent:**
```yaml
spec:
  central:
    db:
      connectionPoolSize:
        minConnections: 10              # Default: 10
        maxConnections: 90              # Default: 90
```

**Notes:**
- Only applicable when operator manages the database (not external)
- Minimum value: 1 for both fields
- Tune based on workload and database resources

---

### Platform-Specific Options

#### OpenShift Version

**CLI Usage:**
```bash
roxctl central generate openshift pvc --openshift-version=4
```

**How to check:**
```bash
# Check if running on OpenShift
oc version
```

**Operator Equivalent:**
```yaml
# Not needed - Operator auto-detects OpenShift vs Kubernetes
```

**Notes:**
- Operator automatically detects the platform
- Generates appropriate resources (Routes, SCCs) for OpenShift
- No manual configuration required

---

#### OpenShift Monitoring

**CLI Usage:**
```bash
roxctl central generate openshift pvc --openshift-monitoring=true
```

**How to check if used:**
```bash
oc get servicemonitor -n stackrox
# Look for: ServiceMonitor resources
```

**Operator Equivalent:**
```yaml
spec:
  monitoring:
    openshift:
      enabled: true                      # Default: true on OpenShift 4
```

**Notes:**
- Default behavior: enabled on OpenShift 4, disabled elsewhere
- Creates ServiceMonitor resources for OpenShift monitoring stack
- Set to `false` to disable integration

---

#### Istio Support

**CLI Usage:**
```bash
roxctl central generate k8s pvc --istio-support=1.7
```

**How to check if used:**
```bash
kubectl get deployment central -n stackrox -o yaml | grep -i istio
# Look for: istio-specific annotations or sidecars
```

**Operator Equivalent:**
```yaml
# Not directly supported - Istio integration happens automatically via:
# 1. Namespace labeling for sidecar injection
# 2. Istio automatically injects sidecars based on namespace labels

# Enable Istio injection on the namespace:
kubectl label namespace stackrox istio-injection=enabled
```

**Notes:**
- Modern Istio versions use namespace labeling for injection
- Older `--istio-support` flag added specific network policies - these are less relevant now
- Configure Istio integration at the namespace level, not in Central CR

---

### Advanced Configuration

#### Offline Mode

**CLI Usage:**
```bash
roxctl central generate k8s pvc --offline=true
```

**How to check if used:**
```bash
kubectl get deployment central -n stackrox -o yaml | grep OFFLINE
# Look for: ROX_OFFLINE_MODE environment variable
```

**Operator Equivalent:**
```yaml
spec:
  egress:
    connectivityPolicy: Offline          # Options: Online, Offline
```

**Notes:**
- `Offline`: Disables external network access, including vulnerability updates
- `Online` (default): Allows internet access for updates
- Offline mode requires manual vulnerability database updates

---

#### Telemetry

**CLI Usage:**
```bash
# Enabled by default on release builds
roxctl central generate k8s pvc --enable-telemetry=true

# Disabled
roxctl central generate k8s pvc --enable-telemetry=false
```

**How to check if used:**
```bash
kubectl get deployment central -n stackrox -o yaml | grep TELEMETRY
```

**Operator Equivalent:**

**Enable (default for release builds):**
```yaml
spec:
  central:
    telemetry:
      enabled: true
      # Optional custom endpoint
      storage:
        endpoint: https://custom-endpoint.example.com
        key: my-api-key
```

**Disable:**
```yaml
spec:
  central:
    telemetry:
      enabled: false
```

**Notes:**
- Default: enabled on release versions, disabled on development builds
- Telemetry sends anonymous usage data to Red Hat
- Storage endpoint/key usually not needed - uses defaults

---

#### Declarative Configuration

**CLI Usage:**
```bash
roxctl central generate k8s pvc \
  --declarative-config-config-maps=my-config-cm \
  --declarative-config-secrets=my-config-secret
```

**How to check if used:**
```bash
kubectl get deployment central -n stackrox -o yaml | grep -A 10 "declarative-config"
# Look for: volume mounts for declarative configuration
```

**Operator Equivalent:**
```yaml
spec:
  central:
    declarativeConfiguration:
      configMaps:
        - name: my-config-cm
        - name: another-config-cm
      secrets:
        - name: my-config-secret
```

**Notes:**
- ConfigMaps and Secrets must exist in the same namespace
- Used for automating Central configuration (notifiers, auth providers, etc.)
- Supports multiple ConfigMaps and Secrets

---

#### Pod Security Policies

**CLI Usage:**
```bash
roxctl central generate k8s pvc --enable-pod-security-policies
```

**How to check if used:**
```bash
kubectl get psp | grep stackrox
# Look for: PodSecurityPolicy resources
```

**Operator Equivalent:**
```yaml
# Not supported - PSPs are deprecated in Kubernetes 1.25+
# Use Pod Security Standards instead at namespace level
```

**Notes:**
- PodSecurityPolicies are deprecated and removed in Kubernetes 1.25+
- For Kubernetes 1.25+, use Pod Security Standards/Admission
- Operator does not create PSP resources

---

## Additional Resources

- [Operator EXTENDING_CRDS.md](operator/EXTENDING_CRDS.md) - How CRD fields are structured
- [Operator DEFAULTING.md](operator/DEFAULTING.md) - Default values and behavior
- [Helm Chart README](image/templates/README.md) - Understanding underlying Helm charts
- Official Red Hat ACS Documentation: https://docs.openshift.com/acs/

## Migration Checklist

- [ ] Identify current roxctl command (from scripts/history)
- [ ] Document current configuration (storage, exposure, images)
- [ ] Install StackRox Operator (Helm or OLM)
- [ ] Configure image overrides if using disconnected cluster (RELATED_IMAGE_* variables)
- [ ] Create Central CR with equivalent configuration
- [ ] Apply Central CR
- [ ] Verify deployment: `kubectl get central -n stackrox`
- [ ] Check pods are running: `kubectl get pods -n stackrox`
- [ ] Verify Central is accessible
- [ ] Test login with admin password
- [ ] Verify sensors can connect
- [ ] Remove old roxctl-generated resources (if applicable)

## Getting Help

If you encounter issues during migration:
1. Check operator logs:
   ```bash
   # Find the operator deployment first
   kubectl get deployment -A | grep rhacs-operator
   # Then check logs (adjust namespace/deployment name as needed)
   kubectl logs -n rhacs-operator deployment/rhacs-operator-controller-manager
   ```
2. Check Central CR status: `kubectl describe central stackrox-central-services -n stackrox`
3. Review conditions: `kubectl get central stackrox-central-services -n stackrox -o yaml`
4. Contact Red Hat support with your Central CR and operator logs
