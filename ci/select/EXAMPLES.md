# CI selection examples

Every example below is a full walkthrough against the production rules in
[`ci/test-selection-rules.yaml`](../test-selection-rules.yaml), with the actual
planner output. Reproduce any of them with:

```sh
ci/select/build.sh
printf '%s\n' ui/src/App.tsx README.md > /tmp/files
ci/select/bin/ci-select resolve --files-file /tmp/files
```

Each example shows the two halves of the trace: the **file section** (which
rules matched each changed path) and the **job section** (the decision per
job, with the paths that caused it).

## Example 1: documentation only

Diff: `README.md`

```
1. changed path: "README.md" matched: documentation-only

1. go, go-postgres, go-bench (+12 more): skip, because every changed path is covered and no run request survived
   - covered by: ["documentation-only"]
```

Every target skips: one rule covers the only changed file for every job.
Full coverage plus no surviving run request is the only way to skip. Jobs
with the same explanation are printed as one grouped line.

## Example 2: a domain README

Diff: `central/sac/README.md`

```
1. changed path: "central/sac/README.md" matched: central-only-domains, documentation-only

1. go: skip, because every changed path is covered and no run request survived
   - covered by: ["documentation-only"]
   - conflict on path 1 (central/sac/README.md): ["central-only-domains"] requests run, ["documentation-only"] marks it safe to skip; the skip wins because it is specific to this file
2. go-postgres, go-bench, e2e-byodb-tests (+10 more): skip, because every changed path is covered and no run request survived
   - covered by: ["documentation-only"]
3. sensor-integration-tests: skip, because every changed path is covered and no run request survived
   - covered by: ["central-only-domains" "documentation-only"]
```

This is the conflict the trace is built to explain. The Central domain rule
matches the file (`^central/sac/`) and requests Go. But the documentation
rule also matches it (Markdown anywhere except test fixtures) and marks it
safe to skip Go. The skip wins because it is reviewed for exactly this
file, while the domain rule only approximates "this directory matters".
Every target skips.

## Example 3: domain code

Diff: `central/sac/auth.go`

```
1. changed path: "central/sac/auth.go" matched: central-only-domains

1. go: run, because a run request survived for a path not marked safe to skip
   - run requested by ["central-only-domains"] for paths: 1
2. go-postgres, go-bench, e2e-byodb-tests (+10 more): run, because not every path is covered, so "remaining" runs the job
   - uncovered paths: 1
3. sensor-integration-tests: skip, because every changed path is covered and no run request survived
   - covered by: ["central-only-domains"]
```

`.go` is not documentation, so the Central request survives on this file and
Go runs. The same rule marks the file safe to skip Sensor integration. Every
other target has no coverage for a Go file, so `remaining` names the path
that forced it.

## Example 4: domain README plus domain code

Diff: `central/sac/README.md`, `central/sac/auth.go`

```
1. changed path: "central/sac/README.md" matched: central-only-domains, documentation-only
2. changed path: "central/sac/auth.go" matched: central-only-domains

1. go: run, because a run request survived for a path not marked safe to skip
   - run requested by ["central-only-domains"] for paths: 2
   - conflict on path 1 (central/sac/README.md): ["central-only-domains"] requests run, ["documentation-only"] marks it safe to skip; the skip wins because it is specific to this file
2. go-postgres, go-bench, e2e-byodb-tests (+10 more): run, because not every path is covered, so "remaining" runs the job
   - uncovered paths: 2
3. sensor-integration-tests: skip, because every changed path is covered and no run request survived
   - covered by: ["central-only-domains" "documentation-only"]
```The same job shows both lines: the surviving request on the code file and the
dropped request on the README. One surviving request is enough to run Go.

## Example 5: one uncovered shared file

Diff: `README.md`, `pkg/shared.go`

```
1. changed path: "README.md" matched: documentation-only
2. changed path: "pkg/shared.go" matched: none

1. go, go-postgres, go-bench (+12 more): run, because not every path is covered, so "remaining" runs the job
   - uncovered paths: 2
```

No rule matches `pkg/shared.go`, so no job can have full coverage and every
target runs. `remaining` states exactly which path forced the run.

## Example 6: UI only

Diff: `ui/src/App.tsx`

```
1. changed path: "ui/src/App.tsx" matched: frontend-only

1. go, go-postgres, go-bench, sensor-integration-tests: skip, because every changed path is covered and no run request survived
   - covered by: ["frontend-only"]
2. e2e-byodb-tests, e2e-db-backup-restore-tests-gke, e2e-db-backup-restore-tests-gke-konflux (+8 more): run, because not every path is covered, so "remaining" runs the job
   - uncovered paths: 1
```

The frontend rule skips the four backend targets for UI files; the e2e
targets have no coverage and run under their usual eligibility.

## Example 7: testdata documentation in a domain

Diff: `central/sac/testdata/fixture.md`

```
1. changed path: "central/sac/testdata/fixture.md" matched: central-only-domains

1. go: run, because a run request survived for a path not marked safe to skip
   - run requested by ["central-only-domains"] for paths: 1
2. go-postgres, go-bench, e2e-byodb-tests (+10 more): run, because not every path is covered, so "remaining" runs the job
   - uncovered paths: 1
```

The documentation rule excludes `testdata/` — fixtures are test inputs, not
prose — so this file is *not* documentation-covered, the Central request
survives, and Go runs. Compare with example 2.

## Example 8: empty diff

Diff: none

```
empty diff: no skip coverage

1. go, go-postgres, go-bench (+12 more): run, because an empty diff provides no coverage, so "remaining" runs the job
```

An empty diff grants no skip coverage; every target runs.

---

Each numbered decision block states the verdict first, with the rule that
decided it: a job runs because a run request survived on a path not marked
safe to skip, runs because not every path is covered and `"remaining"` runs
it, or skips because every changed path is covered and no run request
survived. Evidence bullets below the verdict list the run requests (with
their changed-path numbers), conflicts where a skip overrode a run request
(naming both rules and the file), the covering rules for a skip, and the
uncovered paths for `"remaining"`. Jobs sharing an identical verdict and
evidence print as one grouped block. The trace ends with a
`-- decisions --` summary, one plain `job action` line per target. Long
citation lists collapse after twenty numbers; the changed-path section caps
at fifty paths.
