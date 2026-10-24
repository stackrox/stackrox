# PostgreSQL upgrade backups

`init-entrypoint.sh` creates a physical `backup.tar` before a supported PostgreSQL
major-version upgrade and verifies it by restoring and starting the old server.
Backups live in `<old-major>-<new-major>` directories on the optional backup PVC
(`/var/lib/postgresql/backup`) or the data PVC (`/var/lib/postgresql/data/backup`).

## Single-copy upgrade prototype

This branch includes an opt-in experiment for comparing the existing tar path
with one independently copied directory. Both variants are in the same DB image:

| `ROX_POSTGRES_UPGRADE_BACKUP_MODE` | Behavior |
|---|---|
| Unset or `tar` | Existing archive, extract, verify, delete-verification-copy path |
| `directory` | Copy once, flush, start/stop the copy with the old engine, flush again, retain it |

Directory mode uses `cp -a --reflink=never` and retains
`<backup-root>/<old>-<new>/directory-copy`. It does not produce `backup.tar`.
Source health checks, two-copy free-space preflight, backup-volume selection,
and conversion mode/parallelism remain the same for the comparison. Both the
community and Konflux Dockerfiles already package the modified initializer; no
second image build or helper-script injection is necessary.

This is a fresh-trial prototype, not a supported recovery format. It rejects
existing version-pair artifacts, source symlinks (including external WAL or
tablespaces), and all four recovery flags. A failed attempt must be investigated
and reset from the pristine fixture rather than retried automatically. Tar mode
also refuses to reuse/delete a detected directory-copy artifact for a forward
upgrade or recovery operation. Normal same-major restarts preserve the artifact.
Automatic stale-backup cleanup does not recognize directory-only backups.

Recover prototype data manually by independently copying the retained directory
to an empty working location and starting it with the old engine. The smoke test
below demonstrates this. Older recovery scripts and `RESTORE_BACKUP` cannot
restore this format. Do not run the experiment on customer data.

### Comparing built images on OpenShift

Use the same CI-built DB image **digest** for B and P. In the standalone benchmark
Pod described in the prototype runbook, replace the injected `B.sh`/`P.sh`
commands with the installed initializer:

```bash
# B: legacy path, starting from a fresh old-engine fixture.
oc -n "$BENCH_NS" exec db-upgrade-bench -- \
  env ROX_POSTGRES_UPGRADE_BACKUP_MODE=tar /usr/local/bin/init-entrypoint.sh

# Reset working data and backup artifacts from the pristine seed before P.
oc -n "$BENCH_NS" exec db-upgrade-bench -- \
  env ROX_POSTGRES_UPGRADE_BACKUP_MODE=directory /usr/local/bin/init-entrypoint.sh
```

Use the built image as the benchmark Pod's `image`; script ConfigMap mounts are
unnecessary except for your fixture-verification SQL. Keep the 150Gi working,
300Gi backup, and 100Gi seed claims for the 50 GB experiment, subject to measuring
the physical footprint and free space. Keep Central stopped and reconciliation
paused while using those claims. Start/query the target engine after each
initializer run; the initializer itself does not start the runtime server.

For an experiment using the actual DB Deployment, set the variable on **init-db**,
not just the runtime container. With roxie reconciliation paused, a strategic
merge patch can set it before an intended upgrade:

```bash
oc -n "$BENCH_NS" patch deployment central-db --type=strategic -p \
  '{"spec":{"template":{"spec":{"initContainers":[{"name":"init-db","env":[{"name":"ROX_POSTGRES_UPGRADE_BACKUP_MODE","value":"directory"}]}]}}}}'
```

This patch can trigger a rollout if the Deployment is running. Prepare the
disposable old-engine data and images first. It does not force a major upgrade
on already-upgraded data. No permanent chart or Operator API is added.

### Timing output

Both modes log `PG_UPGRADE_TIMING` lines during a major upgrade:

```text
PG_UPGRADE_TIMING mode=directory phase=directory_copy event=start
PG_UPGRADE_TIMING mode=directory phase=directory_copy event=end elapsed_s=42 exit_code=0
PG_UPGRADE_TIMING mode=directory phase=backup_total event=end elapsed_s=46 exit_code=0
```

Numbers above are illustrative. Each phase has start/end records; failure emits
the active phase's nonzero exit code and an `initializer_total` record. The
`backup_total` summary is emitted only after successful verification and required
flushes/cleanup. It overlaps the individual backup phases; do not add both when
summing durations. Seconds are coarse shell elapsed time. `initializer_total`
excludes normal runtime startup and Central API recovery. Ordinary same-major
starts do not emit upgrade timing. Abrupt kills may leave missing end records.

Compare three fresh B/P pairs with identical data, image digest, storage, resource
limits, and reset/cache preparation. Keep hashing and seed-copy time outside the
measured upgrade interval. Preserve failed attempts. The small smoke test is not
a performance benchmark or crash-durability qualification.

## Automatic cleanup

Cleanup runs only during a forward upgrade from `POSTGRESQL_PREV_VERSION`, after
checking the required binaries and starting and cleanly stopping the source DB.
It preserves older backups until the new backup has been verified. If neither
backup location has enough space for the new backup and its verification copy,
it first tries reclaiming obsolete backups and checks the space again. Cleanup
errors are logged; insufficient space still prevents the upgrade.

A cleanup candidate must be a real directory with a canonical, increasing pair
of positive major versions (up to three digits) and a nonempty, regular,
non-symlink `backup.tar`. Its source version must precede the current source
version, and its destination must not exceed that version. For a `15-16` upgrade,
`13-15` qualifies; `15-16`, `16-17`, and `13-16` are preserved. Symlinked roots,
symlinked directories, unrelated names, and missing or ambiguous archives are
preserved. The layout identifies legacy backups; cleanup does not inspect their
archive contents.

These older backups are outside the current image's supported rollback path.
They may still be useful with an older compatible image. When disk space is
insufficient, reclaiming them before the new backup is verified is a deliberate
tradeoff: the validated source database remains in place, but a later failure can
leave those older recovery points unavailable.

There is no automatic age-based expiration. Ordinary startups and fresh
initialization preserve backups. `RESTORE_BACKUP` and `FORCE_NEW_BACKUP` suppress
automatic cleanup during recovery; their existing explicit actions still apply.
`FORCE_OLD_BINARIES` bypasses the upgrade and its cleanup entirely. In particular,
`FORCE_NEW_BACKUP` still explicitly removes the current upgrade backup.

`FORCE_CLEANUP=true` removes `${PGDATA}-new`, the temporary database directory left
by a previous upgrade attempt, so the upgrade can retry. It does not suppress
automatic backup cleanup: obsolete backups remain eligible for removal under the
same validation and space checks as a normal upgrade. If `RESTORE_BACKUP` or
`FORCE_NEW_BACKUP` is also set to `true`, automatic backup cleanup stays disabled.

## Tests

Run `bats image/postgres/scripts`. Helper tests exercise real filesystem deletion
and preservation. Entrypoint tests use temporary paths and fake PostgreSQL and
space checks to cover cleanup timing, failures, retries, and recovery modes.
Real major-version upgrade and restore validation also requires a PostgreSQL
container with both supported server versions.

`tests/upgrade-backup-smoke.sh` runs both modes against real engines, verifies
rows, sequences, and large objects, modifies the upgraded database, and restores
the original data from each backup. First prepare the seed in an old-major image:
the target image's compatibility binaries do **not** include the old `initdb`.
Use old and new product DB images with the same PostgreSQL user UID, architecture,
and locale configuration. For the current 15→16 case, verify the seed image's
`postgres --version` reports 15. From the repository root:

```bash
export OLD_DB_IMAGE='quay.io/stackrox-io/central-db:YOUR_PG15_BUILD_TAG'
export DB_IMAGE='quay.io/stackrox-io/central-db:YOUR_CI_BUILD_TAG'
export SMOKE_SEED="db-upgrade-smoke-seed-$(date +%s)"
podman run --rm --network none --entrypoint /bin/bash \
  --volume "$SMOKE_SEED:/seed:U" \
  --tmpfs /run/secrets/stackrox.io/secrets:rw,mode=0777 \
  --volume "$PWD/image/postgres/tests:/prototype-tests:ro" \
  "$OLD_DB_IMAGE" /prototype-tests/upgrade-backup-smoke.sh prepare

podman run --rm --network none --entrypoint /bin/bash \
  --volume "$SMOKE_SEED:/seed:ro" \
  --tmpfs /var/lib/postgresql/data:rw,mode=0777 \
  --tmpfs /var/lib/postgresql/backup:rw,mode=0777 \
  --tmpfs /run/secrets/stackrox.io/secrets:rw,mode=0777 \
  --volume "$PWD/image/postgres/tests:/prototype-tests:ro" \
  "$DB_IMAGE" /prototype-tests/upgrade-backup-smoke.sh
```

The prepare step refuses a nonempty seed mount; the run step refuses nonempty
working/backup/secret roots and requires the expected source major. Keep the
stopped seed read-only for the run. Remove the named seed volume after collecting
results with `podman volume rm "$SMOKE_SEED"`.

The permissive mount-root modes are only for empty disposable test tmpfs mounts;
PostgreSQL data directories are initialized with restrictive permissions. The
script never connects to an external database. It is invoked with Bash and does
not need to be executable or included in production images.

The existing CI `shell-unit-tests` job discovers the entrypoint Bats tests,
including both modes and copy/flush/verification failures. The real-engine smoke
test above is a separate image-validation command; it is not automatically run
by that mocked shell test job. The existing Build workflow packages the scripts
and publishes Central DB images on PR events (or explicit dispatch); merely
pushing a non-master branch does not trigger that workflow. No CI result is
implied by passing local tests.
