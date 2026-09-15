# StackRox Cross-Domain Patterns

**Version**: 1.0
**Last Updated**: 2026-03-13
**Repository**: https://github.com/stackrox/stackrox

This document describes recurring architectural and implementation patterns that appear across 50+ packages in the StackRox codebase. Individual package documentation should reference these patterns instead of re-explaining them.

---

## Table of Contents

1. [Datastore Pattern](#datastore-pattern)
2. [Service Pattern](#service-pattern)
3. [Singleton Pattern](#singleton-pattern)
4. [Code Generation Pipeline](#code-generation-pipeline)
5. [SAC (Scoped Access Control)](#sac-scoped-access-control)
6. [Feature Flags](#feature-flags)
7. [Search Integration](#search-integration)
8. [Event Pipeline (Sensor)](#event-pipeline-sensor)

---

## Datastore Pattern

**Applies to**: 100+ datastores across `central/*/datastore/`

### Purpose

The datastore pattern provides a consistent abstraction layer for persisting and querying StackRox entities (alerts, deployments, policies, images, etc.). It separates business logic (datastore layer) from storage implementation (store layer) and enforces security boundaries via SAC.

### Standard Structure

```
central/{entity}/
├── datastore/               # Business logic layer
│   ├── datastore.go         # DataStore interface definition
│   ├── datastore_impl.go    # Implementation with domain logic
│   ├── singleton.go         # Package-level singleton factory
│   ├── internal/store/      # Low-level storage implementation
│   │   └── postgres/        # PostgreSQL-specific store
│   │       ├── store.go     # Store interface
│   │       └── store_impl.go # Generated or hand-written implementation
│   ├── mocks/               # Generated mock implementations
│   └── test/                # Integration tests
```

### Key Interfaces

**DataStore** (Transaction Script pattern):
- Provides domain-specific operations (Search, Upsert, Delete, etc.)
- Enforces business rules and validation
- Coordinates with related datastores
- Never directly exposes storage details

**Store** (Data Access Object pattern):
- Low-level CRUD operations
- Direct PostgreSQL interaction via `pkg/postgres`
- No business logic or cross-entity concerns
- Transaction-aware via context

### Typical DataStore Interface

```go
type DataStore interface {
    // Retrieval
    Get(ctx context.Context, id string) (*storage.Entity, bool, error)
    GetMany(ctx context.Context, ids []string) ([]*storage.Entity, error)
    Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
    Count(ctx context.Context, q *v1.Query) (int, error)

    // Mutation
    Upsert(ctx context.Context, obj *storage.Entity) error
    UpsertMany(ctx context.Context, objs []*storage.Entity) error
    Delete(ctx context.Context, id string) error
    DeleteMany(ctx context.Context, ids []string) error

    // Specialized operations (domain-specific)
    SearchListEntities(ctx context.Context, q *v1.Query) ([]*storage.ListEntity, error)
}
```

### PostgreSQL Store Pattern

All datastores use PostgreSQL with:
- **JSONB serialization**: Full proto stored in `serialized` column
- **Extracted fields**: Searchable/indexed columns (id, name, cluster_id, etc.)
- **Generated schema**: Auto-generated from protobuf definitions via `pkg/postgres/schema`
- **SAC integration**: Row-level security or query filters applied automatically

### Common Optimizations

1. **Keyed Mutex**: For safe concurrent updates to individual entities
   ```go
   type datastoreImpl struct {
       storage store.Store
       keyedMutex *concurrency.KeyedMutex
   }
   ```

2. **View-based queries**: Specialized query methods return lightweight projections
   ```go
   SearchListAlerts() ([]*storage.ListAlert, error) // Only ID, name, severity
   ```

3. **Batch operations**: UpsertMany for bulk inserts with transaction handling

4. **Hash-based change detection**: SHA256 hash column to skip no-op updates (e.g., deployments)

### How to Add a New Datastore

1. Define protobuf message in `proto/storage/{entity}.proto`
2. Run `make proto-generated-srcs`
3. Run `make generate-postgres-schemas` to generate schema
4. Create `central/{entity}/datastore/datastore.go` interface
5. Implement in `datastore_impl.go` with business logic
6. Create singleton in `singleton.go` using `sync.Once`
7. Wire dependencies in `central/startup.go`

See examples: `central/alert/datastore`, `central/deployment/datastore`, `central/policy/datastore`

---

## Service Pattern

**Applies to**: 80+ gRPC services across `central/*/service/`

### Purpose

The service pattern provides a thin gRPC API facade over datastores, enforcing authorization via SAC and translating between API types and storage types. Services are stateless and handle request/response serialization.

### Standard Structure

```
central/{entity}/service/
├── service.go              # Service interface (often matches proto)
├── service_impl.go         # Implementation with authorization checks
├── singleton.go            # Package-level singleton factory
└── mocks/                  # Generated mock implementations
```

### Typical Service Implementation

```go
type serviceImpl struct {
    v1.Unimplemented{Entity}ServiceServer
    datastore datastore.DataStore
}

func (s *serviceImpl) GetEntity(ctx context.Context, req *v1.ResourceByID) (*storage.Entity, error) {
    // 1. Authorization check via SAC
    if ok, err := sac.ForResource(resources.Entity).ReadAllowed(ctx); err != nil || !ok {
        return nil, sac.ErrResourceAccessDenied
    }

    // 2. Delegate to datastore
    entity, found, err := s.datastore.Get(ctx, req.GetId())
    if err != nil {
        return nil, err
    }
    if !found {
        return nil, errors.Wrap(errox.NotFound, "entity not found")
    }

    // 3. Return result
    return entity, nil
}
```

### Authorization Pattern

Services use SAC helpers from `pkg/sac`:

```go
// Single resource check
helper := sac.ForResource(resources.Deployment)
allowed, err := helper.ReadAllowed(ctx)

// Multi-resource check (any of)
multiHelper := sac.ForResources(
    sac.ForResource(resources.Alert),
    sac.ForResource(resources.Deployment),
)
allowed, err := multiHelper.ReadAllowedToAny(ctx)

// Scoped check (cluster/namespace)
scopeChecker := sac.GlobalAccessScopeChecker(ctx)
allowed := scopeChecker.
    AccessMode(storage.Access_READ_ACCESS).
    Resource(resources.Deployment).
    ClusterID(clusterID).
    Namespace(namespace).
    IsAllowed()
```

### Service Registration

Services register with the gRPC server in `central/startup.go`:

```go
alertService := alertService.Singleton()
v1.RegisterAlertServiceServer(grpcServer, alertService)
```

### Common Service Operations

**CRUD operations**:
- `Get{Entity}(ResourceByID) → Entity`
- `List{Entities}(RawQuery) → List{Entities}Response`
- `Count{Entities}(RawQuery) → Count{Entities}Response`
- `Post{Entity}({Entity}) → {Entity}` (create)
- `Put{Entity}({Entity}) → Empty` (update)
- `Delete{Entity}(ResourceByID) → Empty`

**Bulk operations**:
- `Stream{Entities}(RawQuery) → stream Entity`
- `Export{Entities}(Export{Entity}Request) → Export{Entity}Response`

**Specialized operations**: Domain-specific methods (e.g., `MarkAlertsResolved`, `ReassessPolicies`, `DryRunPolicy`)

### How Services Register

1. Service implementation in `central/{entity}/service/service_impl.go`
2. Singleton factory in `singleton.go`
3. Registration in `central/startup.go` or component-specific startup
4. Authorization enforced at method entry via SAC

See examples: `central/alert/service`, `central/deployment/service`, `central/policy/service`

---

## Singleton Pattern

**Applies to**: 200+ singletons across `pkg/` and `central/`

### Purpose

Package-level singletons provide global access to shared resources (datastores, services, clients) while ensuring thread-safe initialization. StackRox uses `sync.Once` for lazy initialization.

### Standard Implementation

```go
var (
    once sync.Once
    as   DataStore
)

// Singleton returns the singleton instance of the DataStore
func Singleton() DataStore {
    once.Do(func() {
        storage := postgres.New(globaldb.GetPostgres())
        as = New(storage)
    })
    return as
}
```

### Dependency Injection Pattern

For testability, singletons often accept injected dependencies:

```go
var (
    once sync.Once
    as   DataStore
)

func initialize() {
    storage := postgres.New(globaldb.GetPostgres())
    as = New(
        storage,
        processBaselineDataStore.Singleton(),
        notifier.Singleton(),
        connectionManager.Singleton(),
    )
}

func Singleton() DataStore {
    once.Do(initialize)
    return as
}
```

### Why Singletons?

1. **Global state**: Database connections, configuration, shared caches
2. **Lazy initialization**: Only initialize when first accessed
3. **Thread safety**: `sync.Once` guarantees single initialization
4. **Dependency management**: Central initialization point for wiring dependencies

### Testing with Singletons

Test code uses dependency injection to avoid singleton:

```go
// Production code uses singleton
ds := datastore.Singleton()

// Test code injects dependencies directly
storage := pgtest.ForT(t)
ds := datastore.New(storage, mockNotifier, mockBaselines)
```

### Anti-Pattern: Global Mutable State

Singletons should be **immutable after initialization**. Avoid:

```go
// BAD: Mutable global state
var GlobalConfig *Config

func UpdateConfig(c *Config) {
    GlobalConfig = c  // Race condition!
}
```

Instead, use context-based or injected configuration.

See examples: `central/alert/datastore/singleton.go`, `central/policy/datastore/singleton.go`

---

## Code Generation Pipeline

**Applies to**: 100+ generated files in `generated/`, `pkg/postgres/schema/`, stores

### Purpose

StackRox uses extensive code generation to maintain consistency between protobuf definitions, database schemas, Go code, and search mappings. This reduces manual maintenance and prevents drift.

### Generation Pipeline

```
Protocol Buffers (proto/storage/*.proto)
         ↓
    [make proto-generated-srcs]
         ↓
    Generated Go Code (generated/storage/*.pb.go)
         ↓
    [walker.Walk() - Reflection on proto types]
         ↓
    PostgreSQL Schema Definitions (pkg/postgres/schema/*.go)
         ↓
    [GORM AutoMigrate or raw SQL]
         ↓
    Database Tables (deployments, alerts, policies, etc.)
```

### Key Tools

1. **protoc**: Protocol buffer compiler
   - Input: `proto/storage/*.proto`, `proto/api/v1/*.proto`
   - Output: `generated/storage/*.pb.go`, `generated/api/v1/*.pb.go`
   - Command: `make proto-generated-srcs`

2. **walker.Walk()**: Schema reflection (`pkg/postgres/walker/`)
   - Input: Go `reflect.Type` of protobuf message
   - Output: `walker.Schema` object (table definition)
   - Parses struct tags: `sql:"pk,index=btree,fk(Image:id)"`

3. **pg-table-bindings**: Store/schema generator (`tools/generate-helpers/pg-table-bindings/`)
   - Input: Protobuf types + templates
   - Output: Schema files, store implementations, search mappings
   - Command: `make generate-postgres-schemas`

4. **mockgen**: Mock generator
   - Input: Go interfaces
   - Output: `mocks/*.go` files
   - Command: `make go-generated-srcs`

### What's Auto-Generated vs Hand-Written

**Auto-Generated** (never edit manually):
- `generated/storage/*.pb.go` - Protobuf Go bindings
- `pkg/postgres/schema/{entity}.go` - Schema definitions
- `pkg/postgres/schema/all.go` - Schema registry (partially)
- Store implementations (when using pg-table-bindings templates)
- `mocks/*.go` - Mock implementations

**Hand-Written**:
- `proto/storage/*.proto` - Protobuf definitions (source of truth)
- `{entity}/datastore/datastore_impl.go` - Business logic
- `{entity}/service/service_impl.go` - API handlers
- Migration files (`migrator/migrations/m_*_to_m_*/`)
- Custom queries and specialized store methods

### Struct Tag Syntax

PostgreSQL schema generation uses `sql:` tags:

```go
type Alert struct {
    Id         string `protobuf:"name=id" sql:"pk"`                          // Primary key
    PolicyId   string `protobuf:"name=policy_id" sql:"index=btree"`          // Indexed
    ClusterId  string `protobuf:"name=cluster_id" sql:"fk(Cluster:id)"`      // Foreign key
    Deployment string `protobuf:"name=deployment_name" sql:"unique"`         // Unique constraint
    State      int32  `protobuf:"name=state" sql:"type(smallint)"`           // Override type
}
```

Tag options:
- `pk`: Primary key
- `index`, `index=btree`, `index=hash`: Index creation
- `fk(TypeName:field)`: Foreign key reference
- `unique`: Unique constraint
- `type(sql_type)`: Override column type
- `ignore_pk`, `ignore_fks`: Suppress constraints in nested tables

### How to Regenerate Code

**After changing protobuf definitions**:
```bash
make proto-generated-srcs          # Regenerate Go bindings
make generate-postgres-schemas     # Regenerate schema/store code
make go-generated-srcs             # Regenerate mocks/other
```

**After schema changes**:
```bash
DESCRIPTION="add column xyz" make bootstrap_migration
# Edit migrations/m_{N}_to_m_{N+1}_{desc}/migration_impl.go
# Add backward-compatible schema changes
```

### Common Pitfalls

1. **Editing generated files**: Always edit source (proto/templates), not generated code
2. **Forgetting to regenerate**: CI will catch this but wastes time
3. **Breaking migrations**: Schema changes must be backward-compatible (see `migrator/README.md`)
4. **Changing NamingStrategy**: Global GORM naming convention affects all tables (`pgutils/utils.go:30`)

See examples: `pkg/postgres/schema/alerts.go`, `pkg/postgres/walker/walker.go`

---

## SAC (Scoped Access Control)

**Applies to**: All datastores, services, and search queries

### Purpose

SAC provides fine-grained, hierarchical authorization across the entire platform. It enables role-based access control (RBAC) at multiple scope levels: global, access mode, resource type, cluster, and namespace.

### Scope Hierarchy

```
GlobalScope
    └── AccessModeScope (READ_ACCESS | READ_WRITE_ACCESS)
        └── ResourceScope (Alert, Deployment, Image, Policy, etc.)
            └── ClusterScope (specific cluster ID)
                └── NamespaceScope (specific namespace name)
```

### Three-State Logic

Access checks return one of three states:
- **Excluded**: No access permitted
- **Partial**: Access to some but not all child scopes
- **Included**: Full access to scope and all descendants

### Common Usage Patterns

**In Services** (authorization check):
```go
// Check if user has read access to resource
if ok, err := sac.ForResource(resources.Deployment).ReadAllowed(ctx); err != nil || !ok {
    return nil, sac.ErrResourceAccessDenied
}

// Check scoped access (cluster/namespace)
scopeChecker := sac.GlobalAccessScopeChecker(ctx)
if !scopeChecker.
    AccessMode(storage.Access_READ_ACCESS).
    Resource(resources.Deployment).
    ClusterID(clusterID).
    Namespace(namespace).
    IsAllowed() {
    return nil, sac.ErrResourceAccessDenied
}
```

**In Datastores** (filtering):
```go
// Get effective access scope for query filtering
scopeChecker := sac.GlobalAccessScopeChecker(ctx)
scopeTree, err := scopeChecker.EffectiveAccessScope(resources.Deployment)
if err != nil {
    return nil, err
}

// Build SAC query filter
sacQueryFilter, err := sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
if err != nil {
    return nil, err
}

// Combine with user query
query := search.ConjunctionQuery(userQuery, sacQueryFilter)
```

**In Search Queries**:
```go
// SAC filters are automatically injected by search layer
// For namespace-scoped resources:
// (ClusterID = "cluster-1" AND Namespace IN ("ns-a", "ns-b")) OR
// (ClusterID = "cluster-2" AND Namespace IN ("ns-x", "ns-y"))
```

### Resource Scope Levels

**Global-scoped resources**:
- `resources.Access` - Authentication/authorization
- `resources.Administration` - Platform configuration
- `resources.Integration` - External integrations

**Cluster-scoped resources**:
- `resources.Cluster`
- `resources.Node`
- `resources.Compliance`

**Namespace-scoped resources**:
- `resources.Deployment`
- `resources.Alert`
- `resources.Image`
- `resources.NetworkPolicy`

### Context-Based Scope Checkers

SAC scope checkers are stored in `context.Context`:

```go
// Retrieve global scope checker from context
scopeChecker := sac.GlobalAccessScopeChecker(ctx)

// Check access
allowed := scopeChecker.
    Resource(resources.Deployment).
    AccessMode(storage.Access_READ_ACCESS).
    IsAllowed()
```

Services attach scope checkers via gRPC interceptors before request handling.

### Effective Access Scope

The `effectiveaccessscope.ScopeTree` represents computed access topology:

```go
type ScopeTree struct {
    State           scopeState                        // Excluded, Partial, Included
    Clusters        map[string]*clustersScopeSubTree  // keyed by cluster name
    clusterIDToName map[string]string
}
```

This is computed by intersecting user permissions with actual cluster/namespace topology.

### Testing with SAC

Test utilities in `pkg/sac/testutils`:

```go
// Create test scope checker with specific access
checker := sac.TestScopeCheckerCoreFromAccessResourceMap(t, []permissions.ResourceWithAccess{
    {Resource: resources.Deployment, Access: storage.Access_READ_ACCESS},
})

// Or use pre-configured test contexts
ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
    sac.AllowFixedScopes(
        sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
        sac.ResourceScopeKeys(resources.Deployment),
        sac.ClusterScopeKeys("cluster-1"),
    ))
```

See: `pkg/sac/README.md`, `central/alert/datastore/datastore_impl.go` (SAC integration)

---

## Feature Flags

**Applies to**: 100+ feature flags across `pkg/env/` and `pkg/features/`

### Purpose

Feature flags control runtime behavior without code changes, enabling gradual rollouts, A/B testing, and emergency kill switches. StackRox uses environment variables (`ROX_*`) for configuration.

### Two Systems

1. **pkg/env**: Environment variable settings (runtime configuration)
2. **pkg/features**: Boolean feature flags (feature toggles)

### Environment Variable Pattern (`pkg/env`)

Typed settings with validation:

```go
// Boolean flag
var ProcessBaselineRisk = env.RegisterBooleanSetting("ROX_PROCESS_BASELINE_RISK", true)

// Integer with validation
var MaxParallelImageScan = env.RegisterIntegerSetting("ROX_MAX_PARALLEL_IMAGE_SCAN_INTERNAL", 30,
    env.WithMin(1), env.WithMax(100))

// Duration
var ScanTimeout = env.RegisterDurationSetting("ROX_SCAN_TIMEOUT", 10*time.Minute)

// String
var CentralEndpoint = env.RegisterSetting("ROX_CENTRAL_ENDPOINT",
    env.WithDefault("central.stackrox.svc:443"))
```

### Usage

```go
// Check setting value
if env.ProcessBaselineRisk.BooleanSetting() {
    // Process baseline risk calculation enabled
}

// Get integer value
maxScans := env.MaxParallelImageScan.IntegerSetting()

// Get duration
timeout := env.ScanTimeout.DurationSetting()
```

### Naming Conventions

- Prefix: `ROX_` for all StackRox settings
- Format: `ROX_{COMPONENT}_{FEATURE}_{SETTING}`
- Examples:
  - `ROX_POSTGRES_DEFAULT_TIMEOUT`
  - `ROX_SENSOR_CONNECTION_RETRY_INTERVAL`
  - `ROX_ENABLE_CENTRAL_DIAGNOSTICS`

### Feature Flags (`pkg/features`)

Simple boolean toggles for features:

```go
// In pkg/features/features.go
var FlattenImageData = registerFeature("ROX_FLATTEN_IMAGE_DATA", false)

// Usage
if features.FlattenImageData.Enabled() {
    // Use flattened image schema
} else {
    // Use legacy image schema
}
```

### When to Use Environment Variables

Use environment variables for:
- **Configuration**: Timeouts, buffer sizes, limits, endpoints
- **Tuning**: Performance parameters that vary by deployment
- **Kill switches**: Emergency disable of expensive features
- **Testing**: Debug modes, scale testing, offline mode

**Don't use for**:
- Feature flags that should be managed via UI/API
- Values that change frequently without restart
- Secrets (use Kubernetes secrets instead)

### How to Add a New Environment Variable

1. **Define in appropriate file** (`pkg/env/{component}.go`):
   ```go
   var MyNewSetting = env.RegisterIntegerSetting(
       "ROX_MY_NEW_SETTING",
       42,  // default value
       env.WithMin(1),
       env.WithMax(100),
   )
   ```

2. **Document in code comment**:
   ```go
   // MyNewSetting controls the maximum number of concurrent operations.
   // Min: 1, Max: 100, Default: 42
   var MyNewSetting = ...
   ```

3. **Add to `pkg/env/README.md`** under appropriate category

4. **Use in code**:
   ```go
   maxOps := env.MyNewSetting.IntegerSetting()
   ```

5. **Expose via Helm chart** (if user-configurable):
   - Add to `image/templates/helm/stackrox-central/values.yaml`
   - Add to `image/templates/helm/stackrox-central/templates/01-central-*.yaml`

### Common Settings by Category

**PostgreSQL**: Timeouts, retry intervals, connection pooling
```go
ROX_POSTGRES_DEFAULT_TIMEOUT=60s
ROX_POSTGRES_QUERY_RETRY_TIMEOUT=5m
```

**Sensor**: Buffer sizes, retry intervals, connection settings
```go
ROX_SENSOR_CONNECTION_RETRY_INITIAL_INTERVAL=10s
ROX_EVENT_PIPELINE_QUEUE_SIZE=1000
```

**Scanning**: Parallelism, timeouts, SBOM limits
```go
ROX_MAX_PARALLEL_IMAGE_SCAN_INTERNAL=30
ROX_SCAN_TIMEOUT=10m
```

**Compliance**: Scan intervals, operator versions
```go
ROX_COMPLIANCE_SCAN_TIMEOUT=15m
ROX_NODE_SCANNING_INTERVAL=4h
```

See: `pkg/env/README.md`, `pkg/features/`, individual package documentation for environment variables

---

## Search Integration

**Applies to**: All entities with search capabilities (100+ resources)

### Purpose

StackRox provides unified search across all entities (deployments, alerts, images, policies, etc.) with query parsing, field mapping, and PostgreSQL translation. Search queries are expressed in a domain-specific language and automatically converted to SQL.

### Search Flow

```
User Query String ("Deployment:nginx Cluster:prod")
         ↓
    [pkg/search/parser] - Parse to search.Query
         ↓
    [pkg/search/postgres/mapping] - Map fields to columns
         ↓
    [pkg/search/postgres/query] - Build SQL WHERE clause
         ↓
    PostgreSQL Query Execution
         ↓
    Results (with SAC filtering applied)
```

### Field Label Registration

Each entity registers searchable fields via schema:

```go
// In pkg/postgres/schema/alerts.go (auto-generated)
schema.SetOptionsMap(search.Walk(v1.SearchCategory_ALERTS, "alert", (*storage.Alert)(nil)))
schema.SetSearchScope(v1.SearchCategory_ALERTS)
mapping.RegisterCategoryToTable(v1.SearchCategory_ALERTS, schema)
```

### Search Query Syntax

**Basic search**:
```
Deployment Name:nginx
```

**Boolean operators**:
```
Deployment Name:nginx AND Cluster:prod
Policy Severity:CRITICAL OR Policy Severity:HIGH
```

**Negation**:
```
NOT Deployment Name:test
```

**Scoped search** (cross-entity):
```
Deployment:nginx+Cluster:prod  // Deployment in specific cluster
Image CVE:CVE-2021-44228       // Images with specific CVE
```

**Field operators**:
```
Risk Score:>50                  // Numeric comparison
Created:>2024-01-01            // Date comparison
Deployment Name:r/^nginx-*/    // Regex matching
```

### Field Mapping

Search fields map to database columns via `OptionsMap`:

```go
type FieldLabel struct {
    Name       string      // "Deployment Name"
    ColumnName string      // "deployments.name"
    Type       DataType    // String, Numeric, DateTime, etc.
    Category   SearchCategory
}
```

Auto-generated from protobuf field names:
- `name` → "Deployment Name"
- `cluster_id` → "Cluster ID"
- `risk_score` → "Risk Score"

### PostgreSQL Translation

Search queries convert to SQL WHERE clauses:

**Example**:
```
Input:  Deployment Name:nginx AND Cluster:prod
Output: deployments.name ILIKE '%nginx%' AND deployments.cluster_id = 'prod-cluster-id'
```

**With SAC filtering**:
```sql
SELECT * FROM deployments
WHERE deployments.name ILIKE '%nginx%'
  AND deployments.cluster_id = 'prod-cluster-id'
  AND (
      (deployments.cluster_id = 'cluster-1' AND deployments.namespace IN ('ns-a', 'ns-b'))
      OR
      (deployments.cluster_id = 'cluster-2' AND deployments.namespace IN ('ns-x', 'ns-y'))
  )
```

### Search Categories

Categories group related entities for scoped search:

```go
v1.SearchCategory_ALERTS
v1.SearchCategory_DEPLOYMENTS
v1.SearchCategory_IMAGES
v1.SearchCategory_POLICIES
v1.SearchCategory_CLUSTERS
v1.SearchCategory_NAMESPACES
v1.SearchCategory_NODES
v1.SearchCategory_VULNERABILITIES
// ... 50+ categories
```

### Cross-Entity Search

Search can join across related entities:

```
Deployment+Image CVE:CVE-2021-44228
```

This finds deployments that use images with the specified CVE.

### How Search Works in Datastores

```go
func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
    // 1. Get SAC scope tree
    scopeTree, err := sac.GlobalAccessScopeChecker(ctx).EffectiveAccessScope(resources.Deployment)

    // 2. Build SAC query filter
    sacFilter, err := sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)

    // 3. Combine with user query
    query := search.ConjunctionQuery(q, sacFilter)

    // 4. Execute search (translates to SQL)
    return ds.storage.Search(ctx, query)
}
```

### Adding Search to New Entity

1. **Protobuf field tags**: Ensure fields have `search:` tags
   ```protobuf
   string name = 1 [(gogoproto.moretags) = "search:\"Deployment Name\""];
   ```

2. **Schema registration**: Auto-generated by `make generate-postgres-schemas`
   ```go
   schema.SetOptionsMap(search.Walk(...))
   schema.SetSearchScope(v1.SearchCategory_DEPLOYMENTS)
   ```

3. **Datastore search methods**: Implement Search/Count operations
   ```go
   func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
   ```

4. **Service exposure**: Expose via gRPC service
   ```go
   func (s *serviceImpl) ListDeployments(ctx context.Context, req *v1.RawQuery) (*v1.ListDeploymentsResponse, error)
   ```

### Derived Fields

Some search fields are computed/aggregated:

```go
// CVE Count field (aggregated from related CVEs)
// Risk Score field (computed from multiple factors)
```

Limitations: `UnsupportedDerivedFieldDataTypes` = `{StringArray, EnumArray, IntArray, Map}`

See: `pkg/search/README.md`, `pkg/search/postgres/`, `pkg/postgres/schema/`

---

## Event Pipeline (Sensor)

**Applies to**: All Kubernetes resource types monitored by Sensor

### Purpose

Sensor monitors Kubernetes resources (Deployments, Pods, Nodes, NetworkPolicies, etc.) and streams events to Central for security analysis. The event pipeline provides a standard flow for processing, deduplicating, and forwarding resource changes.

### Standard Flow

```
Kubernetes API Watch
         ↓
    Listener (K8s event handler)
         ↓
    Resolver (Enrichment with metadata)
         ↓
    Output Queue (Buffered channel)
         ↓
    Detector (Policy evaluation - Central-side)
         ↓
    Deduper (Event deduplication)
         ↓
    Central Storage (PostgreSQL)
```

### Component Breakdown

**1. Listener** (`sensor/kubernetes/listener/resources/`)

Watches Kubernetes API for resource changes:

```go
type Listener interface {
    // Process add/update/delete events from K8s
    ProcessEvent(obj interface{}, action ResourceAction)
}
```

Implemented for each resource type:
- `deployment_listener.go`
- `pod_listener.go`
- `node_listener.go`
- `networkpolicy_listener.go`
- etc.

**2. Resolver** (within Listener or separate)

Enriches events with additional context:
- Cluster ID/name
- Namespace metadata
- Related resources (Pods → Deployment)
- Container image references

**3. Output Queue** (`sensor/common/sensor/`)

Buffers events for transmission to Central:

```go
type outputQueue struct {
    C <-chan *central.MsgFromSensor  // Read-only channel
    stopper concurrency.Stopper
}
```

Buffer sizes configurable via environment variables:
- `ROX_EVENT_PIPELINE_QUEUE_SIZE=1000`
- `ROX_RESPONSES_CHANNEL_BUFFER_SIZE=100000`

**4. Detector** (`central/detection/`)

Central-side policy evaluation:
- Evaluates resources against active policies
- Generates alerts for policy violations
- Applies lifecycle stages (BUILD, DEPLOY, RUNTIME)

**5. Deduper** (`central/sensor/service/pipeline/`)

Prevents duplicate events:
- Tracks event hashes per cluster
- Configurable via `ROX_MAX_EVENT_HASH_SIZE=1000000`
- State synchronized between Sensor and Central

**6. Storage**

Final persistence in PostgreSQL:
- Deployments → `deployments` table
- Alerts → `alerts` table
- Network flows → `network_flows` table

### Event Types by Resource

**Deployments**:
- Add: New deployment created
- Update: Configuration changed (image, replicas, env vars)
- Delete: Deployment removed

**Pods**:
- Add: Pod scheduled
- Update: Pod phase changed (Pending → Running → Succeeded)
- Delete: Pod terminated

**Nodes**:
- Add: Node joined cluster
- Update: Node conditions changed (Ready, DiskPressure, etc.)
- Delete: Node removed from cluster

**NetworkPolicies**:
- Add: New policy created
- Update: Policy rules modified
- Delete: Policy removed

### Resource-Specific Patterns

**Deployments** (most complex):
```
K8s Deployment Event
    → Listener extracts containers
    → Resolver enriches with image metadata
    → Output queue buffers
    → Central receives and stores
    → Detector evaluates policies
    → Alerts generated for violations
    → Risk scores calculated
```

**Process Indicators** (runtime events):
```
Collector (eBPF) detects process
    → Sensor receives via gRPC stream
    → Process filter applies (fan-out limiting)
    → Output queue buffers
    → Central receives
    → Detector checks runtime policies
    → Alerts generated if suspicious
```

**Network Flows**:
```
Collector (eBPF) observes network connection
    → Sensor receives connection updates
    → Network flow computer aggregates
    → Output queue buffers
    → Central receives
    → Network graph updated
    → Baseline learning applied
```

### Offline Resilience

Sensor buffers events when Central is unreachable:

```go
// Buffer sizes during offline mode
ROX_SENSOR_NETFLOW_OFFLINE_BUFFER_SIZE=100
ROX_SENSOR_PROCESS_INDICATOR_BUFFER_SIZE=50000
```

When connection restored:
1. Sensor sends deduper state to Central
2. Buffered events transmitted
3. Deduper filters already-seen events
4. Normal operation resumes

### How to Add New Resource Type

1. **Create Listener** (`sensor/kubernetes/listener/resources/{resource}_listener.go`):
   ```go
   type resourceListener struct {
       outputQueue  queue.OutputQueue
       clusterName  string
   }

   func (l *resourceListener) ProcessEvent(obj interface{}, action ResourceAction) {
       resource := obj.(*v1.Resource)
       // Enrich and convert to proto
       msg := &central.MsgFromSensor{
           Msg: &central.MsgFromSensor_Event{
               Event: &central.SensorEvent{
                   Resource: convertToProto(resource),
                   Action:   action,
               },
           },
       }
       l.outputQueue.Send(msg)
   }
   ```

2. **Register in Dispatcher** (`sensor/kubernetes/listener/dispatcher.go`):
   ```go
   dispatcher.RegisterResourceHandler(ctx, &resourceListener{...})
   ```

3. **Add Central Handler** (`central/sensor/service/pipeline/`):
   ```go
   func (p *pipelineImpl) ProcessResourceEvent(ctx context.Context, event *central.SensorEvent) error {
       // Store in datastore
       // Trigger detection
   }
   ```

4. **Define Protobuf Message** (`proto/internalapi/central/sensor_events.proto`):
   ```protobuf
   message SensorEvent {
       oneof resource {
           storage.Resource resource = N;
       }
   }
   ```

### Common Pipeline Issues

1. **Buffer Overflow**: Events dropped when queue full
   - Increase buffer sizes or reduce event volume
   - Monitor with metrics: `sensor_event_pipeline_queue_size`

2. **Deduper State Loss**: Duplicate events after Sensor restart
   - Deduper state timeout: `ROX_DEDUPER_STATE_TIMEOUT=30s`
   - Events may be reprocessed briefly

3. **Resource Leaks**: Memory growth from unbounded caches
   - Purger cycles clean stale data
   - `ROX_ENRICHMENT_PURGER_UPDATE_CYCLE=30m`

See: `sensor/kubernetes/listener/`, `central/detection/`, `central/sensor/service/pipeline/`

---

---

## Error Handling Patterns

**Applies to**: All packages using errox package

### Purpose

StackRox uses the `pkg/errox` package for consistent error handling across the codebase, providing semantic error types, gRPC status code mapping, and error wrapping conventions.

### Sentinel Errors

Sentinel errors are predefined error values that represent specific error conditions:

```go
// Common sentinel errors from pkg/errox
errox.NotFound         // Entity not found (gRPC: NOT_FOUND)
errox.AlreadyExists    // Duplicate entity (gRPC: ALREADY_EXISTS)
errox.InvalidArgs      // Invalid arguments (gRPC: INVALID_ARGUMENT)
errox.Unauthenticated  // Authentication failed (gRPC: UNAUTHENTICATED)
errox.PermissionDenied // Authorization failed (gRPC: PERMISSION_DENIED)
```

**Usage**:
```go
func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.Entity, bool, error) {
    entity, found, err := ds.storage.Get(ctx, id)
    if err != nil {
        return nil, false, err
    }
    if !found {
        return nil, false, errors.Wrap(errox.NotFound, "entity not found")
    }
    return entity, true, nil
}
```

### gRPC Status Mapping

errox errors automatically map to gRPC status codes:

- `errox.NotFound` → `codes.NotFound`
- `errox.AlreadyExists` → `codes.AlreadyExists`
- `errox.InvalidArgs` → `codes.InvalidArgument`
- `errox.Unauthenticated` → `codes.Unauthenticated`
- `errox.PermissionDenied` → `codes.PermissionDenied`

### Error Wrapping Conventions

**Add context while preserving type**:
```go
if err != nil {
    return errors.Wrap(err, "failed to fetch deployment")
}
```

**Wrap with sentinel error**:
```go
if !found {
    return errors.Wrap(errox.NotFound, "deployment not found")
}
```

**Check error type**:
```go
if errors.Is(err, errox.NotFound) {
    // Handle not found case
}
```

### PostgreSQL Error Translation

PostgreSQL errors are automatically translated to errox types:

- `23505` (unique_violation) → `errox.AlreadyExists`
- `23503` (foreign_key_violation) → `errox.ReferencedObjectNotFound` or `errox.ReferencedByAnotherObject`

This happens in `pkg/postgres/error.go` via `toErrox()`.

See: `pkg/errox/`, `pkg/postgres/error.go`

---

## Transaction Patterns

**Applies to**: All datastores using PostgreSQL

### Purpose

PostgreSQL transaction handling in StackRox uses a nested transaction model where outer transactions control lifecycle while inner transactions are NOOPs. This allows composing multiple store operations in a single transaction without explicit plumbing.

### Transaction Modes

**outer**: Transaction created outside store, passed via context
```go
tx, err := db.Begin(ctx)
ctx = postgres.ContextWithTx(ctx, tx)
// All store operations in this context use the same transaction
deploymentStore.Upsert(ctx, deployment)
imageStore.Upsert(ctx, image)
tx.Commit(ctx)
```

**inner**: Nested transaction (commit/rollback are NOOPs)
```go
// Inside store method
tx, err := postgres.GetTransaction(ctx, ds.storage)
// Uses existing tx from context if present, creates new one if not
// Commit/rollback handled by outer transaction
```

**original**: Transaction created and used only within a single store
```go
// Rare case - store manages its own transaction
tx, err := ds.storage.Begin(ctx)
// ... operations ...
tx.Commit(ctx)
```

### When to Use Transactions

**Use transactions for**:
- Multi-entity updates that must be atomic
- Updates requiring consistency across tables
- Operations that must rollback together on failure

**Don't use transactions for**:
- Single row updates (already atomic)
- Read-only operations (unless consistency required)
- Long-running operations (locks resources)

### Context-Based Transaction Management

Transactions are stored in `context.Context`:

```go
// Create and attach transaction to context
tx, err := db.Begin(ctx)
ctx = postgres.ContextWithTx(ctx, tx)

// Retrieve transaction from context
tx := postgres.TxFromContext(ctx)
```

Store methods automatically use transactions from context when present.

### Commit/Rollback Behavior

- **outer mode**: Rollback tolerates `pgx.ErrTxClosed` (may have been rolled back by inner)
- **inner mode**: Commit/Rollback are NOOPs (parent transaction controls lifecycle)
- Uses `context.WithoutCancel()` to ensure commit/rollback complete even if parent context is cancelled

### Example: Multi-Store Transaction

```go
func (s *serviceImpl) UpdateDeploymentWithImages(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) error {
    // Create transaction
    tx, err := postgres.GetPostgres().Begin(ctx)
    if err != nil {
        return err
    }
    defer func() {
        if err != nil {
            tx.Rollback(ctx)
        }
    }()

    // Attach to context
    ctx = postgres.ContextWithTx(ctx, tx)

    // All operations share transaction
    if err := s.deploymentStore.Upsert(ctx, deployment); err != nil {
        return err
    }

    if err := s.imageStore.UpsertMany(ctx, images); err != nil {
        return err
    }

    // Commit transaction
    return tx.Commit(ctx)
}
```

See: `pkg/postgres/README.md`, `pkg/postgres/tx.go`, `pkg/postgres/context.go`

---

## Testing Patterns

**Applies to**: All test files across the codebase

### Purpose

StackRox uses consistent testing patterns across unit tests, integration tests, and end-to-end tests to ensure reliability and maintainability.

### Testify Suites

Many tests use `github.com/stretchr/testify/suite` for structured test organization:

```go
type DeploymentDataStoreTestSuite struct {
    suite.Suite
    datastore DataStore
    storage   store.Store
    db        *pgxpool.Pool
}

func TestDeploymentDataStore(t *testing.T) {
    suite.Run(t, new(DeploymentDataStoreTestSuite))
}

func (s *DeploymentDataStoreTestSuite) SetupTest() {
    s.db = pgtest.ForT(s.T())
    s.storage = postgres.New(s.db)
    s.datastore = New(s.storage)
}

func (s *DeploymentDataStoreTestSuite) TearDownTest() {
    s.db.Close()
}

func (s *DeploymentDataStoreTestSuite) TestGet() {
    // Test logic using s.datastore
}
```

**Running suite tests**:
- Run entire suite: `go test ./package -run TestDeploymentDataStore`
- Run specific subtest: `go test ./package -run TestDeploymentDataStore/TestGet`
- **Wrong**: `go test ./package -run TestGet` (won't find it - it's not a top-level function)

### SAC Tests

SAC (Scoped Access Control) tests validate authorization logic:

```go
func (s *AlertDataStoreTestSuite) TestSACFiltering() {
    // Create test scope checker with specific access
    checker := sac.TestScopeCheckerCoreFromAccessResourceMap(s.T(), []permissions.ResourceWithAccess{
        {Resource: resources.Alert, Access: storage.Access_READ_ACCESS},
    })

    // Create context with scope checker
    ctx := sac.WithGlobalAccessScopeChecker(context.Background(), checker)

    // Test that queries respect SAC
    results, err := s.datastore.Search(ctx, search.EmptyQuery())
    s.NoError(err)
    // Verify results match expected scope
}
```

### PostgreSQL Integration Tests

Tests requiring PostgreSQL use build tag `sql_integration`:

```go
//go:build sql_integration

package postgres

import (
    "testing"
    "github.com/stackrox/stackrox/pkg/postgres/pgtest"
)

func TestDeploymentStore(t *testing.T) {
    db := pgtest.ForT(t)  // Creates ephemeral test database
    store := New(db)
    // Test logic
} // Database automatically cleaned up via t.Cleanup()
```

**Running PostgreSQL tests**:
```bash
# Start test database
docker run --rm --env POSTGRES_USER="$USER" --env POSTGRES_HOST_AUTH_METHOD=trust --publish 5432:5432 docker.io/library/postgres:15

# Run integration tests
go test -v -tags sql_integration ./central/deployment/datastore/internal/store/postgres
```

### Mock Generation

Mocks are auto-generated using `mockgen`:

```go
//go:generate mockgen -package mocks -destination mocks/datastore.go github.com/stackrox/stackrox/central/deployment/datastore DataStore
```

**Regenerate mocks**: `make go-generated-srcs`

**Using mocks in tests**:
```go
func TestService(t *testing.T) {
    mockDataStore := mocks.NewMockDataStore(gomock.NewController(t))
    mockDataStore.EXPECT().Get(gomock.Any(), "id123").Return(deployment, true, nil)

    service := New(mockDataStore)
    // Test logic
}
```

### End-to-End Tests

E2E tests validate complete workflows in real environments:

**Location**: `tests/` directory
**Framework**: Go with build tags (`test_e2e`, `compliance`, `destructive`, etc.)
**Execution**: GitHub Actions, OpenShift CI

**Common patterns**:
```go
//go:build test_e2e

func TestDeploymentLifecycle(t *testing.T) {
    // Deploy StackRox
    // Create test deployment
    // Wait for detection
    // Verify alerts generated
    // Clean up
}
```

### QA Tests (Groovy/Spock)

Backend integration tests use Spock framework:

**Location**: `qa-tests-backend/`
**Framework**: Spock 2.4 (BDD) + JUnit 5
**Language**: Groovy 4.0

**Test structure**:
```groovy
class DeploymentTest extends BaseSpecification {
    def "should detect deployment and generate alert"() {
        given:
        "a policy that matches the deployment"
        def policy = createPolicy()

        when:
        "deploying a violating workload"
        def deployment = deployNginx()

        then:
        "an alert is generated"
        waitForViolation(deployment.id, policy.id)

        cleanup:
        deleteDeployment(deployment)
    }
}
```

**Test groups**:
- **BAT** (Build Acceptance Tests): Critical paths, 2-3 hours
- **SMOKE**: Core functionality, 5-10 minutes
- **Full suite**: All tests, 6-8 hours

**Running**:
```bash
./gradlew test --tests "DeploymentTest"
./gradlew test -Dgroups="BAT"
```

See: `doc/tests/README.md`, `doc/qa-tests-backend/README.md`, `pkg/postgres/pgtest/`, `pkg/sac/testutils/`

---

## Layer Responsibility Matrix

**Purpose**: Clarify the separation of concerns across StackRox's layered architecture.

| Layer | Responsibility | Example |
|-------|---------------|---------|
| **Proto** | Data definition, API contracts | `proto/storage/deployment.proto` - Defines Deployment message structure |
| **Store** | PostgreSQL CRUD operations, no business logic | `central/deployment/datastore/internal/store/postgres/store_impl.go` - Raw SQL queries, row mapping |
| **DataStore** | Business logic, validation, SAC enforcement, cross-entity coordination | `central/deployment/datastore/datastore_impl.go` - Risk scoring, baseline checks, keyed mutex |
| **Service** | gRPC API handlers, authorization checks, request/response translation | `central/deployment/service/service_impl.go` - SAC checks, API endpoint implementation |
| **Detector** | Policy evaluation, alert generation, threat detection | `central/detection/` - Evaluates deployments against policies |

### Layer Details

**Proto Layer** (proto/storage/, proto/api/):
- Protocol buffer message definitions
- API service definitions
- No implementation code
- Generated code in `generated/`

**Store Layer** (datastore/internal/store/postgres/):
- Direct PostgreSQL interaction via `pkg/postgres`
- CRUD operations: Get, Upsert, Delete, Count, Walk
- Transaction-aware via context
- No business rules or cross-entity logic
- Generated or hand-written implementations

**DataStore Layer** (datastore/):
- Wraps Store with business logic
- Enforces validation and invariants
- Coordinates with related datastores
- Applies SAC filters to queries
- Uses keyed mutex for safe concurrent updates
- Domain-specific operations (e.g., SearchListDeployments)

**Service Layer** (service/):
- Thin gRPC facade over DataStore
- Authorization via SAC (ReadAllowed, WriteAllowed)
- Request validation
- API-level error handling
- Stateless (no business logic)
- Maps between API types and storage types

**Detector Layer** (central/detection/):
- Policy evaluation engine
- Alert generation from violations
- Lifecycle stage filtering (BUILD, DEPLOY, RUNTIME)
- Threat detection algorithms
- Independent of storage/service layers

### Anti-Patterns to Avoid

**Don't put business logic in Store**:
```go
// BAD: Store doing validation
func (s *storeImpl) Upsert(ctx context.Context, obj *storage.Deployment) error {
    if obj.GetRiskScore() > 100 {  // Business logic in Store!
        return errors.New("invalid risk score")
    }
    // ...
}

// GOOD: DataStore does validation
func (ds *datastoreImpl) Upsert(ctx context.Context, obj *storage.Deployment) error {
    if err := ds.validate(obj); err != nil {
        return err
    }
    return ds.storage.Upsert(ctx, obj)
}
```

**Don't put business logic in Service**:
```go
// BAD: Service computing risk
func (s *serviceImpl) GetDeployment(ctx context.Context, req *v1.ResourceByID) (*storage.Deployment, error) {
    deployment, _, err := s.datastore.Get(ctx, req.GetId())
    deployment.RiskScore = computeRisk(deployment)  // Business logic in Service!
    return deployment, err
}

// GOOD: DataStore computes risk
func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.Deployment, bool, error) {
    deployment, found, err := ds.storage.Get(ctx, id)
    if found {
        deployment.RiskScore = ds.computeRisk(deployment)
    }
    return deployment, found, err
}
```

**Don't bypass SAC in DataStore**:
```go
// BAD: Direct store access
func (ds *datastoreImpl) GetAll(ctx context.Context) ([]*storage.Deployment, error) {
    return ds.storage.GetAll(ctx)  // No SAC filtering!
}

// GOOD: Apply SAC filter
func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
    scopeTree, err := sac.GlobalAccessScopeChecker(ctx).EffectiveAccessScope(resources.Deployment)
    sacFilter, err := sac.BuildClusterNamespaceLevelSACQueryFilter(scopeTree)
    query := search.ConjunctionQuery(q, sacFilter)
    return ds.storage.Search(ctx, query)
}
```

See: Datastore Pattern, Service Pattern, SAC sections above

---

## New Developer Path

**Purpose**: Recommended learning path for new developers to understand StackRox patterns.

### Stage 1: Datastore Pattern
**Time**: 2-3 days
**Focus**: Understand the core data access layer

**Study**:
1. Read `doc/PATTERNS.md` Datastore Pattern section
2. Examine `central/alert/datastore/datastore_impl.go`
3. Compare Store interface vs DataStore interface
4. Trace a single Upsert call through the layers

**Exercises**:
- Add a new method to an existing DataStore
- Write a unit test using mocks
- Understand keyed mutex usage for concurrent updates

**Key Files**:
- `central/alert/datastore/datastore.go` - Interface definition
- `central/alert/datastore/datastore_impl.go` - Implementation
- `central/alert/datastore/internal/store/postgres/store.go` - Store interface

### Stage 2: Service Pattern
**Time**: 1-2 days
**Focus**: Learn gRPC API layer and authorization

**Study**:
1. Read Service Pattern section in PATTERNS.md
2. Examine `central/alert/service/service_impl.go`
3. Understand SAC authorization checks
4. Follow request flow from gRPC to DataStore

**Exercises**:
- Add a new gRPC endpoint
- Implement SAC checks for new operation
- Handle error translation (errox patterns)

**Key Files**:
- `central/alert/service/service_impl.go`
- `proto/api/v1/alert_service.proto`
- `pkg/grpc/authz/` - Authorization interceptors

### Stage 3: SAC (Scoped Access Control)
**Time**: 2-3 days
**Focus**: Master authorization and scope checking

**Study**:
1. Read SAC section in PATTERNS.md
2. Read `pkg/sac/README.md` (if available, otherwise examine code)
3. Understand scope hierarchy (Global → Access Mode → Resource → Cluster → Namespace)
4. Trace SAC filter application in Search queries

**Exercises**:
- Write SAC tests for DataStore
- Implement scope-based filtering
- Understand three-state logic (Excluded/Partial/Included)

**Key Files**:
- `pkg/sac/` - SAC implementation
- `pkg/sac/resources/` - Resource metadata
- `central/alert/datastore/datastore_impl.go` - SAC integration example

### Stage 4: Search Integration
**Time**: 2-3 days
**Focus**: Learn search query translation and field mapping

**Study**:
1. Read Search Integration section in PATTERNS.md
2. Examine `pkg/postgres/schema/alerts.go` - Schema registration
3. Understand search query syntax and PostgreSQL translation
4. Trace field mapping from protobuf to database columns

**Exercises**:
- Add searchable field to existing entity
- Write search query tests
- Understand cross-entity search (joins)

**Key Files**:
- `pkg/search/` - Search framework
- `pkg/search/postgres/` - PostgreSQL translation
- `pkg/postgres/schema/` - Schema with search field registration

### Stage 5: Code Generation
**Time**: 1-2 days
**Focus**: Understand proto → schema → store pipeline

**Study**:
1. Read Code Generation Pipeline section in PATTERNS.md
2. Read `pkg/postgres/README.md` - Schema generation details
3. Examine `tools/generate-helpers/pg-table-bindings/` - Generator code
4. Run code generation commands

**Exercises**:
- Add field to protobuf and regenerate
- Understand struct tag syntax (`sql:`, `search:`)
- Review generated schema files

**Commands**:
```bash
make proto-generated-srcs          # Proto → Go bindings
make generate-postgres-schemas     # Schema generation
make go-generated-srcs             # Mocks, stringer, etc.
```

**Key Files**:
- `proto/storage/deployment.proto` - Source definitions
- `generated/storage/deployment.pb.go` - Generated Go code
- `pkg/postgres/schema/deployments.go` - Generated schema
- `pkg/postgres/walker/` - Schema reflection engine

### Learning Resources

**Documentation**:
- `doc/PATTERNS.md` - This document
- `doc/ARCHITECTURE.md` - System architecture
- `doc/pkg/postgres/README.md` - PostgreSQL patterns
- `migrator/README.md` - Database migrations
- `AGENTS.md` - Development workflow

**Example Packages** (from simple to complex):
1. **Simple**: `central/cluster/datastore/` - Minimal business logic
2. **Medium**: `central/alert/datastore/` - Moderate complexity
3. **Complex**: `central/deployment/datastore/` - Risk scoring, baselines, multiple relationships

**Code Walkthroughs**:
1. Trace Alert creation: Service → DataStore → Store → PostgreSQL
2. Trace Search query: Parse → Field mapping → SAC filter → SQL translation
3. Trace Policy evaluation: Detector → DataStore → Alert generation

### Next Steps After Basics

**Advanced Topics** (after 1-2 weeks):
- Transaction patterns (nested transactions, context-based)
- Error handling patterns (errox, gRPC status mapping)
- Testing patterns (Testify suites, SAC tests, PostgreSQL integration)
- Event pipeline (Sensor → Central flow)
- Migrations (backward-compatible schema changes)

**Specialized Areas**:
- Scanner integration (vulnerability scanning)
- Compliance framework (standards, profiles, operators)
- Network graph (flows, baselines, policies)
- Risk scoring (deployment risk, indicators)
- UI integration (React, TypeScript, API consumption)

### Common Pitfalls for New Developers

1. **Editing generated files**: Always edit source (proto/templates), not generated code
2. **Forgetting SAC**: All DataStore Search/Get operations must apply SAC filters
3. **Business logic in wrong layer**: Keep Store simple, DataStore has business rules
4. **Not using transactions**: Multi-entity updates need transactions for consistency
5. **Ignoring error wrapping**: Use `errors.Wrap()` and errox sentinels
6. **Breaking migrations**: Schema changes must be backward-compatible

### Getting Help

- **Slack**: Ask in #development channel
- **Code Review**: Submit small PRs, ask for feedback
- **Pairing**: Pair with experienced developer on first few tasks
- **Documentation**: Keep this document updated with learnings

---

## Summary

These patterns appear consistently across the StackRox codebase:

1. **Datastore Pattern**: Standardized data access with business logic separation
2. **Service Pattern**: Thin gRPC facades with SAC authorization
3. **Singleton Pattern**: Thread-safe lazy initialization for shared resources
4. **Code Generation**: Proto → Schema → Store pipeline for consistency
5. **SAC**: Hierarchical permission checking with three-state logic
6. **Feature Flags**: Runtime configuration via typed environment variables
7. **Search Integration**: Unified search with field mapping and SQL translation
8. **Event Pipeline**: Standard flow for Kubernetes resource monitoring
9. **Error Handling**: errox sentinels, gRPC mapping, error wrapping conventions
10. **Transaction Patterns**: Nested transactions with inner/outer modes
11. **Testing Patterns**: Testify suites, SAC tests, PostgreSQL integration, mocks
12. **Layer Responsibilities**: Clear separation across Proto/Store/DataStore/Service/Detector

Individual package documentation should reference these patterns by name rather than re-explaining them. This ensures consistency and reduces maintenance burden.

For detailed examples, refer to:
- **Datastore**: `doc/central/alert/README.md`, `doc/central/deployment/README.md`
- **Code Generation**: `doc/pkg/postgres/README.md`
- **SAC**: `doc/pkg/sac/README.md`
- **Feature Flags**: `doc/pkg/env/README.md`
- **Testing**: `doc/tests/README.md`, `doc/qa-tests-backend/README.md`
- **Architecture**: `doc/ARCHITECTURE.md`
