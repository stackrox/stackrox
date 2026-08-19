# ACK Model for VM Index Reports

VM index reports flow directly from Sensor to Central; Compliance is not involved (see `compliance/ACK_MODEL.md` for the separate node-scanning ACK model, which does go through Compliance).

`VMScraper` pulls reports from each running VM's roxagent over VSOCK and forwards successful reports to Central through `vmIndex.Handler`, the same handler used by every other producer of `v1.IndexReport`s.

A short tick (`ROX_VIRTUAL_MACHINES_SCRAPER_TICK_INTERVAL`, default 10s) walks the per-VM schedule. Each VM is due at its own `nextAttemptAt`. Success and permanent failures reschedule at `ROX_VIRTUAL_MACHINES_SCRAPER_POLL_INTERVAL`. Retryable pull failures and NACKs share one exponential backoff (`ROX_VIRTUAL_MACHINES_SCRAPER_INITIAL_BACKOFF`, default 10s, doubled each time, capped at `min(poll interval, 30m)`).

Central's `SensorACK` is delivered to Sensor components whose `Accepts()` matches. Only `VMScraper` accepts `SensorACK_VM_INDEX_REPORT`; `Handler` does not.

```mermaid
sequenceDiagram
    participant A as roxagent inside VM
    participant Sc as Sensor VMScraper
    participant H as Sensor Handler
    participant X as Central VM pipeline

    Sc->>A: GetReport (VSOCK, when the VM is due)
    A-->>Sc: index report
    Sc->>H: Send(report)
    H->>X: SensorEvent(vmID, index report)
    X-->>Sc: SensorACK(vmID:vsockCID, ACK|NACK, reason)

    alt NACK
        Sc->>Sc: clear cached generation; apply retry backoff
        Note over Sc,A: next due attempt requests a full report
    end
```

`VMScraper` records ACK/NACK volume (`IndexReportAcksReceived`) and, on NACK, resets its cached `report_generation` so the next due attempt re-sends a full report instead of an "unchanged" delta. There is no payload cache.

## Why the resource ID is `vmID:vsockCID`

Using only the vsock CID is unsafe because a CID can be reused by a different VM later, so a stale ACK could otherwise get attributed to the wrong VM's report. Pairing it with `vmID` removes that ambiguity when parsing logs or correlating an ACK back to a specific VM and connection.

`VMScraper`'s NACK handling only needs the `vmID` half - it extracts that portion (`vmIDFromResourceID`) and looks the VM up by ID in its own store; the CID is not used for VSOCK routing on this path. It stays in the pair purely for debugging/observability, not because pull mode's retry logic requires it.

One limitation remains: `vmID:vsockCID` is not a per-report unique ID - multiple in-flight reports from the same VM with the same CID are still not fully distinguishable, so a stale ACK can match the latest entry instead of the one it was actually for.
