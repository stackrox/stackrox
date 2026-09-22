# CI-minimal vulnerability bundle

This is test data, not a production vulnerability feed. Keep native importer
records: an NVD enrichment is not a matching vulnerability.

## Coverage

The generator retains test CVEs and advisories (including decorated Ubuntu names,
OSV aliases, and identities in advisory links), the existing Alpine 3.9 package
selection, and the package identities in `qa-packages.json`. The latter contains Ubuntu binary/source packages and
Maven packages indexed from the amd64 Struts image
`quay.io/rhacs-eng/qa-multi-arch@sha256:4dd78f23f89cc7e6da2efe939b14d8f855adbfa7eb62ffd5d508ec57cdf30873`.
All records for those identities are retained, so aggregate QA assertions are not reduced to a list of
named CVEs. Manual records are retained in full. NVD and Red Hat CSAF enrichment
is selected by the identifiers and advisory links of the retained records.
RHEL advisory selection is closed over native CVE identities so unaffected
(`Invert`) ranges are retained even when they do not carry an advisory link.

The Scanner corpus is `scanner/e2etests/testdata/image_tests.json`, which
`TestImage` actually loads; `image_tests.orig.json` is not an additional suite.
Native sources also include Amazon Linux, Oracle Linux, and Photon, required by
the active Scanner cases. ALAS, RHSA, RHBA, GO, and GHSA identifiers are selected
as well as CVEs. A CVE may appear only in an Oracle or Photon advisory link.
Mocked UI data is not a source of bundle requirements.
`TestCIMinimalBundle` reads the active Scanner corpus and checks that every
expected vulnerability/advisory has native records, in addition to the explicit
package/distribution and backend QA checks. This does not replace image matching.

The required QA cases are:

| Image/component | Expectation |
| --- | --- |
| `qa-multi-arch:struts-app` | At least 138 findings; `CVE-2017-5638` on `org.apache.struts:struts2-core`; vulnerability risk factor |
| `qa-multi-arch:ubi9-minimal-9.6-1760515502`, `openssl-libs` | `CVE-2025-15467` |
| `qa:oci-manifest`, Ubuntu 16.04 `systemd` | `CVE-2021-33910` |
| `qa:list-image-oci-manifest`, Ubuntu 22.04 `libc6` | `CVE-2023-4911` |
| `qa:ubi9-9.7-1769417801-amd64`, `python3` | `CVE-2025-11468`, Moderate, CVSS 4.5 |
| `qa:ubuntu-22.04-amd64`, `gpgv` | `CVE-2022-3219`, Low, CVSS 3.3 |
| Amazon Linux 2, `nss-sysinit` | `ALAS2-2024-2442` |
| Oracle Linux 8, `libgcrypt` | `CVE-2021-33560`, `CVE-2021-40528` |
| Photon 3.0, `curl` | `CVE-2023-38546` |

QA image names above use `quay.io/rhacs-eng/`. The backend QA and Scanner tests
are the authority for these expectations. Do not lower counts, change scores, or fabricate records
to make a reduced bundle pass. Validate against the same full-source snapshot
when an expectation fails.

## Regeneration

The repair uses the 2026-09-22 source snapshot:

- URL: `https://definitions.stackrox.io/v4/vulnerability-bundles/dev/vulnerabilities.zip?generation=1790074423522380`
- SHA256: `979abbb54646d7baa320bee2dd507d5e32b869f1ac3c834d20b1b08859fb9083`
- Size: 236,433,828 bytes.

Keep the downloaded archive; generation reads it without refreshing it. An absent
`SOURCE_BUNDLE_ZIP` is downloaded from `SOURCE_BUNDLE_URL`. Supply the checksum to
reject an unexpected snapshot. Both root members and production `bundles/`
members are supported; ambiguous duplicates are rejected.

From the repository root, generate and validate isolated outputs first:

```sh
SOURCE_BUNDLE_ZIP=/path/to/source.zip \
SOURCE_BUNDLE_SHA256=979abbb54646d7baa320bee2dd507d5e32b869f1ac3c834d20b1b08859fb9083 \
CI_MINIMAL_OUTPUT_PATHS=/tmp/ci-minimal-one.zip:/tmp/ci-minimal-two.zip \
scanner/updater/ci/generate-ci-minimal-bundle.sh --check-reproducible

bats scanner/updater/ci/generate-ci-minimal-bundle_test.bats
CI_MINIMAL_BUNDLE_PATH=/tmp/ci-minimal-one.zip \
go test ./scanner/updater -run '^TestCIMinimalBundle$' -count=1
```

Omit `CI_MINIMAL_OUTPUT_PATHS` to regenerate both checked-in copies after
validation. `CI_MINIMAL_TEST_CVE_PATHS` and `CI_MINIMAL_PACKAGE_SELECTION` allow
isolated generator fixtures; they do not alter the production updater config.

To refresh `qa-packages.json`, index the image with the current Scanner indexer,
record its digest, and extract its dpkg binary/source names and Maven names from
the index report. Keep distribution and repository identities separate. Validate
the updated inventory through real matching, not only by counting source records.

## Runtime validation and publication

The 2026-09-22 generated bundle was validated before replacing the checked-in
copies: both reproducible outputs and the native-scan candidate have SHA256
`cd2db5c4ef0960ee9668f7df91ce54eb2a309b5c11e21f545ef210eee225ca5e` and are
606 KiB. The isolated matcher imported all 11 sources in 5.221420973 seconds.
Seven saved image scans completed with 921 (Struts), 19 (Python), 14 (OpenSSL),
4 (gpgv), 7 (libc), 8 (nginx), and 8 (systemd) vulnerability records. Struts
had 921 normalized package/vulnerability matches, equal to the unfiltered
reference set; this includes CVE-2017-5638 on
`org.apache.struts:struts2-core`. Named QA records and severities were present,
including Python CVE-2025-11468 (Moderate, CVSS 4.5) and gpgv CVE-2022-3219
(Low, CVSS 3.3).

This validates native import and matching only. No backend GraphQL QA or full
42-image E2E run was performed, and a cold full-feed import was not timed
because it exceeded the available 45-minute window.

Use fresh, isolated matcher databases for full/minimal comparisons. Original
updater fingerprints are preserved; importing a subset over an already imported
full snapshot can skip operations with the same fingerprint.

Run `TestCIMinimalBundleStruts` in the Scanner E2E suite and the backend QA suites
`DefaultPoliciesTest`, `ImageScanningTest`, `VulnMgmtTest`, and
`VulnScanWithGraphQLTest`. Compare archive size and cold-import duration using the
same source snapshot and database configuration. Archive checks alone do not
establish runtime coverage.

The matcher-specific CI allowlist must include every archive source. The Helm
customization order applies matcher-specific environment variables after global
ones; the standalone Scanner E2E chart has no allowlist and imports all members.

Publishing is separate from local generation: publish the validated bundle
commit first, then update every immutable bundle URL (CI values, Scanner E2E
values, installation tests, and the GitHub accessibility test). Verify the remote SHA256 before
rerunning CI. Until those pins change, CI still downloads the previous bundle.
