# Audit Logging

**Primary Packages**: `central/audit`, `compliance-app/collection/auditlog`
**Components**: Audit message creation, notifier integration, K8s audit log collection

## Overview

Creates comprehensive audit trails for security-relevant operations: (1) StackRox administration events (API calls, configuration changes, user actions) and (2) Kubernetes audit logs (cluster-level API server activity). All audit messages sent to configured notifiers for external storage, SIEM integration, or compliance archival.

**Capabilities**: Audit all mutating API operations, capture user identity/permissions/source IP, track authentication/authorization failures, collect K8s cluster audit logs from master nodes, send to external systems via notifiers, filter service-to-service noise while capturing security events, support audit without exposing permissions (privacy option).

## Architecture

### Central Audit

**Interface** at `pkg/audit.Auditor`: `SendAuditMessage(ctx, req, grpcMethod, authError, requestError)`.

**Implementation** at `central/audit/audit.go`: `audit` struct holds notifications processor and withoutPermissions flag (controlled by `ROX_AUDIT_LOG_WITHOUT_PERMISSIONS`). `SendAuditMessage` skips if no audit notifiers enabled, filters service-to-service except allowlisted endpoints (M2M token generation, API token creation), creates audit message, sends async to notifiers via `ProcessAuditMessage`.

**Message Creation** at `newAuditMessage`: Extracts identity from context, determines method (API/UI/CLI based on headers and service context), maps gRPC method to interaction type (CREATE/UPDATE/DELETE), captures source IP (X-Forwarded-For or RemoteAddr or peer address), calculates status from auth/request errors (AUTH_FAILED if authorization error or scope denial, REQUEST_FAILED if other error, REQUEST_SUCCEEDED otherwise), builds user info optionally without permissions.

**Interceptors**: gRPC interceptor at `UnaryServerInterceptor` handles request, extracts auth status from context, sends audit message async (non-blocking). HTTP interceptor at `PostAuthHTTPInterceptor` wraps response writer, handles request, audits if mutating method (POST/PUT/DELETE/PATCH) and status >=400 indicates error.

### Compliance Audit Log Collection

**Purpose**: Collect K8s audit logs from master nodes for runtime threat detection.

**Implementation** at `compliance/collection/auditlog/auditlog_impl.go`: `AuditLogCollector` interface with `StartReader(ctx, state)` and `StopReader()`. Collection process: Sensor sends `MsgToCompliance_AuditLogCollectionRequest_StartReq` to master node, Compliance starts reader on master, reader tails audit log files and processes events, events sent to Sensor then forwarded to Central, can resume from saved state after restarts.

**Master Node Detection**: Only runs on nodes with labels `node-role.kubernetes.io/master` or `node-role.kubernetes.io/control-plane`.

**Saved State**: File position stored for resumption, prevents duplicate event processing, state persisted in compliance container.

## Audit Event Types

### StackRox Admin Events

Captured: user login/logout, API token generation, policy create/modify/delete, integration config changes, cluster register/delete, image scanning, configuration changes (admission control, notifiers), role/permission changes, backup/restore operations.

Event structure (proto at `proto/storage/audit.proto`): timestamp, method (API/UI/CLI), request (endpoint, interaction type CREATE/UPDATE/DELETE, payload), status (SUCCESS/AUTH_FAILED/REQUEST_FAILED), user info (username, roles, permissions), source IP.

### Kubernetes Audit Events

Captured from K8s audit logs: pod creations/deletions, secret access, ConfigMap changes, RBAC modifications, service account token requests, persistent volume operations, network policy changes, admission controller decisions.

Event structure: id, timestamp, audit data (stage RequestReceived/ResponseComplete, verb get/create/update/delete, user, user groups, source IPs, object reference, response status HTTP code).

## Service-to-Service Filtering

Default: service-to-service gRPC calls NOT audited to reduce noise.

Allowlisted endpoints at `auditableServiceEndpoints`: `/v1.AuthService/ExchangeAuthMachineToMachineToken`, `/v1.APITokenService/GenerateToken`. These security-sensitive internal endpoints audited even for service-to-service calls.

## User Information

Structure: username, roles list, permissions list (optional based on ROX-20288). Privacy option via `ROX_AUDIT_LOG_WITHOUT_PERMISSIONS=true` strips permission details from user info, reduces log size and security concerns, still captures username and roles, useful for compliance where permission details are sensitive.

## Method Detection

Request method types: UI (HTTP with Referer header), API (HTTP without Referer or service-to-service), CLI (gRPC from user context like roxctl).

## Source IP Tracking

IP capture handles: direct connections (RemoteAddr), proxied connections (X-Forwarded-For first IP as client), gRPC connections (peer address).

## Notifier Integration

Notifier types supporting audit: Splunk (HTTP Event Collector), Syslog (RFC 5424), Generic Webhook (JSON POST), Email (summaries), PagerDuty (critical events), Slack (notifications).

Processing at `notifierProcessor.ProcessAuditMessage`: gets audit notifiers, sends to each async with error logging, non-blocking to avoid request delays.

## Code Locations

**Central Audit**: `central/audit/audit.go` (service), `central/audit/interceptors.go` (gRPC/HTTP), `central/audit/userinfo.go` (user data), `central/audit/status.go` (status calculation).

**Compliance**: `compliance/collection/auditlog/auditlog_impl.go` (collector), `compliance/collection/auditlog/state.go` (state management).

**Integration**: `pkg/notifier/processor.go` (notifier processor), `pkg/audit/auditor.go` (interface).

## Environment Variables

**Central**: `ROX_AUDIT_LOG_WITHOUT_PERMISSIONS` (strip permissions, default: false).

**Compliance**: `ROX_ENABLE_K8S_AUDIT_LOG_COLLECTION` (enable K8s audit collection, default: false), `ROX_K8S_AUDIT_LOG_PATH` (audit log file path, default: `/var/log/kube-apiserver/audit.log`).

## Recent Changes

Jira ROX-33041 added audit events for internal token generation including M2M token exchange and service account token generation. ROX-20285 added source IP to audit messages with support for proxied requests via X-Forwarded-For. ROX-20288 added audit log without permissions privacy option via `ROX_AUDIT_LOG_WITHOUT_PERMISSIONS`.

## Use Cases

**Compliance**: SOC 2 audit trail, PCI-DSS sensitive config access, HIPAA policy access tracking, GDPR data modification tracking.

**Security Operations**: Detect unauthorized access, identify privilege escalation, track configuration changes, investigate incidents, monitor API token usage.

**Troubleshooting**: Debug authorization failures, track policy changes over time, identify configuration drift source, audit user actions during incidents.

## Best Practices

**Notifiers**: Configure multiple for redundancy, send to SIEM (Splunk/Elasticsearch) for analysis, archive for compliance (7+ years), alert on critical events (auth failures, policy deletions), test regularly.

**Log Management**: Enable `ROX_AUDIT_LOG_WITHOUT_PERMISSIONS` if permissions sensitive, use source IP for geographic analysis and anomaly detection, configure log rotation in external systems, define retention based on compliance requirements, restrict access to authorized personnel.

**K8s Audit Logs**: Collection runs on control-plane nodes only, plan storage for large audit logs, monitor impact on master node I/O, ensure state persistence to avoid duplicate processing.

## Troubleshooting

**Audit Messages Not Appearing**: Check notifier enabled for audit (`kubectl logs -n stackrox deploy/central | grep "audit notifier"`), verify operations are mutating (CREATE/UPDATE/DELETE), read-only operations not audited, service-to-service calls filtered by default, check notifier health status in UI, review Central logs for processing errors.

**K8s Audit Logs Not Collected**: Verify compliance pod on master (`kubectl get pods -n stackrox -l app=compliance -o wide` should show pods on control-plane nodes), check audit log path correct (default `/var/log/kube-apiserver/audit.log`, varies by K8s distribution), ensure compliance container can read audit log (file mounted and readable), verify collection requested in Sensor configuration.

**Missing Source IPs**: If using reverse proxy ensure X-Forwarded-For set, source IP should be peer address for direct connections, service-to-service calls may not have meaningful source IP.

## Performance

**Central Audit**: Async processing (no request blocking), notifier failures don't block requests, separate goroutine per message send, minimal memory usage (messages not buffered).

**Compliance Collection**: Tailing audit logs adds disk I/O on master nodes, K8s audit logs can be very large (GB/day), file position tracked to avoid re-reading, events streamed to Central (consider volume).
