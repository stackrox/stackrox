# Proto Enrichment Workflow

A self-contained prompt for subagents enriching a single ACS service proto file.

---

## Instructions for Subagent

You are enriching the ACS API proto file for **{{SERVICE_NAME}}** to Stripe-level documentation quality.

Your inputs:
- Service name: `{{SERVICE_NAME}}`
- Proto file: `{{PROTO_FILE}}`  (e.g., `proto/api/v1/alert_service.proto`)
- Implementation: `{{IMPL_FILE}}`  (e.g., `central/alert/service/service_impl.go`)
- Storage proto: `{{STORAGE_FILE}}`  (e.g., `proto/storage/alert.proto`, or "none")
- Swagger output: `{{SWAGGER_FILE}}`  (e.g., `generated/api/v1/alert_service.swagger.json`)
- Beads task ID: `{{TASK_ID}}`

---

## Step 1 — Read Source Files

Read all of the following in one pass:
1. `{{PROTO_FILE}}` — the proto to enrich
2. `{{IMPL_FILE}}` — the Go implementation (to understand actual behavior)
3. `{{STORAGE_FILE}}` — storage proto for field semantics (if applicable)
4. `{{SWAGGER_FILE}}` — current Swagger output (to see what already has comments)

If the impl file path is wrong, search: `central/**/service*impl*.go` or `central/**/service.go`.

---

## Step 2 — Understand What Each RPC Does

For each `rpc` in the proto, answer:
1. What does it return and what does it filter/query?
2. What are the required vs optional request fields?
3. What errors can it return (not found, invalid, permission denied)?
4. Are there any side effects, ordering guarantees, or pagination?
5. What auth/RBAC is required (check impl for permission checks)?

Derive answers from the implementation code, NOT from guessing. If uncertain, write "Returns X based on Y" where Y is something you can see in the code.

---

## Step 3 — Write Enriched Comments

### Comment Pattern

**RPC with summary only** (for simple, self-evident operations):
```proto
// GetAlert returns the alert with the given ID.
rpc GetAlert(ResourceByID) returns (storage.Alert) {}
```

**RPC with summary + description** (for operations with nuance, errors, or context):
```proto
// ListAlerts returns a paginated list of alerts matching the query.
//
// Alerts can be filtered by severity, policy name, cluster, namespace, and
// state (ACTIVE, SNOOZED, RESOLVED). Uses StackRox search query syntax
// (e.g. "Severity:CRITICAL+Cluster:production").
//
// Returns NOT_FOUND if the requested cluster does not exist.
// Returns INVALID_ARGUMENT if the query syntax is malformed.
rpc ListAlerts(ListAlertsRequest) returns (ListAlertsResponse) {}
```

**Service-level comment** (required for all services, goes above `service ServiceName {`):
```proto
// AlertService manages security alerts triggered by policy violations.
//
// Alerts are created automatically when deployed workloads violate active
// security policies. They can be listed, filtered by severity or cluster,
// resolved individually or in bulk, and queried as time series data.
//
// Authentication: all endpoints require a valid API token with read access
// to the Alert resource. Resolve and delete operations require write access.
service AlertService {
```

**Field comments** (goes above each field in message definitions):
```proto
message ListAlertsRequest {
  // query filters alerts using StackRox search syntax.
  // Example: "Severity:CRITICAL+Cluster:production"
  string query = 10;

  // pagination controls the page size and offset.
  Pagination pagination = 11;
}
```

**Enum value comments**:
```proto
enum RequestGroup {
  UNSET = 0;
  // CATEGORY groups alert counts by policy category.
  CATEGORY = 1;
  // CLUSTER groups alert counts by cluster.
  CLUSTER = 2;
}
```

### Quality Checklist

Before writing, verify each comment:
- [ ] Derived from actual code behavior (not guessed)
- [ ] States WHAT it does, not just restates the name
- [ ] Mentions key filters/parameters when relevant
- [ ] Notes error conditions for non-trivial operations
- [ ] Service comment describes the domain and auth requirements
- [ ] No hallucinated endpoint paths or field names
- [ ] Uses present tense ("Returns" not "Will return")
- [ ] Field comments explain semantics, not just type ("Unix timestamp in seconds" not "int64 field")

### What NOT to do

- Do NOT repeat the method name verbatim: `// GetAlert gets an alert.` ❌
- Do NOT guess behavior — if unsure, check the impl file
- Do NOT invent error codes that aren't in the impl
- Do NOT add comments to fields that are truly self-evident (e.g., `id`, `name` on well-known resources)
- Do NOT change any proto fields, options, or RPC signatures — only add/modify comments

---

## Step 4 — Apply Changes

Edit `{{PROTO_FILE}}` directly:
1. Add/update the service-level comment
2. Add/update each RPC comment (summary + description where warranted)
3. Add comments to request/response message fields (focus on non-obvious ones)
4. Add comments to key enum values

Make atomic edits — one RPC at a time if needed to keep the diff readable.

---

## Step 5 — Validate

After editing, run a quick sanity check:
```bash
# Verify proto syntax is valid (no parse errors)
grep -c "rpc " {{PROTO_FILE}}  # should match expected count
# Confirm no import or syntax errors introduced
head -5 {{PROTO_FILE}}
```

The proto is NOT regenerated as part of individual enrichment — regeneration happens in bulk after all services are done (stackrox-5rl: Review Phase 1 Enrichment & Regenerate Swagger).

---

## Step 6 — Commit and Close

```bash
git add {{PROTO_FILE}}
git commit -m "docs(proto): enrich {{SERVICE_NAME}} API documentation

Add service-level description, RPC summaries/descriptions, field comments.

Part of Phase 0 API enrichment (stackrox-{{TASK_ID}}).
Generated with AI assistance."

bd close {{TASK_ID}} --reason "Enriched: service comment, X RPCs with summaries, Y RPCs with descriptions, Z field comments added." --json
```

---

## Service → File Mapping

| Service | Proto | Impl Dir | Storage |
|---------|-------|----------|---------|
| AlertService | proto/api/v1/alert_service.proto | central/alert/service/ | proto/storage/alert.proto |
| DeploymentService | proto/api/v1/deployment_service.proto | central/deployment/service/ | proto/storage/deployment.proto |
| ImageService | proto/api/v1/image_service.proto | central/image/service/ | proto/storage/image.proto |
| PolicyService | proto/api/v1/policy_service.proto | central/policy/service/ | proto/storage/policy.proto |
| ClusterService | proto/api/v1/cluster_service.proto | central/cluster/service/ | proto/storage/cluster.proto |
| CVEService | proto/api/v1/cve_service.proto | central/cve/service/ | proto/storage/cve.proto |
| SearchService | proto/api/v1/search_service.proto | central/search/service/ | none |
| ComplianceService | proto/api/v1/compliance_service.proto | central/compliance/service/ | proto/storage/compliance.proto |
| ConfigService | proto/api/v1/config_service.proto | central/config/service/ | proto/storage/config.proto |
| ReportService | proto/api/v1/report_service.proto | central/reports/service/ | proto/storage/report_configuration.proto |
| VulnMgmtService | proto/api/v1/vuln_mgmt_service.proto | central/vulnmgmt/service/ | none |
| AuthService | proto/api/v1/auth_service.proto | central/auth/service/ | none |
| AuthProviderService | proto/api/v1/authprovider_service.proto | central/authprovider/service/ | proto/storage/auth_provider.proto |
| ApiTokenService | proto/api/v1/api_token_service.proto | central/apitoken/service/ | proto/storage/api_token.proto |
| ClusterService | proto/api/v1/cluster_service.proto | central/cluster/service/ | proto/storage/cluster.proto |
| ClusterInitService | proto/api/v1/cluster_init_service.proto | central/clusterinit/service/ | none |
| BackupService | proto/api/v1/backup_service.proto | central/backup/service/ | none |
| CloudSourceService | proto/api/v1/cloud_source_service.proto | central/cloudsources/service/ | proto/storage/cloud_source.proto |
| ComplianceManagementService | proto/api/v1/compliance_management_service.proto | central/compliance/service/ | none |
| GroupService | proto/api/v1/group_service.proto | central/group/service/ | proto/storage/group.proto |
| ImageIntegrationService | proto/api/v1/image_integration_service.proto | central/imageintegration/service/ | proto/storage/image_integration.proto |
| IntegrationHealthService | proto/api/v1/integration_health_service.proto | central/integrationhealth/service/ | none |
| MetadataService | proto/api/v1/metadata_service.proto | central/metadata/service/ | none |
| MitreService | proto/api/v1/mitre_service.proto | central/mitre/service/ | none |
| NamespaceService | proto/api/v1/namespace_service.proto | central/namespace/service/ | proto/storage/namespace.proto |
| NetworkBaselineService | proto/api/v1/network_baseline_service.proto | central/networkbaseline/service/ | none |
| NetworkGraphService | proto/api/v1/network_graph_service.proto | central/networkgraph/service/ | none |
| NetworkPolicyService | proto/api/v1/network_policy_service.proto | central/networkpolicies/service/ | none |
| NodeService | proto/api/v1/node_service.proto | central/node/service/ | proto/storage/node.proto |
| NotifierService | proto/api/v1/notifier_service.proto | central/notifier/service/ | proto/storage/notifier.proto |
| PolicyCategoryService | proto/api/v1/policy_category_service.proto | central/policycategory/service/ | none |
| ProcessBaselineService | proto/api/v1/process_baseline_service.proto | central/processbaseline/service/ | proto/storage/process_baseline.proto |
| ProcessService | proto/api/v1/process_service.proto | central/process/service/ | none |
| RBACService | proto/api/v1/rbac_service.proto | central/rbac/service/ | proto/storage/rbac.proto |
| ReportConfigurationService | proto/api/v1/report_configuration_service.proto | central/reportconfigurations/service/ | none |
| ResourceCollectionService | proto/api/v1/resource_collection_service.proto | central/resourcecollection/service/ | none |
| RoleService | proto/api/v1/role_service.proto | central/role/service/ | proto/storage/role.proto |
| SecretService | proto/api/v1/secret_service.proto | central/secret/service/ | proto/storage/secret.proto |
| SensorUpgradeService | proto/api/v1/sensor_upgrade_service.proto | central/sensor/service/ | none |
| SignatureIntegrationService | proto/api/v1/signature_integration_service.proto | central/signatureintegration/service/ | proto/storage/signature_integration.proto |
| TelemetryService | proto/api/v1/telemetry_service.proto | central/telemetry/service/ | none |
| UserService | proto/api/v1/user_service.proto | central/user/service/ | proto/storage/user.proto |

---

## Tips for Fast, Accurate Enrichment

1. **Read the impl first** — the `Handle` or service method implementations show exactly what the RPC does, what it queries, what errors it returns.
2. **Grep for permission checks** — `sac.With(permissions.View(...))` or similar tells you what auth is needed.
3. **Check existing comments in storage protos** — storage fields often have better comments than service fields.
4. **Use the Swagger JSON** — `{{SWAGGER_FILE}}` shows existing summaries; keep them if accurate, improve if thin.
5. **Batch your edits** — read all RPCs first, draft all comments, then write all at once.
