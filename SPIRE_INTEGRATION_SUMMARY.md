# SPIRE Integration for Sensor → Central - Hackathon Summary

## 🎯 What Was Built

A proof-of-concept SPIRE integration that allows Sensor to authenticate to Central using SPIFFE workload identities instead of manually-distributed certificates. This demonstrates **zero-trust workload identity** as an alternative to the current certificate-based mTLS.

## ✅ Completed Work

### 1. Go Dependencies
- Added `github.com/spiffe/go-spiffe/v2@v2.5.0` to `go.mod`
- All dependencies resolved and tidied

### 2. Central-side Changes (SPIRE Identity Extraction)

**Files Created:**
- `central/sensor/service/spiffe_extractor.go` - Extracts SPIFFE IDs from client certificates
- `central/sensor/service/spiffe_identity.go` - Implements `authn.Identity` for SPIFFE identities

**Files Modified:**
- `central/main.go` - Registered SPIFFE extractor alongside existing mTLS extractor

**How it Works:**
- Central's gRPC server now has a SPIFFE identity extractor
- When Sensor connects, Central checks for SPIFFE ID in the certificate's URI SAN
- Accepts `spiffe://stackrox.local/ns/stackrox/sa/sensor` as `SENSOR_SERVICE`
- Logs: `✅ SPIRE: Authenticated SENSOR_SERVICE via SPIFFE ID: spiffe://...`
- Falls back to certificate-based mTLS if no SPIFFE ID found

### 3. Sensor-side Changes (SPIRE Connection)

**Files Created:**
- `sensor/common/centralclient/spire_connection.go` - SPIRE-based gRPC connection helper

**Files Modified:**
- `sensor/common/centralclient/grpc_connection.go` - Modified to try SPIRE first, then fall back

**How it Works:**
- Sensor checks for SPIRE socket at `/spire-workload-api/spire-agent.sock`
- If found, obtains X.509-SVID from SPIRE Workload API
- Uses SPIRE credentials to create gRPC connection to Central
- Logs: `🔐 SPIRE: Socket found, attempting SPIRE-based connection to Central`
- Logs: `🎉 SPIRE: Successfully created gRPC connection to Central`
- Falls back to traditional mTLS if SPIRE unavailable

### 4. Helm Chart Changes

**Files Modified:**
- `image/templates/helm/stackrox-central/templates/01-central-13-deployment.yaml.htpl`
  - Added SPIRE CSI volume (OpenShift only)
  - Added volumeMount to central container at `/spire-workload-api`

- `image/templates/helm/stackrox-secured-cluster/templates/sensor.yaml.htpl`
  - Added SPIRE CSI volume (OpenShift only)
  - Added volumeMount to sensor container at `/spire-workload-api`

**Files Created:**
- `image/templates/helm/stackrox-central/templates/spire-registrations.yaml`
  - ClusterSPIFFEID for Central workloads

- `image/templates/helm/stackrox-secured-cluster/templates/spire-registrations.yaml`
  - ClusterSPIFFEID for Sensor workloads

**How it Works:**
- On OpenShift, SPIFFE CSI driver automatically mounts the SPIRE agent socket
- ClusterSPIFFEID resources tell SPIRE which pods should get SVIDs
- SPIRE Agent attests the pods and provides X.509-SVIDs via the CSI volume

### 5. Demo Script

**File Created:**
- `scripts/demo-spire.sh` - Automated verification and demo presentation script

**What it Does:**
- Verifies SPIRE components are running
- Checks ClusterSPIFFEID registrations
- Verifies SPIRE volumes are mounted in pods
- Searches logs for SPIRE authentication messages
- Provides formatted demo summary with key talking points

## 🚀 Next Steps (When OpenShift 4.19 Cluster is Ready)

### 1. Install SPIRE on OpenShift 4.19

In the OpenShift Web Console:
1. Navigate to **OperatorHub**
2. Search for **"Zero Trust Workload Identity Manager"**
3. Install the operator (keep defaults)

Then apply SPIRE components:
```bash
cat <<EOF | oc apply -f -
apiVersion: operator.openshift.io/v1alpha1
kind: SpireServer
metadata:
  name: cluster
spec:
  trustDomain: stackrox.local
  clusterName: demo-cluster
---
apiVersion: operator.openshift.io/v1alpha1
kind: SpireAgent
metadata:
  name: cluster
spec:
  trustDomain: stackrox.local
  clusterName: demo-cluster
  nodeAttestor:
    k8sPSATEnabled: "true"
  workloadAttestors:
    k8sEnabled: "true"
---
apiVersion: operator.openshift.io/v1alpha1
kind: SpiffeCSIDriver
metadata:
  name: cluster
spec:
  agentSocketPath: '/run/spire/agent-sockets/spire-agent.sock'
EOF
```

Verify:
```bash
oc get pods -n zero-trust-workload-identity-manager
oc exec -n zero-trust-workload-identity-manager spire-server-0 -- \
  /opt/spire/bin/spire-server healthcheck
```

### 2. Build and Deploy StackRox

```bash
# Push code to CI for building (assuming you have CI access)
git add -A
git commit -m "feat: Add SPIRE/SPIFFE integration for Sensor-Central authentication

Implements SPIFFE workload identity as an alternative to certificate-based mTLS
for Sensor to Central communication. When SPIRE is available on OpenShift,
Sensor automatically uses SPIRE-issued X.509-SVIDs for authentication, eliminating
the need for manual certificate distribution.

Changes:
- Add SPIFFE identity extractor in Central
- Add SPIRE connection logic in Sensor
- Add SPIRE CSI volume mounts in Helm charts
- Add ClusterSPIFFEID registrations for automatic workload enrollment

This is a Technology Preview feature for OpenShift-only deployments.
Fully backward compatible - falls back to cert-based mTLS when SPIRE unavailable.

Partially generated by AI."

# Push and wait for CI to build images
git push
```

Deploy to your cluster:
```bash
# Deploy Central
helm install -n stackrox --create-namespace stackrox-central-services \
  ./image/templates/helm/stackrox-central \
  --set imagePullSecrets.useExisting="your-pull-secret"

# Deploy Sensor
helm install -n stackrox stackrox-secured-cluster-services \
  ./image/templates/helm/stackrox-secured-cluster \
  --set clusterName=demo-cluster \
  --set centralEndpoint=central.stackrox:443 \
  --set imagePullSecrets.useExisting="your-pull-secret"
```

### 3. Run Demo

```bash
./scripts/demo-spire.sh
```

### 4. Key Demo Points

**Look for these log messages:**

In Central:
```
✅ SPIRE: Authenticated SENSOR_SERVICE via SPIFFE ID: spiffe://stackrox.local/ns/stackrox/sa/sensor
```

In Sensor:
```
🔐 SPIRE: Socket found, attempting SPIRE-based connection to Central
✅ SPIRE: Successfully obtained X.509-SVID from SPIRE Workload API
🎉 SPIRE: Successfully created gRPC connection to Central
```

**Demo Talking Points:**
- ✅ Zero certificate distribution required
- ✅ Fully automated workload identity attestation
- ✅ Short-lived credentials (SVIDs expire in hours, not years)
- ✅ Backward compatible (falls back to cert-based mTLS)
- ✅ OpenShift-native integration (Tech Preview)
- ✅ Minimal code changes (extractor + connection helper)

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ OpenShift Cluster                                           │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ SPIRE Server (zero-trust-workload-identity-manager)  │  │
│  │  - Manages trust domain: stackrox.local              │  │
│  │  - Issues X.509-SVIDs to attested workloads         │  │
│  └──────────────────────────────────────────────────────┘  │
│                            │                                │
│                            │ gRPC                           │
│       ┌────────────────────┼────────────────────┐          │
│       │                    │                    │          │
│       ▼                    ▼                    ▼          │
│  ┌─────────┐         ┌─────────┐         ┌─────────┐     │
│  │ SPIRE   │         │ SPIRE   │         │ SPIRE   │     │
│  │ Agent   │         │ Agent   │         │ Agent   │     │
│  └────┬────┘         └────┬────┘         └────┬────┘     │
│       │                   │                   │           │
│  ┌────┴────────────────────┴───────────────────┴────┐    │
│  │ SPIFFE CSI Driver                               │    │
│  │ (mounts /spire-workload-api in pods)            │    │
│  └─────────────────────────────────────────────────┘    │
│                                                           │
│  ┌──────────────────────┐        ┌───────────────────┐  │
│  │ Central Pod          │        │ Sensor Pod        │  │
│  │                      │◄───────┤                   │  │
│  │ SPIFFE Extractor     │ mTLS   │ SPIRE Connection  │  │
│  │  ├─ Validates URI SAN│  with  │  ├─ Gets X509-SVID│  │
│  │  ├─ Checks SPIFFE ID │ SPIFFE │  ├─ Uses SPIRE    │  │
│  │  └─ Returns Identity │   ID   │  │   trust bundle │  │
│  │                      │        │  └─ Connects to   │  │
│  │ Volume:              │        │     Central       │  │
│  │ /spire-workload-api  │        │                   │  │
│  └──────────────────────┘        │ Volume:           │  │
│                                   │ /spire-workload-  │  │
│                                   │       api         │  │
│                                   └───────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## 📝 Technical Details

### SPIFFE ID Format
```
spiffe://stackrox.local/ns/{namespace}/sa/{serviceaccount}
```

Example:
- Central: `spiffe://stackrox.local/ns/stackrox/sa/central`
- Sensor: `spiffe://stackrox.local/ns/stackrox/sa/sensor`

### ClusterSPIFFEID Matching
```yaml
apiVersion: spire.spiffe.io/v1alpha1
kind: ClusterSPIFFEID
metadata:
  name: sensor-stackrox
spec:
  spiffeIDTemplate: "spiffe://stackrox.local/ns/{{ .PodMeta.Namespace }}/sa/{{ .PodSpec.ServiceAccountName }}"
  podSelector:
    matchLabels:
      app: sensor
```

### Fallback Behavior
1. **Sensor** checks for `/spire-workload-api/spire-agent.sock`
   - If present → Use SPIRE
   - If absent → Use traditional mTLS certificates

2. **Central** checks for SPIFFE ID in client certificate URI SAN
   - If present and recognized → Authenticate via SPIFFE
   - If absent → Try other extractors (mTLS, JWT, etc.)

## ⚠️ Limitations & Future Work

### Current Limitations
- OpenShift-only (conditional on `._rox.env.openshift`)
- Single trust domain: `stackrox.local`
- Namespace must match between Central and Sensor (both in `stackrox`)
- No support for multi-cluster federation yet

### Future Enhancements
- SPIRE Federation for multi-cluster scenarios
- Extend to other components (Admission Controller → Sensor already has the infrastructure)
- Make trust domain configurable
- Support vanilla Kubernetes (upstream SPIRE Helm chart)
- Replace internal CA entirely with SPIRE for all internal services

## 🧪 Testing Notes

- Code compiles successfully ✅
- Central sensor service builds ✅
- Sensor centralclient builds ✅
- Linter passes ✅
- **Integration testing pending** (requires OpenShift 4.19 with SPIRE)

## 📚 References

- SPIFFE Spec: https://github.com/spiffe/spiffe
- SPIRE Documentation: https://spiffe.io/docs/latest/spire/
- OpenShift Zero Trust Workload Identity Manager: https://docs.openshift.com/container-platform/4.19/security/zero-trust-workload-identity-manager.html
- go-spiffe library: https://github.com/spiffe/go-spiffe

---

**Time Investment:** ~2.5 hours of coding
**Status:** Code complete, pending integration testing
**Next Step:** Deploy to OpenShift 4.19 with SPIRE operator
