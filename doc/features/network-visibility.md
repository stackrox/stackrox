# Network Visibility

Passive network flow monitoring, baseline learning, anomaly detection, and Kubernetes NetworkPolicy generation.

**Primary Packages**: `central/networkgraph`, `central/networkpolicies`, `pkg/networkgraph`

## What It Does

StackRox observes network flows and builds real-time communication maps across clusters. The system provides:

- Interactive network graph visualization of service-to-service communication
- Automatically learned baseline traffic patterns for deployments
- Anomaly detection with alerts for unexpected connections
- NetworkPolicy YAML generation from observed traffic
- External entity visualization (connections to external IPs and domains)
- Flow details (protocol, port, ingress/egress direction)
- Simulation mode for previewing policy impact
- CIDR block management for grouping external IPs

## Architecture

### Network Graph Core

The `central/networkgraph/` manages flow aggregation, storage, and graph building. Key modules:

**Aggregator** (`aggregator/`): Combines flows into deployment-level edges, merging connections with same source/dest/protocol/port and updating timestamps.

**Entity Manager** (`entity/`): Manages deployment and external entity lifecycle, mapping pods to deployments and matching external IPs against CIDR blocks using radix tree from `pkg/networkgraph/tree/`.

**Flow Datastore** (`flow/datastore/`): Stores raw and aggregated flows in PostgreSQL with time-based partitioning.

**Graph Updater** (`updater/`): Maintains in-memory graph, prunes edges based on retention, periodically serializes to database.

### Data Model

**NetworkFlow**: Source (deployment/external), destination (deployment/external/internet), protocol (TCP/UDP/ICMP), destination port, ingress/egress flag, first/last seen timestamps, clustered flag.

**NetworkBaseline**: Deployment reference, peer set with protocols/ports, observation period duration, locked state (manual vs automatic), forbidden peers.

**NetworkEntity**: Type (Deployment/ExternalSource/Internet), deployment reference or external source (CIDR/IP), namespace and cluster scope.

**ExternalNetworkSource**: Human-readable name (e.g., "AWS us-east-1"), CIDR range, default flag (system-provided vs user-created).

Graph representation: nodes are deployments/external sources/Internet meta-node, edges are NetworkFlow objects, properties (protocol/port/direction) stored on edges.

### Flow Collection

1. **Collection**: Collector (eBPF/kernel module) captures pod-level connections, sends NetworkConnection messages to Sensor, which buffers and aggregates.

2. **Upload**: Sensor periodically sends NetworkFlowUpdate to Central via gRPC stream.

3. **Ingestion**: `central/networkgraph/flow/datastore/` receives flows, validates and enriches with cluster/namespace metadata, stores in PostgreSQL.

4. **Graph Building**: Entity manager maps pods to deployments and matches external IPs against CIDR blocks. Aggregator groups flows by deployment pair. Graph updater maintains in-memory graph and prunes based on retention.

5. **Baseline Learning**: New deployments enter observation mode (default 1 hour), all flows recorded. After observation period, baseline auto-locks. Locked baseline defines "normal" traffic. Runtime flows compared against locked baseline, deviations trigger policy violations.

### Network Policy Generation

1. **Request**: User selects deployments in UI, request sent to `central/networkpolicies/service.go`.

2. **Analysis**: `central/networkpolicies/graph/evaluator.go` queries network graph, identifies all peers that communicated with target, filters based on options (include baselines, external, etc.).

3. **Generation**: `central/networkpolicies/generator/` creates NetworkPolicy YAML with ingress rules (allow from observed sources) and egress rules (allow to observed destinations), pod selector matching target.

4. **Undo Tracking**: `central/networkpolicies/undo/` stores generated policy metadata, tracks StackRox-created vs user-created policies, enables selective rollback.

5. **Application**: User applies generated YAML via kubectl, StackRox monitors effectiveness via continued flow observation.

### External Source Management

Users define named CIDR blocks (e.g., "Corporate VPN: 10.0.0.0/8") stored in `central/networkgraph/config/datastore/`. The `pkg/networkgraph/externalsrcs/store.go` maintains radix tree of CIDRs. Incoming external IPs match against tree in O(log n). Matched IPs labeled with CIDR block name in graph. System includes default blocks for major cloud providers.

## Configuration

**Central**:
- `ROX_NETWORK_GRAPH_ENABLED`: Enable network graph (default: true)
- `ROX_NETWORK_BASELINE_LOCK_DURATION`: Hours before baseline auto-locks (default: 1)
- `ROX_NETWORK_FLOW_RETENTION`: Days to retain raw flows (default: 7)
- `ROX_NETWORK_GRAPH_EXTERNAL_SRCS_ENABLED`: Enable external source tracking (default: true)

**Sensor**:
- `ROX_NETWORK_FLOW_COLLECTION`: Enable flow collection (default: true)
- `ROX_NETWORK_FLOW_BUFFER_SIZE`: Flow buffer size before flush (default: 1000)
- `ROX_NETWORK_FLOW_UPLOAD_INTERVAL`: Seconds between uploads (default: 30)

**Collector**:
- `COLLECTION_METHOD`: "ebpf" or "kernel-module" (default: ebpf)
- `ENABLE_NETWORK_FLOWS`: Enable network flow collection (default: true)
- `NETWORK_CONNECTION_IFACE`: Network interface to monitor (default: all)

Network graph settings via UI/API: baseline lock time, flow retention, external source matching, graph display filters. Policy generation options: include baseline only, allow external sources, namespace scoping, delete existing policies flag.

## Testing

**Unit Tests**:
- `central/networkgraph/aggregator/*_test.go`: Flow aggregation
- `central/networkgraph/entity/*_test.go`: Entity management
- `pkg/networkgraph/tree/*_test.go`: Radix tree CIDR matching
- `central/networkpolicies/graph/*_test.go`: Graph analysis
- `central/networkpolicies/generator/*_test.go`: Policy YAML generation

**Integration** (PostgreSQL, `//go:build sql_integration`): `central/networkgraph/flow/datastore/internal/store/postgres/*_test.go`

**E2E**: `NetworkGraphTest.groovy`, `NetworkBaselineTest.groovy`, `NetworkPolicyGeneratorTest.groovy` in `qa-tests-backend/`

## Known Limitations

**Performance**: Clusters with 10,000+ pods generate millions of flows. Large graphs (>500 nodes) slow in UI. Many unique external IPs fragment graph.

**Accuracy**: Connections <1 second may be missed. UDP flows less reliable than TCP. Traffic outside cluster not always visible. Service mesh (Envoy sidecars) obfuscates actual pod-to-pod communication.

**Behavior**: Long-running deployments may have stale baselines. Generated policies may be too permissive or restrictive. Cannot undo manually modified policies.

**Features**: No packet inspection or payload analysis. DNS queries not always associated with subsequent connections. Network policy enforcement relies on Kubernetes CNI support. Cross-cluster flows not fully supported.

**Workarounds**: Use `ROX_NETWORK_FLOW_RETENTION=30` to increase history for better baseline learning. Filter graph view by namespace to reduce rendering load. Define CIDR blocks for known external services to reduce "Internet" node size. Manually review and edit generated policies before applying. Unlock baselines periodically in dynamic environments. Use network policy simulation mode before enforcement.

## Implementation

**Flow Aggregation**:
- Flow ingestion: `central/networkgraph/flow/datastore/flow_impl.go` (CreateFlowStore, RemoveFlow)
- Aggregation logic: `central/networkgraph/aggregator/aggregator_impl.go` (Aggregate, mergeFlows)
- Entity resolution: `central/networkgraph/entity/networktree/manager_impl.go` (createOrGetDeploymentEntity, createOrGetExternalSrcEntity)
- CIDR matching: `pkg/networkgraph/tree/networktree.go` (radix tree implementation)
- Graph updates: `central/networkgraph/updater/updater_impl.go` (prune, updateGraph)

**Core**: `central/networkgraph/aggregator/`, `central/networkgraph/entity/`, `central/networkgraph/flow/datastore/`, `central/networkgraph/updater/`
**Policy Generation**: `central/networkpolicies/generator/`, `central/networkpolicies/graph/`, `central/networkpolicies/undo/`
**Utilities**: `pkg/networkgraph/tree/`, `pkg/networkgraph/externalsrcs/`
**API**: `proto/api/v1/network_graph_service.proto`, `proto/api/v1/network_baseline_service.proto`, `proto/api/v1/network_policy_service.proto`
**Storage**: `proto/storage/network_flow.proto`, `proto/storage/network_baseline.proto`
