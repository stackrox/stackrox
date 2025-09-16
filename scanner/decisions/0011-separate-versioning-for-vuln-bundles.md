# 0011 - Separate Versioning Stream for Vulnerability Bundles

- **Author(s):** J. Victor Martins <jvdm@sdf.org>
- **Created:** [2024-07-22 Mon]

## Status

Status: Accepted

Updates: [#0009](0009-scannerv4-read-manual-vulnerabilities.md)

## Context

A Vulnerability Bundle is an archive containing updates to vulnerability data in Scanner V4 (aka. Scanner).  They are constantly updated by the Vulnerability Exporter (aka. Exporter), which runs periodically in GitHub Workflows and uses ClairCore to fetch the security data to convert it to a JSON-based schema.  Then, the workflow bundles the data into a ZIP archive.  ClairCore owns the schema.

The Scanner Matcher, using ClairCore, runs the Vulnerability Updater (aka. Updater), which fetches the bundle, parses the data, and updates the vulnerability database.

Suppose a backward-incompatible schema change is made in ClairCore and incorporated into Scanner.  In that case, the following Exporter execution will update the bundle adhering to the new schema.  Existing Scanner instances can only parse the changed data if they upgrade to the new ClairCore version that understands the schema changes.[^1]

Due to the tight coupling between the ClairCore version and schema, one bundle per StackRox release was introduced for Scanner V4.

Each bundle is created with an Exporter running the same ClairCore version used to build the release, ensuring compatibility.  The Exporter serves bundles at `https://definitions.stackrox.io/v4/<version>/vulnerabilities.zip` where `<version>` is the supported StackRox release (such as "4.5.0").  Scanner uses its release version tagged during build time to determine which URL to fetch from.

In offline mode, one offline bundle per release was introduced for Scanner V4, coexisting with the "latest" Scanner V2 offline bundles.  To facilitate user migration to per-release offline bundles, the latest offline bundle used by Scanner V2 (at the time of this writing served at https://install.stackrox.io/scanner/scanner-vuln-updates.zip) was changed to contain Scanner V4's offline bundles for `4.[45][.x]` releases.  A command `roxctl scanner download-db` was created to facilitate users fetching the right offline bundle for their release.

This approach needs to be revised for the following reasons:

1.  The vulnerability schema rarely changes, and maintaining multiple bundles is expensive.  It requires multiple GitHub workflow runs, which increases the chances of failure and pressure on the security sources' endpoints.
2.  It can be challenging to backport fixes and updates to previous releases.  This requires patch releases, which trigger the release workflow.  Improving the release process can be complex, as it may require patch release cuts, changes to different branches, and integration between logic that is managed in the master branch, executing other releases binaries.
3.  The latest offline bundles does not support post `4.5` releases.  Bundle embedding for each release increases the cost to store and consume offline bundles.  The `roxctl` interface and documentation today needs to be clearer on these limitations.


## Decision

We will implement a versioning stream for vulnerability bundles, bumped when we determine that ClairCore's schema has changed.

Initially, two version streams will exist (table 1 below):

| Stream | Source Ref   | Description                                                     |
|--------|--------------|-----------------------------------------------------------------|
| dev    | heads/master | This stream tracks the latest vulnerability schema              |
| v1     | 4.5.2        | This stream tracks the schema used by the last StackRox release |

On the subsequent releases (for example, `4.6.0`), one of the two will be carried out:

1.  The schema hasn't changed, so v1 now tracks the new release branch (`tags/4.6.0`).

   | Stream | Source Ref   |
   |--------|--------------|
   | dev    | heads/master |
   | v1     | 4.6.0   |

2.  The schema has changed, so v1 stays at the current git ref (`tags/4.5.0`), and a new version, v2, is created to track the next release (`tags/4.6.0`).

   | Stream | Source Ref   |
   |--------|--------------|
   | dev    | heads/master |
   | v1     | 4.5.2        |
   | v2     | 4.6.0        |

This process could be either manual or automated.

### Building versioned bundles

#### Online mode

The GitHub workflow will have a map with "bundle version" as keys and "source reference" as values, similar to Table 1 above.  This map can be stored statically in the matrix definition or as a file in the source repository.

A source reference can be one of the following:

1.  A branch reference, `heads/<branch-name>`, example: `heads/release-4.6`
2.  A tag reference, `tags/<tag-name>`, example: `tags/4.6.0`.
3.  A StackRox release, `<stackrox-release>`, example: `4.6.0`

Both (1.) and (2.) will determine a specific commit to use to build the vulnerability exporter.[^2]

(3.) will select a desired release build based on a StackRox target release.  The workflow will find the best matching Git tag for the specified `<stackrox-release>`, by checking if the version exists as a tag or matching it against release candidate (`-rc`) or patch (`.x`) tags.

For each version in the mapping, a bundle will be built based on the determined git ref and stored at `https://definitions.stackrox.io/v4/<bundle-version>/vulnerabilities.zip`.

In every release cut, the key specified in `scanner/VULNERABILITY_VERSION` will be updated in the mapping to the current release reference (e.g., `4.6.0`).  Notice that the key is still updated if the version is the same.   If it's new, it will be added, and the previous version will stay.

For example, in release cut 4.6 we would look into `scanner/VULNERABILITY_VERSION`, let's say its contents are:

```
$ cat scanner/VULNERABILITY_VERSION
v1
```

The mapping should be updated so `v1` points to the current release branch reference.  Let's say the mapping was:

bundle version | git reference
--- | ---
`dev` | `master`
`v1` | `4.5.0`

Then, `v1` is updated to the new release (in this example, `4.6.0`):

bundle version | git reference
--- | ---
`dev` | `master`
`v1` | `4.6.0`

If in 4.6 the bundle version was updated to:

```
$ cat scanner/VULNERABILITY_VERSION
v2
```

On 4.6.0 release cut, the final mapping would be:

bundle version | git reference
--- | ---
`dev` | `master`
`v1` | `4.5.0`
`v2` | `4.6.0`

#### Offline mode

Per-release offline bundles will continue to be published.  They will be versioned by StackRox releases, but they will not be based on Y-stream (`4.5`) but rather on Z-stream patch releases (`4.5.0`).  The bundle will contain the correct vulnerability bundle for that particular Z-stream release.  The `manifest.json` file will continue to have a "version" attribute, but it shows the vulnerability schema version. Except for `4.[45].x` releases, where it points to the Y-stream of that offline bundle for backward compatibility.  An additional field `release_versions` will provide the full list of Z-stream releases supported by this bundle.  For example, for patch release `4.6.1` the offline bundle could be:

```
{
  "version": "v1",
  "created": "2024-08-20T21:00:30+00:00",
  "release_versions": ["4.6.0", "4.6.1", ...],
}
```

The central definitions handler will validate the offline bundle POST against `release_versions` to ensure the running Central version is supported.  The same will happen on offline bundle GET, but against `vulnerability_version`.  This will guarantee the offline bundles are supported by the Scanner's schema version.

We will continue to offer the latest offline bundles.  They will contain all offline bundles versioned by schema, and the existing `4.5` bundle to continue to be added for backward compatibility.  Example:

```
unzip -l scanner-vuln-updates.zip
Archive:  scanner-vuln-updates.zip
  Length      Date    Time    Name
---------  ---------- -----   ----
171516200  2024-08-20 21:00   scanner-defs.zip
   248088  2024-08-20 21:00   k8s-istio.zip
191673607  2024-08-20 21:00   scanner-v4-defs-v1.zip
191673030  2024-08-20 21:00   scanner-v4-defs-v2.zip
191673030  2024-08-20 21:00   scanner-v4-defs-4.5.zip
---------                     -------
363437895                     3 files
```

The `roxctl scanner download-db` will continue to support downloading the correct bundle per release using the `--version` command line option.

### Ingesting versioned bundles

Scanner will know the vulnerability version it requires based on hard-coded information added during build time.  A file at `scanner/VULNERABILITY_VERSION` will determine the version on release builds and nightlies.  On non-release builds this will hard-coded to "dev".

The definitions handler in Central can be kept the same.  It already supports arbitrary versions in the GET request from Scanner.  [The check for nightlies](https://github.com/stackrox/stackrox/blob/e332cd19a639f59d9931414a4f4e561981396dad/central/scannerdefinitions/handler/handler.go#L661) and other per-release checks in the ingestion path are unnecessary and will be removed.

The bundle version will be configured in the vulnerability URL by a new variable called `ROX_VULNERABILITY_VERSION` to specify the expected bundle version to fetch.  The previous variable `ROX_VERSION` will be left for backward compatibility.

The vulnerability bundle will include a `manifest.json` file, which will carry the StackRox version and bundle version used to build it.  This file will be used by Scanner to prevent ingestion of unsupported bundle versions.

### Changing schema

When a schema bump is detected (e.g., a ClairCore bump that brings in incompatible schema changes to the vulnerability structs), `scanner/VULNERABILITY_VERSION` should be updated to the new schema version.

Once a StackRox release is out of support, we will remove the entry from the mapping after a grace period, effectively turning off updates for that stream.  The grace period duration will be documented but not specified on this decision record.

## Consequences

- Reduces the number of vulnerability fetching and parsing in the Exporter workflows.  The number will drop to once or twice, as schema changes happen in ClairCore.
- Reduces the number of vulnerability bundles per version.
- Updates to the Exporter can happen outside the release cycle.  No need to cut patch releases to update previous versions.  Moving the git ref to a separate branch or adding commits to the release branch without releasing them would suffice.  It is OK to wait until the next patch release since the Exporter runs in the GitHub workflow.
- It's possible that a Scanner deployment tracking "dev" bundles will get backward incompatible vulnerability updates, but that's an exception since "dev" is considered unstable and used for development.  But, this will not affect nightlies anymore.
- It's possible to bump schemas in between patch releases: the last bundle version would be tied to `tags/<ref>` for that.
- The support for `<stackrox-version>` allows the mapping file to be updated once per Y-stream release bump rather than every Z-stream bump.  It also allows vulnerability builds to happen before the tag is cut (on RC releases, etc).
- Offering individual versioned offline bundles using StackRox releases simplifies the existing `roxctl download-db|upload-db` workflows, minimizing changes to the validation logic at the expense of additional storage consumption.  Alternatively, use vulnerability versions but serve (e.g., in GCP) the mapping to determine which offline bundle to use per release.  Validation would have to change in Central, and Scanner would have to serve its "supported" bundle version (through an API, etc).

[^1]: There are other changes in ClairCore updaters that can cause such incompatibilities.  For example, addition of security sources (and their corresponding matchers/updaters) will add information to the bundle that is unknown to previous releases, or removal of updaters in new releases would prevent updates to older relases.  To simplify the discussion in this ADR, any incompatible change in the security bundle content will be treated (and referenced to) as a "schema change".

[^2]: References using `heads/*` will actually pick the current commit references by that branch, but it's specific in the sense that one branch reference is unambiguous in a given point in time.
