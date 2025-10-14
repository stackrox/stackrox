# RBAC: How Permissions Work in Central

How StackRox enforces role-based access control on the backend - explained for frontend engineers.

## Table of Contents
- [Overview](#overview)
- [Permission Evaluation Flow](#permission-evaluation-flow)
- [Resource Hierarchy](#resource-hierarchy)
- [Scoped vs Global Access](#scoped-vs-global-access)
- [How Permissions Filter Data](#how-permissions-filter-data)
- [Why API Calls Return 403](#why-api-calls-return-403)
- [Common Scenarios](#common-scenarios)
- [Performance Impact](#performance-impact)

## Overview

When you call `usePermissions()` in the UI, you're checking permissions that were **evaluated and enforced by Central**. Understanding how Central handles RBAC helps you:
- Understand why users see different data
- Debug permission issues faster
- Build UIs that match backend capabilities
- Have better conversations with backend engineers

**Key Insight**: Permissions aren't just UI controls - they actively **filter database queries** on the backend.

## Permission Evaluation Flow

What happens when a user makes an API request?

```
User makes request: GET /v1/alerts
  ↓
1. Central extracts JWT token
  ↓
2. Central looks up user's roles
  ↓
3. Central aggregates permissions from all roles
  ↓
4. Central checks: Does user have READ_ACCESS to 'Alert'?
   ├─ YES → Continue to step 5
   └─ NO  → Return 403 Forbidden
  ↓
5. Central applies scope filtering to SQL query
  ↓
6. Query returns only alerts user can access
```

### Step 1-2: Who Is This User?

```
JWT Token contains:
  ├─ User ID: "user@example.com"
  └─ Roles: ["Security-Analyst", "Platform-Viewer"]
```

**Code location**: `/central/auth/`

### Step 3: What Can This User Do?

Central aggregates permissions from **all assigned roles**:

```
Role: Security-Analyst
  ├─ Alert: READ_WRITE_ACCESS
  ├─ WorkflowAdministration: READ_ACCESS
  └─ Scope: Cluster=production, Namespace=*

Role: Platform-Viewer
  ├─ Cluster: READ_ACCESS
  ├─ Node: READ_ACCESS
  └─ Scope: (global - all clusters)

Combined Permissions:
  ├─ Alert: READ_WRITE_ACCESS (highest wins)
  ├─ Cluster: READ_ACCESS
  ├─ WorkflowAdministration: READ_ACCESS
  └─ Scope: Cluster=production OR global
```

**Rule**: If ANY role grants access, user has access (least restrictive wins).

**Code location**: `/central/role/`

### Step 4: Resource Check

```go
// Pseudocode of what Central does
func CheckPermission(user User, resource string, accessLevel AccessLevel) bool {
    permissions := GetUserPermissions(user)

    userAccess := permissions[resource]

    if accessLevel == READ_ACCESS {
        return userAccess == READ_ACCESS || userAccess == READ_WRITE_ACCESS
    }

    if accessLevel == READ_WRITE_ACCESS {
        return userAccess == READ_WRITE_ACCESS
    }

    return false
}
```

If check fails → **403 Forbidden** (API call rejected)

### Step 5: Scope Filtering

Even if user has access to `Alert`, **which alerts** can they see?

```sql
-- Without scoping (Admin user):
SELECT * FROM alerts WHERE state = 'ACTIVE'

-- With scoping (Scoped user):
SELECT * FROM alerts
WHERE state = 'ACTIVE'
  AND deployment_id IN (
    SELECT id FROM deployments
    WHERE cluster_id IN ('production-cluster-id')
      AND namespace_id IN (user's allowed namespaces)
  )
```

**This is why two users with the same "Alert: READ_ACCESS" see different data.**

## Resource Hierarchy

StackRox resources form a hierarchy:

```
Cluster
  └─ Namespace
       └─ Deployment
            ├─ Image
            │    └─ CVE (Vulnerability)
            └─ Secret

Alert (references Deployment or Image)
Policy (can be scoped to Cluster/Namespace)
```

### Global Resources

These apply across the entire system:

| Resource | What It Controls |
|----------|------------------|
| `Access` | User and role management |
| `Administration` | System configuration |
| `Integration` | External integrations (Slack, Jira, etc.) |
| `WorkflowAdministration` | Policies and policy categories |

**Global resources cannot be scoped** - you either have access to all or none.

### Scopable Resources

These can be restricted to specific clusters/namespaces:

| Resource | Scopable To |
|----------|-------------|
| `Alert` | Cluster, Namespace |
| `Deployment` | Cluster, Namespace |
| `Image` | Cluster, Namespace (via deployment) |
| `Compliance` | Cluster |
| `NetworkGraph` | Cluster, Namespace |

## Scoped vs Global Access

### Global Access (Admin Role)

```
Role: Admin
  ├─ All resources: READ_WRITE_ACCESS
  └─ Scope: <empty> (global)

SQL query for alerts:
  SELECT * FROM alerts  -- No filtering!
```

**Sees**: All alerts across all clusters

### Scoped Access (Development Team Role)

```
Role: Dev-Team-Backend
  ├─ Alert: READ_ACCESS
  ├─ Deployment: READ_ACCESS
  └─ Scope:
       ├─ Cluster: production
       └─ Namespaces: backend, backend-staging

SQL query for alerts:
  SELECT * FROM alerts
  WHERE deployment_id IN (
    SELECT id FROM deployments
    WHERE cluster_id = 'production-cluster-id'
      AND namespace_id IN ('backend-ns-id', 'backend-staging-ns-id')
  )
```

**Sees**: Only alerts in `production/backend` and `production/backend-staging`

### Why Scoping Exists

**Use case**: Multi-tenant environments

```
Company has 3 teams:
  ├─ Frontend Team
  │   └─ Access: production/frontend, staging/frontend
  ├─ Backend Team
  │   └─ Access: production/backend, staging/backend
  └─ Platform Team (SRE)
      └─ Access: All clusters, all namespaces

Each team sees only their own violations and deployments.
```

This prevents teams from seeing each other's security issues.

## How Permissions Filter Data

### Example 1: Listing Alerts

**User**: Frontend Team (scoped to `production/frontend`)

**API call**: `GET /v1/alerts`

**What Central does**:

```sql
-- 1. Check permission
User has READ_ACCESS to 'Alert'? → YES

-- 2. Build scoped query
SELECT alerts.*, deployments.name, policies.name
FROM alerts
JOIN deployments ON alerts.deployment_id = deployments.id
JOIN policies ON alerts.policy_id = policies.id
WHERE deployments.cluster_id = 'production-cluster-id'
  AND deployments.namespace_id IN ('frontend-ns-id')
ORDER BY alerts.time DESC
LIMIT 20
```

**Result**: User sees 15 alerts (only from their namespace)

**Global admin would see**: 1,247 alerts (all clusters, all namespaces)

### Example 2: Getting a Specific Alert

**User**: Frontend Team (scoped)

**API call**: `GET /v1/alerts/backend-alert-id`

**What Central does**:

```sql
-- 1. Fetch the alert
SELECT * FROM alerts WHERE id = 'backend-alert-id'

-- 2. Check scope
Alert's deployment is in namespace: backend
User has access to namespace: frontend

-- 3. Scope check fails
```

**Result**: **403 Forbidden** - Even though the alert exists, user cannot access it.

**From UI perspective**: User never sees this alert in lists, and direct access is denied.

### Example 3: Cross-Resource Requirements

**API call**: `GET /v1/deployments/{id}/images`

**What Central checks**:

```
1. Does user have READ_ACCESS to 'Deployment'?
2. Does user have READ_ACCESS to 'Image'?
3. Is deployment within user's scope?
```

If ANY check fails → **403 Forbidden**

**Why this matters for UI**:
- Button to view images should check BOTH `Deployment` and `Image` permissions
- This is why `/ui/CLAUDE.md` emphasizes cross-resource permission checks

## Why API Calls Return 403

### Scenario 1: Missing Resource Permission

```
User role:
  ├─ Alert: READ_ACCESS
  └─ No access to 'WorkflowAdministration'

API call: GET /v1/policies
Result: 403 Forbidden
```

**Fix**: User needs a role with `WorkflowAdministration: READ_ACCESS`

### Scenario 2: Out-of-Scope Access

```
User role:
  ├─ Alert: READ_ACCESS
  └─ Scope: Cluster=production, Namespace=frontend

API call: GET /v1/alerts/staging-alert-id
Result: 403 Forbidden
```

**Fix**: User needs scope expanded to include `staging` cluster

### Scenario 3: Insufficient Access Level

```
User role:
  └─ WorkflowAdministration: READ_ACCESS

API call: POST /v1/policies (create new policy)
Result: 403 Forbidden
```

**Fix**: User needs `WorkflowAdministration: READ_WRITE_ACCESS`

### Scenario 4: Deleted or Moved Resource

```
User role:
  ├─ Deployment: READ_ACCESS
  └─ Scope: Cluster=production, Namespace=backend

API call: GET /v1/deployments/xyz
Result: 403 Forbidden

Why? Deployment was moved to 'frontend' namespace yesterday.
```

**Fix**: User's scope needs to include new namespace

## Common Scenarios

### Scenario 1: "Why does Admin see more alerts than me?"

**You see**: 15 alerts

**Admin sees**: 1,247 alerts

**Reason**: You have scoped access, admin has global access.

```
Your query:
  SELECT * FROM alerts WHERE namespace_id IN (your namespaces)

Admin query:
  SELECT * FROM alerts  -- No scope filtering
```

### Scenario 2: "Why can't I resolve this alert?"

**You have**: `Alert: READ_ACCESS`

**You need**: `Alert: READ_WRITE_ACCESS`

```
GET /v1/alerts/{id}       → 200 OK (read works)
PATCH /v1/alerts/{id}/resolve → 403 Forbidden (write denied)
```

**Central checks**: Does user have `READ_WRITE_ACCESS` to `Alert`?

### Scenario 3: "I can see the deployment, but not its compliance data"

**You have**:
- `Deployment: READ_ACCESS`

**You're missing**:
- `Compliance: READ_ACCESS`

Even though the deployment is in your scope, compliance data requires additional permission.

### Scenario 4: "Why do counts differ between users?"

**Dashboard shows**:
- You: "3 Critical Violations"
- Teammate: "8 Critical Violations"

**Reason**: Different scopes

```
Your scope: production/frontend
Teammate scope: production/backend, production/frontend
```

You both see production cluster, but teammate sees more namespaces.

### Scenario 5: "Why can't I create a report?"

**API call**: `POST /v1/reports/configurations`

**Requirements**:
- `WorkflowAdministration: READ_WRITE_ACCESS` (create report config)
- `Image: READ_ACCESS` (report needs to read image data)

Missing either → **403 Forbidden**

## Performance Impact

### Global Access = Faster Queries

```sql
-- Admin user (no scope filtering)
SELECT * FROM alerts
ORDER BY time DESC
LIMIT 20

-- Execution time: ~50ms
```

Simple query, well-indexed, fast.

### Scoped Access = Slower Queries

```sql
-- Scoped user
SELECT * FROM alerts
WHERE deployment_id IN (
  SELECT id FROM deployments
  WHERE cluster_id IN ('prod-id', 'staging-id')
    AND namespace_id IN (... 15 namespace IDs ...)
)
ORDER BY time DESC
LIMIT 20

-- Execution time: ~200ms
```

Additional subquery and filtering adds overhead.

### Why This Matters

1. **Dashboard performance** - Scoped users may see slower dashboards
2. **Large result sets** - More namespaces = slower queries
3. **Nested resources** - Image vulnerabilities for scoped deployments require multiple joins

**Optimization**: Central indexes scope columns (`cluster_id`, `namespace_id`) to speed up these queries.

## Permission Caching

### Backend Optimization

Central **caches user permissions** for 5 minutes:

```
1. User logs in
   ↓
2. Central loads roles and calculates permissions
   ↓
3. Permissions stored in cache (key: user ID)
   ↓
4. Next 5 minutes: permissions read from cache
   ↓
5. After 5 minutes: cache expires, permissions reloaded
```

**Why this matters**:
- Changing a user's role takes **up to 5 minutes** to take effect
- User might see 403 errors for a few minutes after role change
- Logging out and back in **does not** clear the cache (it's server-side)

## Key Takeaways for Frontend Engineers

1. **Permissions are enforced at query time** - Not just UI controls
2. **Users see filtered data** - Two users with same resource access may see different results due to scoping
3. **403 means permission denied** - Not "not found" - the resource exists but user can't access it
4. **Scoping filters SQL queries** - Adds WHERE clauses to database queries
5. **Global resources can't be scoped** - Admin, Integration, WorkflowAdministration
6. **Permission changes take up to 5 minutes** - Backend caching
7. **Cross-resource checks are backend-enforced** - UI should mirror these requirements

## Debugging Permission Issues

### Check User's Effective Permissions

In the UI, go to **Access Control** → **My Permissions**

Shows:
- All assigned roles
- Effective permissions per resource
- Scope (clusters and namespaces)

### Check API Response

```bash
# Call API with verbose logging
curl -H "Authorization: Bearer $TOKEN" \
     https://central/v1/alerts

# If 403:
# - Check response body for error message
# - Message often indicates missing resource
```

### Common Error Messages

```json
{
  "error": "Permission denied for resource: WorkflowAdministration",
  "code": 7
}
```

**Meaning**: User lacks access to `WorkflowAdministration` resource

```json
{
  "error": "Not found",
  "code": 5
}
```

**Meaning**: Either doesn't exist OR out of user's scope (Central doesn't reveal which)

## Related Documentation

- [Central APIs & Data Flow](./01-central-apis.md) - How Central processes requests
- [Architecture Overview](./architecture-overview.md) - How RBAC fits in the system
- Backend code: `/central/role/` - RBAC implementation

---

**Last Updated**: 2025-10-09
**For Frontend Engineers** - Understanding backend RBAC helps you build permission-aware UIs!
