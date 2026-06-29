# VSOCK Pull Mode — Production Design (v2)

**Epic:** [ROX-34590](https://redhat.atlassian.net/browse/ROX-34590) — Change VSOCK mode from push to pull  
**POC:** [PR #21351](https://github.com/stackrox/stackrox/pull/21351) — validated end-to-end pull path  
**Previous context:** [v1 spec](2026-06-18-vsock-pull-mode-context.md) | [TLS analysis](2026-06-23-vsock-tls-options-comparison.md)  
**Focused docs:** [1. Architecture](2026-06-23-vsock-pull-mode-1-architecture.md) | [2. Protocol](2026-06-23-vsock-pull-mode-2-protocol.md) | [3. Compatibility](2026-06-23-vsock-pull-mode-3-compatibility.md) | [4. TLS](2026-06-23-vsock-pull-mode-4-tls.md) | [5. Status](2026-06-23-vsock-pull-mode-5-status.md)  
**Date:** 2026-06-23

---

## 1. Problem

ACS uses VSOCK to transmit security telemetry from VM-based workloads (roxagent inside
VMs) to the host. The current push model (roxagent → relay → Sensor) breaks when Linux
kernel VSOCK namespace isolation is enabled — VMs can no longer reach the relay across
namespace boundaries.

## 2. Solution: Pull via KubeVirt VSOCK API

Reverse the data flow. Sensor connects INTO the VM using the KubeVirt VSOCK subresource
API. roxagent listens on a VSOCK port and responds with scan data on demand.

```
Sensor (VMScraper, 5-min timer)
  → websocket dial to KubeVirt subresource API
  → virt-api → virt-handler → VSOCK into VM (TLS)
  → roxagent serve (protocol handler)
  → VMServiceResponse with cached report
  → Sensor validates, reads, forwards via existing handler
  → Central (same SensorEvent as push path)
```

## 3. Scope

**In scope:**
- Production-quality pull path with KubeVirt TLS
- Extensible wire protocol (protobuf envelope, multi-method, bidirectional capabilities)
- roxagent `serve` mode as the only deployed mode (push code removed from agent)
- Sensor VMScraper component with full observability
- Generation-based deduplication with mandatory 4h refresh to Central
- Bounded scraping concurrency with configurable parallelism
- Push suppression for VMs actively scraped via pull mode

**Out of scope (separate tickets):**
- Relay removal from Compliance and Sensor (kept for old agents during transition)
- Reactive scanning (ROX-34984 — agent-side change detection + more frequent polling)

## 4. Wire Protocol

### 4.1 Framing

```
[4 bytes: payload length, big-endian uint32][payload: protobuf message]
```

Framing is implemented in a shared package (`pkg/vsockframing`) used by both Sensor
and roxagent. `ReadFrame` enforces a caller-provided `maxSize` to reject oversized payloads
before allocating memory.

Connection lifecycle:
1. Sensor dials VM (websocket via KubeVirt API)
2. Sensor sends framed `VMServiceRequest`
3. roxagent sends framed `VMServiceResponse`
4. Connection closes

### 4.2 Proto Definitions

File: `proto/internalapi/virtualmachine/v1/vm_service.proto`

```protobuf
syntax = "proto3";
package virtualmachine.v1;

import "google/protobuf/timestamp.proto";
import "internalapi/scanner/v4/index_report.proto";

// --- Envelope ---

message VMServiceRequest {
  RequestMeta meta = 1;
  oneof method {
    GetReportRequest get_report = 2;
  }
}

message VMServiceResponse {
  ResponseMeta meta = 1;
  oneof result {
    GetReportResponse get_report = 2;
    ErrorResponse error = 3;
  }
}

// --- Metadata ---

message RequestMeta {
  string request_id = 1;
  repeated string capabilities = 2;
  map<string, string> facts = 3;
}

message ResponseMeta {
  string agent_version = 1;
  google.protobuf.Timestamp report_generated_at = 2;
  uint32 report_generation = 3;
  repeated string supported_methods = 4;
  map<string, string> facts = 5;
}

// --- Methods ---

message GetReportRequest {
  uint32 if_newer_than_generation = 1;
}

message GetReportResponse {
  scanner.v4.IndexReport index_report = 1;
  bool unchanged = 2;
}

// --- Errors ---

message ErrorResponse {
  ErrorCode code = 1;
  string message = 2;
  map<string, string> details = 3;
}

enum ErrorCode {
  ERROR_CODE_UNSPECIFIED = 0;
  ERROR_CODE_UNKNOWN_METHOD = 1;
  ERROR_CODE_NOT_READY = 2;
  ERROR_CODE_INTERNAL = 3;
}
```

### 4.3 Bidirectional Capabilities

Feature negotiation is capability-based and bidirectional. No protocol version field.

- **Wire format** is fixed: `[4-byte length][protobuf]`. This never changes.
- **Both sides advertise what they support:**
  - `RequestMeta.capabilities` — what Sensor understands (e.g. `"report_v1"`)
  - `ResponseMeta.supported_methods` — what methods the agent accepts (e.g. `"get_report"`)

**How it works:**

1. Sensor sends `GetReportRequest` with `RequestMeta{capabilities: ["report_v1"]}`
2. Agent reads Sensor's capabilities → knows what response format to send
3. Agent responds with `ResponseMeta{supported_methods: ["get_report"]}` + report
4. Sensor caches the agent's supported methods for future requests

**Compatibility:**
- Old agent + New Sensor: agent ignores unknown capabilities; advertises fewer methods
- Old Sensor + New agent: agent downgrades response format; Sensor ignores unknown methods
- Unknown method received by agent: returns `ErrorResponse{code: UNKNOWN_METHOD}`

**Adding new features:**
- New response format: Sensor adds to `capabilities`; agent checks before using
- New request method: agent adds to `supported_methods`; Sensor checks before sending
- No version bumps, no converters, no deprecation ceremonies

## 5. Security

### 5.1 KubeVirt TLS

Transport encryption and caller authentication via KubeVirt's built-in TLS:

1. Sensor passes `useTLS: true` in the VSOCK subresource request
2. virt-handler wraps the VSOCK connection in TLS (as client), presenting a cert
   signed by KubeVirt's internal CA (CN: `kubevirt.io:system:client:...`)
3. roxagent validates the client cert:
   - Fetches KubeVirt CA via gRPC `System.CABundle` RPC on VSOCK CID 2, port 1
     (virt-handler's CA distribution service). Uses a raw-bytes gRPC codec to avoid
     importing `kubevirt.io/client-go` (glog `-v` flag conflict)
   - Periodically refreshes the CA (configurable, default 1h) for seamless rotation
   - TLS config uses `RequireAndVerifyClientCert` with `GetConfigForClient` callback
     that reads the latest CA pool on each handshake — no restart needed for rotation
4. roxagent presents an ephemeral self-signed ECDSA certificate (regenerated on each
   start, never persisted). No party validates this cert — not its identity, signature,
   or expiry. It exists solely to satisfy TLS protocol requirements (key exchange needs
   a certificate). Authentication flows in the opposite direction (agent verifies
   virt-handler's client cert)
5. TLS is mandatory: sensor always dials with TLS, and roxagent fails to start if the
   initial KubeVirt CA fetch fails. There is no plaintext fallback.
6. RBAC: Sensor's ServiceAccount needs `virtualmachineinstances/vsock` permission
   on `subresources.kubevirt.io`

### 5.2 Identity

In pull mode, Sensor explicitly dials a specific VMI by namespace/name — VM identity
is established by the KubeVirt API routing (same trust as `kubectl exec`).
`vsock_cid` is not transmitted in the response — Sensor knows who it dialed.

### 5.3 RBAC

```yaml
- apiGroups: ["subresources.kubevirt.io"]
  resources: ["virtualmachineinstances/vsock"]
  verbs: ["get"]
```

## 6. Component Architecture

### 6.1 Sensor Side

```
┌─────────────────────────────────────────────────────────────┐
│ Sensor (runs in K8s management cluster)                     │
│                                                             │
│  ┌──────────────────┐    ┌─────────────┐    ┌───────────┐   │
│  │ VMScraper        │───▶│ VMI Store   │    │ Handler   │   │
│  │ (SensorComponent)│    │(ListRunning)│    │ (Send→C)  │───┼──▶ Central
│  │ 5-min timer      │    └─────────────┘    └─────▲─────┘   │
│  │ + immediate poll │                              │        │
│  └──────┬───────────┘                              │ Send() │
│         │ Dial(ns, name, port, tls=true)           │        │
│         ▼ (bounded: errgroup, limit=20)            │        │
│  ┌──────────────────┐                              │        │
│  │ MultiDialer      │── websocket ──▶ K8s API ─────┼──┐     │
│  │ (plain websocket,│                              │  │     │
│  │  no kubecli dep) │                              │  │     │
│  └──────┬───────────┘                              │  │     │
│         │ VMServiceResponse                        │  │     │
│         ▼                                          │  │     │
│  ┌──────────────────┐                              │  │     │
│  │ Protocol Client  │── capabilities + dedup ──────┘  │     │
│  └──────┬───────────┘                                 │     │
│         │                                             │     │
│  ┌──────▼───────────┐                                 │     │
│  │ ReportCheck      │ (diagnostics — logs warnings)   │     │
│  └──────────────────┘                                 │     │
│                                                       │     │
│  ┌──────────────────────────────────────────────────┐ │     │
│  │ Push path (gRPC service + relay) — old agents    │ │     │
│  │ PullActiveChecker suppresses push for scraped VMs│ │     │
│  └──────────────────────────────────────────────────┘ │     │
└───────────────────────────────────────────────────────┼─────┘
                                                        │
            KubeVirt VSOCK subresource (TLS)            │
            virt-api → virt-handler → VSOCK             │
                                                        │
┌───────────────────────────────────────────────────────┼────┐
│ VM (roxagent serve — runs inside each virtual machine)│    │
│                                                       │    │
│  ┌──────────────────┐                                 │    │
│  │ VSOCK Listener   │◀────────────────────────────────┘    │
│  │ port 818, TLS    │                                      │
│  └────────┬─────────┘                                      │
│           │                                                │
│  ┌────────▼─────────┐    ┌──────────────┐                  │
│  │ Protocol Handler │───▶│ Report Cache │                  │
│  │ (read req,       │    │ (atomic snap)│                  │
│  │  write resp)     │    └──────┬───────┘                  │
│  └──────────────────┘           │ rescan (4h)              │
│                        ┌────────▼───────┐                  │
│                        │ Scanner        │                  │
│                        │ (NodeIndexer)  │                  │
│                        └────────────────┘                  │
└────────────────────────────────────────────────────────────┘
```

**VMScraper** is a standalone `common.SensorComponent`:
- Gated behind `features.VirtualMachines.Enabled()`
- Polls immediately on start, then every 5 minutes (fixed interval; currently 15s for dev/testing)
- Iterates `VMIStore.ListRunning()`, dials each VM concurrently via `errgroup`
  with configurable parallelism (default 20, env `ROX_VIRTUAL_MACHINES_SCRAPER_CONCURRENCY`)
- Per-VM timeout: 30 seconds (context deadline propagated to websocket read/write)
- Calls `handler.Send(ctx, indexReport)` — same path as push mode
- Errors: log + skip + metric; next cycle retries automatically
- **Deduplication with mandatory refresh:** Sensor tracks `report_generation` and
  `last_forwarded_at` per VM. Rules:
  - Generation changed → forward immediately (new data)
  - Generation unchanged, last forward >4h ago → re-dial with `ifNewerThan=0` to
    force a full report, then forward (Central must receive at least one report per 4h
    so Scanner can match against new vulnerability definitions)
  - Generation unchanged, last forward <4h ago → skip
  - Generation comparison uses strict equality (not `>=`) so that after an agent restart
    (when generation resets to 1), a sensor holding a higher generation from the previous
    instance receives the full report instead of a false "unchanged"
- **Report validation:** Before forwarding, `reportcheck.IsViable()` checks for nil
  reports, zero packages, suspiciously low package counts (<5), and oversized reports
  (>2 MiB warning). Non-viable reports are not forwarded.
- **Stale state pruning:** After each cycle, vmState entries for VMs no longer in the
  running set are removed to prevent unbounded memory growth.
- **Isolation:** VMScraper must not share state, goroutines, or lifecycle with the
  push-mode gRPC service. They are independent components — one must never block,
  crash, or degrade the other.
- **Pull preference / push suppression:** If the same VM reports via both push and pull
  (during transition), pull-mode reports take precedence. VMScraper maintains an
  `activeVMs` set updated atomically at the end of each cycle — the old set remains in
  effect during scraping to prevent a suppression gap. The push-mode gRPC service
  (`vmIndex.Service`) checks `PullActiveChecker.IsActivelyScraped(vsockCID)` and drops
  push reports for actively-scraped VMs. Both `namespace/name` keys and CID strings are
  registered for matching.

**MultiDialer** (from POC, production-ready):
- Plain websocket using `k8s.io/client-go/rest` + `gorilla/websocket`
- Cannot use `kubevirt.io/client-go` (glog `-v` flag conflict — upstream issue)
- Uses Sensor's existing REST config for TLS and auth
- Context deadline propagated to websocket read/write deadlines after dial
- wsStream adapter handles cross-message-boundary reads, treats websocket close as EOF
- WebSocket read limit: configurable via `ROX_VIRTUAL_MACHINES_PULL_MAX_RESPONSE_SIZE_KB`
  (default 16 MiB, matching push-mode limit)
- WebSocket I/O buffer: 1 MiB (not a message limit; larger messages are chunked internally)

### 6.2 roxagent Side

```
┌─────────────────────────────┐
│ VM (roxagent serve)         │
│                             │
│  ┌───────────────────────┐  │
│  │ VSOCK Listener        │  │
│  │ port 818, TLS (KV CA) │  │
│  │ semaphore: max 1 conn │  │
│  └────────┬──────────────┘  │
│           │ accept           │
│  ┌────────▼──────────────┐  │
│  │ Protocol Handler      │  │
│  │ - read VMServiceReq   │  │
│  │ - dispatch by method  │  │
│  │ - check capabilities  │  │
│  │ - write VMServiceResp │  │
│  │ conn deadline: 30s    │  │
│  └────────┬──────────────┘  │
│           │                  │
│  ┌────────▼──────────────┐  │
│  │ Report Cache          │  │
│  │ atomic.Pointer to     │  │
│  │ immutable snapshot    │  │
│  │ (report, generation,  │  │
│  │  generatedAt, facts)  │  │
│  └────────┬──────────────┘  │
│           │ rescan (4h)      │
│  ┌────────▼──────────────┐  │
│  │ Scanner (NodeIndexer) │  │
│  └───────────────────────┘  │
└─────────────────────────────┘
```

**roxagent changes:**
- Push mode code is **removed** (vsock client, push timer, relay dependency — all deleted)
- `serve` is the only subcommand deployed
- Protocol handler: reads framed request (max 1 MiB), dispatches by method, writes framed response
- TLS server: accepts connections, validates KubeVirt client cert (CA refreshed hourly via
  gRPC on CID 2 / port 1), presents ephemeral self-signed ECDSA cert
- **Single connection enforcement:** semaphore limits to 1 concurrent connection; additional
  connections are rejected immediately. Each accepted connection has a 30-second deadline.
- **Graceful shutdown:** WaitGroup drains in-flight connections when context is cancelled
- Report cache: `atomic.Pointer[reportSnapshot]` — immutable snapshot struct holding the
  report, generation counter, generatedAt timestamp, and facts. Single writer (rescan loop),
  multiple readers (connection handlers). `SetReport` atomically stores a new snapshot with
  incremented generation; readers never observe partial (new report + stale facts) state.
- Rescan: configurable periodic timer (default 4h, `--rescan-interval` flag). Scans on
  startup before accepting connections. Facts (detected OS, version, activation status,
  DNF metadata status) are rediscovered on each rescan.
- Agent version: injected via `-ldflags` at build time

## 7. Error Handling

| Scenario | Agent behavior | Sensor behavior |
|----------|---------------|-----------------|
| Agent still scanning (no report cached) | `ErrorResponse{code: NOT_READY}` | Log debug, skip, retry next cycle |
| Unknown method sent by Sensor | `ErrorResponse{code: UNKNOWN_METHOD}` | Log warning, don't send that method to this agent again |
| Malformed request (unmarshal failure) | `ErrorResponse{code: INTERNAL}` | N/A |
| Plaintext connection to TLS listener | Detect TLS record error, log warning, close | N/A |
| VM not running | N/A (dial fails) | Log warning, skip, metric |
| VSOCK not enabled on VMI | N/A (dial fails) | Log warning, skip, metric |
| Agent crashed / port not bound | N/A (connection refused) | Log warning, skip, metric |
| Per-VM timeout (30s) | N/A | Cancel context, log warning, metric (`timeout`) |
| Report too large | N/A | Frame read fails at maxSize limit, log error |
| Report not viable (nil, zero packages) | N/A | Log warning, skip, metric (`invalid_report`) |
| Connection closed / EOF | N/A | Log debug (agent may be restarting), metric (`read_error`) |
| Concurrent connection attempt | Reject (semaphore full), close | Re-dials next cycle |

## 8. Observability

### 8.1 Timing (Prometheus histograms)

| Metric | Description |
|--------|-------------|
| `vsock_pull_dial_duration_seconds` | Time to establish websocket connection per VM |
| `vsock_pull_read_duration_seconds` | Time to receive full response from agent per VM |
| `vsock_pull_total_duration_seconds` | End-to-end per VM (dial + read + send to Central) |
| `vsock_pull_cycle_duration_seconds` | Full poll cycle across all VMs |

### 8.2 Sizes (Prometheus histograms)

| Metric | Description |
|--------|-------------|
| `vsock_pull_report_bytes` | Response payload size in bytes |
| `vsock_pull_report_packages` | Package count per report |

### 8.3 Counters

| Metric | Labels | Description |
|--------|--------|-------------|
| `vsock_pull_requests_total` | `status` | Per-VM attempts: `success`, `unchanged`, `dial_error`, `read_error`, `invalid_report`, `send_error`, `not_ready`, `unknown_method`, `timeout` |
| `vsock_pull_cycles_total` | | Poll cycles executed |

### 8.4 Gauges

| Metric | Description |
|--------|-------------|
| `vsock_pull_vms_in_cycle` | Number of running VMs in the last poll set |

### 8.5 Cardinality

**No per-VM labels** — aggregate metrics only. Customers can run up to 10,000 VMs;
per-VM labels would cause cardinality explosion. Per-VM debugging uses structured
Sensor logs (with namespace/name).

## 9. Configuration

### 9.1 Sensor Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ROX_VIRTUAL_MACHINES_SCRAPER_CONCURRENCY` | 20 | Max VMs scraped concurrently per cycle |
| `ROX_VIRTUAL_MACHINES_PULL_MAX_RESPONSE_SIZE_KB` | 16384 (16 MiB) | Max response size accepted from agent |
| `ROX_VIRTUAL_MACHINES_VSOCK_PORT` | 818 | VSOCK port to dial on VMs |

### 9.2 roxagent CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | 818 | VSOCK port to listen on |
| `--host-path` | `/` | Root filesystem path for indexing |
| `--rescan-interval` | 4h | Interval between rescans |
| `--repo-cpe-url` | (built-in URL) | Repository to CPE mapping URL |

### 9.3 Hardcoded Constants

| Constant | Value | Location |
|----------|-------|----------|
| Poll interval | 5 min (currently 15s for dev) | VMScraper |
| Per-VM timeout | 30s | VMScraper |
| Mandatory refresh interval | 4h | VMScraper |
| CA refresh interval | 1h | CARefresher |
| Max concurrent agent connections | 1 | Server |
| Agent connection deadline | 30s | Server |
| Max request size (agent side) | 1 MiB | Handler |
| WS I/O buffer | 1 MiB | MultiDialer |

## 10. Backward Compatibility

### 10.1 Push Path During Transition

| Component | Push code | Reason |
|-----------|-----------|--------|
| roxagent | **Removed** | Serve-only in next release |
| Compliance (relay) | **Kept** | Old agents on existing VMs may still push |
| Sensor (gRPC service) | **Kept** | Receives reports forwarded by relay |
| Sensor (push suppression) | **Added** | `PullActiveChecker` drops push reports for pull-scraped VMs |
| Central | **Unchanged** | Same `SensorEvent_VirtualMachineIndexReport` from both paths |

### 10.2 Removal Timeline

The feature is tech preview — no backward compatibility guarantees between tech preview
and GA. Target: push-mode removal in the next release (pull-only GA). Transition period
for internal developers/testers only.

## 11. File Structure

| File | Responsibility |
|------|----------------|
| `proto/internalapi/virtualmachine/v1/vm_service.proto` | Protocol definitions |
| `pkg/vsockframing/framing.go` | Shared length-prefixed frame read/write |
| `sensor/common/virtualmachine/vmscraper/scraper.go` | VMScraper SensorComponent |
| `sensor/common/virtualmachine/vsockclient/client.go` | Protocol client (send request, read response) |
| `sensor/kubernetes/virtualmachine/vsockdialer/dialer.go` | Websocket dialer + wsStream adapter (KubeVirt API) |
| `sensor/common/virtualmachine/reportcheck/check.go` | Report viability diagnostics |
| `sensor/common/virtualmachine/metrics/metrics.go` | Prometheus metrics (push + pull) |
| `sensor/common/virtualmachine/index/service.go` | Push-mode gRPC service with `PullActiveChecker` |
| `compliance/virtualmachines/roxagent/cmd/serve.go` | roxagent serve subcommand + self-signed cert |
| `compliance/virtualmachines/roxagent/vsockserver/server.go` | VSOCK listener + TLS + single-conn enforcement |
| `compliance/virtualmachines/roxagent/vsockserver/protocol.go` | Protocol handler + `ReportCache` |
| `compliance/virtualmachines/roxagent/vsockserver/tls.go` | KubeVirt CA fetch (gRPC) + `CARefresher` |
| `compliance/virtualmachines/roxagent/discovery/discovery.go` | VM fact discovery (OS, activation, DNF) |
| `pkg/env/virtualmachine.go` | Environment variable definitions |
| `image/templates/helm/stackrox-secured-cluster/templates/sensor-rbac.yaml` | RBAC for VSOCK access |

## 12. Decisions Log

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | KubeVirt TLS | Zero infra overhead; same trust model as KubeVirt guest agents |
| 2 | Protobuf envelope protocol | Type-safe, extensible via oneof, consistent with codebase |
| 3 | Bidirectional capabilities (no protocol version) | Wire format is fixed; features discovered dynamically; no converters needed |
| 4 | roxagent push code removed | Simplify agent; relay stays for old agents during transition |
| 5 | Generation-based dedup + mandatory 4h refresh | Frequent polling catches changes quickly; dedup prevents spam; 4h refresh ensures Scanner can match new CVEs |
| 6 | Strict equality for generation comparison | Agent restart resets counter to 1; `>=` would falsely report "unchanged" when sensor holds a higher generation from the previous instance |
| 7 | Standalone SensorComponent (VMScraper) | Push is inbound gRPC, pull is outbound VSOCK — must not interfere |
| 8 | Plain websocket dialer (no kubecli) | `kubevirt.io/client-go` panics due to glog `-v` flag conflict |
| 9 | gRPC raw-bytes codec for CA fetch | Avoids importing kubevirt proto types (same glog conflict) |
| 10 | Full metrics from day 1 | Cheap; critical for customer issue resolution |
| 11 | No per-VM metric labels | Up to 10k VMs; cardinality explosion risk |
| 12 | Pull reports take precedence over push | During transition, prefer pull when both paths deliver for same VM |
| 13 | Single concurrent agent connection | Agent serves one Sensor poller; simplifies reasoning about concurrency |
| 14 | Shared framing package (`pkg/vsockframing`) | Both Sensor and roxagent need identical framing; DRY |
| 15 | Ephemeral self-signed agent cert | Satisfies TLS protocol requirement; no party validates it; auth flows in reverse (agent verifies virt-handler) |
| 16 | Immutable report snapshot via `atomic.Pointer` | Single-writer/multi-reader without locks; readers never see partial state |
| 17 | Bounded scraping concurrency (default 20) | Reduces cycle time for large VM counts without overwhelming network |
| 18 | Immediate poll on start | VMs don't wait a full interval before first scrape |

## 13. Future Work

- **Reactive scanning** ([ROX-34984](https://redhat.atlassian.net/browse/ROX-34984)): agent detects DNF changes, rescans, increments
  generation. Sensor polls more frequently. Generation counter + `unchanged`
  response makes this cheap. Separate ticket.
- **Avoid redundant re-dial on mandatory refresh** ([ROX-35362](https://redhat.atlassian.net/browse/ROX-35362)):
  when the 4h mandatory refresh is due, Sensor currently makes two connections
  per VM (first returns `unchanged`, then re-dials with `ifNewerThan=0`). Since
  Sensor can check `lastForwardedAt > 4h` before dialing, it should send
  `ifNewerThan=0` on the first connection directly, eliminating the redundant
  round trip. The `if_newer_than_generation` proto field itself is already
  implemented — Sensor sends the last known generation and the agent responds
  `unchanged` instead of re-serializing the full report.
