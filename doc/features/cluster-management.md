# Cluster Management

Kubernetes cluster registration, health monitoring, and lifecycle operations across StackRox's multi-cluster deployment model.

**Primary Packages**: `central/cluster`, `central/clusterinit`, `pkg/clusterhealth`

## What It Does

The cluster management system handles the complete lifecycle of monitored Kubernetes clusters:

- Register clusters using init bundles or Cluster Registration Secrets (CRS)
- Track health status of Sensor, Collector, and Admission Controller components
- Monitor sensor connectivity and report upgrade progress
- Coordinate cascading deletion across all cluster-related data
- Prevent cluster ID reuse during network partitions using 3-minute grace period

## Architecture

### Cluster DataStore

The `central/cluster/datastore/` module serves as the central coordinator for cluster data. Implementation in `datastore_impl.go` uses keyed mutexes for concurrent access: different clusters update in parallel while same-cluster updates serialize.

The storage.Cluster proto tracks comprehensive metadata: unique ID, name, type (K8S/OPENSHIFT), managed status, image configurations, admission controller settings, tolerations, network policies, Central endpoint, and Helm config.

Recent work in ROX-32667 added label-based scoping support for fine-grained access control.

### Init Bundle System

The `central/clusterinit/backend/` module issues init bundles and CRS for secure onboarding. Responsibilities include cryptographic certificate generation, metadata tracking (ID, name, created_at, expiration), revocation support, and certificate chain validation for incoming sensors.

Init bundles contain service CA certificate, sensor client certificate, sensor private key, and Central endpoint address. CRS provides dynamic certificate generation with configurable max_registrations (ROX-26769) and time limits.

**Bundle Comparison**:

| Aspect | Init Bundle | CRS |
|--------|-------------|-----|
| Certificate | Static, embedded | Dynamic, on-demand |
| Lifetime | Long-lived | Time-limited (24h default) |
| Cluster Count | One per cluster | Multiple (max_registrations) |
| Revocation | Manual via API | Auto on expiration |
| Use Case | Production | Ephemeral/dev/automation |

### Health Monitoring

The `pkg/clusterhealth/` module calculates health status using component-specific thresholds defined in `clusterhealth.go`:

**Sensor Health Thresholds**:
- Healthy: disconnected < 1 minute
- Degraded: 1-3 minutes disconnected
- Unhealthy: > 3 minutes
- Uninitialized: never connected

**Collector Health** (fraction of desired pods ready):
- Healthy: 100% ready
- Degraded: ≥80% ready
- Unhealthy: <80% ready

**Admission Controller Health**:
- Healthy: 100% ready
- Degraded: ≥66% ready
- Unhealthy: <66% ready

Overall cluster status uses priority: UNINITIALIZED > UNHEALTHY > DEGRADED > HEALTHY. The `PopulateOverallClusterStatus` function in `pkg/clusterhealth/clusterhealth.go` implements this logic.

### Cluster Move Grace Period

To prevent cluster ID reuse during network partitions, the datastore maintains a TTL cache of deleted clusters in `datastore_impl.go`. When clusters are removed, IDs enter a 3-minute grace period before reuse, preventing data corruption if the original Sensor reconnects.

## Registration Flow

1. User creates init bundle or CRS via API/UI
2. User deploys Sensor with credentials
3. Sensor connects to Central with certificate
4. Central validates certificate chain
5. Central registers or looks up cluster
6. Sensor receives cluster ID
7. Sensor starts streaming events

CRS workflow dynamically generates certificates during registration, supporting multiple clusters from a single secret.

## Cascade Deletion

When removing clusters, the datastore coordinates deletion across multiple datastores defined in `datastore_impl.go`:

- Alerts, deployments, namespaces, nodes, pods
- Network entities, flows, baselines
- RBAC resources (roles, bindings, secrets, service accounts)
- Image integrations, compliance data, CVE edges

Deletion order matters: dependent data removes first, cluster record last, preventing orphaned database entries.

## Integration Points

**Sensor**: Cluster registration on first connection, health updates every 60 seconds, upgrade status reporting, component version tracking.

**UI**: Cluster list with health indicators, configuration management, init bundle generation, upgrade status display.

**roxctl**: Commands for `init-bundles generate/list/revoke`, `crs generate` (feature-flagged), `cluster delete`.

**Operator**: SecuredCluster CR reconciliation, automatic upgrades, certificate rotation, health monitoring.

## Implementation

**Core**: `central/cluster/datastore/datastore_impl.go`, `central/cluster/datastore/health.go`
**Storage**: `central/cluster/datastore/internal/store/postgres/`
**Init Bundles**: `central/clusterinit/backend/backend_impl.go`, `central/clusterinit/store/postgres/`
**Health**: `pkg/clusterhealth/clusterhealth.go`, `pkg/clusterhealth/constants.go`
**Connection**: `central/sensor/service/connection/`
