# Offline Scanner E2E vulnerability bundle

Regular E2E runs use the checked-in CI-minimal bundle. Nightlies use the
production feed selected by the deployed Scanner version. The generator creates
native Scanner update data from explicit fixtures; it does not download or filter
a production feed, inspect test expectations, or change production updater
configuration.

## Regenerate

From the repository root:

```sh
go generate ./scanner/updater/ci/generate
go run ./scanner/updater/ci/generate -check
```

The generated file is
`scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip`. The
`make go-generated-srcs` and `make generated-srcs` targets also rebuild it, and
CI checks for uncommitted generated changes.

The generator uses ClairCore's `jsonblob.Store` to write the same record format
as production export, and both use `scanner/updater/bundle.WriteCompressed`.
CI still groups, deduplicates and sorts native payloads, derives fingerprints and
references from their canonical content, and normalizes only the store's random
`Ref` and current `Date`. Unknown envelope fields and complete payloads are
preserved. Each operation uses a separate store to avoid map iteration order.
Fixed ZIP metadata and CI compression settings keep repeated generation
byte-identical; production keeps its default compression settings.

## Update fixtures

The `generate/fixtures_*.go` files are the source of truth. For each scenario,
record the consuming test and image digest in a fixture comment, then verify the
package and advisory relationship against source data. Preserve package/source
identity, distribution or repository, aliases, version ranges, fixed versions,
and any unaffected ranges. Add NVD or CSAF enrichment only when it matches a
native vulnerability record; enrichment by itself does not create findings.

Keep test inputs and expected results independent from the generator. In
particular, do not derive fixture selection or expectations by reading tests or
the bundle being generated. The Struts matching test uses
`generate/testdata/struts-packages.json` as a separate dpkg inventory and checks
the aggregate result in test code. Details about that inventory are in
`generate/testdata/README.md`.

After changing fixtures, regenerate the ZIP and run the focused generator and
bundle checks:

```sh
go test ./scanner/updater/ci/generate
CI_MINIMAL_BUNDLE_PATH="$PWD/scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip" \
  go test ./scanner/updater -run '^TestCIMinimalBundle$' -count=1
go test ./scanner/matcher/updater/vuln -run '^TestCIMinimalBundleValid$' -count=1
```

The optional `scanner_db_integration` test in `generate/matching_test.go` exercises
the production importer and matchers against disposable PostgreSQL databases.
Set `SCANNER_CI_TEST_DB` only to an explicitly selected disposable admin
connection; the test creates and drops databases. See the test's comments for
the optional baseline comparison.

## Update consumers

When bundle bytes change, commit the ZIP first. Then update the immutable commit
pins in the consumer values and tests, verify the pinned download against the
checked-in ZIP, and commit those pin changes separately. Search for
`ci-minimal/vulnerabilities.zip` to find current references; the known consumers
include:

- `deploy/common/ci-values.yaml`
- `scanner/e2etests/helmchart/values.yaml`
- `scanner/matcher/updater/vuln/ci_minimal_bundle_test.go`
- `tests/e2e/bats/scanner_v4_operator_bundle.bats`

Do not infer full E2E or production-feed acceptance from generator tests. The
active image corpus, backend QA suites, and nightly production-import logs cover
separate behavior.
