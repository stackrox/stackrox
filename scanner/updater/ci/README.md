# Scanner CI vulnerability bundle

This frozen dataset contains the package/advisory relationships needed by Scanner
and backend tests. Update it when a test needs new coverage or a dependency or
format change requires compatible fixtures. There is no periodic data refresh.
The author of the affected test or compatibility change maintains its fixtures.

## Origin and source of truth

The original minimal ZIP was filtered from a pinned production snapshot using
CI package and vulnerability requirements. During the
[offline generator implementation](https://github.com/stackrox/stackrox/commit/d79122fe33533aaa62330ba22373d640d82c65c0),
a one-off transcription helper converted that ZIP into typed Go fixture tables.
The input ZIP's SHA-256 was
`41c160a910cfa9600f9e6e7b69af0e8ba26032f35bd207373ae3206a6577aa35`.
That temporary helper is not a maintained extraction tool.

Today, the source of truth is the `fixtures_*.go` files under [generate/](generate/).
The [generator](generate/main.go) builds the ZIP offline from those tables;
it does not read a production bundle, tests, or the network. “Synthetic” refers
to the assembled test dataset: package/advisory relationships come from
upstream records, rather than invented findings. The checked-in ZIP lets tests
read the dataset without regeneration or network access.

## Add or update a CVE or scenario

1. Identify the consuming test and image digest or package/version being tested.
   Define its expected findings independently of fixture generation; do not
   lower existing counts or change expected scores to accommodate missing data.
2. Prefer records from a pinned production bundle. Record its immutable URL or
   object generation and SHA-256 in a nearby fixture comment. If it lacks the
   needed record, use authoritative vendor, NVD, or OSV data and record the
   source. Do not replace unrelated fixtures with a newer feed snapshot.
3. Find the CVE or advisory through names, aliases, and advisory links. Preserve
   relevant native records: package and source-package identity, distribution,
   repository/CPE, architecture/module constraints, affected/fixed versions,
   and unaffected (`Invert`) ranges. A CVE can have several applicable records.
4. Add the records to the appropriate source's `fixtures_*.go` table. Add the
   required NVD/CSAF enrichment with matching tags, descriptions, links, and
   CVSS. Enrichment alone does not produce a finding. Preserve empty native
   descriptions when they should use NVD fallback text. For a new source group,
   register its table in [fixtures()](generate/fixtures.go).
5. Add or extend independent matching expectations in
   [matching_test.go](generate/matching_test.go) and the consuming test. Record
   the scenario and source provenance alongside the fixture. Existing tables
   provide examples of native records and enrichment payloads.

## Regenerate and validate

Advance `defaultBundleRevision` in [main.go](generate/main.go) beyond every
existing ZIP member timestamp before replacing changed bundle bytes. Changes
to compression dependencies can also change the bytes and require a revision
bump. From the repository root:

```sh
go generate ./scanner/updater/ci/generate
go run ./scanner/updater/ci/generate -check
sha256sum scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip
```

Run the generator package tests:

```sh
go test ./scanner/updater/ci/generate ./scanner/updater/bundle
CI_MINIMAL_BUNDLE_PATH="$PWD/scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip" \
  go test ./scanner/updater -run '^TestCIMinimalBundle$'
go test ./scanner/matcher/updater/vuln -run '^TestCIMinimalBundleValid$'
```

For changes to matching, enrichments, or importer compatibility, run the
[database-backed matching and persistent-update tests](generate/matching_test.go).
Set `SCANNER_CI_TEST_DB` to a disposable PostgreSQL admin connection and run
`make -C scanner db-integration-test`. These tests create and drop databases;
the [Scanner DB integration workflow](../../../.github/workflows/scanner-db-integration-tests.yaml)
also runs them. Run the affected E2E scenario to check actual image findings.
Passing serialization/importer tests does not prove every new ClairCore field
is covered: review relevant field and format changes and add expectations where
needed. Fixtures do not replace production-feed testing.

## Publish and run

Commit fixtures, independent tests, and the regenerated ZIP together. The [Publish Scanner PR CI bundle workflow](../../../.github/workflows/scanner-ci-bundle-pr.yaml)
validates and publishes the exact ZIP from an upstream `stackrox/stackrox` PR,
including drafts, when the ZIP changes. Fixture-only changes do not trigger
publication. Fork PRs cannot publish new bundles; use an upstream branch.
There is no master-push or scheduled CI-bundle publication.

The immutable URL is:

```text
https://definitions.stackrox.io/v4/vulnerability-bundles/ci-minimal-<sha256>/vulnerabilities.zip
```

Existing objects are verified rather than overwritten. Wait for publication to
succeed, then rerun affected E2E jobs that started before the bundle was ready.
The workflow summary lists the source commit, digest, and URL. There is no
automatic waiting, retesting, or production fallback. If the tested merged
checkout has a different ZIP, update the branch so those bytes are published.
