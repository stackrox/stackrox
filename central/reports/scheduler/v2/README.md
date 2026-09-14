# Report scheduler ownership

Central runs report scheduling when Central Worker is disabled. Otherwise the Worker
owns it. Each scheduler acquires the PostgreSQL report advisory lock before recovering
pending reports and schedules. Lock contention, connection errors, and failed initial
discovery queries are retried every five seconds until shutdown.

`Start` starts one background lifecycle. `Stop` cancels acquisition and running reports,
waits for report execution to finish, stops cron, and releases ownership. Reports
interrupted by scheduler shutdown remain pending for the successor to recover; explicit
user cancellations still become failures. A stopped scheduler cannot be restarted.

Worker `/readyz` requires initialized scheduler ownership; `/healthz` remains independent.
The `rox_report_scheduler_ready` metric is 1 on the active, initialized scheduler and 0
on an inactive, waiting, or stopping process. When reports are delegated, Central's
metric is 0 and the Worker's metric should be 1. Central's API readiness remains
independent so a replacement Central can become ready during a rolling update before
the old Central releases scheduling ownership.

The Central DB NetworkPolicy retains the same-namespace Worker selector even when the
Worker is disabled. Removing that access during deletion can prevent a terminating
Worker from releasing its session-level locks. Worker shutdown stops reporting and
pruning concurrently, within the pod's 120-second termination grace period.
