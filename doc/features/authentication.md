# Authentication and Authorization

**Primary Packages**: `pkg/auth`, `central/auth`, `central/apitoken`

## For API Consumers

This section explains how to authenticate when calling StackRox APIs programmatically (for automation, CI/CD pipelines, scripts, or external integrations).

### Generate an API Token

**Option 1: Via UI**
1. Log in to the StackRox UI
2. Navigate to **Platform Configuration → Integrations → API Token**
3. Click **Generate Token**
4. Provide a name and select role(s)
5. Optionally set an expiration time
6. Click **Generate** and copy the token (shown only once)

**Option 2: Via roxctl**
```bash
roxctl --endpoint central.example.com:443 \
       --password <admin-password> \
       central generate token \
       --name "CI Pipeline Token" \
       --role Admin \
       --expiration 24h
```

**Example output:**
```
eyJhbGciOiJSUzI1NiIsImtpZCI6IjEyMyJ9.eyJpc3MiOiJjZW50cmFsIiwiYXVkIjpbImNlbnRyYWwiXSwianRpIjoiYWJjZGVmZ2giLCJpYXQiOjE3MDAwMDAwMDAsImV4cCI6MTcwMDAwMDAwMCwicm9sZXMiOlsiQWRtaW4iXSwibmFtZSI6IkNJIFBpcGVsaW5lIFRva2VuIn0.signature
```

**Important:** Store the token securely. It cannot be retrieved again after generation.

### Authenticate API Calls

#### REST API (HTTP)

Set the `Authorization` header with `Bearer` scheme:

```bash
curl -X GET \
  https://central.example.com/v1/alerts \
  -H "Authorization: Bearer <API_TOKEN>" \
  -H "Content-Type: application/json"
```

**Example with POST:**
```bash
curl -X POST \
  https://central.example.com/v1/policies \
  -H "Authorization: Bearer <API_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Policy",
    "severity": "HIGH_SEVERITY",
    "lifecycleStages": ["DEPLOY"]
  }'
```

#### gRPC API

**Go client example:**
```go
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/metadata"
)

// Create TLS credentials
creds := credentials.NewClientTLSFromCert(certPool, "central.example.com")

// Dial with TLS
conn, err := grpc.Dial(
    "central.example.com:443",
    grpc.WithTransportCredentials(creds),
)
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

// Create context with token in metadata
ctx := metadata.AppendToOutgoingContext(
    context.Background(),
    "authorization", "Bearer <API_TOKEN>",
)

// Make API call
client := v1.NewAlertServiceClient(conn)
resp, err := client.ListAlerts(ctx, &v1.ListAlertsRequest{})
```

**Python client example:**
```python
import grpc
from google.protobuf import json_format

# Create secure channel
credentials = grpc.ssl_channel_credentials()
channel = grpc.secure_channel('central.example.com:443', credentials)

# Create metadata with token
metadata = [('authorization', 'Bearer <API_TOKEN>')]

# Make API call
stub = AlertServiceStub(channel)
response = stub.ListAlerts(
    ListAlertsRequest(),
    metadata=metadata
)
```

#### roxctl CLI

**Method 1: Environment variable**
```bash
export ROX_API_TOKEN="<API_TOKEN>"
roxctl --endpoint central.example.com:443 deployment check --file deployment.yaml
```

**Method 2: Command-line flag**
```bash
roxctl --endpoint central.example.com:443 \
       --api-token "<API_TOKEN>" \
       deployment check --file deployment.yaml
```

**Method 3: Token file**
```bash
echo "<API_TOKEN>" > /path/to/token-file
roxctl --endpoint central.example.com:443 \
       --api-token-file /path/to/token-file \
       deployment check --file deployment.yaml
```

**Method 4: Config file**
Create `~/.roxctl/config.yaml`:
```yaml
central:
  endpoint: central.example.com:443
  api_token: <API_TOKEN>
```

Then run:
```bash
roxctl deployment check --file deployment.yaml
```

### Token Management

#### List API Tokens
```bash
roxctl --endpoint central.example.com:443 \
       --api-token "<API_TOKEN>" \
       central token list
```

#### Revoke API Token
```bash
# Via roxctl (need token ID from list command)
roxctl --endpoint central.example.com:443 \
       --api-token "<API_TOKEN>" \
       central token revoke <TOKEN_ID>

# Via UI
# Platform Configuration → Integrations → API Token → Click trash icon
```

#### Automatic Expiration
Tokens with expiration are automatically revoked after expiry. Configure expiration during token generation:
- Default: No expiration (token valid indefinitely)
- Recommended: Set expiration for CI/CD tokens (e.g., 90 days, 1 year)
- Short-lived: For temporary access (e.g., 24 hours, 7 days)

### Security Best Practices

1. **Principle of least privilege**: Grant minimum required roles
   - Use `Analyst` for read-only access
   - Use `Admin` only when write access is needed
   - Create custom roles for specific permissions

2. **Token rotation**: Regularly rotate long-lived tokens
   - Set expiration on all tokens
   - Rotate before expiration
   - Revoke old tokens immediately after rotation

3. **Secure storage**:
   - Never commit tokens to Git repositories
   - Use secret management systems (Vault, Kubernetes Secrets, GitHub Secrets)
   - Restrict file permissions for token files (`chmod 600`)

4. **Environment-specific tokens**:
   - Use separate tokens for dev/staging/prod environments
   - Use separate tokens per application/service
   - Name tokens descriptively (e.g., "GitHub Actions - Prod Pipeline")

5. **Monitoring**:
   - Review API token usage via audit logs
   - Revoke unused or suspicious tokens
   - Enable token expiration notifications (environment variable `ROX_TOKEN_EXPIRATION_NOTIFIER_INTERVAL`)

### Troubleshooting

**Problem: 401 Unauthorized**
- Verify token is valid (not expired or revoked)
- Check `Authorization` header format: `Bearer <token>` (note the space)
- Ensure token has sufficient permissions for the API endpoint

**Problem: 403 Forbidden**
- Token is valid but lacks required role/permission
- Check token roles: UI → Platform Configuration → Integrations → API Token
- Generate new token with appropriate role or create custom role

**Problem: TLS/Certificate errors**
- For self-signed certificates, add `--insecure-skip-tls-verify` flag (development only)
- For custom CA, use `--ca <path-to-ca-cert>` flag
- Production: Use proper TLS certificates (Let's Encrypt, corporate CA)

**Problem: Token not working after generation**
- Tokens are active immediately (no propagation delay)
- Verify token was copied completely (no truncation)
- Check Central logs for authentication errors: `kubectl logs deploy/central | grep auth`

### Examples by Use Case

#### CI/CD Pipeline (GitHub Actions)
```yaml
- name: Scan deployment
  env:
    ROX_API_TOKEN: ${{ secrets.ROX_API_TOKEN }}
    ROX_ENDPOINT: central.example.com:443
  run: |
    curl -O -L https://mirror.openshift.com/pub/rhacs/assets/latest/bin/Linux/roxctl
    chmod +x roxctl
    ./roxctl deployment check \
      --file k8s/deployment.yaml \
      --output json
```

#### Infrastructure as Code (Terraform)
```hcl
provider "roxctl" {
  endpoint  = "central.example.com:443"
  api_token = var.rox_api_token
}

resource "roxctl_policy" "privileged_containers" {
  name     = "Deny Privileged Containers"
  severity = "HIGH"
  # ... policy configuration
}
```

#### Kubernetes Operator / Controller
```go
// Read token from mounted secret
tokenBytes, err := ioutil.ReadFile("/var/run/secrets/stackrox.io/token")
token := strings.TrimSpace(string(tokenBytes))

// Create gRPC client
conn, _ := grpc.Dial(
    "central.stackrox.svc:443",
    grpc.WithTransportCredentials(creds),
)

ctx := metadata.AppendToOutgoingContext(
    context.Background(),
    "authorization", "Bearer "+token,
)

client := v1.NewPolicyServiceClient(conn)
policies, _ := client.ListPolicies(ctx, &v1.RawQuery{})
```

#### Scheduled Jobs / Cronjobs
```bash
#!/bin/bash
# weekly-vulnerability-report.sh

# Token from environment or secret management system
TOKEN="${ROX_API_TOKEN:-$(cat /etc/secrets/rox-token)}"

# Generate vulnerability report
curl -X POST \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  https://central.example.com/v1/reports/vulnerability \
  -d '{
    "reportScope": {
      "clusters": ["*"]
    },
    "format": "PDF"
  }' \
  -o "/reports/vuln-report-$(date +%Y%m%d).pdf"
```

---

## Overview (For Implementers)

Multi-provider authentication framework supporting user authentication (OIDC, SAML, OpenShift OAuth, basic auth, client certificates), machine-to-machine (M2M) authentication (external OIDC tokens), and API tokens for programmatic access. Built on JWT token issuance with role-based access control integration.

**Capabilities**: Multiple identity providers with pluggable architecture, JWT issuance/validation with RSA-256 signatures, role mapping from external groups to StackRox roles, API tokens for CI/CD and automation, M2M auth for GitHub Actions/Kubernetes ServiceAccounts/generic OIDC, certificate-based authentication (mTLS).

## Architecture

Component hierarchy: `pkg/auth/` contains core framework with `authproviders/` (provider framework + backends for oidc/saml/openshift/userpki/basic/iap), `tokens/` (JWT issuance/validation), `permissions/` (role resolution), `user/` (attribute verification). `central/auth/` handles M2M authentication with `m2m/` (token exchange for GitHub/K8s/OIDC), `internaltokens/` (service-to-service tokens), `userpass/` (user/password auth). `central/apitoken/` manages API token lifecycle with `backend/` (issuance/revocation), `datastore/` (token metadata storage), `expiration/` (auto-expiration worker), `service/` (gRPC API).

### Authentication Flow

**User Authentication**: User initiates auth, registry routes to provider backend, provider handles IdP-specific flow (OAuth/SAML), external claims extracted (username/email/groups), attribute verification (optional), role mapping (external groups → StackRox roles), token issuance (signed JWT with claims), subsequent requests use token validation.

**M2M Authentication**: External ID token presented (GitHub OIDC, K8s SA token), token verification (signature/expiration/audience), claim extraction (repo/namespace), role resolution via pattern matching, StackRox token issuance, API access with resolved roles.

**API Token Authentication**: User generates token via API (with roles, optional expiry), token metadata stored in DB (not raw token), raw token returned once (only time visible), client presents token in Authorization header, token validated (signature + revocation check), request proceeds with token's roles.

## Key Code Locations

### Core Framework

**Provider Interface** at `authproviders/provider.go` and `provider_impl.go`: Provider extends tokens.Source with Name, Type, Enabled, Backend, GetOrCreateBackend, RoleMapper, Issuer, AttributeVerifier, ApplyOptions, Active, MarkAsActive methods.

**Backend Interface** at `authproviders/backend.go`: LoginURL, ProcessHTTPRequest, ExchangeToken, Validate, OnEnable, OnDisable methods. File: `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/auth/authproviders/backend.go`

**Registry** at `authproviders/registry_impl.go`: Central hub for providers, routes HTTP requests, manages backend factories, issues tokens via common infrastructure, handles refresh token cookies. File: `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/auth/authproviders/registry_impl.go`

**Token Issuance** at `tokens/issuer.go`: IssuerFactory creates Issuer with CreateIssuer and UnregisterSource. Issuer has Issue method for creating tokens with claims. File: `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/auth/tokens/issuer.go`

**Token Validation** at `tokens/validator.go`: Validator interface with Validate method. Parses JWT, verifies signature, checks audience/issuer/expiration, returns validated claims + sources. File: `/Users/rc/go/src/github.com/stackrox/stackrox/pkg/auth/tokens/validator.go`

### Provider Implementations

**OIDC** at `authproviders/oidc/backend_impl.go`: Full OpenID Connect 1.0 implementation with authorization code/implicit/hybrid flows, response modes (fragment/query/form_post), auto mode selection based on client secret, refresh tokens via offline_access scope, custom claim mapping from IdP token, userinfo endpoint enrichment for groups, nonce verification for id_tokens.

**SAML** at `authproviders/saml/backend_impl.go`: SAML 2.0 Service Provider with metadata URL or manual configuration, Assertion Consumer Service, multiple attribute mapping (friendly names + URIs), extracts email/name/groups.

**OpenShift** at `authproviders/openshift/backend_impl.go`: OpenShift OAuth Server integration with service account token authentication, auto-discovery of OAuth endpoints, certificate watching (service CA, internal CA, injected CA), refresh token support via Dex connector, client ID `system:serviceaccount:<namespace>:central`, monitors 3 CA paths for certificate updates, recreates connector on cert changes.

**User PKI** at `authproviders/userpki/backend.go`: mTLS client certificate authentication, certificate fingerprint verification, chain validation against configured CAs, attributes from certificate subject, expiration from certificate validity.

**Basic Auth** at `authproviders/basic/backend.go`: Username/password authentication, htpasswd backend (bcrypt only), browser HTTP Basic Auth challenge, time-limited login URLs (5s window), integration with `pkg/grpc/authn/basic`.

### M2M Authentication

**Token Exchanger** at `central/auth/m2m/exchanger.go`: TokenExchanger interface with ExchangeToken, Provider, Config methods. Exchange flow: verify ID token (signature/expiration/audience), extract claims (type-specific: GitHub, K8s, generic), resolve roles via pattern matching, create StackRox claims, issue signed token, return access token. File: `/Users/rc/go/src/github.com/stackrox/stackrox/central/auth/m2m/exchanger.go`

**Claim Extractors**: Generic OIDC at `generic_claim_extractor.go` (sub/email/groups), GitHub Actions at `github_claim_extractor.go` (repository/repository_owner/workflow_ref/actor), Kubernetes at `kube_claim_extractor.go` (service-account:namespace:name, SA namespace/name).

**Role Mapping** at `central/auth/m2m/role_mapper.go`: For each mapping in config, extract claim value from ID token, match against regex pattern, if match assign configured role, accumulate all matching roles.

**DataStore** at `central/auth/datastore/datastore_impl.go`: Stores M2M config in PostgreSQL, maintains in-memory set of token exchangers, initializes exchangers on startup, updates set on config changes.

### API Tokens

**Backend** at `central/apitoken/backend/backend.go`: IssueRoleToken, RevokeToken, GetTokens operations.

**DataStore** at `central/apitoken/datastore/datastore_impl.go`: PostgreSQL table `api_tokens` with dual storage (main + scheduled revocation), schema includes id/name/roles/issued_at/expiration/revoked, SAC enforcement via Integration resource permissions.

**Expiration Worker** at `central/apitoken/expiration/`: Background worker monitors expiration, automatically revokes expired tokens, cleanup of expired metadata.

**Service** at `central/apitoken/service/service_impl.go`: gRPC endpoints GenerateToken, GetAPITokens, RevokeAPIToken.

## Token Structure

JWT claims structure: iss (issuer-id), aud (source-id array), jti (uuid), iat (issued at), exp (expiration), roles array, external_user (user_id, full_name, email, attributes with groups/email/userid), name (for API tokens). Technical details: RSA-256 signature via go-jose/go-jose/v4, validation checks audience claim for source acceptance, revocation via in-memory layer with expiry-based cleanup.

## Configuration

Provider registration at `registry.RegisterBackendFactory` with backend factory functions. Provider creation via `registry.CreateProvider` with options for Type, Name, Config, Enabled.

OIDC config keys: issuer (IdP issuer URL), client_id, client_secret (optional for fragment mode), mode (auto/fragment/query/post), extra_scopes, disable_offline_access_scope.

API token issuance: `roxctl central generate token --name "CI Pipeline Token" --role Admin --expiration 24h` or via API `POST /v1/apitokens/generate` with name, roles, expiration.

## Security Considerations

**Token Security**: Raw tokens NEVER stored (only metadata), HTTPS required for all auth flows, configurable expiration (default 24h for API tokens), immediate revocation propagation to in-memory layer.

**Provider Security**: OIDC nonces not persisted (vulnerable to restarts), revocation in-memory only (not distributed across Central replicas), client secret storage redacted in UI views and merged on update, refresh tokens use HTTP-only cookies (not encrypted at rest).

**M2M Security**: Regex validation for mapping patterns from user input (validate complexity), overly broad patterns grant excessive access, external tokens must be for StackRox (audience validation), strict issuer checking with HTTPS required.

**Certificate Security**: Fingerprint verification required for certificate acceptance, full chain validated against configured CAs, certificate validity checked on each auth.

## Known Issues

Jira items addressed: OIDC issuer URL could cache successful issuer to avoid retries (oidc/provider.go:32), OIDC hybrid flow could expose admin knob (oidc/provider.go:111), helper bugs documented in test cases at oidc/internal/endpoint/helper_test.go to prevent unconscious behavior changes.

Complexity concerns: Backend creation async with 30s timeout and 10s retry interval supports provider registration before backend factories available (provider_impl.go). Provider update validation invalidates tokens issued before update with 10s leeway for clock skew/mark active race, can cause temporary auth failures after config changes. OpenShift cert watching has global registry of backends for cert update notifications, backend recreated on every cert change, no clear cleanup on backend deletion (watcher.go). Token exchanger lifecycle lacks graceful shutdown, may have open HTTP connections on shutdown, should implement Close method. Configuration validation compiles regex on each M2M config upsert, should validate before save, pre-compilation stored in exchanger not validated upfront.

Recent changes: Multiple iterations on internal role structure and permissions, enhanced token claims for multiple internal roles, improved permission modeling, better separation of concerns. Policy enhancements: enhanced token policy validation, better role verification, improved error handling. OpenShift enhancements (ROX-26042): certificate watching and auto-reconnection, service account token authentication, duplicate name validation. OIDC enhancements: ROX-23628 extra scopes configuration, ROX-25055/25056 retry OIDC provider creation, ROX-21611 GitLab groups fix. Token infrastructure: ROX-33014 common issuer source refactoring, ROX-33191 use proxy.Transport in auth providers for proxy support, removed ExpireAt from tokens.

## Testing

Unit tests use mock generators via go:generate mockgen-wrapper, test utilities in `pkg/testutils/roletest`, testify suites used extensively. Integration points require database for provider storage, HTTP server for OAuth/SAML callbacks, certificate file monitoring for OpenShift. Test mode with query parameter `test=true` returns user metadata instead of token for UI auth provider testing without full login flow.

## Related Components

`pkg/grpc/authn` has gRPC authentication middleware using validators, `pkg/grpc/authz` handles authorization based on permissions, `pkg/sac` implements scope-based access control, `pkg/jwt` provides low-level JWT signing/validation, `pkg/cryptoutils` handles nonce generation and certificate fingerprinting, `central/role` manages role database storage, `central/group` handles group-based role mapping, `central/user` manages users.

## Performance

Provider lookup: O(1) map lookup by ID. Token validation: cryptographic signature verification (RSA-256). Revocation check: O(1) map lookup with periodic cleanup. Backend creation: async with 30s timeout, cached after first success. Certificate watching: file polling via fsnotify (OpenShift only).
