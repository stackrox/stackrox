# Master Options List - roxctl central generate

This document tracks all options available in `roxctl central generate` commands and their impact on generated manifests.

**Legend:**
- ✓ = Available in this mode
- ✗ = Not available in this mode

---

## Storage-Specific Options (PVC modes only)

### --db-name
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✗, k8s-hostpath ✗
**Default:** `central-db`
**Description:** External volume name for Central DB
**Impact:** Changes PVC name and its reference in StatefulSet
**Affected files:**
- `central/01-central-11-db-pvc.yaml` - PVC metadata.name
- `central/01-central-12-central-db.yaml` - PVC claimName reference in volumes
**Detection method:**
```bash
kubectl get pvc -n stackrox -l app=central-db -o jsonpath='{.items[0].metadata.name}'
```
**Expected output:**
- Default: `central-db`
- If custom: the custom name specified

---

### --db-size
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✗, k8s-hostpath ✗
**Default:** `100`
**Description:** External volume size in Gi for Central DB
**Impact:** Changes PVC storage request size
**Affected files:**
- `central/01-central-11-db-pvc.yaml` - spec.resources.requests.storage
**Detection method:**
```bash
kubectl get pvc -n stackrox -l app=central-db -o jsonpath='{.items[0].spec.resources.requests.storage}'
```
**Expected output:**
- Default: `100Gi`
- If custom: `<value>Gi` where value is the specified size

---

### --db-storage-class
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✗, k8s-hostpath ✗
**Default:** (none - optional if default StorageClass exists)
**Description:** Storage class name for Central DB
**Impact:** Sets explicit storageClassName in PVC
**Affected files:**
- `central/01-central-11-db-pvc.yaml` - spec.storageClassName
**Detection method:**
```bash
kubectl get pvc -n stackrox -l app=central-db -o jsonpath='{.items[0].spec.storageClassName}'
```
**Expected output:**
- Default: empty or cluster default storage class
- If custom: the specified storage class name

---

## Storage-Specific Options (HostPath modes only)

### --db-hostpath
**Available in:** openshift-pvc ✗, k8s-pvc ✗, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `/var/lib/stackrox-central`
**Description:** Path on the host for database storage
**Impact:** Changes hostPath volume path in central-db Deployment
**Affected files:**
- `central/01-central-12-central-db.yaml` - volumes[disk].hostPath.path
**Detection method:**
```bash
kubectl get deployment -n stackrox central-db -o jsonpath='{.spec.template.spec.volumes[?(@.name=="disk")].hostPath.path}'
```
**Expected output:**
- Default: `/var/lib/stackrox-central`
- If custom: the specified path

---

### --db-node-selector-key
**Available in:** openshift-pvc ✗, k8s-pvc ✗, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (none)
**Description:** Node selector key (e.g. kubernetes.io/hostname)
**Impact:** Adds nodeSelector to central-db Deployment (requires --db-node-selector-value)
**Affected files:**
- `central/01-central-12-central-db.yaml` - spec.template.spec.nodeSelector
**Detection method:**
```bash
kubectl get deployment -n stackrox central-db -o jsonpath='{.spec.template.spec.nodeSelector}'
```
**Expected output:**
- Default: `{}` (empty)
- If set: `{"<key>":"<value>"}`

---

### --db-node-selector-value
**Available in:** openshift-pvc ✗, k8s-pvc ✗, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (none)
**Description:** Node selector value
**Impact:** Used together with --db-node-selector-key to add nodeSelector
**Affected files:**
- Same as --db-node-selector-key
**Detection method:**
- Same as --db-node-selector-key

---

## Platform-Specific Options (OpenShift only)

### --openshift-monitoring
**Available in:** openshift-pvc ✓, k8s-pvc ✗, openshift-hostpath ✓, k8s-hostpath ✗
**Default:** `auto`
**Description:** Integration with OpenShift 4 monitoring
**Impact:** Controls whether OpenShift monitoring resources are created
**Affected files:**
- `central/99-openshift-monitoring.yaml`
- `central/99-scanner-v4-openshift-monitoring.yaml`
**Detection method:**
```bash
kubectl get servicemonitor -n stackrox
```
**Expected output:**
- Default (auto): ServiceMonitor and related monitoring resources exist
**Note:** OpenShift only. With default `auto`, monitoring resources are created.

---

### --openshift-version
**Available in:** openshift-pvc ✓, k8s-pvc ✗, openshift-hostpath ✓, k8s-hostpath ✗
**Default:** `0`
**Description:** The OpenShift major version (3 or 4) to deploy on
**Impact:** Minimal in manifests (may affect Helm values and scripts)
**Affected files:**
- Helm values files
**Detection method:** Not easily detectable from deployed resources
**Note:** Primarily affects generation logic, not deployed manifests significantly. Default `0` means auto-detect.

---

## Global Options (Available in all modes)

### --backup-bundle
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (none)
**Description:** Path to the backup bundle from which to restore keys and certificates
**Impact:** Affects TLS secrets and certificates
**Affected files:**
- `central/01-central-05-tls-secret.yaml` and related TLS secrets
**Detection method:**
```bash
kubectl get secret -n stackrox central-tls -o jsonpath='{.data}'
```
**Note:** This is a one-time import operation. Replaces auto-generated certificates with those from backup bundle.

---

### --ca
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (none)
**Description:** Path to a custom CA certificate to use (PEM format)
**Impact:** Replaces auto-generated CA certificate in TLS secrets
**Affected files:**
- `central/01-central-05-tls-secret.yaml` and related TLS secrets
**Detection method:**
```bash
kubectl get secret -n stackrox central-tls -o jsonpath='{.data.ca\.pem}' | base64 -d
```
**Expected output:**
- Default: Auto-generated StackRox CA
- If custom: The provided custom CA certificate
**Note:** Requires manual inspection of certificate to determine if custom CA was used

---

### --central-db-image
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (determined by --image-defaults)
**Description:** The central-db image to use
**Impact:** Overrides central-db image in Deployment
**Affected files:**
- `central/01-central-12-central-db.yaml`
**Detection method:**
```bash
kubectl get deployment -n stackrox central-db -o jsonpath='{.spec.template.spec.containers[0].image}'
```
**Expected output:**
- Default (rhacs): `registry.redhat.io/advanced-cluster-security/rhacs-central-db-rhel8:4.x.x`
- Default (opensource): `quay.io/stackrox-io/central-db:4.x.x`
- If custom: The specified custom image

---

### --declarative-config-config-maps
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `[]`
**Description:** List of config maps to add as declarative configuration mounts in central
**Impact:** Mounts additional configmaps into Central deployment
**Affected files:**
- `central/01-central-13-deployment.yaml`
**Detection method:**
```bash
kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.volumes[*].configMap.name}' | tr ' ' '\n' | sort
```
**Expected output:**
- Default: Only standard ConfigMaps (e.g., `additional-ca-volume`)
- If set: Additional ConfigMap names mounted at `/run/stackrox.io/declarative-configuration/<configmap-name>`
**Format:** Comma-separated list (e.g., `cm1,cm2,cm3`)

---

### --declarative-config-secrets
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `[]`
**Description:** List of secrets to add as declarative configuration mounts in central
**Impact:** Mounts additional secrets into Central deployment
**Affected files:**
- `central/01-central-13-deployment.yaml`
**Detection method:**
```bash
kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.volumes[*].secret.secretName}' | tr ' ' '\n' | sort
```
**Expected output:**
- Default: Only standard secrets (htpasswd, tls, monitoring-tls)
- If set: Additional secret names mounted at `/run/stackrox.io/declarative-configuration/<secret-name>`
**Format:** Comma-separated list (e.g., `secret1,secret2,secret3`)
**Note:** Secrets are marked as `optional: true` in volume definitions

---

### --default-tls-cert
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (none)
**Description:** PEM cert bundle file
**Impact:** Replaces auto-generated TLS certificate in Central TLS secret
**Affected files:**
- `central/01-central-05-tls-secret.yaml`
**Detection method:**
```bash
kubectl get secret -n stackrox central-tls -o jsonpath='{.data.cert\.pem}' | base64 -d
```
**Note:** Requires manual inspection of certificate to determine if custom cert was used. Must be used together with --default-tls-key.

---

### --default-tls-key
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (none)
**Description:** PEM private key file
**Impact:** Replaces auto-generated TLS private key in Central TLS secret
**Affected files:**
- `central/01-central-05-tls-secret.yaml`
**Detection method:**
```bash
kubectl get secret -n stackrox central-tls -o jsonpath='{.data.key\.pem}'
```
**Note:** Private key content cannot be easily validated. Must be used together with --default-tls-cert.

---

### --disable-admin-password
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `false`
**Description:** Disable the administrator password (only use if IdP configured)
**Impact:** Minimal - htpasswd secret still created but admin password may not be used for auth
**Affected files:**
- `central/01-central-04-htpasswd-secret.yaml` (different hash but secret exists)
**Detection method:** Not easily detectable from manifests - check Central auth configuration at runtime
**Note:** Does NOT remove htpasswd secret, likely affects Central runtime auth behavior only

---

### --enable-pod-security-policies
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `false`
**Description:** Create PodSecurityPolicy resources (for pre-v1.25 Kubernetes)
**Impact:** Creates PodSecurityPolicy resources (NEW FILES)
**Affected files:**
- **NEW:** `central/01-central-02-psps.yaml`
- **NEW:** `central/01-central-02-db-psps.yaml`
- **NEW:** `scanner/02-scanner-01-psps.yaml`
- **NEW:** `scanner-v4/02-scanner-v4-01-psps.yaml`
**Detection method:**
```bash
kubectl get psp | grep stackrox
```
**Expected output:**
- Default: No PSPs
- If enabled: stackrox-related PSPs listed
**Note:** Only needed for pre-v1.25 Kubernetes with PSP admission controller enabled

---

### --enable-telemetry
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `true`
**Description:** Whether to enable telemetry
**Impact:** Controls telemetry-related environment variables (NOT ROX_OFFLINE_MODE)
**Affected files:**
- `central/01-central-13-deployment.yaml`
**Detection method:**
```bash
# Check if telemetry is enabled
kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="ROX_TELEMETRY_ENDPOINT")].value}'
# Or check if explicitly disabled
kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="ROX_TELEMETRY_STORAGE_KEY_V1")].value}'
```
**Expected output:**
- Default (enabled): ROX_TELEMETRY_ENDPOINT and ROX_TELEMETRY_API_WHITELIST are present
- If disabled: ROX_TELEMETRY_STORAGE_KEY_V1="DISABLED" is set, telemetry endpoint vars removed
**Note:** This is SEPARATE from --offline. Telemetry controls data collection, offline controls internet connectivity.

---

### --image-defaults
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `rhacs`
**Description:** Default container images settings (rhacs, opensource)
**Impact:** Changes image repositories, names, and tags across ALL deployments (MAJOR IMPACT)
**Affected files:**
- `central/01-central-13-deployment.yaml`
- `central/01-central-12-central-db.yaml`
- `central/02-config-controller-02-deployment.yaml`
- `scanner/02-scanner-06-deployment.yaml`
- `scanner-v4/*-deployment.yaml`
- Helm charts and setup scripts
**Detection method:**
```bash
kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.containers[0].image}'
```
**Expected output:**
- Default (rhacs): `registry.redhat.io/advanced-cluster-security/rhacs-main-rhel8:4.x.x`
- If opensource: `quay.io/stackrox-io/main:4.x.x`
**Image patterns:**
- rhacs: `registry.redhat.io/advanced-cluster-security/rhacs-<component>-rhel8:<version>`
- opensource: `quay.io/stackrox-io/<component>:<version>`
**Note:** This is a MAJOR change affecting ALL component images (Central, DB, Scanner, Scanner-v4)

---

### --istio-support
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (none)
**Description:** Generate deployment files supporting the given Istio version (valid: 1.0-1.7)
**Impact:** Appends Istio DestinationRule resources to service YAML files to disable Istio mTLS on specific ports (since StackRox uses built-in mTLS)
**Affected files:**
- `central/01-central-14-service.yaml` - Appends DestinationRule `central-internal-no-istio-mtls` (disables mTLS on port 443)
- `scanner/02-scanner-07-service.yaml` - Appends two DestinationRules:
  - `scanner-internal-no-istio-mtls` (disables mTLS on ports 8080, 8443)
  - `scanner-db-internal-no-istio-mtls` (disables mTLS on port 5432)
- `scanner-v4/02-scanner-v4-08-db-service.yaml` - Appends DestinationRule `scanner-v4-db-internal-no-istio-mtls`
- `scanner-v4/02-scanner-v4-08-indexer-service.yaml` - Appends DestinationRule `scanner-v4-indexer-internal-no-istio-mtls`
- `scanner-v4/02-scanner-v4-08-matcher-service.yaml` - Appends DestinationRule `scanner-v4-matcher-internal-no-istio-mtls`
**Detection method:**
```bash
kubectl get destinationrule -n stackrox
```
**Expected output:**
- Default: No DestinationRules
- If set: Multiple DestinationRules with names like `*-internal-no-istio-mtls`
**Purpose:** Each DestinationRule configures Istio to disable its own mTLS (`tls.mode: DISABLE`) for specific service ports because StackRox components use their own built-in mTLS implementation.

---

### --lb-type
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `none`
**Description:** The method of exposing Central
**Valid values:** 
- OpenShift: route, lb, np, none
- K8s: lb, np, none
**Impact:** Creates Service or Route resources for exposing Central
**Affected files:**
- **NEW (when lb/np):** `central/01-central-15-exposure.yaml`
**Detection method:**
```bash
# For LoadBalancer
kubectl get svc -n stackrox central-loadbalancer 2>/dev/null && echo "lb" || echo "not-lb"
# For Route (OpenShift)
kubectl get route -n stackrox central 2>/dev/null && echo "route" || echo "not-route"
# For NodePort - check service type
kubectl get svc -n stackrox central -o jsonpath='{.spec.type}'
```
**Expected output:**
- Default (none): No LoadBalancer service, no Route, service type is ClusterIP
- If `lb`: Service `central-loadbalancer` with type LoadBalancer exists
- If `route`: Route `central` exists (OpenShift only)
- If `np`: Service type is NodePort

---

### --main-image (-i)
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (determined by --image-defaults)
**Description:** The main image to use
**Impact:** Overrides main/Central image in Central deployment and config controller
**Affected files:**
- `central/01-central-13-deployment.yaml`
- `central/02-config-controller-02-deployment.yaml`
- Scanner-v4 init containers
**Detection method:**
```bash
kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.containers[?(@.name=="central")].image}'
```
**Expected output:**
- Default: Determined by --image-defaults (rhacs: `quay.io/rhacs-eng/main:4.x.x`, opensource: `quay.io/stackrox-io/main:4.x.x`)
- If custom: The specified custom image
**Note:** Also affects Scanner-v4 init containers that use the main image

---

### --offline
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `false`
**Description:** Whether to run StackRox in offline mode
**Impact:** Sets ROX_OFFLINE_MODE environment variable in Central deployment
**Affected files:**
- `central/01-central-13-deployment.yaml`
**Detection method:**
```bash
kubectl get deploy -n stackrox central -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="ROX_OFFLINE_MODE")].value}'
```
**Expected output:**
- Default: `false`
- If `--offline=true`: `true`

---

### --output-dir
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (none - outputs to stdout if not specified)
**Description:** The directory to output the deployment bundle to
**Impact:** Does not affect manifest content, only output location
**Detection method:** N/A - does not affect manifests

---

### --output-format
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `kubectl`
**Description:** The deployment tool to use (kubectl, helm, helm-values)
**Impact:** Changes output format but maintains same configuration
**Detection method:** N/A - affects output structure, not deployed resource content
**Note:** Values: kubectl (YAML manifests), helm (Helm chart), helm-values (values file only)

---

### --password (-p)
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (autogenerated)
**Description:** Administrator password
**Impact:** Sets specific admin password hash in htpasswd secret
**Affected files:**
- `central/01-central-04-htpasswd-secret.yaml`
- `password` file
**Detection method:** Not detectable - password is hashed
**Note:** Password hash changes every time even with same password. Cannot reliably detect if custom password was specified.

---

### --plaintext-endpoints
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (none)
**Description:** The ports or endpoints to use for plaintext (unencrypted) exposure
**Impact:** Not fully tested - may add plaintext endpoint configuration
**Detection method:** To be determined
**Note:** Untested in Phase 4

---

### --scanner-db-image
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (determined by --image-defaults)
**Description:** The scanner-db image to use
**Impact:** Overrides scanner-db image in Scanner deployment
**Affected files:**
- `scanner/02-scanner-06-deployment.yaml` (scanner-db container)
**Detection method:**
```bash
kubectl get deploy -n stackrox scanner-db -o jsonpath='{.spec.template.spec.containers[0].image}'
```
**Expected output:**
- Default (rhacs): `registry.redhat.io/advanced-cluster-security/rhacs-scanner-db-rhel8:4.x.x`
- Default (opensource): `quay.io/stackrox-io/scanner-db:4.x.x`
- If custom: The specified custom image

---

### --scanner-image
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (determined by --image-defaults)
**Description:** The scanner image to use
**Impact:** Overrides scanner image in Scanner deployment
**Affected files:**
- `scanner/02-scanner-06-deployment.yaml`
**Detection method:**
```bash
kubectl get deploy -n stackrox scanner -o jsonpath='{.spec.template.spec.containers[?(@.name=="scanner")].image}'
```
**Expected output:**
- Default (rhacs): `registry.redhat.io/advanced-cluster-security/rhacs-scanner-rhel8:4.x.x`
- Default (opensource): `quay.io/stackrox-io/scanner:4.x.x`
- If custom: The specified custom image

---

### --scanner-v4-db-image
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (determined by --image-defaults)
**Description:** The scanner-v4-db image to use
**Impact:** Overrides scanner-v4-db image in Scanner V4 DB deployment
**Affected files:**
- `scanner-v4/02-scanner-v4-07-db-deployment.yaml`
**Detection method:**
```bash
kubectl get deploy -n stackrox scanner-v4-db -o jsonpath='{.spec.template.spec.containers[0].image}'
```
**Expected output:**
- Default (rhacs): `registry.redhat.io/advanced-cluster-security/rhacs-scanner-v4-db-rhel8:4.x.x`
- Default (opensource): `quay.io/stackrox-io/scanner-v4-db:4.x.x`
- If custom: The specified custom image

---

### --scanner-v4-image
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (determined by --image-defaults)
**Description:** The scanner-v4 image to use
**Impact:** Overrides scanner-v4 image in Matcher and Indexer deployments
**Affected files:**
- `scanner-v4/02-scanner-v4-07-matcher-deployment.yaml`
- `scanner-v4/02-scanner-v4-07-indexer-deployment.yaml`
**Detection method:**
```bash
kubectl get deploy -n stackrox scanner-v4-matcher -o jsonpath='{.spec.template.spec.containers[0].image}'
kubectl get deploy -n stackrox scanner-v4-indexer -o jsonpath='{.spec.template.spec.containers[0].image}'
```
**Expected output:**
- Default (rhacs): `registry.redhat.io/advanced-cluster-security/rhacs-scanner-v4-rhel8:4.x.x`
- Default (opensource): `quay.io/stackrox-io/scanner-v4:4.x.x`
- If custom: The specified custom image

---

## Meta/Client Options (Do not affect manifest content)

### --direct-grpc
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `false`
**Description:** Use direct gRPC (roxctl client behavior)
**Impact:** Client-side only - does not affect generated manifests
**Detection method:** N/A

---

### --endpoint (-e)
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `localhost:8443`
**Description:** Endpoint for service to contact (roxctl client behavior)
**Impact:** Client-side only - does not affect generated manifests
**Detection method:** N/A

---

### --force-http1
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `false`
**Description:** Always use HTTP/1 for all connections (roxctl client behavior)
**Impact:** Client-side only - does not affect generated manifests
**Detection method:** N/A

---

### --help (-h)
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `false`
**Description:** Show help
**Impact:** Client-side only - does not affect generated manifests
**Detection method:** N/A

---

### --insecure
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `false`
**Description:** Enable insecure connection options (roxctl client behavior)
**Impact:** Client-side only - does not affect generated manifests
**Detection method:** N/A

---

### --insecure-skip-tls-verify
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `false`
**Description:** Skip TLS certificate validation (roxctl client behavior)
**Impact:** Client-side only - does not affect generated manifests
**Detection method:** N/A

---

### --no-color
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `false`
**Description:** Disable color output (roxctl client behavior)
**Impact:** Client-side only - does not affect generated manifests
**Detection method:** N/A

---

### --plaintext
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `false`
**Description:** Use plaintext connection (roxctl client behavior)
**Impact:** Client-side only - does not affect generated manifests
**Detection method:** N/A

---

### --server-name (-s)
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (derived from endpoint)
**Description:** TLS ServerName to use for SNI (roxctl client behavior)
**Impact:** Client-side only - does not affect generated manifests
**Detection method:** N/A

---

### --token-file
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** (none)
**Description:** Use API token in file to authenticate (roxctl client behavior)
**Impact:** Client-side only - does not affect generated manifests
**Detection method:** N/A

---

### --use-current-k8s-context
**Available in:** openshift-pvc ✓, k8s-pvc ✓, openshift-hostpath ✓, k8s-hostpath ✓
**Default:** `false`
**Description:** Use current kubeconfig context (roxctl client behavior)
**Impact:** Client-side only - does not affect generated manifests
**Detection method:** N/A

---

## Random/Non-Deterministic Elements (Phase 3 Findings)

The following values change between identical runs and should be **ignored** when comparing manifests:

1. **Admin password hash** - in `central/01-central-04-htpasswd-secret.yaml`
   - Field: `stringData.htpasswd`
   - Pattern: `admin:$2a$05$...`

2. **TLS Certificates and Keys** - in `central/01-central-05-tls-secret.yaml` and `central/01-central-05-db-tls-secret.yaml`
   - CA certificate serial numbers and keys
   - Service certificates (Central, Central DB) and private keys
   - All PEM-encoded certificate and key content

3. **JWT signing keys** - in `central/01-central-05-tls-secret.yaml`
   - Field: `stringData.jwt-key.pem`
   - RSA private key for JWT signing

4. **Password file** - `password` file containing plaintext admin password

**Pattern to ignore in diffs:** Any changes to Secret resources containing certificates, keys, or passwords.

---

## Summary Statistics

**Total options:** 42
**Options affecting manifests:** 29
**Client-side only options:** 11
**Output control options:** 2 (--output-dir, --output-format)
**Platform-specific:** 2 (OpenShift only)
**Storage-specific:** 6 (3 for PVC, 3 for HostPath)
**Fully tested:** 24
**Partially tested/untested:** 5 (--password, --plaintext-endpoints, --ca, --default-tls-cert, --default-tls-key, --backup-bundle)

---

## Key Findings for Migration Tool

### High-Priority Options (Commonly Used)

These options are frequently used and have significant impact on deployments:

1. **Storage configuration:** 
   - `--db-size`, `--db-storage-class` (PVC modes)
   - `--db-hostpath`, `--db-node-selector-key/value` (HostPath modes)

2. **Exposure:** 
   - `--lb-type` (especially `route` on OpenShift)

3. **Images:** 
   - `--image-defaults` (affects ALL images)
   - `--main-image`

4. **Operational:** 
   - `--offline`
   - `--enable-telemetry`

5. **OpenShift-specific:** 
   - `--openshift-monitoring`

### Options Creating New Resources

These options create entirely new files/resources when enabled:

- `--lb-type` → Creates `01-central-15-exposure.yaml` (Service or Route)
- `--enable-pod-security-policies` → Creates 4 new PSP YAML files
- `--openshift-monitoring` → Creates ServiceMonitor resources

### Options With Cumulative/Multi-Value Effects

These options can have multiple values:

- `--declarative-config-secrets` → Comma-separated list of secrets
- `--declarative-config-config-maps` → Comma-separated list of ConfigMaps
- Image overrides → Can override multiple component images independently

### Detection Strategy for Migration Tool

The migration tool should:

1. **Query deployed resources** using kubectl commands from this document
2. **Compare values against known defaults** to detect customizations
3. **Infer which options were likely specified** based on detected values
4. **Generate equivalent Central CR fields** from inferred options

### Migration Considerations

**Options that cannot be directly migrated:**
- `--backup-bundle` - One-time import, not ongoing configuration
- `--ca`, `--default-tls-cert`, `--default-tls-key` - Custom CA/certs must be provided as separate secrets before CR creation
- `--password` - Admin password is auto-generated by operator (not configurable in CR)
- `--enable-pod-security-policies` - Deprecated in modern Kubernetes, operator may not support

**Options requiring manual intervention:**
- Custom certificates - User must create secrets before creating CR
- Declarative configuration - ConfigMaps/Secrets must exist before CR creation
- Image pull secrets - Must be configured appropriately

**Options with detection challenges:**
- `--disable-admin-password` - Affects runtime behavior, not easily detectable from manifests
- `--password` - Cannot reliably detect if custom password was used (hashes differ each time)
- `--openshift-version` - Primarily affects generation logic, minimal manifest impact

### Remaining Untested Options

The following options have not been fully tested:

- `--password` - Likely only affects htpasswd secret hash (not detectable)
- `--plaintext-endpoints` - May add plaintext endpoint configuration
- Certificate options (`--ca`, `--default-tls-cert`, `--default-tls-key`, `--backup-bundle`) - Affect TLS secrets but detection requires manual certificate inspection
