# RedHat Based main image

The RedHat based main image is currently used for the RedHat marketplace as well as for DoD customers.

This image is built in opinionated way based on the DoD Centralized Artifacts Repository (DCAR) requirements outlined [here](https://dccscr.dsop.io/dsop/dccscr/tree/master/contributor-onboarding)

## PostgreSQL clients

`download.sh` fetches PostgreSQL client RPMs from PGDG for amd64, arm64, and
ppc64le. PGDG publishes no s390x packages, so for that architecture the
`Makefile` builds the `downloads` stage on CentOS Stream instead and
`download.sh` takes the clients from its AppStream repository.

`PG_VERSION` selects the client major version and must match the PostgreSQL
server in `central-db`, since `pg_dump` refuses to dump a newer server. On
s390x the module stream has to be enabled explicitly, because the CentOS
default is older than that server. The final stage runs `pg_dump` and
`pg_restore` and checks their versions in the runtime filesystem, so missing
libraries or an older client fail the image build.
