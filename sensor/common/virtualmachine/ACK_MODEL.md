# ACK Model for VM Index Reports

VM index reports flow directly from Sensor to Central; Compliance is not involved (see `compliance/ACK_MODEL.md` for the separate node-scanning ACK model, which does go through Compliance).

`VMScraper` pulls reports from each running VM's roxagent over VSOCK and forwards successful reports to Central through `vmIndex.Handler`, which wraps the Scanner V4 payload as a `v1.IndexReport`.

A successful forward already returns the VM to poll cadence. Central's ACK is counted (`IndexReportAcksReceived`) and otherwise ignored. A NACK clears cached `report_token` and applies the shared retry backoff so the next attempt pulls a full report. There is no payload cache and no ACK timeout.

A short tick (`ROX_VIRTUAL_MACHINES_SCRAPER_TICK_INTERVAL`, default 10s) walks the per-VM schedule. Each VM is due at its own `nextAttemptAt`. Success and permanent failures reschedule at `pollInterval + U(0, W)`, where `pollInterval` is `ROX_VIRTUAL_MACHINES_SCRAPER_POLL_INTERVAL` (default 4h) and `W` is `ROX_VIRTUAL_MACHINES_SCRAPER_STEADY_SPREAD_FRACTION` times that interval (default 2/3).

`C` is `newVMIndexReportWindow`: `min(20m, pollInterval / 3)` (20m at the default 4h poll interval). Newly tracked VMs are first due at `now + U(0, C)`. A tick starts at most `startBudget` of the due set (`ceil(nTracked × tick / C)`, at least 1), then capped by concurrency. `nTracked` is the fleet in `vmState`, not the leftover due pile, so the drain rate stays stable as due VMs are started.

Retryable pull failures and NACKs share one exponential backoff (`ROX_VIRTUAL_MACHINES_SCRAPER_INITIAL_BACKOFF`, default 10s, doubled each time, capped at `min(pollInterval, 30m)`).

Central's `SensorACK` is delivered to Sensor components whose `Accepts()` matches. `Handler` advertises `SensorACKSupport` so Central sends these ACKs; only `VMScraper` accepts `SensorACK_VM_INDEX_REPORT`.

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

    alt ACK
        Note over Sc: recorded; schedule unchanged
    else NACK
        Sc->>Sc: clear cached token; apply retry backoff
        Note over Sc,A: next due attempt requests a full report
    end
```

## Why the resource ID is `vmID:vsockCID`

Using only the vsock CID is unsafe because a CID can be reused by a different VM later, so a stale ACK could otherwise get attributed to the wrong VM's report. Pairing it with `vmID` removes that ambiguity when parsing logs or correlating an ACK back to a specific VM and connection.

`VMScraper`'s NACK handling only needs the `vmID` half: it extracts that portion (`vmIDFromResourceID`) and looks the VM up by ID in its own store. The CID is not used for VSOCK routing on this path; it stays in the pair for debugging.

The pair is not a per-report unique ID: multiple in-flight reports from the same VM with the same CID are still not fully distinguishable, so a stale ACK can match the latest entry instead of the one it was actually for.

A NACK that changes `lastToken` or backoff before `commitVMState` runs is not overwritten by that commit. If `lastToken` is already empty and backoff is already at the cap, `nextBackoff` is a no-op, so that NACK is invisible to the commit.
