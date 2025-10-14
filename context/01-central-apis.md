# Central: APIs & Data Flow

How Central serves API requests and manages data - explained for frontend engineers.

## Table of Contents
- [Overview](#overview)
- [Central's Role](#centrals-role)
- [API Architecture](#api-architecture)
- [Request Lifecycle](#request-lifecycle)
- [Data Storage](#data-storage)
- [Performance Considerations](#performance-considerations)
- [Real-Time vs Cached Data](#real-time-vs-cached-data)
- [Pagination Strategy](#pagination-strategy)
- [Common Scenarios](#common-scenarios)

## Overview

**Central** is the brain of StackRox. When your React component makes an API call, it goes to Central. Understanding how Central processes these requests helps you:
- Build UIs that match backend capabilities
- Understand why some queries are slow
- Debug API issues more effectively
- Have better conversations with backend engineers

## Central's Role

Central is a **Go-based API server** that:
1. **Receives API requests** from the UI (and roxctl, other clients)
2. **Authenticates and authorizes** the request
3. **Queries PostgreSQL** for data
4. **Processes and aggregates** data (if needed)
5. **Returns JSON** to the client

```
┌─────────┐     HTTPS/gRPC      ┌─────────┐     SQL     ┌────────────┐
│   UI    │ ─────────────────> │ Central │ ──────────> │ PostgreSQL │
│ (React) │ <───────────────── │  (Go)   │ <────────── │            │
└─────────┘     JSON            └─────────┘    Rows     └────────────┘
```

### What Central Does NOT Do

- **Does not store data itself** - Everything goes to PostgreSQL
- **Does not scan images** - Scanner does that
- **Does not monitor clusters** - Sensor does that

Central is the **aggregator and coordinator** - it receives data from Sensors and Scanner, stores it in PostgreSQL, and serves it via APIs.

## API Architecture

Central exposes **three types of APIs**:

### 1. REST APIs (v1, v2)

**Endpoint Pattern**: `/v1/{resource}` or `/v2/{resource}`

**Examples**:
- `GET /v1/alerts` - List alerts
- `GET /v1/policies/{id}` - Get a single policy
- `POST /v1/policies` - Create a policy
- `PATCH /v1/alerts/{id}/resolve` - Resolve an alert

**Backend Implementation**:
- Written in Go
- Located in `/central/*/service/service.go` files
- Uses gRPC internally, wrapped with HTTP/JSON layer
- Most services follow CRUD pattern

**Why REST for most operations?**
- Simple, well-understood
- Easy to cache
- Works well for standard CRUD operations

### 2. GraphQL API

**Endpoint**: `/api/graphql`

**Examples**:
```graphql
query {
  deployments(query: "Cluster:production") {
    id
    name
    images {
      name
      vulns {
        cve
        severity
      }
    }
  }
}
```

**Backend Implementation**:
- Located in `/central/graphql/resolvers/`
- Built on top of the same data layer as REST
- Resolvers query PostgreSQL

**Why GraphQL for complex queries?**
- Fetch nested/related data in one request
- Reduces over-fetching (client specifies exactly what it needs)
- Great for complex UIs like deployment details

**Trade-off**: GraphQL queries can be expensive if not careful. Each nested field is a potential database query.

### 3. gRPC APIs (Internal)

**Used internally** between:
- Sensor → Central
- Scanner → Central
- Central services ↔ Central services

**Not exposed to UI** - The UI uses REST/GraphQL, which wrap gRPC internally.

## Request Lifecycle

What happens when you call `GET /v1/alerts`?

### Step 1: Authentication

```
UI Request → Central
  ├─ Extract JWT token from header
  ├─ Validate token signature
  └─ Extract user identity
```

**Code location**: `/central/auth/`

If auth fails → **401 Unauthorized**

### Step 2: Authorization (RBAC)

```
Central checks:
  ├─ What role does this user have?
  ├─ What resources can they access? (Alert, Cluster, etc.)
  └─ What scope? (All clusters? Specific namespaces?)
```

**Code location**: `/central/role/`

If authz fails → **403 Forbidden**

**Key insight**: Authorization filters data at query time. If you have access to only `Cluster:production`, you'll only see alerts from that cluster - even if the API endpoint is the same.

### Step 3: Query PostgreSQL

```
Central service:
  ├─ Builds SQL query based on request parameters
  │  ├─ Filters (search query)
  │  ├─ Sorting
  │  ├─ Pagination (offset, limit)
  │  └─ RBAC scoping (WHERE cluster_id IN ...)
  ├─ Executes query
  └─ Fetches rows
```

**Code location**: `/central/*/datastore/` and `/central/*/store/postgres/`

### Step 4: Process & Transform

```
Central:
  ├─ Converts database rows → Go structs
  ├─ Applies business logic (e.g., calculate risk scores)
  ├─ Aggregates data (if needed)
  └─ Converts to protobuf/JSON
```

### Step 5: Return Response

```
Central → UI
  └─ JSON response with data
```

**Total time**: Typically 50ms - 500ms, but can be longer for complex queries.

## Data Storage

### PostgreSQL Schema

All StackRox data lives in **PostgreSQL**:

```
┌──────────────────────────────────────┐
│          PostgreSQL                  │
├──────────────────────────────────────┤
│ alerts                               │
│ ├─ id, policy_id, time, state        │
│ └─ deployment_id, cluster_id         │
│                                      │
│ policies                             │
│ ├─ id, name, severity, enabled       │
│                                      │
│ deployments                          │
│ ├─ id, name, namespace_id            │
│ └─ cluster_id                        │
│                                      │
│ images                               │
│ ├─ id, name, scan_time               │
│                                      │
│ vulnerabilities                      │
│ ├─ cve, severity, cvss               │
│                                      │
│ ... (50+ tables)                     │
└──────────────────────────────────────┘
```

**Key tables**:
- `alerts` - Violations/alerts
- `policies` - Security policies
- `deployments` - Kubernetes deployments
- `images` - Container images
- `image_cves` - Image vulnerabilities
- `clusters` - Kubernetes clusters
- `namespaces` - Kubernetes namespaces

**Relationships**:
- Alerts reference Policies and Deployments
- Deployments reference Clusters and Images
- Images reference CVEs (vulnerabilities)

### How Data Gets There

```
1. Sensor watches Kubernetes cluster
   ↓
2. Sensor sends deployment data to Central (gRPC)
   ↓
3. Central stores in PostgreSQL
   ↓
4. Scanner scans images, sends CVE data to Central
   ↓
5. Central stores CVEs in PostgreSQL
   ↓
6. Policy engine evaluates policies → generates alerts
   ↓
7. Central stores alerts in PostgreSQL
```

This is why there's a delay between deploying something in Kubernetes and seeing it in the UI - the data has to flow through this pipeline.

## Performance Considerations

### Why Some Queries Are Slow

1. **Large result sets**
   - Fetching 10,000 deployments takes time
   - Solution: Use pagination

2. **Complex joins**
   ```sql
   -- This query joins 5 tables:
   SELECT alerts.*, policies.name, deployments.name,
          clusters.name, images.name
   FROM alerts
   JOIN policies ON alerts.policy_id = policies.id
   JOIN deployments ON alerts.deployment_id = deployments.id
   JOIN clusters ON deployments.cluster_id = clusters.id
   JOIN images ON deployments.image_id = images.id
   WHERE alerts.state = 'ACTIVE'
   ```
   - Each join adds cost
   - Solution: Indexed columns, optimized queries

3. **Aggregations**
   ```sql
   -- Counting alerts by severity across all clusters
   SELECT severity, COUNT(*)
   FROM alerts
   GROUP BY severity
   ```
   - Has to scan all rows
   - Solution: Pre-computed counts (coming in future versions)

4. **RBAC filtering**
   ```sql
   -- User with namespace-scoped access
   SELECT * FROM alerts
   WHERE deployment_id IN (
     SELECT id FROM deployments
     WHERE namespace_id IN (user's allowed namespaces)
   )
   ```
   - Scoped users require additional filtering
   - Solution: Indexed scoping columns

### What Makes Queries Fast

✅ **Fetching by ID**: `GET /v1/policies/{id}` - Single row lookup, very fast

✅ **Simple filters**: `GET /v1/alerts?query=Cluster:production` - Indexed column, fast

✅ **Pagination**: `GET /v1/alerts?pagination.limit=20` - Only fetches 20 rows

✅ **Cached data**: Dashboard widgets cache results for 30 seconds

## Real-Time vs Cached Data

| Data Type | Freshness | Example |
|-----------|-----------|---------|
| **Real-time** | Instant | Alert counts on dashboard (polls every 30s) |
| **Near real-time** | 1-5 seconds | Deployment status from Sensor |
| **Delayed** | Minutes to hours | Vulnerability scan results |
| **Static** | Changed only on user action | Policies, roles, integrations |

### Why Vulnerability Data Is Delayed

```
1. Image deployed in cluster
   ↓ (seconds)
2. Sensor reports to Central
   ↓ (seconds)
3. Central stores deployment
   ↓ (minutes - scanner has to pull image)
4. Scanner scans image for CVEs
   ↓ (seconds)
5. Scanner reports CVEs to Central
   ↓ (seconds)
6. Central stores CVEs
   ↓ NOW visible in UI
```

**Total time**: 2-10 minutes for first scan

**Subsequent scans**: Faster (image layers cached)

### Why Alert Counts Change

Alerts are **generated in real-time** when:
- A new deployment violates a policy
- A runtime violation occurs (e.g., unauthorized process)
- A policy changes and is re-evaluated

The UI polls `/v1/alerts/summary/counts` every 30 seconds to show fresh counts.

## Pagination Strategy

### Backend Implementation

Central uses **offset-based pagination**:

```
GET /v1/alerts?pagination.offset=0&pagination.limit=20
```

**How it works in PostgreSQL**:
```sql
SELECT * FROM alerts
ORDER BY time DESC
LIMIT 20 OFFSET 0  -- First page (results 0-19)
```

Next page:
```sql
LIMIT 20 OFFSET 20  -- Second page (results 20-39)
```

### Why Offset Pagination?

✅ **Simple**: Easy to implement jump to page 5, 10, etc.

❌ **Slow for large offsets**: `OFFSET 10000` has to scan 10,000 rows to skip them

❌ **Inconsistent during inserts**: If new alert arrives while you're on page 2, page 3 might have duplicates

### Performance Tips

- **Keep limit reasonable**: 20-100 rows per page
- **Avoid large offsets**: Don't jump to page 100
- **Use sorting**: Always specify `sortOption` for consistent results

## Common Scenarios

### Scenario 1: "Why don't I see all alerts?"

**Likely cause**: RBAC scoping

```
User has access to:
  ├─ Cluster: production
  └─ Namespaces: default, backend

Central filters query:
  SELECT * FROM alerts
  WHERE cluster_id IN ('production-cluster-id')
    AND namespace_id IN ('default-id', 'backend-id')
```

You only see alerts for resources you have access to.

### Scenario 2: "Why is the dashboard slow?"

**Likely causes**:
1. **Too many clusters** - Aggregating data across 50 clusters takes time
2. **Too many alerts** - Counting 100,000 alerts is expensive
3. **Complex widgets** - Each widget is a separate query

**Solutions**:
- Dashboard queries are cached for 30 seconds
- Consider narrowing scope (filter to specific clusters)
- Backend team can optimize slow queries

### Scenario 3: "Why did my API call return 403?"

**Cause**: Failed authorization

```
User requests: GET /v1/policies
Central checks: Does user have READ_ACCESS to 'WorkflowAdministration'?
Result: NO → 403 Forbidden
```

Always check RBAC permissions before making API calls.

### Scenario 4: "Why is image scanning slow?"

**Not an API issue** - Scanning is a separate process:

```
Scanner pipeline:
  1. Pull image layers (slow - network download)
  2. Extract files from layers (slow - decompression)
  3. Scan files for CVEs (fast - indexed database)
  4. Report to Central (fast)
```

First scan: 2-10 minutes
Cached scans: 30 seconds - 2 minutes

## Understanding API Response Times

| Endpoint | Typical Time | Why |
|----------|--------------|-----|
| `GET /v1/policies/{id}` | 10-50ms | Single row lookup |
| `GET /v1/policies` | 50-200ms | List query with filtering |
| `GET /v1/alerts?pagination.limit=20` | 100-300ms | Paginated query with joins |
| `GET /v1/alerts/summary/counts` | 200-500ms | Aggregation across all alerts |
| `POST /v1/policies` | 100-300ms | Insert + validation |
| GraphQL nested query | 200-1000ms | Multiple queries + data assembly |

**Slow query threshold**: > 1 second (report to backend team)

## Key Takeaways for Frontend Engineers

1. **Central is stateless** - All state lives in PostgreSQL
2. **Every API call goes through auth + authz** - RBAC filtering happens at query time
3. **Some data is delayed** - Vulnerability scans take minutes, not seconds
4. **Pagination is offset-based** - Don't jump to large page numbers
5. **RBAC filters query results** - Users see only what they have access to
6. **Complex queries are expensive** - GraphQL can generate many DB queries
7. **Dashboard data is cached** - Counts update every 30 seconds

## Related Documentation

- [Architecture Overview](./architecture-overview.md) - How Central fits in the system
- [RBAC Backend](./02-rbac-backend.md) - How permissions are enforced
- Backend code: `/central/` - Central service implementation

---

**Last Updated**: 2025-10-09
**For Frontend Engineers** - You don't need to write Go, but understanding this helps you build better UIs!
