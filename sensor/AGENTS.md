# AGENTS.md - StackRox Sensor

## Project Overview

Sensor is a critical ACS (Advanced Cluster Security) component that runs in the secured cluster and serves as the primary data collection and processing engine. It gathers cluster information from various sources and forwards enriched data to Central for analysis and storage.

### Core Functions
- **Data Collection**: Gathers information from Kubernetes, Compliance, Collector, and local Scanner
- **Event Processing**: Enriches and transforms raw cluster events before forwarding to Central
- **Policy Detection**: Performs system policy violation detection locally in the cluster
- **Resource Monitoring**: Tracks deployments, pods, services, roles, and other Kubernetes resources

## Architecture Overview

### Main Components

#### 1. Services vs Components Architecture
- **Services**: Handle gRPC communication with other ACS components (Compliance, Collector, Admission Control)
- **Components**: Implement `SensorComponent` interface and communicate with Central via message channels

#### 2. Core Interfaces

**SensorComponent Interface** (`sensor/common/component.go`):
```go
type SensorComponent interface {
    Notifiable           // Receives state change notifications
    CentralSender       // Sends messages to Central via ResponsesC()
    CentralReceiver     // Processes messages from Central via ProcessMessage()
    Component           // Basic lifecycle (Start/Stop/Capabilities/Name)
}
```

**Component States**:
- `SensorComponentEventCentralReachable`: Online mode
- `SensorComponentEventOfflineMode`: Offline mode
- `SensorComponentEventSyncFinished`: Initial sync complete

#### 3. Key Directory Structure

```
sensor/
├── admission-control/     # Admission Control pod (separate from Sensor)
├── common/               # Core Sensor components
│   ├── sensor/          # Main Sensor implementation
│   ├── detector/        # Policy detection engine
│   ├── compliance/      # Compliance service
│   ├── image/          # Image scanning components
│   ├── networkflow/    # Network flow processing
│   └── ...
├── kubernetes/          # Kubernetes-specific implementations
│   ├── sensor/         # K8s Sensor creation and management
│   ├── eventpipeline/  # K8s event processing pipeline
│   ├── listener/       # K8s resource listeners
│   └── ...
├── debugger/           # Debug utilities
├── upgrader/           # Sensor upgrade logic
└── tests/             # Integration tests
```

### Data Processing Pipeline

#### Event Pipeline Flow
1. **Listener** (`sensor/kubernetes/eventpipeline/`): Receives Kubernetes events via informers
2. **Dispatcher**: Performs initial enrichment with object-specific data
3. **Resolver**: Enriches with adjacent/dependent resources (RBAC, services, etc.)
4. **Output Queue**: Forwards to Central and triggers policy detection

#### Resource Transformation
- **Kubernetes Resources** → **ACS Deployments**: Maps K8s objects (Deployment, DaemonSet, Job, etc.) to unified ACS deployment model
- **Memory Stores**: In-memory resource stores for deployments, pods, services, roles, etc.
- **Hierarchy Tracking**: Maintains parent-child relationships (deployment → replicaset → pods)

## Development Guidelines

### Code Style
- Follow existing patterns in `sensor/common/` for component implementations
- Implement the `SensorComponent` interface for new components
- Use buffered channels to prevent pipeline blocking
- Handle both online and offline modes for resilient operation

### Component Development Pattern

```go
type MyComponent struct {
    responsesC chan *message.ExpiringMessage
    // ... other fields
}

func (c *MyComponent) Start() error {
    // Start component logic
}

func (c *MyComponent) Stop() {
    // Cleanup logic
}

func (c *MyComponent) ResponsesC() <-chan *message.ExpiringMessage {
    return c.responsesC
}

func (c *MyComponent) ProcessMessage(ctx context.Context, msg *central.MsgToSensor) error {
    // Handle messages from Central
}

func (c *MyComponent) Notify(e common.SensorComponentEvent) {
    // Handle state changes (online/offline mode)
}
```

### Building and Testing

#### Build Commands
```bash
# Build Sensor binary
make sensor

# Build with Docker
make image

# Run unit tests
go test ./sensor/...

# Run specific component tests
go test ./sensor/common/detector/...

# Run integration tests (requires Kind cluster)
make sensor-integration-test
```

#### Integration Test Setup
Integration tests require a Kubernetes cluster (Kind is used in CI):

```bash
# Create Kind cluster for integration tests
kind create cluster --config kind-config.yaml

# Set KUBECONFIG environment variable
export KUBECONFIG="$(kind get kubeconfig-path)"

# Run integration tests
make sensor-integration-test
```

**Integration Test Configuration:**
- Timeout: 15 minutes (`-timeout 15m`)
- Race detection enabled (`-race`)
- Test parallelism: 1 (`-p 1`)
- Debug logging enabled (`LOGLEVEL=debug`)
- CGO enabled with enhanced checking (`GOEXPERIMENT=cgocheck2`)

#### Key Test Locations
- Unit tests: `*_test.go` files alongside source
- Integration tests: `sensor/tests/`
  - `sensor/tests/connection/`: Central-Sensor connection tests
  - `sensor/tests/pipeline/`: Event pipeline tests
  - `sensor/tests/resource/`: Resource processing tests
  - `sensor/tests/complianceoperator/`: Compliance operator tests
- Pipeline benchmarks: `sensor/tests/pipeline/bench_test.go`

### Important Implementation Notes

#### Memory Management
- All resource stores are in-memory and rebuilt on restart
- Use buffered channels to prevent blocking pipelines
- Implement graceful degradation when buffers are full

#### Error Handling
- Components should handle Central disconnection gracefully
- Log degraded states when dropping events due to full buffers
- Support both online and offline operational modes

#### Message Flow
- All component response channels are multiplexed into a single Central stream
- Messages are wrapped in `ExpiringMessage` for timeout handling
- Deduplication occurs at multiple levels (detector and gRPC wrapper)

### Security Considerations
- Sensor runs with elevated cluster permissions for resource monitoring
- Certificate management handled by `sensor/kubernetes/certrefresh/`
- mTLS communication with Central
- Service account tokens managed in `pkg/satoken/`

### Common Patterns

#### Adding a New Service
1. Create gRPC service definition
2. Implement service in `sensor/common/` or `sensor/kubernetes/`
3. Add connection handling in main sensor creation
4. Register with gRPC server routes

#### Adding a New Component
1. Implement `SensorComponent` interface
2. Add to component creation in `sensor/kubernetes/sensor/sensor.go`
3. Handle online/offline modes appropriately
4. Add to component registry and startup sequence

### Debugging and Monitoring
- Enable debug mode with environment variables
- Use `sensor/debugger/` tools for troubleshooting
- Monitor component health via health checks
- Metrics exposed for Prometheus collection

## Key Dependencies
- Kubernetes client-go for cluster API access
- gRPC for Central communication
- Protocol Buffers for message serialization
- Helm for deployment configuration

## Related Documentation
- See `sensor_symbols.yaml` for complete API reference
- Check `meeting.txt` for team discussions and context
- Review component-specific README files where available