# Central rollback boundary E2E

`postgres_run.sh` extends the upgrade journey with current → pinned unsupported release (rejected) →
N−3 (release-matched smoke tests) → current (HEAD smoke tests). Central DB is
never downgraded; Scanner stays at the fixture version until N−3 smoke setup.

`versions` derives N−3 from production compatibility metadata and validates pinned
`4.9.0` as unsupported using initial GA sequences from local Git tags. This proves
old-release rejection, not the exact N−4 boundary. Missing tags, metadata mismatches,
or a rejected release sharing the minimum sequence fail before deployment changes.
Old binaries enforce sequence limits, not release streams.

Maintain `PREVIOUS_RELEASES` explicitly: its last stream must be supported by the
current target. A stale upgrade list fails before deployment changes.

Rejection requires a non-ready Central, a nonzero exit, the expected compatibility
diagnostic, and an unchanged database version record. Evidence is saved separately
under `rollback-rejection`; it does not weaken the normal log checks.

Local checks: `go test -tags test ./tests/upgrade/versions` and
`bats tests/upgrade/rollback.bats`. Run the full journey through the existing
`e2e-gke-upgrade-tests` job; local checks do not validate historical image or chart
compatibility with the cluster.
