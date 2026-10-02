# ScannerDB Image

ScannerDB is a PostgreSQL-based image.

To see the version of PostgreSQL used, check `ARG PG_VERSION` in the Dockerfile.

The image also ships the previous major version's server binaries under
`/usr/lib64/pgsql/postgresql-<major>/bin`, which is what makes the in-place
major-version upgrade in `scripts/init-entrypoint.sh` possible: when the upgrade
cannot be performed safely it leaves the data directory alone, and
`scripts/docker-entrypoint.sh` starts the server matching whatever is on disk
rather than crashlooping. See `scripts/tests/` for the unit tests.

The image requires a pg_hba.conf (client authentication configuration) file and postgresql.conf (server configuration) file.
These are specified via ConfigMap for the ScannerDB Kubernetes deployment.
