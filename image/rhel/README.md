# RedHat Based main image

The RedHat based main image is currently used for the RedHat marketplace as well as for DoD customers.

This image is built in opinionated way based on the DoD Centralized Artifacts Repository (DCAR) requirements outlined [here](https://dccscr.dsop.io/dsop/dccscr/tree/master/contributor-onboarding)

## PostgreSQL clients

`Dockerfile` downloads PostgreSQL client RPMs from PGDG for amd64, arm64, and
ppc64le. For s390x, `download.sh` adds the CentOS Stream 9 AppStream repository
to its download commands because PGDG does not publish s390x packages and UBI's
repositories do not provide the PostgreSQL 15 module. The download and runtime
stages use UBI on every architecture.

`PG_VERSION` selects the client major version and must match the PostgreSQL
server in `central-db`, since `pg_dump` refuses to dump a newer server. The
`runtime-base` stage runs `pg_dump` and `pg_restore` and checks their versions in
the runtime filesystem, so missing libraries or an older client fail the image
build.
