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

Local checks: `go test -tags test ./tests/upgrade/versions` and
`bats tests/upgrade/rollback.bats`. Run the full journey through the existing
`e2e-gke-upgrade-tests` job; local checks do not validate historical image or chart
compatibility with the cluster.

## CI Central DB reservation

Only `CI=true CI_JOB_NAME=gke-upgrade-tests-central` uses the proposed 2-CPU
Central DB request with no CPU limit; memory is 8Gi requested / 8Gi limited.
Temporary chart defaults are customized before install and both chart upgrades,
because omitted/null values otherwise restore the default CPU limit. Both
`central-db` and `init-db` use these resources. Explicit retained release CPU
limits fail before upgrade. The pinned historical scale script is patched before
execution and restored on exit; only its CI DB resources change.
The defaults file contains embedded Helm templates, so customization uses a
checked patch rather than parsing it as plain YAML. Helm arguments set the CPU
request and both memory values, including when reusing stored release values.

Validate locally with `bats tests/upgrade/rollback.bats` and render charts from
the initial, N−3 and HEAD CLIs: both DB containers must request 2000m with no CPU
limit, 8Gi memory request and limit, and no resource changes in other containers. Rendering
alone does not validate Helm `--reuse-values` across releases.

With separate authorization, run the Central job on a fresh four-node cluster
with matching published images. Capture resources after install, scale, N−3 and
HEAD chart application; require Collector counts 4/4/4/4 and all existing
migration, rejection, recovery and smoke assertions. Compare migration duration,
DB CPU, restarts/OOMs and node reservations. The request is not performance-tuned:
removing the limit allows bursting but does not guarantee throughput or placement.
The 8Gi memory limit lowers the OOM threshold from the chart default of 16Gi.
