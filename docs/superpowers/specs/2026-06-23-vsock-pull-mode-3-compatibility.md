# VSOCK Pull Mode — 3. Compatibility

**Parent design:** [Production Design v2](2026-06-23-vsock-pull-mode-production-design.md)  
**Previous:** [2. Protocol](2026-06-23-vsock-pull-mode-2-protocol.md)  
**Next:** [4. TLS](2026-06-23-vsock-pull-mode-4-tls.md)  
**Audience:** PR reviewers — how version mismatches are handled

---

## How the protocol handles versioning

The protocol has no version field. Instead, both sides advertise what they support
using two separate, complementary mechanisms:

| Direction | Field | Example | Purpose |
|-----------|-------|---------|---------|
| Sensor → Agent | `RequestMeta.capabilities` | `["report_v1"]` | What response **formats** Sensor understands |
| Agent → Sensor | `ResponseMeta.supported_methods` | `["get_report"]` | What request **methods** agent accepts |

These are **not** the same concept:

- **`capabilities`** controls the *format/version* of the response. The agent reads
  Sensor's capabilities to decide *how* to respond (e.g., which fields to include,
  which encoding to use).
- **`supported_methods`** controls *what methods* Sensor can call. Sensor reads this
  to decide *what* to request from the agent.

### Why two fields, not one?

Consider the evolution case: the `get_report` method stays the same, but the response
format changes (e.g., adding compressed reports). Sensor sends `capabilities: ["report_v1",
"report_v2"]`, still calls `get_report`. The agent sees `report_v2` in capabilities and
responds with the new format. The method name doesn't change — only the format does.

If we versioned methods instead (e.g., `get_report_v1`, `get_report_v2`), every format
change would require a new method, a new proto `oneof` case, and duplicated dispatch
logic on the agent. Splitting format negotiation into `capabilities` keeps the protocol
clean.

---

## Scenario 1: Agent is older than Sensor

A newer Sensor (e.g. 4.9) talks to an older agent (e.g. 4.7). Two things can happen:

### 1a. Sensor sends a method the old agent doesn't know

Sensor calls a new method (e.g. `trigger_rescan`) that the old agent doesn't support.
The agent doesn't recognize the method in the request's `oneof` and responds with an error:

```json
{
  "meta": {
    "agent_version": "4.7.0-12-gabcdef",
    "report_generation": 3,
    "supported_methods": ["get_report"],
    "facts": { "detected_os": "RHEL", "os_version": "9.4", "...": "..." }
  },
  "error": {
    "code": "ERROR_CODE_UNKNOWN_METHOD",
    "message": "unknown or unset method"
  }
}
```

**What Sensor does:** Logs a warning, records an `unknown_method` metric. Reads
`supported_methods` from the response and falls back to methods the agent supports
(e.g., continues using `get_report`). A newer Sensor can safely attempt new methods
against a fleet of mixed-version agents — old agents say "I don't know that method"
and Sensor adapts.

### 1b. Sensor sends a capability the old agent doesn't know

Sensor sends `capabilities: ["report_v1", "report_v2"]` but the old agent only
understands `report_v1`:

```json
// Sensor request:
{
  "meta": {
    "request_id": "...",
    "capabilities": ["report_v1", "report_v2"],
    "facts": {}
  },
  "get_report": { "if_newer_than_generation": 0 }
}

// Agent response (old agent, only knows report_v1):
{
  "meta": {
    "agent_version": "4.7.0-12-gabcdef",
    "report_generation": 1,
    "supported_methods": ["get_report"],
    "facts": { "detected_os": "RHEL", "os_version": "9.4", "...": "..." }
  },
  "get_report": {
    "index_report": { "...": "(v1 format, ~450 KiB)" },
    "unchanged": false
  }
}
```

**No error.** The agent ignores the unknown `report_v2` capability and responds in the
v1 format it knows. Sensor receives a valid v1 response and processes it normally.

**Challenge: how does Sensor know which format it got back?** There is currently
nothing in the response that explicitly says "this is v1 format". Today there is
only one format, so this works. But if a second format is introduced, Sensor needs
to distinguish them.

**Solution: use a new `oneof` result case.** When a v2 format is needed, add a
`GetReportV2Response` as a new `oneof` case in `VMServiceResponse`:

```protobuf
message VMServiceResponse {
  ResponseMeta meta = 1;
  oneof result {
    GetReportResponse get_report = 2;     // v1 format
    ErrorResponse error = 3;
    GetReportV2Response get_report_v2 = 4; // future v2 format
  }
}
```

The proto `oneof` itself acts as a type discriminator — Sensor's switch statement
already knows which type it received. No additional format field needed.

**Why this works with protobuf:** Protobuf `oneof` is wire-compatible as long as
field numbers are stable. If one side has a superset of the `oneof` cases, it can
still parse messages from the other side that only uses a subset:

**Old agent (knows fields 2, 3) → New Sensor (knows fields 2, 3, 4):**

```
Agent's proto:                    Sensor's proto:
oneof result {                    oneof result {
  GetReportResponse = 2;  ───▶     GetReportResponse = 2;    ✅ matches
  ErrorResponse = 3;               ErrorResponse = 3;
}                                   GetReportV2Response = 4;  (never received, that's fine)
                                  }
```

Agent sends field 2 (`get_report`). Sensor matches field 2 → `GetReportResponse`.
Field 4 is simply absent from the wire data. Works perfectly.

**New agent (knows fields 2, 3, 4) → Old Sensor (knows fields 2, 3):**

```
Agent's proto:                    Sensor's proto:
oneof result {                    oneof result {
  GetReportResponse = 2;  ───▶     GetReportResponse = 2;    ✅ matches
  ErrorResponse = 3;               ErrorResponse = 3;
  GetReportV2Response = 4; ───▶    (unknown field 4)          ⚠️ oneof unset
}                                 }
```

If agent sends field 2 (`get_report`) → old Sensor matches it, works fine.
If agent sends field 4 (`get_report_v2`) → old Sensor doesn't know field 4, the
`oneof` appears unset, `resp.GetResult()` returns `nil`, hits the `default` case
in the switch, returns `"unexpected response type"` — handled gracefully as a
read error, retry next cycle. No crash, no data corruption.

**The contract: both sides must understand all formats, old and new.**

When a new format is introduced, both the agent and Sensor must be able to produce
and consume all existing formats plus the new one. This guarantees that regardless
of which side is newer, there is always at least one common format they both understand.

The agent actively chooses the best format based on Sensor's `capabilities`:

| Sensor capabilities | Agent knows | Agent responds with |
|---------------------|-------------|---------------------|
| `["report_v1"]` | v1 only | `GetReportResponse` (v1) |
| `["report_v1"]` | v1 + v2 | `GetReportResponse` (v1) — downgrades to match Sensor |
| `["report_v1", "report_v2"]` | v1 only | `GetReportResponse` (v1) — best it can do |
| `["report_v1", "report_v2"]` | v1 + v2 | `GetReportV2Response` (v2) — picks best mutual format |

The agent never sends a format Sensor hasn't advertised. The `default` case in
Sensor's switch is a defensive fallback for buggy agents, not the expected path.

---

## Scenario 2: Sensor is older than Agent

An older Sensor (e.g. 4.7) talks to a newer agent (e.g. 4.9). Two things can happen:

### 2a. Agent supports new methods that old Sensor doesn't know about

The agent responds with `supported_methods: ["get_report", "trigger_rescan"]`, but the
old Sensor only knows how to call `get_report`:

```json
{
  "meta": {
    "agent_version": "4.9.0-1-g123456",
    "report_generation": 2,
    "supported_methods": ["get_report", "trigger_rescan"],
    "facts": { "detected_os": "RHEL", "os_version": "9.4", "...": "..." }
  },
  "get_report": {
    "index_report": { "...": "(~450 KiB)" },
    "unchanged": false
  }
}
```

**No error.** Old Sensor ignores the unknown `trigger_rescan` in `supported_methods`
and continues using `get_report`. The new agent happily serves `get_report` — it still
supports it.

### 2b. Agent could respond in a newer format, but Sensor only advertises old capabilities

Sensor sends `capabilities: ["report_v1"]`. The new agent supports both v1 and v2 but
sees that Sensor only understands v1, so it downgrades its response:

```json
// Sensor request (old Sensor):
{
  "meta": {
    "request_id": "...",
    "capabilities": ["report_v1"],
    "facts": {}
  },
  "get_report": { "if_newer_than_generation": 0 }
}

// Agent response (new agent, downgrades to v1):
{
  "meta": {
    "agent_version": "4.9.0-1-g123456",
    "report_generation": 1,
    "supported_methods": ["get_report", "trigger_rescan"],
    "facts": { "detected_os": "RHEL", "os_version": "9.4", "...": "..." }
  },
  "get_report": {
    "index_report": { "...": "(v1 format, not v2)" },
    "unchanged": false
  }
}
```

**No error.** The agent checks Sensor's capabilities and uses the best format Sensor
understands. Old Sensor gets a response it can parse.

### 2c. No common format or method — total incompatibility

In an extreme case, Sensor and agent are so far apart in versions that they share
no common capabilities or methods. For example:

- Sensor sends `capabilities: ["report_v3"]` and calls method `get_report_v3`
- Agent only knows `capabilities: ["report_v1"]` and method `get_report`

The agent receives an unknown method → responds with `UNKNOWN_METHOD`. Sensor reads
`supported_methods: ["get_report"]` but doesn't know how to call `get_report` (it was
removed in this hypothetical future Sensor).

**Result:** Communication fails completely. This should be reported clearly:

- **Agent side:** Logs `UNKNOWN_METHOD` error for the unknown method it received
- **Sensor side:** Logs an error indicating no compatible method is available for this
  VM. Records a metric (`unknown_method`). Skips this VM on subsequent cycles until
  the agent is upgraded.

**In practice this scenario is unlikely** — it requires skipping multiple major versions
and dropping support for old methods. The contract (both sides must support all old
formats + new ones) prevents this as long as upgrades happen within the supported
version window. If it does occur, the errors on both sides make the root cause clear:
the agent and Sensor versions are too far apart.

---

## Scenario 3: Same methods, but report format has changed

This is the most subtle scenario. Both sides agree on the method (`get_report`) and
the method is present in `supported_methods`, but the *structure* of the response
payload has changed.

### 3a. New optional fields added to the response (protobuf handles this)

A new agent adds fields to `GetReportResponse` or `ResponseMeta` (e.g., a new
`compression` field). Protobuf ignores unknown fields by default, so the old Sensor
simply doesn't see them:

```json
// New agent response (has extra field):
{
  "meta": {
    "agent_version": "4.9.0-1-g123456",
    "report_generation": 1,
    "supported_methods": ["get_report"],
    "facts": { "detected_os": "RHEL", "os_version": "9.4", "...": "..." }
  },
  "get_report": {
    "index_report": { "...": "(standard report)" },
    "unchanged": false,
    "compression": "zstd"
  }
}
```

**No error.** Old Sensor ignores the unknown `compression` field. This is protobuf's
built-in forward compatibility — adding optional fields is always safe.

### 3b. Entirely new response format (capabilities handle this)

If the report format changes fundamentally (e.g., different encoding, restructured
fields), the agent should gate it behind a capability check:

1. New agent checks if `report_v2` is in Sensor's capabilities
2. If yes → respond in v2 format
3. If no → respond in v1 format (downgrade)

This is the purpose of `capabilities` — the agent never sends a format the Sensor
can't understand.

### 3c. Semantic change in existing fields (the gap)

What if the *meaning* of an existing field changes? For example, if
`if_newer_than_generation` changes from strict equality to `>=` comparison.
Neither `capabilities` nor `supported_methods` can express this.

**Our approach:** Avoid semantic changes to existing fields. If behavior needs to
change, add a new method (e.g., `get_report_v2` as a new `oneof` case) and let
`UNKNOWN_METHOD` handle the fallback for old agents.

**Prevention:** Compatibility tests should be maintained that run an older agent
against a newer Sensor and vice versa, verifying that:
- Known methods produce parseable responses in both directions
- Unknown methods return `UNKNOWN_METHOD` gracefully
- Capability negotiation results in a format both sides can process
- No semantic regressions in existing fields (e.g., generation comparison behavior)

These tests act as a safety net against accidental semantic changes that would
otherwise go undetected until a mixed-version deployment in production.

---

## Compatibility matrix (summary)

| Scenario | Mechanism | Error? | Data loss? |
|----------|-----------|--------|------------|
| Old agent, new Sensor calls unknown method | `UNKNOWN_METHOD` error + `supported_methods` fallback | Yes (graceful) | No |
| Old agent, new Sensor sends unknown capability | Agent ignores, responds in known format | No | No |
| New agent, old Sensor doesn't know new methods | Sensor ignores unknown `supported_methods` | No | No |
| New agent, old Sensor only has old capabilities | Agent downgrades response format | No | No |
| New optional proto fields added | Protobuf ignores unknown fields | No | No |
| New response format (fundamental change) | `capabilities` negotiation | No | No |
| Semantic change to existing fields | Not supported — use new method instead | N/A | N/A |
| Agent restart (generation resets) | Strict equality catches mismatch | No | No |
