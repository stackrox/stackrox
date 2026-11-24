# ci-select

`ci-select` turns a CI selection rules file plus a pull request's changed
paths into a **plan**: a `run` or `skip` decision for every CI target.

This is the skeleton of a larger effort (central change detection + rule
engine + job selection). Today only the skeleton exists:

- **CLI** (`main.go`) — the `plan` command.
- **Engine seam** (`internal/ruleengine`) — the `Engine` interface and the
  `Decision` output type callers depend on.
- **Placeholder engine** (`internal/ruleengine/alwaysrun`) — reads the
  target inventory from the rules file and plans `run` for every target,
  ignoring the changed paths. A real path-based engine replaces it behind
  the same seam.

Change detection is not done here: the changed paths are detected upstream
(the existing `.github/workflows/detect-changes.yml`, which uses the GitHub
API) and passed in with `--files-file`. The placeholder engine ignores them
for now; the real engine will match them against path rules.

## Build

```sh
make -C ci/select        # builds ci/select/bin/ci-select
```

## Usage

```sh
git diff --name-only origin/master...HEAD > changed.txt
ci/select/bin/ci-select plan --rules ci/test-selection-rules.yaml --files-file changed.txt
```

`--files-file` holds the changed paths, one per line; omit it for an empty change set.

## Example

`ci/test-selection-rules.yaml` is a stupid-simple example: it lists the
targets and nothing else. The engine plans `run` for each:

```
-- plan --
e2e-qa-tests-gke run
go run
go-postgres run
```

## GitHub Actions

`.github/workflows/ci-select.yaml` calls `detect-changes.yml` for the changed
paths, builds the binary, and prints the plan on every pull request. It is a
**preview only** — it gates no jobs. Wiring the plan into the test workflows
happens once a real engine is in place.
