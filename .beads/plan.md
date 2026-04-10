# ACS Soft-Delete for Deployments Implementation Plan

## Goal: Implement soft-delete for deployments to improve auditability of ephemeral workloads

### Type
epic

### Priority
1

### Description
Implement a soft-delete mechanism for deployments that creates tombstone markers instead of immediately purging data. This enables ServiceNow integration to capture vulnerabilities in short-lived deployments and provides auditability within ACS itself. Deployments marked as soft-deleted retain vulnerability data for a configurable TTL (~24h) before permanent removal.

### References
- JIRA: ROX-33816
- Design Doc: ACS Soft-Delete for Deployments: High-Level Feature Approach (2026-04-08)

---

## Feature: Proto Definitions and Schema

### Type
feature

### Priority
1

### Description
Define the protobuf schema for tombstone tracking and deployment lifecycle state. Add necessary database indices for efficient querying and pruning.

### Parent
Goal: Implement soft-delete for deployments to improve auditability of ephemeral workloads

---

## Task: Add Tombstone message to proto definition

### Type
task

### Priority
1

### Description
Add the Tombstone message to `proto/storage/deployment.proto`:

```protobuf
message Tombstone {
  // deleted_at is the time the resource was soft-deleted.
  google.protobuf.Timestamp deleted_at = 1; // @gotags: search:"Tombstone Deleted At,hidden"
  
  // expires_at is the time after which the resource will be permanently purged.
  // This value is pre-computed as deleted_at plus the configured TTL.
  google.protobuf.Timestamp expires_at = 2; // @gotags: search:"Tombstone Expires At,hidden"
}
```

Add the tombstone field to the Deployment message:
```protobuf
message Deployment {
  ...
  Tombstone tombstone = 36;
}
```

Run `make proto-generated-srcs` to regenerate Go code.

### Parent
Feature: Proto Definitions and Schema

---

## Task: Add LifecycleStage enum and deprecate inactive field

### Type
task

### Priority
1

### Description
Based on design review feedback, add a LifecycleStage enum to `proto/storage/deployment.proto` for better extensibility:

```protobuf
enum LifecycleStage {
  ACTIVE = 0;
  DELETED = 1;
  // Future: SCALED_DOWN, SUSPENDED, etc.
}
```

Add the lifecycle_stage field to Deployment message:
```protobuf
message Deployment {
  ...
  LifecycleStage lifecycle_stage = 37;
  bool inactive = XX [deprecated = true]; // Document deprecation
}
```

This provides a cleaner API surface for the UI to query deployment status directly without constructing it from tombstone timestamps.

Run `make proto-generated-srcs` to regenerate.

### Parent
Feature: Proto Definitions and Schema

---

## Task: Add database index on tombstone expires_at field

### Type
task

### Priority
1

### Description
Following design review feedback (David Shrewsberry), add an index on `expires_at` to make pruning efficient.

Read `migrator/README.md` for the migration workflow. Create a new migration in `migrator/migrations/` that adds an index on `deployments.tombstone.expires_at`.

The pruner will query for expired deployments using `WHERE expires_at < NOW()`, so this index is critical for performance.

Reference similar migrations in the migrations directory for examples.

After creating the migration, run code generation: `make proto-generated-srcs go-generated-srcs` (can run in background).

### Parent
Feature: Proto Definitions and Schema

---

## Task: Add configuration for tombstone TTL

### Type
task

### Priority
2

### Description
Add a configuration field for tombstone TTL (time-to-live) duration. Default should be 24 hours based on design doc.

Identify the appropriate config proto (likely in `proto/api/v1/` or Central config) and add:
```protobuf
// Tombstone retention duration for soft-deleted deployments.
// Default: 24h. After this duration, deployments are permanently purged.
google.protobuf.Duration tombstone_ttl = <field_number>;
```

Document the field and update any config documentation/defaults.

### Parent
Feature: Proto Definitions and Schema

---

## Feature: Core Tombstone Management

### Type
feature

### Priority
1

### Description
Implement core soft-delete logic: mark deployments as deleted with tombstone markers instead of immediate removal, compute expiration times, and add pruning for expired tombstones.

### Parent
Goal: Implement soft-delete for deployments to improve auditability of ephemeral workloads

---

## Task: Update deployment deletion handler to create tombstones

### Type
task

### Priority
1

### Description
Locate the deployment deletion handler in Central (likely in `central/deployment/` or similar). When a deployment delete event is received from Sensor:

1. Instead of calling `Delete()`, call `MarkAsDeleted()` (new method)
2. Set `tombstone.deleted_at = now()`
3. Set `tombstone.expires_at = deleted_at + configured_ttl`
4. Set `lifecycle_stage = DELETED`
5. Update the deployment in the database

Verify that process indicator queue still removes the deployment reference (design review comment #7).

Write unit tests covering:
- Tombstone timestamps are set correctly
- expires_at = deleted_at + TTL
- lifecycle_stage transitions to DELETED

### Parent
Feature: Core Tombstone Management

---

## Task: Implement tombstone pruner/garbage collector

### Type
task

### Priority
1

### Description
Create a background goroutine in Central that periodically queries for expired deployments and permanently deletes them.

File: `central/deployment/pruner/tombstone_pruner.go`

Logic:
1. Query deployments WHERE `expires_at < NOW()` (uses the new index)
2. For each expired deployment, call permanent delete
3. Run every 1 hour (configurable)

Handle shutdown gracefully and add metrics for:
- Number of deployments pruned
- Last prune time
- Errors during pruning

Write tests:
- Pruner deletes deployments past expires_at
- Pruner does NOT delete deployments before expires_at
- Pruner handles empty results

### Parent
Feature: Core Tombstone Management

---

## Task: Update deployment datastore interface for tombstone queries

### Type
task

### Priority
2

### Description
Update the deployment datastore interface (likely `central/deployment/datastore/datastore.go`) to support querying by lifecycle stage.

Add methods:
```go
// GetActiveDeploy ments returns deployments with lifecycle_stage = ACTIVE
GetActiveDeployments(ctx context.Context) ([]*storage.Deployment, error)

// GetSoftDeletedDeployments returns deployments with lifecycle_stage = DELETED
GetSoftDeletedDeployments(ctx context.Context) ([]*storage.Deployment, error)

// GetExpiredDeployments returns deployments where expires_at < now
GetExpiredDeployments(ctx context.Context) ([]*storage.Deployment, error)
```

Implement these methods in the Postgres and RocksDB stores.

Write integration tests (tag with `//go:build sql_integration`).

### Parent
Feature: Core Tombstone Management

---

## Feature: ServiceNow Integration

### Type
feature

### Priority
1

### Description
Enable ServiceNow data ingestion to query both active and soft-deleted deployments via export APIs and GraphQL. The tombstone status indicates deployment state.

### Parent
Goal: Implement soft-delete for deployments to improve auditability of ephemeral workloads

---

## Task: Update export APIs to include soft-deleted deployments

### Type
task

### Priority
1

### Description
Locate the export APIs used by ServiceNow integration (likely REST endpoints in `central/` returning deployment data).

Add a query parameter `include_deleted=true` (default false for backward compatibility).

When `include_deleted=true`:
- Include deployments with lifecycle_stage = DELETED
- Return the tombstone field in the response

Update API documentation and OpenAPI spec if applicable.

Write tests:
- Default behavior excludes soft-deleted
- include_deleted=true returns both active and deleted
- Tombstone fields are correctly serialized

### Parent
Feature: ServiceNow Integration

---

## Task: Update GraphQL schema to expose tombstone and lifecycle_stage

### Type
task

### Priority
1

### Description
Update the GraphQL schema (likely in `central/graphql/`) to expose:
- `tombstone` field with `deletedAt` and `expiresAt`
- `lifecycleStage` field (enum: ACTIVE, DELETED)

Add a filter to deployment queries:
```graphql
deployments(filter: {lifecycleStage: [ACTIVE, DELETED]}): [Deployment!]!
```

Default behavior should filter to ACTIVE only for backward compatibility.

Write GraphQL query tests covering:
- Filter by lifecycle stage
- Tombstone fields are returned for deleted deployments
- Null tombstone for active deployments

### Parent
Feature: ServiceNow Integration

---

## Feature: Vulnerability Management UI

### Type
feature

### Priority
1

### Description
Add filter for "Deleted deployments" to vulnerability management dashboard. Default shows only active deployments. Users can toggle to include soft-deleted deployments.

### Parent
Goal: Implement soft-delete for deployments to improve auditability of ephemeral workloads

---

## Task: Add lifecycle stage filter to VM dashboard UI

### Type
task

### Priority
1

### Description
Update the vulnerability management dashboard UI (React component in `ui/apps/platform/src/Containers/Vulnerabilities/` or similar).

Add a filter dropdown/toggle with options:
- "Active deployments only" (default)
- "Include deleted deployments"
- "Deleted deployments only"

Reference the prototype screenshot from stackrox/pull/19621 for the desired UX.

The filter should map to the deployment status filter:
- Active only: lifecycle_stage = ACTIVE
- Include deleted: lifecycle_stage IN (ACTIVE, DELETED)
- Deleted only: lifecycle_stage = DELETED

Add visual indicator (badge/pill) showing deployment status (Active/Deleted) in the results table.

### Parent
Feature: Vulnerability Management UI

---

## Task: Update VM API queries to respect lifecycle stage filter

### Type
task

### Priority
1

### Description
Update the backend API endpoints that serve the VM dashboard (likely in `central/vulnerabilities/` or `central/deployment/`) to:

1. Accept lifecycle_stage filter parameter
2. Default to ACTIVE if not specified (backward compatibility)
3. Query deployments with the appropriate lifecycle_stage filter

Update the corresponding GraphQL resolvers or REST handlers.

Write tests:
- Default returns only active deployments
- Filter correctly includes/excludes deleted deployments
- Vulnerability data for deleted deployments is retained

### Parent
Feature: Vulnerability Management UI

---

## Task: Add E2E test for VM deleted deployments filter

### Type
task

### Priority
2

### Description
Write an end-to-end test in `ui/apps/platform/src/test-utils/` or appropriate location:

1. Create a deployment
2. Scan for vulnerabilities
3. Delete the deployment (should create tombstone)
4. Navigate to VM dashboard
5. Verify default view excludes the deleted deployment
6. Toggle "Include deleted deployments" filter
7. Verify the deployment and its vulnerabilities appear
8. Verify the deployment shows "Deleted" status badge

### Parent
Feature: Vulnerability Management UI

---

## Feature: Alert and Policy Integration

### Type
feature

### Priority
1

### Description
Update policy evaluation to exclude soft-deleted deployments. When a deployment is soft-deleted, resolve all open alerts associated with it.

### Parent
Goal: Implement soft-delete for deployments to improve auditability of ephemeral workloads

---

## Task: Exclude soft-deleted deployments from policy evaluation

### Type
task

### Priority
1

### Description
Locate the policy evaluation engine (likely in `central/detection/` or `central/alerts/`).

Update policy evaluation queries to filter deployments WHERE lifecycle_stage = ACTIVE.

This affects:
- Scheduled policy re-evaluation ("Reassess all")
- Real-time policy evaluation on deployment updates
- Compliance reporting

Design review comment from Khushboo: ensure all policy evaluation paths are covered.

Write tests:
- Reassess all skips soft-deleted deployments
- Policy violations are not created for soft-deleted deployments
- Existing violations are not re-evaluated for soft-deleted deployments

### Parent
Feature: Alert and Policy Integration

---

## Task: Resolve alerts when deployment is soft-deleted

### Type
task

### Priority
1

### Description
Update the deployment deletion handler to resolve all open alerts when a deployment transitions to DELETED lifecycle stage.

Locate alert resolution logic (likely in `central/alerts/` or co-located with deployment deletion).

When marking deployment as deleted:
1. Query all open alerts WHERE deployment_id = <id>
2. Transition alerts to RESOLVED state
3. Add resolution reason: "Deployment deleted"

This matches current behavior for hard deletes (design review comment #9).

Write tests:
- Open alerts are resolved when deployment deleted
- Alert retention policy is unchanged (design review comment #11)
- Already resolved alerts are not affected

### Parent
Feature: Alert and Policy Integration

---

## Task: Update alert datastore queries to filter active deployments

### Type
task

### Priority
2

### Description
Review alert listing/counting queries in `central/alerts/datastore/` to ensure they default to filtering by active deployments only.

Add a query parameter to optionally include deleted deployments for audit/historical views.

Write tests:
- Default alert queries exclude deployments with lifecycle_stage = DELETED
- Historical queries can include deleted deployments via parameter

### Parent
Feature: Alert and Policy Integration

---

## Feature: Risk Integration

### Type
feature

### Priority
1

### Description
Exclude soft-deleted deployments from risk score calculation and risk UI displays. No risk should be shown or calculated for soft-deleted deployments.

### Parent
Goal: Implement soft-delete for deployments to improve auditability of ephemeral workloads

---

## Task: Exclude soft-deleted deployments from risk calculation

### Type
task

### Priority
1

### Description
Locate the risk score calculation logic (likely in `central/risk/` or similar).

Update risk calculation queries to filter WHERE lifecycle_stage = ACTIVE.

Design review comment #12 (David Shrewsberry): verify the calculation excludes soft-deleted deployments.

Write tests:
- Risk scores are not calculated for soft-deleted deployments
- Deleting a deployment removes it from risk calculations
- Risk scores are recalculated when deployment transitions to DELETED

### Parent
Feature: Risk Integration

---

## Task: Update risk UI to hide soft-deleted deployments

### Type
task

### Priority
2

### Description
Update risk dashboard UI components (React components in `ui/apps/platform/src/Containers/Risk/` or similar) to:

1. Filter out soft-deleted deployments from risk views
2. Update risk queries to include lifecycle_stage = ACTIVE filter

No user-visible option to include deleted deployments in risk views (per design decision).

Write UI tests verifying soft-deleted deployments do not appear in risk dashboards.

### Parent
Feature: Risk Integration

---

## Feature: Documentation and Migration Guide

### Type
feature

### Priority
2

### Description
Document the soft-delete feature, API changes, and migration considerations for users and operators.

### Parent
Goal: Implement soft-delete for deployments to improve auditability of ephemeral workloads

---

## Task: Write user documentation for soft-delete feature

### Type
task

### Priority
2

### Description
Create or update documentation in `docs/` explaining:

1. What is soft-delete and why it exists
2. Default TTL and how to configure it
3. How to view soft-deleted deployments in the UI
4. Impact on ServiceNow integration
5. How to query soft-deleted deployments via API/GraphQL

Follow the existing docs structure and style.

### Parent
Feature: Documentation and Migration Guide

---

## Task: Write operator guide for tombstone configuration

### Type
task

### Priority
3

### Description
Document for cluster operators:

1. How to configure tombstone TTL
2. Performance implications (database size increase during TTL window)
3. How to monitor pruning metrics
4. Troubleshooting failed prunes

Add to operator/admin documentation.

### Parent
Feature: Documentation and Migration Guide

---

## Task: Create upgrade migration notes

### Type
task

### Priority
2

### Description
Document migration behavior when upgrading to the soft-delete version:

1. Existing deployments will have lifecycle_stage = ACTIVE (default)
2. No retroactive tombstones created
3. Database schema changes (new index on expires_at)
4. API backward compatibility (default behavior unchanged)

Add to release notes and upgrade guide.

### Parent
Feature: Documentation and Migration Guide

---

## Feature: Testing and Quality Assurance

### Type
feature

### Priority
1

### Description
Comprehensive testing coverage for soft-delete functionality across all integration points.

### Parent
Goal: Implement soft-delete for deployments to improve auditability of ephemeral workloads

---

## Task: Write integration tests for tombstone lifecycle

### Type
task

### Priority
1

### Description
Write integration tests in `qa-tests-backend/` (Groovy/Spock) covering the full lifecycle:

1. Deploy a workload
2. Verify deployment is ACTIVE
3. Delete the deployment via k8s API
4. Verify tombstone is created with correct timestamps
5. Verify lifecycle_stage = DELETED
6. Verify vulnerability data is retained
7. Wait for TTL expiration (or manipulate time)
8. Verify deployment is permanently deleted
9. Verify vulnerability data is purged

Tag with appropriate test categories.

### Parent
Feature: Testing and Quality Assurance

---

## Task: Write tests for backward compatibility

### Type
task

### Priority
1

### Description
Verify that default API behavior is unchanged:

1. Existing API clients (without lifecycle filter) only see active deployments
2. GraphQL queries without filter default to ACTIVE
3. Export APIs default to exclude deleted deployments
4. UI dashboards default to active deployments only

Write tests covering:
- REST API backward compatibility
- GraphQL backward compatibility
- UI default views

### Parent
Feature: Testing and Quality Assurance

---

## Task: Performance test with high deployment churn

### Type
task

### Priority
2

### Description
Test performance impact of soft-delete with high deployment turnover:

1. Create a test cluster with frequent deployment creation/deletion (simulate ephemeral workloads)
2. Monitor database size growth during TTL window
3. Verify pruner keeps up with expiration
4. Measure query performance with and without the expires_at index
5. Validate that the 24h TTL window does not cause unacceptable database bloat

Document findings and adjust TTL or pruner frequency if needed.

### Parent
Feature: Testing and Quality Assurance

---

## Task: Test process indicator queue interaction

### Type
task

### Priority
2

### Description
Design review comment #7 (David Shrewsberry): Verify the process indicator queue still removes the deployment when it is deleted.

Locate the process indicator queue logic and write tests:
1. Create deployment with running processes
2. Delete deployment (soft-delete)
3. Verify process indicators are removed from the queue
4. Verify no orphaned process indicators remain

This ensures soft-delete behaves identically to hard-delete from the process indicator perspective.

### Parent
Feature: Testing and Quality Assurance
