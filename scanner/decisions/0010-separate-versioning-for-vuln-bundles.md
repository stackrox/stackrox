# 0010 - Separate Versioning Stream for Vulnerability Bundles

- **Author(s):** J. Victor Martins <jvdm@sdf.org>
- **Created:** [2024-07-22 Mon]

## Status

Status: Accepted

Updates: [#0009](0009-scannerv4-read-manual-vulnerabilities.md)

## Context

A Vulnerability Bundle is an archive containing updates to vulnerability data in Scanner V4 (aka. Scanner).  They are constantly updated by the Vulnerability Exporter (aka. Exporter), which runs periodically in GitHub Workflows and uses ClairCore to fetch the security data to convert it to a JSON-based schema.  Then, the workflow bundles the data into a ZIP archive.  ClairCore owns the schema.

The Scanner Matcher, using ClairCore, runs the Vulnerability Updater (aka. Updater), which fetches the bundle, parses the data, and updates the vulnerability database.

Suppose a backward-incompatible schema change is made in ClairCore and incorporated into Scanner.  In that case, the following Exporter execution will update the bundle adhering to the new schema.  Existing Scanner instances can only parse the changed data if they upgrade to the new ClairCore version that understands the schema changes.

Due to the tight coupling between the ClairCore version and schema, Scanner created one bundle per StackRox release.

Each bundle is created with an Exporter running the same ClairCore version used to build the release, ensuring compatibility.  The Exporter serves bundles at `https://definitions.stackrox.io/v4/<version>/vulnerabilities.zip` where `<version>` is the supported StackRox release (such as "4.5.0").  Scanner uses its release version tagged during build time to determine which URL to fetch from.

In offline mode, one offline bundle per release was introduced for Scanner V4.  Coexisting with the "latest" Scanner V2 offline bundles.  To facilitate user migration to per-release offline bundles, the latest offline bundle used by Scanner V2 (at the time of this writing served at https://install.stackrox.io/scanner/scanner-vuln-updates.zip) was changed to contain Scanner V4's offline bundles for `4.[45][.x]` releases.  A command `roxctl scanner download-db` was created to facilitate users fetching the right offline bundle for their release.

This approach needs to be revised for the following reasons:

1.  The vulnerability schema rarely changes, but maintaining multiple bundles is expensive.  It requires multiple GitHub workflow runs, which increases the chances of failure and pressure on the security sources' endpoints.
2.  It can be challenging to backport fixes and updates to previous releases.  This requires patch releases, which trigger the release workflow.  Improving the release process can be complex, as it may require patch release cuts, changes to different branches, and integration between logic that is managed in the master branch, executing other releases binaries.
3.  The latest offline bundles does not support post `4.5` releases.  Bundle embedding for each release increases the cost to store and consume offline bundles.  The `roxctl` interface and documentation today needs to be clearer on these limitations.

## Decision

We will implement a versioning stream for vulnerability bundles, bumped when we determine that ClairCore's schema has changed.

Initially, two version streams will exist (table 1 below):

| Stream | Git Ref      | Description                                                        |
|--------|--------------|--------------------------------------------------------------------|
| dev    | heads/master | This stream tracks the latest vulnerability schema                 |
| v1     | heads/4.4.5  | This stream tracks the schema used by the current StackRox release |

On the subsequent releases (for example,. `4.4.6`), one of the two will be carried out:

1.  The schema hasn't changed, so v1 now tracks the new release (`heads/4.4.6`).
2.  The schema has changed, so v1 stays at the current release (`heads/4.4.5`), and a new version, v2, is created to track the next release (`heads/4.4.6`).

### Building versioned bundles

The GitHub workflow will have a map with "bundle version" as keys and "git reference" as values, similar to Table 1 above.  This can be stored statically in the matrix definition.

For each version in the mapping, a bundle will be built based on the specified git ref and stored at `https://definitions.stackrox.io/v4/<bundle-version>/vulnerabilities.zip`.

In every release cut, the key specified in `scanner/VULNERABILITY_VERSION` will be updated in the mapping to the current release branch reference (e.g., `heads/4.4.6`).  Notice that if the version is the same, it will be updated.  If it's new, it will be added, and the previous version will stay.

Offline bundles per release will not be created anymore, but per bundle version.  The latest offline bundle will ship all bundle versions currently supported by StackRox.

### Ingesting versioned bundles

Scanner will know the vulnerability version it requires based on hard-coded information added during build time.  A file at `scanner/VULNERABILITY_VERSION` will determine the version on release builds and nightlies.  On non-release builds this will hard-coded to "dev".

The definitions handler in Central can be kept the same.  It already supports arbitrary versions in the GET request from Scanner.  [The check for nightlies](https://github.com/stackrox/stackrox/blob/e332cd19a639f59d9931414a4f4e561981396dad/central/scannerdefinitions/handler/handler.go#L661) and other per-release checks in the ingestion path are unnecessary and will be removed.

The bundle version will be configured in the vulnerability URL by a new variable called `ROX_VULNERABILITY_VERSION` to specify the expected bundle version to fetch.  The previous variable `ROX_VERSION` will be left for backward compatibility.

The vulnreability bundle will include a `manifest.json` file, which will carry the StackRox version and bundle version used to build it.  This file will be used by Scanner to prevent ingestion of unsupported bundle versions.

In offline mode, the `roxctl` method for fetching bundles will be modified to fetch only the latest bundle.

### Changing schema

When a schema bump is detected (e.g., a ClairCore bump that brings in incompatible schema changes to the vulnerability structs), `scanner/VULNERABILITY_VERSION` should be updated to the new schema version.

Once a StackRox release is out of support, we will remove the entry from the mapping after a grace period, effectively turning off updates for that stream.  The grace period duration will be documented but not specified on this decision record.

## Consequences

- Only necessary vulnerability fetching and parsing workflows will run at every update interval.  This is likely to happen once or twice until better schema handling is implemented in ClairCore.
- Only necessary bundle formats and versions will exist.
- Minimal offline bundle size, ability to deprecate versioned offline bundles.
- Updates to the Exporter can happen outside the release cycle.  No need to cut patch releases to update previous versions.  Moving the git ref to a separate branch or adding commits to the release branch without releasing them would suffice.  It is OK to wait until the next patch release since the Exporter runs in the GitHub workflow.
- We will continue to maintain multiple workflows, so there are still some potential points of failure and higher maintainability costs.
- It's possible that a Scanner deployment tracking "dev" bundles will get backward incompatible vulnerability updates, but that's a exception since "dev" is considered unstable and used for development.
- It's possible to bump schemas in between patch releases: the last bundle version would be tied to `tags/<ref>` for that.
