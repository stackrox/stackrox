# CI Job Cartography Tool

This CLI tool analyzes the current landscape of Prow and GHA CI jobs and outputs them as JSON. Alternatively, it can output an interactive HTML file that has the ability to search and filter jobs for ease of use.

## Usage

> [!NOTE]
> All commands are assumed to be run from the root of the stackrox/stackrox repo

To see what's currently running in stackrox/prow master/main branches and output to an interactive HTML file (this is probably your most common use case):

```bash
go run ./tools/job-matrix -o html --out /tmp/job-matrix.html
open /tmp/job-matrix.html
```

To view something on a prow branch or stackrox branch:
```bash
go run ./tools/job-matrix --branch kirk/test-gha-changes --prow-branch rox-1234/my-prow-changes
```

To view against a fork:
```bash
go run ./tools/job-matrix --repo iamkirkbater/stackrox-fork --branch my-branch --prow-repo iamkirkbater/release-fork --branch rox-1234/my-changes
```

To view against locally cloned repos and skip shelling out to a shallow clone:
```bash
go run ./tools/job-matrix --workflows-dir .workflows --prow-dir /path/to/my/openshift/release/ci-operator/config/stackrox/stackrox
```
