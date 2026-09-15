# Workload Visibility

Kubernetes workload discovery, tracking, and security analysis enabling all downstream security features.

**Primary Packages**: `central/deployment`, `sensor/kubernetes`

## What It Does

StackRox discovers and tracks all Kubernetes workloads (Deployments, DaemonSets, StatefulSets, Jobs) across monitored clusters. The system provides comprehensive pod lifecycle tracking, container-to-image association for vulnerability scanning, deployment configuration history for change detection, and real-time plus historical security analysis enabling policy evaluation, risk scoring, and network analysis.

## Architecture

### Central Components

The `central/deployment/` DataStore serves as central repository for deployment configurations. Implementation in `datastore/datastore_impl.go` persists deployments across clusters, maintains deployment-to-image relationships, tracks deployment hash for change detection (ROX-32722), integrates with risk/baseline/network flow datastores, and provides optimized query views for UI and API.

**storage.Deployment proto**: Unique ID, name/namespace/cluster_id/cluster_name, labels/annotations (key-value metadata), containers (specs with images), risk_score (calculated value), hash (SHA256 of deployment config for change detection), created timestamp, ports (exposed configs), and service_account.

**storage.Container proto**: Container name, image reference, id_v2 (Image ID in v2 schema, migration field), resources (CPU/memory requests/limits), security_context, volumes (mounts), and ports.

Database storage in PostgreSQL `deployments` table: id (primary key), name/namespace/cluster_id, serialized (jsonb) for full deployment proto, hash (bytea) as SHA256 of config, created timestamp, and priority for ranking.

Deleted deployment cache in `cache/` uses 7-minute TTL preventing race conditions during deletion, accounting for sensor-to-central communication delays, and filtering stale events for deleted deployments.

### Sensor Components

The `sensor/kubernetes/` implements three-phase event processing.

#### Phase 1: Listener

Located in `sensor/kubernetes/listener/`, watches Kubernetes resources and converts events into ResourceEvents. Uses Kubernetes informers (client-go library) with SharedInformer (single resource type), SharedIndexInformer (adds indexing), and DynamicInformer (watches CRDs).

Initial sync: List phase fetches all existing resources from API server, watch phase starts watching for changes (additions, updates, deletions), bookmark event signals initial list completion.

Event handlers defined in `listener/resources/`: OnAdd sends SYNC_RESOURCE during initial sync or CREATE_RESOURCE otherwise, OnUpdate sends UPDATE_RESOURCE, OnDelete sends REMOVE_RESOURCE.

Action types: SYNC_RESOURCE (initial sync, don't enforce policies), CREATE_RESOURCE (new resource, enforce policies), UPDATE_RESOURCE (changed, re-evaluate policies), REMOVE_RESOURCE (deleted, cleanup).

Deployment dispatcher in `listener/resources/deployments.go` converts K8s Deployment to storage.Deployment, updates deployment store, updates pod store with deployment's pods, and creates ResourceEvent with deployment reference.

Supported workload types: Deployments, DaemonSets, StatefulSets, ReplicaSets, Pods (including init/ephemeral containers), Jobs, CronJobs, and OpenShift DeploymentConfigs.

#### Phase 2: Resolver

Located in `sensor/kubernetes/eventpipeline/resolver/resolver_impl.go`, resolves deployment references into complete objects with related resources.

Resolution steps: fetch base deployment, resolve parent for ReplicaSets, gather all pods, resolve services, resolve network policies, resolve RBAC (service account permissions), and send to output.

Deduplication tracks deployment versions to avoid redundant processing. Related resource changes trigger reprocessing of affected deployments when Services or NetworkPolicies change.

#### Phase 3: Output

Located in `sensor/kubernetes/eventpipeline/output/output_impl.go`, queues resolved deployments for detector evaluation and forwards to Central.

Processing flow: receive resolved deployment from resolver, send to detector for policy evaluation via `detector.ProcessDeployment(ctx, deployment, action)`, receive alert results from detector, forward to Central via response channel.

Direct message forwarding bypasses resolver for: namespace changes, node updates, service changes (may trigger deployment reprocessing), and RBAC changes.

## Pod Tracking

The `sensor/kubernetes/listener/resources/` maintains in-memory map of all pods, associates pods with parent deployments, tracks pod status (Running/Pending/Failed), and monitors container statuses and restart counts.

Pod-to-deployment association: pod.OwnerReferences → ReplicaSet → Deployment.

Container conversion in `listener/resources/convert.go` creates storage.Container from corev1.Container extracting name, parsed image name, resources, security context, volumes, ports, and probes.

Image name parsing extracts registry, repository, tag, and digest, normalizes for consistent matching, and handles DockerHub shorthand (e.g., `nginx` → `docker.io/library/nginx`).

## Image Association

Deployment ↔ Image relationship: Deployment (many) ← containers → (many) Images.

Image V2 migration (ROX-29921, ROX-30663): Old format uses container.image (legacy reference), new format uses container.id_v2 (image v2 schema), with both fields populated during transition.

Image cache in Sensor maintains local cache of image metadata for admission control decisions (pre-deployment scanning), updated when Central sends image scan results.

Scan data association: When image scanned, Central updates image with scan results, triggers deployment risk recalculation for all deployments using image, and propagates risk scores to deployments.

Propagation path: Image Scan → Image Risk → Deployment Risk → UI Display.

## Event Pipeline Flow

### Deployment Discovery

1. Kubernetes API Server (watch event)
2. Sensor Informer (OnAdd/OnUpdate/OnDelete)
3. Deployment Dispatcher (convert K8s → storage.Deployment)
4. Deployment Store (in-memory)
5. ResourceEvent created
6. Resolver (gather related resources)
7. Complete Deployment
8. Detector (policy evaluation)
9. Central (persistence + analysis)
10. UI (display)

### Related Resource Flow

Service Change → ServiceDispatcher updates service store → Find deployments matching service selector → Create ResourceEvent with deployment references → Resolver fetches deployments → Includes new service in deployment.Services → Send to detector.

## Change Detection

### Deployment Hash (ROX-32722)

Detects actual configuration changes vs Kubernetes metadata updates. Hash calculation: `hash := sha256.Sum256(canonicalJSON(deployment.Spec))`, stored as `deployment.Hash`.

Change detection in `datastore_impl.go`: compares existing.Hash with deployment.Hash, skips processing if hashes match, avoiding unnecessary risk recalculations, database writes, and improving performance for large deployments.

### Resource Version Tracking

Kubernetes ResourceVersion provides monotonically increasing version from etcd for change detection in informers, preventing duplicate event processing.

Deduplication in resolver maintains `processedVersions` map of deploymentID → resourceVersion, only processes when currentVersion > lastProcessed.

## Performance

### Query Optimization (ROX-32722, ROX-33178)

View-based queries use specialized database views for list operations. DeploymentIDView contains minimal projection (only ID and basic metadata), significantly faster than full deployment queries, used for UI list views and filtering. CountByClusterView provides aggregated counts (deployment counts per cluster) for dashboard metrics.

Hash-based efficiency: Fast `bytes.Equal()` comparison instead of deep object comparison, SHA256 computation amortized over deployment lifetime, prevents cascading recalculations for no-op updates.

Database indexes: (cluster_id, namespace, name) for cluster context lookup, (hash) for fast change detection, (risk_score) for sorted risk queries, full-text search on labels/annotations/names.

Keyed mutex for concurrency in `datastore_impl.go` uses `concurrency.KeyedMutex` enabling concurrent updates for different deployments, serialized updates for same deployment preventing races, and better throughput than global lock.

## Recent Changes

Work in 2024 Q4 addressed commit 854fc111dc separating storage.Deployment into API and storage types, reducing coupling. 2024 Q3 completed ROX-32722 (hash column for efficient change detection), ROX-33178 (optimized ListDeployment with column selection via view-based queries), and fixed GetImagesForDeployment to use imageIdV2s correctly. 2024 Q2's ROX-29921 and ROX-30663 set container image idV2 field during ingestion, migrated images to images_v2 table during first reprocessing, and updated sensor and admission controller image caches. 2024 Q1's ROX-31217 and ROX-31535 removed datastore-level SAC checks and moved security enforcement to PostgreSQL RLS.

## Integration Points

**Policy Evaluation**: Deployments sent to detector for checking, violations tracked and linked, risk scores updated based on violations.

**Vulnerability Management**: Container images scanned, scan results associated with deployments, deployment risk scores reflect image vulnerabilities.

**Network Visualization**: Deployment connectivity tracked via network flows, services and network policies associated, network graph visualization built from relationships.

**Compliance**: Deployments evaluated against standards, configuration baselines maintained, runtime compliance checks executed on pods.

## Implementation

**Event Pipeline**:
- Listener phase: `sensor/kubernetes/listener/resources/deployments.go` (OnAdd, OnUpdate, OnDelete handlers)
- Deployment dispatcher: `sensor/kubernetes/listener/resources/deployments.go` (DispatchDeployment)
- Pod conversion: `sensor/kubernetes/listener/resources/convert.go` (convertContainer, parseImageName)
- Resolver phase: `sensor/kubernetes/eventpipeline/resolver/resolver_impl.go` (resolveDeployment, gatherPods)
- Output phase: `sensor/kubernetes/eventpipeline/output/output_impl.go` (processEvent, sendToDetector)
- Pipeline orchestration: `sensor/kubernetes/eventpipeline/pipeline_impl.go` (Start, Stop)

**Central**: `central/deployment/datastore/datastore_impl.go`, `central/deployment/cache/deleted_deployment_cache.go`, `central/deployment/service/service_impl.go`
**Storage**: `central/deployment/datastore/internal/store/postgres/`
**Sensor**: `sensor/kubernetes/eventpipeline/`, `sensor/kubernetes/listener/`, `sensor/kubernetes/listener/resources/`
**Dispatchers**: `sensor/kubernetes/listener/resources/deployments.go`, `sensor/kubernetes/listener/resources/daemonsets.go`, `sensor/kubernetes/listener/resources/pods.go`
**Related**: `central/imagev2/datastore/`, `central/risk/datastore/`, `central/processbaseline/datastore/`, `central/networkgraph/flow/datastore/`
