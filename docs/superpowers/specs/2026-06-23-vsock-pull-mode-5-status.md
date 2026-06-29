# VSOCK Pull Mode — 5. Status & Follow-ups

**Parent design:** [Production Design v2](2026-06-23-vsock-pull-mode-production-design.md)  
**Previous:** [4. TLS](2026-06-23-vsock-pull-mode-4-tls.md)  
**Audience:** project planners, future contributors — what's done, what's not, what's next

---

## 1. Current status

| Area | Status | Notes |
|------|--------|-------|
| Wire protocol (protobuf envelope, framing) | Done | Shared `pkg/vsockframing`, proto in `internalapi/virtualmachine/v1` |
| roxagent `serve` mode | Done | Push code removed; serve-only deployment |
| Sensor VMScraper | Done | Standalone `SensorComponent`, bounded concurrency, dedup, push suppression |
| KubeVirt TLS | Done | CA refresh, ephemeral server cert, client cert validation |
| WebSocket dialer (no kubecli) | Done | Plain `gorilla/websocket` + `k8s.io/client-go/rest` |
| Metrics (Prometheus) | Done | Histograms, counters, gauges — no per-VM labels |
| Report validation | Done | `reportcheck.IsViable()` — nil, zero packages, low count, oversized |
| RBAC (Helm) | Done | `virtualmachineinstances/vsock` permission in sensor RBAC template |
| Push suppression | Done | `PullActiveChecker` drops push reports for pull-scraped VMs |
| Unit tests | Done | Protocol handler, TLS handshake/rotation, client, scraper, report check |

---

## 2. Before release

| Item | Detail |
|------|--------|
| Set production poll interval | `defaultPollInterval` is `15s` for dev; change to `5 min` before release (`scraper.go`) |
| Quadlet image tag | `roxagent.container` has `Image=...main:latest` with a TODO to pin to the release tag |

---

## 3. Testing gaps

### E2E tests

Existing VM E2E tests (`qa-tests-backend/`) cover the push path but need to be
adapted and extended for pull mode:

- **Adapt existing tests:** update VM reconciliation tests to work with pull-mode
  Sensor (reports arrive via VMScraper, not relay push)
- **New pull-path test:** deploy a VM with roxagent in serve mode, wait for
  Sensor to scrape it, verify the report reaches Central and is visible in the
  API (end-to-end happy path)
- **Push suppression test:** run a VM reachable via both push and pull; verify
  only pull-mode reports are forwarded to Central once VMScraper is active
- **TLS failure E2E:** verify the agent starts cleanly with a valid KubeVirt CA
  and rejects connections without valid client certs (may require a custom
  virt-handler configuration or test harness)

### Compatibility tests

The protocol uses bidirectional capabilities (no version field), so we need to
verify behavior when Sensor and roxagent are at different versions:

| Scenario | Expected behavior | How to test |
|----------|-------------------|-------------|
| Old agent + New Sensor | Agent ignores unknown capabilities; advertises fewer methods; Sensor handles gracefully | Deploy old agent image with new Sensor |
| New agent + Old Sensor | Agent downgrades response; Sensor ignores unknown methods | Deploy new agent image with old Sensor |
| Agent receives unknown method | Returns `ErrorResponse{code: UNKNOWN_METHOD}` | Unit test exists; E2E: send crafted request |
| Sensor receives unknown fields in response | Proto ignores unknown fields (forward compat) | Unit test with extended proto |
| Agent restart mid-scrape | Generation resets to 1; Sensor re-fetches full report | Kill agent during scrape cycle |

**Recommended approach:** build a compatibility test matrix that deploys specific
agent and Sensor image versions (e.g., current release vs. next release). The capability-
based protocol should handle mismatches gracefully, but this needs to be proven
with real deployments, not just unit tests.

### Integration tests

Missing integration coverage for:

- `MultiDialer` — websocket dial against a real or mock KubeVirt API (currently
  untested; the dialer has no test file)
- CA fetch over real VSOCK (`FetchKubeVirtCA` is unit-tested with a mock fetch
  function but not against a live virt-handler)

### What is tested (unit level)

| Package | Test file | Coverage |
|---------|-----------|----------|
| `vsockserver` | `protocol_test.go` | Protocol handler: dispatch, error responses, capabilities |
| `vsockserver` | `tls_test.go` | CA refresh, rotation, failure resilience, TLS handshake, wrong CA rejection |
| `vsockclient` | `client_test.go` | Request/response roundtrip, error mapping |
| `vmscraper` | `scraper_test.go` | Poll cycle, dedup, mandatory refresh, push suppression, stale pruning |
| `reportcheck` | `check.go` | Report viability (nil, zero pkgs, low count, oversized) |

---

## 4. Follow-up work

### Reactive scanning ([ROX-34984](https://redhat.atlassian.net/browse/ROX-34984))

The agent currently rescans on a fixed 4h interval. Reactive scanning would have
the agent detect filesystem changes (e.g. DNF transactions) and rescan
immediately, incrementing the generation counter. Sensor would poll more
frequently to pick up changes faster. The protocol already supports this — the
generation counter and `unchanged` response make frequent polling cheap.


### Avoid redundant re-dial on mandatory refresh ([ROX-35362](https://redhat.atlassian.net/browse/ROX-35362))

When the mandatory 4h refresh is due, Sensor currently makes **two** connections
per VM in a single cycle: the first returns `unchanged`, then Sensor re-dials
with `ifNewerThan=0` to force the full report. Since Sensor can check
`lastForwardedAt > 4h` before dialing, it should send `ifNewerThan=0` on the
first connection directly, eliminating the redundant round trip.

### roxagent installation and upgrade path

Currently roxagent is deployed manually via Quadlet files (`install.sh` +
systemd units). Customers need a supported way to install and upgrade the agent
on their VMs — including initial deployment, image version pinning, and rolling
upgrades across a fleet of VMs without downtime. This may involve packaging
(RPM), an operator-driven approach, or integration with existing VM provisioning
tooling.

### Relay removal

The push-mode relay in Compliance and the gRPC service in Sensor are kept during
the transition period for old agents. Target: remove in the next release after
pull mode is GA. The feature is tech preview — no backward compatibility
guarantees between tech preview and GA.

### Known TODOs in code

| Location | TODO |
|----------|------|
| `central/virtualmachine/service/service_impl.go` | Handle specific error cases with proper error codes (duplicate ID) |
| `central/virtualmachine/datastore/datastore_impl.go` | Move default sorting over multiple columns to store (ROX-31024) |
| `central/virtualmachine/datastore/datastore_impl_test.go` | Test concurrent writes |
| `sensor/common/virtualmachine/index/handler_impl.go` | Send retry message via sensor relay when VM not found |
