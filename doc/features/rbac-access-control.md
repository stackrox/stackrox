# RBAC & Access Control

Authentication via username/password, SSO (OIDC/SAML), and API tokens with fine-grained authorization using three-state scoped access control (SAC).

**Primary Packages**: `pkg/auth`, `pkg/sac`, `central/role`, `central/auth`

## What It Does

Users authenticate through multiple methods, receive role assignments with granular permissions, and access resources scoped to specific clusters/namespaces. The system provides API token management for services, automatic role assignment via SSO groups, audit logging of access, and effective access visibility in the UI.

Three-state logic (Allow/Deny/Unknown) drives access decisions enforced at API, UI, and database query levels.

## Architecture

### Authentication

The `pkg/auth/` framework provides core interfaces, token issuance, and validation. Auth providers in `authproviders/` implement OIDC, SAML, basic auth, and API token authentication. Token management in `tokens/` handles JWT generation, validation, and claims. The `central/auth/` service manages user authentication, login flow, and machine-to-machine auth.

### Authorization (SAC)

The `pkg/sac/` implements three-state logic, effective access scope calculation, and query filtering. Resources in `sac/resources/` define resource types (deployments, policies, images, alerts, clusters). Scope keys in `observe/` generate cluster/namespace identifiers. Query filters in `pkg/search/scoped/` construct SAC-aware database queries.

### Role Management

The `central/role/` service handles role CRUD, permission set management, and access scope configuration. The datastore in `role/datastore/` persists roles, permission sets, and access scopes. Role resolver in `role/mapper/` translates user→roles→permissions→effective access. Group mapping in `central/group/` maps SSO groups to roles.

### Data Model

**User**: Auth provider ID, user ID (email/username/subject), attributes (name, email, groups), and assigned role names.

**Role**: Name identifier, permission set reference, access scope reference, and description.

**PermissionSet**: Name, permissions map of resource→access level (NO_ACCESS/READ_ACCESS/READ_WRITE_ACCESS for DEPLOYMENT, POLICY, IMAGE, ALERT, CLUSTER, etc.), and built-in flag.

**AccessScope**: Name, inclusion/exclusion rules (cluster ID or label selector, namespace or label selector), and built-in flag.

**AuthProvider**: Type (OIDC/SAML2/basic/API token/user certificate), name, provider-specific config (OIDC client ID, SAML metadata), enabled state, and group mappings.

**APIToken**: Name, assigned roles, optional expiration, revoked flag, and unique token ID in JWT.

**Identity** (in request context): User ID and attributes, resolved role list, flattened permission map, and effective access scope from all roles.

**ScopeKey**: Cluster UUID and namespace name for SAC checks and query filtering.

## Data Flow

### Username/Password Login

1. **Request**: User submits credentials to `/v1/login`, received by `central/auth/service/`
2. **Provider Lookup**: System iterates configured providers, validates password hash for basic auth
3. **User Creation**: If first login, creates user record and updates attributes
4. **Role Resolution**: `central/role/mapper/` resolves user→roles, checks direct assignments, applies default roles
5. **Token Issuance**: `pkg/auth/tokens/` generates JWT with claims (user ID, roles, expiration), signs with StackRox key
6. **Response**: Returns JWT to client for subsequent `Authorization: Bearer` header inclusion

### OIDC/SAML SSO Login

1. **Initiate**: User clicks SSO provider, redirects to provider with auth request
2. **Provider Authentication**: User authenticates with external IdP, redirects back with authorization code (OIDC) or SAML assertion
3. **Token Exchange**: Central exchanges auth code for ID/access token (OIDC), validates signature and claims
4. **Assertion Validation**: Central validates SAML assertion signature, extracts attributes
5. **Group Extraction**: Reads groups from OIDC claims or SAML attributes, `central/group/datastore/` looks up group→role mappings
6. **Role Assignment**: Assigns roles based on group memberships, falls back to default role
7. **Token Issuance**: Issues StackRox JWT with user identity and roles

### API Token Authentication

1. **Creation**: Admin creates token via UI/API, assigns roles and optional expiration, `pkg/auth/tokens/` generates JWT with token ID
2. **Usage**: Client sends token in `Authorization: Bearer` header, `pkg/grpc/authn/` extracts and validates
3. **Validation**: Checks signature, verifies not expired, checks not revoked in datastore
4. **Identity Resolution**: Loads roles from token claims, creates Identity with service account user

### Authorization (SAC)

1. **Request Reception**: gRPC/HTTP request arrives, `pkg/grpc/authz/` interceptor invoked
2. **Identity Extraction**: JWT validated and claims extracted, user ID and role list loaded, `pkg/auth/` creates Identity object
3. **Permission Resolution**: `central/role/mapper/` resolves roles→permission sets, combines permissions from multiple roles (union)
4. **Access Scope Resolution**: Resolves roles→access scopes, combines scopes (union of allowed clusters/namespaces), creates effective access scope
5. **Resource Access Check**: Request specifies resource type, `pkg/sac/` checks required permission (READ_ACCESS/READ_WRITE_ACCESS), returns Allow/Deny/Unknown
6. **Scope Filtering**: For scoped resources (deployments, alerts), applies scope filter with cluster ID and namespace, `pkg/sac/effectiveaccessscope/` generates scope checker, database query includes SAC filter
7. **Query Execution**: Database enforces row-level SAC, results filtered to allowed scopes
8. **Response**: Returns data visible to user based on effective access

### Three-State Logic

SAC uses Allow (explicit permission + scope match), Deny (no permission or excluded scope), and Unknown (cannot determine, e.g., resource not yet scoped).

Example: User has READ_DEPLOYMENT for cluster "prod". Request for deployment in "prod" → Allow. Request for deployment in "staging" → Deny. Request for policy (not scoped) → Allow if has READ_POLICY, else Deny.

## Configuration

**Central Environment**:
- `ROX_AUTH_PROVIDER`: Default auth provider (basic/oidc/saml)
- `ROX_ADMIN_PASSWORD`: Initial admin password
- `ROX_OIDC_ISSUER`: OIDC issuer URL
- `ROX_OIDC_CLIENT_ID`: OIDC client identifier
- `ROX_OIDC_CLIENT_SECRET`: OIDC client secret
- `ROX_SAML_METADATA_URL`: SAML IdP metadata URL

**Token Settings**:
- `ROX_JWT_EXPIRATION`: JWT token lifetime in hours (default: 24)
- `ROX_API_TOKEN_EXPIRATION`: API token default expiration (default: never)

Group mappings: Map SSO groups to roles (e.g., Group "DevOps"→Role "Continuous Integration", Group "Security"→Role "Analyst", Group "Admins"→Role "Admin").

## Testing

**Unit Tests**:
- `pkg/auth/*_test.go`: Token validation, identity resolution
- `pkg/auth/authproviders/*_test.go`: OIDC/SAML/basic providers
- `pkg/sac/*_test.go`: Three-state logic, scope matching
- `pkg/sac/effectiveaccessscope/*_test.go`: Scope resolution
- `central/role/datastore/*_test.go`: CRUD operations

**E2E**: `AuthTest.groovy`, `RBACTest.groovy`, `APITokenTest.groovy`, `SACTest.groovy` in `qa-tests-backend/`

## Known Limitations

**Performance**: Many roles slow request processing. Large access scopes (100+ clusters/namespaces) generate large SQL queries. JWT validation adds latency on every request.

**Behavior**: Role permission changes require user re-login. API token revocation not immediate (cached briefly). Overlapping access scopes can confuse (union logic).

**Compatibility**: Not all providers support OIDC refresh tokens correctly. SAML single logout not fully supported. Client certificate auth limited to specific cases.

**Features**: No deny rules (only exclusions from scope). No attribute-based access control on resource attributes (labels/annotations). No time-based access grants. OIDC/SAML groups only sync at login, not continuously.

**Workarounds**: Use specific access scopes instead of "deny" access. Create separate roles for different permission levels instead of complex scopes. Force user re-login after role changes (invalidate session). Use short JWT expiration for frequently changing permissions. Monitor audit logs for unexpected access patterns. Automate API token rotation periodically.

## Implementation

**Authentication**: `pkg/auth/`, `pkg/auth/authproviders/`, `pkg/auth/tokens/`, `central/auth/`
**Authorization**: `pkg/sac/`, `pkg/sac/resources/`, `pkg/sac/effectiveaccessscope/`, `pkg/search/scoped/`
**Role Management**: `central/role/datastore/`, `central/role/mapper/`, `central/group/`
**Middleware**: `pkg/grpc/authz/`, `pkg/grpc/authn/`, `central/auth/middleware/`
**API**: `proto/api/v1/auth_service.proto`, `proto/api/v1/role_service.proto`, `proto/storage/role.proto`
