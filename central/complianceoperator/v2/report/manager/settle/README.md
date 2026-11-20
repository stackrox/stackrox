# Settle-audit-snapshot: compliance reports in their own scheduler

This package (plus the report scheduler in the manager) is the proof of
concept for **proposal 1** ("decouple reporting from scan completion") of the
*compliance reporting on partial scan cycles* design (see
`compliance-report-partial-cycles.md`). It is gated behind the
`ROX_COMPLIANCE_REPORTS_OWN_SCHEDULER` feature flag and covers both report
paths:

- **On-demand reports** (`manager.handleReportRequestSettled`): the report
  settles, audits and snapshots instead of subscribing to the
  `ScanConfigWatcher`.
- **Scheduled reports** (`manager.reportScheduler` + the same settled path):
  reports run on their own schedule, fired by the report scheduler — scan
  completion no longer spawns report cycles at all
  (`manager.handleReadyScan` skips the config-watcher path when the flag is
  on).

## The problem being prototyped away

Today reports are triggered by scan completion and wait for *all* scans of a
scan configuration across *all* clusters (the `ScanConfigWatcher`'s wait set).
When only a subset of scans runs — a "partial cycle", which is exactly what
node-role reconciliation, sensor restarts, new MachineConfigPools and
staggered clusters produce — the report waits for scans that will never
arrive, times out (`ROX_COMPLIANCE_SCAN_SCHEDULE_WATCHER_TIMEOUT`, 45 min by
default), and is then delivered labeled as a success with a dangling
`error_msg`. Which scans belong to the current cycle is only knowable in
sensor and the Compliance Operator; Central cannot observe it.

## The primitive

1. **Settle** (`Settler.Settle`): find the scans of the configuration that are
   *currently running* — scan phase in the scan datastore, the source of
   truth — and wait for them to finish, bounded by
   `ROX_COMPLIANCE_REPORT_SETTLE_TIMEOUT` (default 10 min, polled every
   `ROX_COMPLIANCE_REPORT_SETTLE_POLL_INTERVAL`). A scan that was running and
   did not finish is a real failure and is returned as *unfinished*; a scan
   that disappears while waited on is returned as *removed*.
2. **Audit** (`Settler.Audit`): for every scan of the configuration, per
   cluster, classify it as
   - *fresh*: started within the expected schedule window,
   - *unfinished*: was waited on and did not finish,
   - *overdue*: has not started within the expected window — the expectation
     is computed from the configuration's cron plus each scan's
     `LastStartedTime`, both of which Central owns, with
     `ROX_COMPLIANCE_REPORT_SCHEDULE_GRACE` absorbing operator/sensor
     scheduling delays.

   Findings are folded into `report.FailedCluster` entries (the shape the
   report machinery already consumes) with reasons derived from the
   validator's taxonomy: sensor disconnected (cluster status), compliance
   operator not installed / wrong version (compliance integration), scan
   removed, settle timeout, overdue.
3. **Snapshot** (in the manager): read the latest results per scan from the
   check-results datastore — unchanged, the results aggregator — and deliver.
   `NumFailedClusters` drives the existing labeling: any finding marks the
   report `PARTIAL_SCAN_ERROR_*`, every cluster failing marks it `FAILURE`.
   The report is never delivered as a plain success with a dangling timeout
   error.

## The report scheduler

`manager.reportSchedulerManager` owns **one `configScheduler` per scan
configuration** and fires scheduled report jobs on their own schedule:

- Each scheduler sleeps until the report for the current schedule fire is due
  at **fire + `ROX_COMPLIANCE_REPORT_SCHEDULE_DELAY`** (default 1 h) and
  submits it through the settle-audit-snapshot path. The implicit offset is
  the zero-UX option; a separate user-facing report cron is the cleaner
  long-term option and remains an open product decision in the proposal. The
  delay must stay well below the scan interval; the settle phase and the audit
  absorb the imprecision.
- The scheduler re-reads its configuration from the datastore on every
  wake-up, so cron edits, notifier changes and cluster changes are picked up
  without restarting it.
- **One-time scan configurations** have no cron; their single implicit fire is
  the creation time, so they get exactly one scheduled report (one delay after
  creation).
- **Dedup**: at most one scheduled report per fire. In memory for the running
  scheduler; across restarts and respawns, via the most recent `SCHEDULED`
  snapshot of the configuration in the snapshot store (on-demand snapshots do
  not discharge the scheduled one). A failed attempt is also recorded (as a
  `FAILURE` snapshot), so a fire is attempted at most once. If Central was
  down across several fires, the most recent fire is reported (a catch-up
  report: the audit flags everything that is overdue in it).

### Lifecycle and supervision

- **Pipeline hooks**: the scan-config service calls
  `OnScanConfigUpsert`/`OnScanConfigDeleted` on create/update/delete, so a new
  configuration gets its scheduler immediately. The hooks are no-ops unless
  this Central is scheduling (feature disabled, or another instance holds the
  advisory lock).
- **Startup**: the manager seeds the schedulers from the datastore.
- **Resync safety net**: every
  `ROX_COMPLIANCE_REPORT_SCHEDULER_RESYNC_INTERVAL` (default 1 min) the
  scheduler set is reconciled against the datastore, covering missed pipeline
  events and rebuilding after a lock failover.
- **Supervision**: each scheduler runs under its own
  `concurrency.Stopper`, and a watcher goroutine waits on
  `Stopped().Done()`/`Err()`. The scheduler map
  (`map[config-id]scheduler`) is the authority for "was this exit
  commanded?": delete and manager shutdown remove (or replace) the entry
  before stopping, so a watcher that no longer finds *its own* scheduler
  instance in the map does not respawn. A scheduler that reports
  `errScanConfigGone` (its configuration vanished without a delete event)
  deregisters itself. Any other exit is respawned after a backoff, so a
  scheduler that dies repeatedly cannot hot-spin. The loops themselves are
  self-healing first (transient datastore errors and failed submissions are
  retried with backoff; only a stop request or the configuration's
  disappearance exits them) — the respawn is a belt-and-braces for bugs.
- **HA**: `start()` acquires the `ComplianceReportSchedulerLockID` PostgreSQL
  advisory lock, so only one Central instance schedules reports (the same
  pattern as the vulnerability report scheduler).
- With the flag on, `handleReadyScan` no longer creates `ScanConfigWatcher`s
  or automatic snapshots — the entire completion-triggered cycle machinery is
  dormant. Scan watchers remain for settle detection (old-results cleanup and
  stale marking), which also kills the phantom cycles a stale re-delivery
  (e.g. a NOT-APPLICABLE scan after a sensor restart) used to spawn.

## Content semantics

- **Unfinished** scans are excluded from the report content (their mid-run
  rows are incoherent) and named in the failed-clusters section.
- **Overdue** scans keep their latest results in the report — *mixed
  freshness is the intended semantics*: a report may contain
  master-from-Monday next to infra-from-Wednesday, and each row already
  discloses its `AssessmentTime`. This is what `FailedCluster.IncludeResults`
  signals to the results aggregator.
- **Clusters with no scans at all** are flagged with the derivable reasons.
  This is the failure visibility the wait-for-all set used to provide (and
  more: today, when no scan reaches Central at all, the machinery never
  starts and nothing is reported).

## Known PoC limitations

- The schedule window and the report fire times are computed in UTC; the
  Compliance Operator evaluates the cron in the cluster's local timezone.
  The grace period absorbs the difference for modest offsets; a per-cluster
  timezone would remove it.
- The report-cron offset (fire + delay) assumes the delay stays well below
  the scan interval, so fire windows do not overlap.
- Suspended scan configurations are not excluded from the expectation window
  (suspend/resume state does not currently live on the scan configuration in
  Central).
- The settle phase holds a report-execution concurrency slot while waiting
  (bounded by the settle timeout).
- Re-runs no longer auto-trigger report emails (on-demand reports remain
  available); scheduling a report a fixed delay after a re-run is an open
  product decision in the proposal.
