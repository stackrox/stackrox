# Central rollback boundary E2E

`postgres_run.sh` extends the upgrade journey with current → N−4 (rejected) →
N−3 (release-matched smoke tests) → current (HEAD smoke tests). Central DB is
never downgraded; Scanner stays at the fixture version until N−3 smoke setup.

`versions` derives the boundary from the production compatibility metadata and
reads initial GA sequences from local Git tags. Missing tags, metadata mismatches,
or N−4 sharing the minimum sequence fail before deployment changes. Old binaries
only enforce sequence limits, so shared sequences cannot prove N−4 rejection.

Rejection requires a non-ready Central, a nonzero exit, the expected compatibility
diagnostic, and an unchanged database version record. Evidence is saved separately
under `rollback-rejection`; it does not weaken the normal log checks.

Local checks: `go test ./tests/upgrade/versions` and
`bats tests/upgrade/rollback.bats`. Run the full journey through the existing
`e2e-gke-upgrade-tests` job; local checks do not validate historical image or chart
compatibility with the cluster.
