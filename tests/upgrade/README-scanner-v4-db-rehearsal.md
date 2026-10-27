<!--
TODO(cdu): review note, remove before the PR. The Results section at the
bottom is an empty template -- I cannot fill it in without a cluster carrying
a real vulnerability store. Either you run the rehearsal and fill it in on
this branch before opening the PR, or the tables go in the ROX-37428 Jira
instead and this file keeps only the procedure. ROX-37430 should not merge
until those numbers exist somewhere.
-->

# scanner-v4-db PostgreSQL 15 → 16 rehearsal (ROX-37428)

Procedure for rehearsing the in-place `pg_upgrade` against a **production-sized**
vulnerability store, before `scanner/image/db/Dockerfile` is flipped to PostgreSQL 16
(ROX-37430).

The design validation (ROX-37423) ran against a synthetic 50 000-row cluster. That proved
the mechanism works; it says nothing about how long it takes on a real vulnerability
store, which is the number that decides whether the upgrade costs seconds or minutes of
scanner downtime in the field. This rehearsal produces that number, and re-runs the
negative paths where the stakes are higher than a toy cluster.

Driven by [`scanner_v4_db_rehearsal.sh`](./scanner_v4_db_rehearsal.sh). This is **not** a
CI test — populating the vulnerability store takes too long and too much disk for a
per-PR job. CI coverage is `postgres_upgrade_run.sh` (ROX-37429); unit coverage is
`scanner/image/db/scripts/tests/init-entrypoint.bats` (ROX-37427).

## Prerequisites

- A cluster with Central services deployed and `STORAGE=pvc`, so scanner-v4-db has a real
  volume rather than the `emptyDir` the secured-cluster path uses.
- scanner-v4-db running the **current** (PostgreSQL 15) image, with the matcher having
  fully populated the vulnerability store. Confirm `scanner-v4-matcher` is ready and has
  stopped logging vulnerability-load progress.
- A candidate scanner-v4-db image built at PostgreSQL 16, i.e. the ROX-37430 branch with
  the ROX-37424/37425/37426 changes in it.
- `KUBECONFIG`, and for the scan-continuity step `API_ENDPOINT` and `ROX_API_TOKEN`.

Run **both** transfer modes. `--clone` needs a reflink-capable filesystem (XFS with
`reflink=1`, Btrfs, ZFS); `--copy` is what most installs will actually take. Which one a
given cluster's StorageClass produces is not something the script can choose, so this is
two cluster runs, not one. Record which filesystem each run used:

```bash
kubectl -n stackrox exec deploy/scanner-v4-db -c db -- df -PT /var/lib/postgresql/data
```

## Happy path

```bash
export KUBECONFIG=...
PG16_IMAGE=quay.io/stackrox-io/scanner-v4-db:<your-tag>

tests/upgrade/scanner_v4_db_rehearsal.sh baseline
tests/upgrade/scanner_v4_db_rehearsal.sh snapshot          # optional, see below
tests/upgrade/scanner_v4_db_rehearsal.sh upgrade "$PG16_IMAGE"
tests/upgrade/scanner_v4_db_rehearsal.sh verify
tests/upgrade/scanner_v4_db_rehearsal.sh restart
tests/upgrade/scanner_v4_db_rehearsal.sh scans
tests/upgrade/scanner_v4_db_rehearsal.sh report
```

What each step is for:

| Step | Asserts |
|---|---|
| `baseline` | the cluster really is PostgreSQL 15 and initialized; records row counts, sizes, free space |
| `upgrade` | flips the image and keeps the init container log, which is otherwise lost with the pod |
| `verify` | `PG_VERSION` is 16, the server is 16, `${PGDATA}-new`/`-old` are gone, `.init-ready` is present, every row count matches, and `pg_statistic` is non-empty |
| `restart` | the regression test for the `.init-ready` footgun — without the sentinel written before the swap, *this* restart is what would wipe the upgraded database |
| `scans` | the acceptance criterion from ROX-36454: scans succeed immediately, with no `FailedPrecondition` from the matcher |
| `report` | wall-clock for `pg_upgrade` and `vacuumdb` separately, derived from `kubectl logs --timestamps` |

`snapshot` tars the populated PostgreSQL 15 directory aside so the negative cases can each
start from the same cluster instead of re-populating. It needs room for a second copy of
the data; if the volume cannot take one, skip it and re-populate between cases.

## Negative paths

The governing property is that **none of these may crashloop the pod**. In every case the
scanner must come up — on PostgreSQL 15 if necessary — with the data intact. Restore the
PostgreSQL 15 snapshot between cases:

```bash
tests/upgrade/scanner_v4_db_rehearsal.sh restore
tests/upgrade/scanner_v4_db_rehearsal.sh negative force-old-binaries "$PG16_IMAGE"

tests/upgrade/scanner_v4_db_rehearsal.sh restore
tests/upgrade/scanner_v4_db_rehearsal.sh negative stale-new "$PG16_IMAGE"

tests/upgrade/scanner_v4_db_rehearsal.sh restore
tests/upgrade/scanner_v4_db_rehearsal.sh negative stale-old "$PG16_IMAGE"

# Only meaningful on a filesystem without reflink support, where --clone --check
# fails and the copy ladder is actually reached.
tests/upgrade/scanner_v4_db_rehearsal.sh restore
tests/upgrade/scanner_v4_db_rehearsal.sh negative no-space "$PG16_IMAGE"
```

### Mid-run `pg_upgrade` failure

Not automated, because making the real `pg_upgrade` fail *after* `--check` has passed
needs a purpose-built image. Build one on top of the candidate:

```dockerfile
FROM <PG16_IMAGE>
USER root
RUN mv /usr/bin/pg_upgrade /usr/bin/pg_upgrade.real && \
    printf '%s\n' '#!/bin/bash' \
      'for a in "$@"; do [ "$a" = --check ] && exec /usr/bin/pg_upgrade.real "$@"; done' \
      'echo "simulated failure"; exit 1' > /usr/bin/pg_upgrade && \
    chmod +x /usr/bin/pg_upgrade
USER 70:70
```

Then run `upgrade` with that image and confirm: the init container logs
`Upgrade failed, staying on PostgreSQL 15`, dumps the server log, removes `${PGDATA}-new`,
and exits **zero**; the pod comes up on PostgreSQL 15; row counts are unchanged. This is
the case that proves the no-backup decision is safe — neither `--clone` nor `--copy`
mutates the old cluster, so `$PGDATA` is still authoritative after a failed run.

## Results

Fill this in and attach to [ROX-37428](https://issues.redhat.com/browse/ROX-37428). It is
the gate for ROX-37430.

### Environment

| | Clone run | Copy run |
|---|---|---|
| Cluster / StorageClass | | |
| Filesystem (`df -PT`) | | |
| Vulnerability store size (`pg_database_size`) | | |
| `$PGDATA` on disk | | |
| Rows (total across user tables) | | |

### Timings

| | Clone run | Copy run |
|---|---|---|
| clean shutdown + `--check` | | |
| `pg_upgrade` | | |
| `vacuumdb --all --analyze-only` | | |
| total init container | | |
| scanner-v4-db unavailable, end to end | | |

If `vacuumdb` dominates and delays readiness noticeably, the fallback is
`--analyze-in-stages` with only stage 1 run offline — see §4.4 and the risk table of the
design document.

### Outcomes

| Case | Pod came up | Data intact | PG version after | Notes |
|---|---|---|---|---|
| Happy path, `--clone` | | | 16 | |
| Happy path, `--copy` | | | 16 | |
| Second restart | | | 16 | |
| Scans immediately after | n/a | n/a | n/a | no `FailedPrecondition` expected |
| `FORCE_OLD_BINARIES=true` | | | 15 | |
| Stale `${PGDATA}-new` | | | 16 | discarded, then upgraded |
| Stale `${PGDATA}-old`, `$PGDATA` gone | | | 16 | recovered, then upgraded |
| Volume too full to copy | | | 15 | deferred, retried next restart |
| `pg_upgrade` fails mid-run | | | 15 | server log dumped |
