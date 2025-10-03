# 0017 - RC Version Handling in Vulnerability Bundles

- **Author(s):** J. Victor Martins <jvdm@sdf.org>
- **Created:** 2025-08-27

## Status

Status: Accepted

Updates: [#0011](0011-separate-versioning-for-vuln-bundles.md)

## Context

Versioned vulnerability bundle builds, by default, fallback to the latest release candidate tag when the release tag for a given version was not created yet. This means that broken or incompatible RC changes can overwrite production bundles. Although this allows testing during release candidate validation, incompatible changes would also affect production deployments tracking stable vulnerability version streams.

## Decision

Release candidate bundles will now be produced only through manual `workflow_dispatch` runs. The manual dispatch will accept inputs to specify the bundle version stream and references. Dispatching the workflow with these inputs is equivalent the execution of one stream + reference specification found in the `scanner/updater/version/VULNERABILITY_BUNDLE_VERSION` file; in fact, the single dispatch run will effectively replace that file.

References will accept two, mutually exclusive, inputs:

1. General available reference.
2. Release candidate reference.

When a GA reference is selected, the workflow builds a bundle for the selected reference and uploads to the GA bucket location based on the specified version stream (as of today `https://definitions.stackrox.io/v4/<version-stream>/vulnerabilities.zip`).

When RC reference is selected, the workflow builds a bundle for the selected reference, but uploads it to a RC bucket location based on the specified version stream with `-rc` appended (`https://definitions.stackrox.io/v4/<version-stream>-rc/vulnerabilities.zip`).

Additionally, when RC reference is used, the workflow will resolve any "release version" reference (i.e. the non-git references on the format `X.Y.Z`) to the latest matching `-rc.N` tag for that version, failing if there is no RC candidate tag.  Other run types (including scheduled) will fail if a "release version" reference cannot be directly resolved to a git tag reference (that is, no matching `-rc.N` will be performed).

Finally, Scanner deployments can now opt-in to automatically ingest candidate bundles by setting a new Matcher configuration flag. The matcher's updater will attempt to download `<stream>-rc` and fall back to `<stream>` if the candidate bundle does not exist (status 404).  Any other HTTP status different from the accepted successful returns from the HTTP client (usually >= 400) will be reported immediately for any of candidate URLs.

## Consequences

- Stable scheduled builds no longer consume release candidate code.
- RC bundles are not built periodically, and require explicit invocation to be updated. The release process still need to dispatch the workflow to build the `<stream>-rc` bundles for testing (manual or automated), not covered here.
- RC deployments can switch between stable and RC bundles without redeploying, allowing quick testing during pre-release.
- The code change in the Updater to implement the conditional fetch adds some complexity to the download logic. It's hard to know which bundle (production or RC) was used.
- The new mechanism for dispatching bundle creation open the doors for manual upload of bundles of any version and tag. It allows flexibility for operators to overwrite data if needed. But, if misused, can cause vulnerability bundle outages.
