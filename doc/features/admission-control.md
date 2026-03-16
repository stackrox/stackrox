# Admission Control

**Primary Packages**: `sensor/admission-control`, `pkg/admissioncontrol`
**Binary**: `admission-control` (separate pod from Sensor)

## Overview

Validates Kubernetes resource creation/update requests via a ValidatingWebhookConfiguration before persistence to etcd. Provides pre-deployment policy enforcement with millisecond response times via in-memory policy cache, isolated deployment for crash protection, and independent scaling.

**Core Capabilities**: Block violating resources before deployment, fast policy decisions via cache, bypass annotations for emergencies, image scanning integration at admission time.

## Architecture

The admission controller runs at `sensor/admission-control/main.go` as an isolated pod exposing HTTPS on port 8443. API server calls `/admissioncontroller` with AdmissionReview JSON. The service at `sensor/admission-control/service/service.go` handles TLS endpoint and JSON marshaling.

**Manager** (`sensor/admission-control/manager/manager_impl.go`): Orchestrates policy evaluation with in-memory caches for policies (from Sensor), runtime class configs, and image scan results (200 MB LRU cache). Policy evaluation runs against converted deployment objects enriched with cached image data.

**Alert Sender** (`sensor/admission-control/alerts/alert_sender.go`): Queues violations and batch-sends to Sensor via gRPC every second, with retry on failure.

### Settings Synchronization

Three sources with precedence: (1) Sensor gRPC stream (authoritative), (2) File mount watch (Helm values), (3) ConfigMap watch (user overrides). Watcher at `sensor/admission-control/settingswatch/message_push.go` receives policies and cluster config via gRPC. ConfigMap watch at `sensor/admission-control/settingswatch/k8s_watch.go` monitors `admission-control-settings` for user changes. File mount watch at `sensor/admission-control/settingswatch/mount_watch.go` handles Helm-based deployments at `/run/secrets/stackrox.io/admission-control/`. Settings persistence at `sensor/admission-control/settingswatch/persister.go` caches to ConfigMap for faster startup.

### Request Flow

API server POST includes AdmissionReview with uid, kind, operation, and object spec. `serviceImpl.HandleValidate` extracts resource reference and delegates to manager. Manager converts to storage.Deployment, enriches with image scan cache hits, runs `policyDetector.DetectDeployment`, determines admission decision (block if enforcing violations exist), queues alerts, and returns AdmissionResponse. Alerts batch-send with max 100 per batch to Sensor.

**Policy Cache**: Manager stores policies slice with RWMutex for concurrent access at `managerImpl.policies` and `managerImpl.policiesLock`. Updates come from Sensor's policy stream. Image cache at `ImageCache` struct uses LRU with 200 MB default, 1-hour TTL, and explicit invalidation on scan updates. RuntimeClass manager applies default runtime class if pod doesn't specify one.

### Bypass Annotations

Emergency bypass via `admission.stackrox.io/break-glass: "ticket-12345"` annotation allows deployment to proceed while still generating alerts for audit trail. Implementation checks deployment annotations in `shouldBypass` function.

### Readiness and Health

Ready when: (1) settings received from Sensor or loaded from ConfigMap, (2) policy cache populated, (3) gRPC connection to Sensor established. Readiness check at `managerImpl.IsReady` verifies policies exist and initial resource sync complete. Endpoint at `GET /ready` on port 8443.

Graceful shutdown: SIGTERM marks not ready, waits 15 seconds for in-flight requests, flushes alert queue to Sensor, then exits. Implementation at `mainCmd` with grace period timer.

### Webhook Configuration

ValidatingWebhookConfiguration at `stackrox` with `policyeval.stackrox.io` webhook calling `service: admission-control, namespace: stackrox, path: /admissioncontroller`. Rules match CREATE/UPDATE on deployments, pods, replicasets, daemonsets, statefulsets. FailurePolicy defaults to Ignore (fail-open) to prevent cluster lockout. Timeout: 10 seconds.

## Performance

Typical latencies: policy evaluation 1-5 ms, image cache lookup <1 ms, total webhook 5-20 ms. Capacity: 1000+ requests/second per pod, scale horizontally. Fast path uses in-memory policy cache and LRU image cache. Slow path: image not in cache requires Sensor request, complex policy evaluation with regex/CVE checks.

## Code Locations

**Core**: `sensor/admission-control/main.go` (entry), `manager/manager_impl.go` (orchestration), `service/service.go` (HTTP endpoint), `alerts/alert_sender.go` (batching).

**Settings Sync**: `settingswatch/message_push.go` (gRPC), `settingswatch/k8s_watch.go` (ConfigMap), `settingswatch/mount_watch.go` (file), `settingswatch/persister.go` (cache).

**Shared**: `pkg/admissioncontrol/` has ConfigMap keys and conversion utilities.

## Environment Variables

- `ROX_SENSOR_ENDPOINT`: Sensor gRPC endpoint (default: `sensor.stackrox:9443`)
- `ROX_ADMISSION_CONTROL_LISTEN_ON_CREATION`: Enable on CREATE (default: true)
- `ROX_ADMISSION_CONTROL_LISTEN_ON_UPDATES`: Enable on UPDATE (default: false)
- `ROX_ADMISSION_CONTROL_LISTEN_ON_EVENTS`: Enable K8s event recording (default: false)

## Troubleshooting

**Blocks all resources**: Check policies loaded (`kubectl logs -n stackrox deploy/admission-control | grep "policies loaded"`), verify gRPC connection to Sensor.

**Not enforcing**: Check webhook config (`kubectl get validatingwebhookconfiguration stackrox -o yaml`), verify pod readiness, inspect settings ConfigMap.

**High latency**: Monitor image cache hit rate (`admission_control_image_cache_hit_rate` metric), check queue depth (`admission_control_alert_queue_size`).

**Debug mode**: `kubectl set env -n stackrox deploy/admission-control ROX_LOG_LEVEL=debug`

## Metrics

Prometheus on port 9090: `admission_control_request_count{allowed, resource}`, `admission_control_request_duration_seconds{resource}`, `admission_control_policy_violations{policy, resource}`, `admission_control_image_cache_hit_rate`, `admission_control_alert_queue_size`.
