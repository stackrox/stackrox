# Runtime Detection

Real-time monitoring of container runtime behavior using eBPF or kernel modules to detect security anomalies without modifying applications.

**Primary Packages**: `collector/`, `sensor/common`, `central/detection/runtime`

## What It Does

StackRox monitors running containers for security violations: unexpected process execution, network anomalies, file access patterns, privilege escalation, and deviations from learned baselines. Users see runtime alerts, process discovery with real-time process lists, automatically learned process baselines with locking, live network connections, file access monitoring, and execution events (execve, setuid, setgid, pod exec, port-forward).

## Architecture

### Collector

The `collector/` agent captures runtime data using eBPF probes or kernel modules. Core components include libsinsp/libscap libraries from Falco/Sysdig in `collector/lib/`, ProcessSignalFormatter in `lib/ProcessSignalFormatter.cpp` for formatting events, NetworkSignalHandler in `lib/NetworkSignalHandler.cpp` for network events, ConnectionTracker in `lib/ConnectionTracker.cpp` tracking active connections, and gRPC client in `lib/GRPCUtil.cpp` sending data to Sensor.

### Sensor Event Pipeline

The `sensor/common/` implements Listener→Resolver→Output→Detector architecture. Key modules: detector in `detector/` processes runtime indicators and maintains baselines, networkflow in `networkflow/` aggregates connections, and processbaseline in `processbaseline/` manages baseline lifecycle.

### Central Detection

The `central/detection/runtime/` evaluates runtime policies. Related modules: processindicator in `central/processindicator/` stores execution data, processbaseline in `central/processbaseline/` manages baseline storage and locking, and networkbaseline in `central/networkbaseline/` handles network baseline management.

### Data Model

**ProcessIndicator**: Process (name, args, path, user/group ID), container (ID, pod UID, deployment ID), signal type (created, exec'd, terminated), lineage (parent tree), timestamps (start, signal time), and baseline flag (whether in baseline).

**ProcessBaseline**: Deployment reference, element set (allowed process names/args/paths), stack by strategy (process name, args, user), user locked vs automatic locking, and created time.

**NetworkFlow**: Source/destination (pod, external entity, service), protocol/port (TCP/UDP with dest port), ingress/egress flag, first/last seen timestamps, and clustered flag.

**KubernetesEvent**: Type (pod exec, port-forward, privileged action), object (pod, deployment, namespace), user (who performed), and timestamp.

**Collector Signal Types**: SIGNAL_EXEC (execve), SIGNAL_FORK, SIGNAL_EXIT (termination), SIGNAL_SETUID (UID change), SIGNAL_SETGID (GID change). Network signals for connection/endpoint/close. File signals for open/modification.

## Data Flow

### Process Detection

1. **Syscall Capture**: eBPF probes attach to kernel hooks (execve, fork, setuid) or kernel module intercepts, libsinsp/libscap processes raw events, events enriched with container metadata via cgroup parsing.

2. **Signal Formatting**: ProcessSignalFormatter creates ProcessSignal proto including process tree, args, executable hash, with deduplication (only send unique processes) and batching.

3. **Transmission**: gRPC stream sends ProcessSignal messages to Sensor, connection resilient to interruptions, buffering if Sensor unavailable.

4. **Event Pipeline**: Listener receives signals from Collector, Resolver enriches with Kubernetes metadata (pod→deployment mapping), Output forwards to Central or stores locally, Detector performs local policy evaluation.

5. **Baseline Checking**: Sensor queries local baseline cache, if process not in baseline and baseline locked: marks as anomaly, creates ProcessIndicator with not_in_baseline=true.

6. **Upload**: ProcessIndicators sent to Central via gRPC, `central/sensor/service/` receives, stores in `central/processindicator/datastore/`.

7. **Policy Evaluation**: `central/detection/runtime/` evaluates RUNTIME policies checking process name/args/path, unexpected execution, privilege escalation, baseline deviation. Violations create alerts.

8. **Alert**: `central/alert/` creates record linking deployment, policy, process indicator, sends notifications to integrations.

### Network Detection

1. **Tracking**: eBPF probes on connect/accept/close syscalls, ConnectionTracker maintains active connection table, periodic flush of updates.

2. **Signal Generation**: NetworkSignalHandler creates NetworkConnection proto with source/dest IP, port, protocol, timestamps, and direction determination.

3. **Flow Aggregation**: `sensor/common/networkflow/` receives from Collector, aggregates by deployment pair, buffers before sending.

4. **Upload**: Periodic NetworkFlowUpdate to Central, `central/networkgraph/` receives and processes, flows compared against network baseline.

5. **Policy Evaluation**: Runtime policies detect unexpected connections, baseline deviations trigger alerts.

### Baseline Learning

1. **Observation**: New deployment enters observation mode (default: 1 hour), all process executions recorded, baseline elements created dynamically.

2. **Locking**: After observation period, baseline auto-locks. User can manually lock earlier or unlock for re-learning. Locked baseline stored in `central/processbaseline/datastore/`.

3. **Anomaly Detection**: Processes not in locked baseline marked as anomalies, runtime policies enforce "unexpected process" violations, alerts generated for anomalous behavior.

### Offline Mode

Sensor operates offline if Central unreachable: maintains local cache of baselines and policies, continues collecting indicators locally, buffers data for eventual upload when reconnected, uses local disk storage for buffered indicators.

## Configuration

**Collector**:
- `COLLECTION_METHOD`: "ebpf", "kernel-module", or "core-bpf" (default: ebpf)
- `GRPC_SERVER`: Sensor gRPC endpoint (e.g., "sensor:8443")
- `ENABLE_CORE_DUMP`: Enable core dumps on crash
- `LOG_LEVEL`: Logging verbosity (info, debug, trace)
- `MODULE_DOWNLOAD_BASE_URL`: URL for kernel module downloads
- `COLLECTOR_CONFIG`: Path to collector config file

**Sensor**:
- `ROX_PROCESSES_LISTEN_ON_CREATES`: Enable process detection on creates (default: true)
- `ROX_PROCESSES_LISTEN_ON_UPDATES`: Enable on updates (default: false)
- `ROX_BASELINE_GENERATION_DURATION`: Hours before baseline locks (default: 1)
- `ROX_NETWORK_DETECTION_ENABLED`: Enable network flow collection (default: true)
- `ROX_OFFLINE_MODE`: Operate without Central (default: false)

**Central**:
- `ROX_RUNTIME_DETECTION_ENABLED`: Enable runtime policy evaluation (default: true)
- `ROX_PROCESSES_RETENTION`: Days to retain process indicators (default: 7)
- `ROX_BASELINE_LOCK_ENABLED`: Enable automatic baseline locking (default: true)

Baseline settings per deployment: stack by strategy (process name, args, user, UID), auto-lock enable, manual element add/remove.

Runtime policy configuration: lifecycle stage RUNTIME, enforcement inform only (cannot block running processes), scope (cluster, namespace, deployment filters). Common policies: "Unauthorized Process Execution", "Privilege Escalation", "Shell Spawned in Container", "Netcat Execution", "Crypto Mining".

## Testing

**Unit Tests**:
- `sensor/common/detector/*_test.go`: Detector logic
- `sensor/common/processbaseline/*_test.go`: Baseline management
- `sensor/common/networkflow/*_test.go`: Flow aggregation
- `central/detection/runtime/*_test.go`: Runtime policy evaluation
- `central/processindicator/*_test.go`: Indicator storage

**Integration** (PostgreSQL, `//go:build sql_integration`): Requires PostgreSQL on port 5432.

**E2E**: `ProcessBaselineTest.groovy`, `RuntimePolicyTest.groovy`, `NetworkDetectionTest.groovy` in `qa-tests-backend/`

## Known Limitations

**Performance**: eBPF overhead causes 5-10% CPU on high-frequency syscalls. Chatty containers generate thousands of indicators. Large deployments have large baseline storage. Kernel module compilation requires matching kernel headers.

**Accuracy**: Processes <100ms may be missed. Very deep process trees (>10 levels) truncated. Cgroup parsing can fail in exotic runtimes. High connection rate can overwhelm tracker.

**Compatibility**: eBPF requires Linux kernel 4.14+, some features need 4.19+. Kernel module must match exact kernel version. Works with Docker, containerd, CRI-O; limited with other runtimes. Some SELinux/AppArmor profiles block Collector probes.

**Behavior**: Long-running apps evolve beyond baseline. Noisy deployments have overly permissive baselines. Cannot sync new policies while offline. If Collector crashes, in-flight signals lost.

**Workarounds**: Use `COLLECTION_METHOD=core-bpf` for newer eBPF features. Increase `ROX_BASELINE_GENERATION_DURATION` for dynamic workloads. Filter process indicators by name to reduce volume. Manually curate baselines to remove one-time processes. Restart Collector pod to reload kernel module after kernel upgrade. Use `ROX_PROCESSES_RETENTION=3` to reduce indicator storage. Unlock baselines periodically in CI/CD environments.

## Process Filtering and Dropping

To reduce noise and data volume, StackRox filters processes at multiple stages:

### Sensor Process Filter

The `sensor/common/processfilter/` implements client-side process filtering to drop repetitive or uninteresting processes before sending to Central:

**Filter Configuration** (`pkg/env/process_filter_mode.go`):
- `ROX_PROCESS_FILTER_MODE`: Preset modes (baseline, verbose, compact)
- `ROX_PROCESS_FILTER_MAX_EXACT_PATH_MATCHES`: Max unique process paths (default: 5000)
- `ROX_PROCESS_FILTER_FAN_OUT_LEVELS`: Depth for fan-out tracking (default: [100, 50, 25])

**Filter Logic** (`pkg/process/filter/filter.go`):
- Tracks unique process paths in bloom filter
- Drops processes after path limit exceeded
- Fan-out tracking for process trees (prevents fork bombs from overwhelming system)
- Exact path matching for frequently executed binaries

**Usage**: Filter initialized in `sensor/common/processfilter/filter.go` (Singleton), applied before sending ProcessIndicator to Central.

### Central Indicator Filtering

Central applies additional filtering in `central/processindicator/datastore/`:
- Deduplicates identical indicators within time window
- Applies retention policy (default 7 days)
- Filters by deployment scope for SAC

### Network Flow Filtering

Network flows are also filtered to reduce data volume:

**Flow Dropping** (`sensor/common/networkflow/manager.go`):
- Aggregates connections by deployment pair before sending
- Drops short-lived connections (<1s duration)
- Merges flows with same source/dest/protocol/port
- Rate limits flow updates per cluster

**Central Flow Filtering** (`central/networkgraph/flow/datastore/flow_impl.go`):
- Time-based partitioning drops old flows automatically
- Filters flows by cluster/namespace scope
- Prunes flows outside retention window

## Implementation

**Collector**: `collector/lib/ProcessSignalFormatter.cpp`, `collector/lib/NetworkSignalHandler.cpp`, `collector/lib/ConnectionTracker.cpp`
**Sensor**: `sensor/common/detector/`, `sensor/common/processbaseline/`, `sensor/common/networkflow/`
**Central**: `central/detection/runtime/`, `central/processindicator/`, `central/processbaseline/`
**API**: `proto/api/v1/process_service.proto`, `proto/api/v1/process_baseline_service.proto`, `proto/storage/process_indicator.proto`, `proto/internalapi/sensor/signal_service.proto`
