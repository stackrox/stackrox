# PostgreSQL upgrade backups

`init-entrypoint.sh` creates a physical `backup.tar` before a supported PostgreSQL
major-version upgrade and verifies it by restoring and starting the old server.
Backups live in `<old-major>-<new-major>` directories on the optional backup PVC
(`/var/lib/postgresql/backup`) or the data PVC (`/var/lib/postgresql/data/backup`).

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
