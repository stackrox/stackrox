# Diagnose node repository and CPE assignments

The temporary instrumentation is disabled unless `ROX_NODE_INDEX_DEBUG_DIR` is
set. It captures indexing evidence in the Compliance container. It does not
change repository selection, package filtering, CPE mapping, or matching.

A node can have several CPEs. RPM product CPEs come from repository metadata and
the repository-to-CPE mapping. The synthetic RHCOS image package has a separate
CPE derived from image labels or os-release. Investigate these paths separately.

## Enable and extract captures

Build the main image from this branch using your normal build and image
distribution workflow. Deploy that image as the **compliance** container in the
secured cluster's Collector DaemonSet. Scanner itself does not need this patch.
Enable Scanner V4 node scanning in the ACS deployment.

These commands are for the disposable reproduction cluster. They roll the
Collector DaemonSet. Set the namespace and image to your deployment's values:

```sh
NS=stackrox
MAIN_IMAGE='<main-image-containing-this-patch>'
oc -n "$NS" set image daemonset/collector compliance="$MAIN_IMAGE"
oc -n "$NS" set env daemonset/collector --containers=compliance \
  ROX_NODE_INDEX_DEBUG_DIR=/tmp/node-index-debug \
  ROX_NODE_INDEX_CACHE_PATH=/tmp/node-index-debug-cache \
  ROX_NODE_INDEX_CACHE_DURATION=0s \
  ROX_NODE_SCANNING_MAX_INITIAL_WAIT=0s
oc -n "$NS" rollout status daemonset/collector
```

The distinct cache path avoids serving an existing cached report. Changing the
cache duration alone does not expire a report already on disk: its expiry was
stored when it was created. A capture is produced only when indexing runs;
cache hits do not run these hooks. Later scans still follow the configured
scanning interval.

Select the Collector pod on the node you are investigating:

```sh
NODE='<node-name>'
oc -n "$NS" get pods --field-selector "spec.nodeName=$NODE" -o wide
POD='<collector-pod-on-that-node>'
oc -n "$NS" logs "$POD" -c compliance | rg 'Node index diagnostics'
oc -n "$NS" exec "$POD" -c compliance -- ls /tmp/node-index-debug
oc -n "$NS" cp -c compliance "$POD:/tmp/node-index-debug" ./node-index-debug
oc -n "$NS" exec "$POD" -c compliance -- cat /tmp/node-index-debug-cache > node-index-cache.json
```

Each `scan-<UTC-time>-<random>` directory contains:

- `scan.json`: node name, effective paths, package filter, mapping source, and
  embedded Go build information, including dependency versions when available.
- `index-root/` and `identity-root/`: snapshots of content manifests, image
  labels, os-release, and DNF history databases visible through each filesystem.
  `*-files.json` records missing files and capture errors. `*-manifest-globs.json`
  records manifest candidates.
- `mapping-response.json`: response bytes consumed by the actual repository
  scanner request, with HTTP status and selected response metadata in
  `mapping-http.json`. This does not issue a second download. A local seed
  mapping, if configured, is captured as `mapping-seed.json`. On an HTTP failure
  or seed fallback, inspect both sources and `mapping-http-error.json`.
- `repositories.json`: scanner output with original repository IDs encoded in
  the URI's `repoid` query parameter and the resulting CPEs.
- `packages-raw.json`: RPM scanner output, including DNF-enriched
  `repository_hint`, source package, EVR, architecture, and package database,
  before ACS filtering, deduplication, and ID assignment.
- `packages-filtered.json`: packages retained by ACS with assigned IDs.
- `coalesced.json` and `rpm-assignments.json`: RPM package environments and
  their assigned repositories, with the coalescer rule explained per package.
- `rhcos-identity.json`: selected image identity source, name, version, and CPE,
  or the error preventing image package creation.
- `index-report.json`: the final V4 report returned to the caller, serialized
  with Go's JSON encoder, like the report inside the existing node-index cache.
  `error.json` records indexing failures; earlier stages remain available.

Captures use private file permissions and accumulate until you remove them or
the pod's temporary volume disappears. They can contain a large package inventory
and DNF history. Copy them before replacing the pod. A failed diagnostic write
does not fail indexing; inspect warnings and file status before relying on an
incomplete capture.

## Trace one RPM step by step

1. Read `scan.json`. In the current OCP chart, the RPM scanner sees `/hostindex`
   and image identity sees `/host`. The `/hostindex` mount exposes host
   `/usr/share` and `/usr/lib`; it does not mount `/var/lib/dnf`. Confirm the
   running pod's mounts and the capture's visibility results before concluding
   that host DNF history was used.
2. Find the manifest selected by ClairCore. It takes the first lexically sorted
   match of the first nonempty glob, in this order:
   `root/buildinfo/content_manifests/*.json`,
   `usr/share/buildinfo/content_manifests/*.json`, `usr/share/buildinfo/*.json`.
   The last glob also matches labels and other JSON files. Inspect the actual
   selected file, rather than assuming `content_manifest.json` was selected.
3. If there is no manifest, or `from_dnf_hint` is true, repository IDs and
   package hints come from DNF history. If that field is absent or false in a
   valid manifest, repository IDs come from `content_sets` and DNF enrichment
   is skipped. A JSON syntax error is treated as no manifest by the pinned
   ClairCore; other decoding errors can fail indexing.
4. For each discovered repository ID, look up `data[repoid].cpes` in the mapping
   used by the scan. Unknown repository IDs contribute no repositories. Invalid
   CPEs are skipped. `repositories.json` shows the accepted results, with the
   original repository ID in each URI.
5. Find the installed package in `packages-raw.json`. This is the RPM database
   inventory, enriched with DNF provenance only when step 3 selected DNF. DNF
   history is provenance, not the authoritative list of currently installed
   packages. Follow it through `packages-filtered.json` to check filtering or
   duplicate collapse.
6. Read its entry in `rpm-assignments.json`. If its URL-encoded hint contains
   `repoid`, only repositories with that same decoded URI parameter are
   assigned. A hint whose repository has no mapping can leave the package with
   no repositories. With no `repoid` hint, **all repositories in the layer are
   assigned**. The coalescer uses the first value if the hint has several.
7. Compare those CPEs with `index-report.json`. For a missing RPM vulnerability,
   then inspect Scanner's loaded advisory products, package names/kinds, and
   version ranges. An assigned CPE alone does not prove that a matching advisory
   exists. Preserve the advisory bundle/update identity as part of the repro.

For example, after selecting a local capture directory:

```sh
SCAN='./node-index-debug/<scan-directory>'
jq --arg name glibc 'to_entries[] | select(.value.name == $name)' \
  "$SCAN/rpm-assignments.json"
jq --arg name glibc '.[] | select(.name == $name)' "$SCAN/packages-raw.json"
REPO='<decoded-repoid-from-the-package-hint-or-repository-URI>'
jq --arg repo "$REPO" '.data[$repo]' "$SCAN/mapping-response.json"
```

The implementation being traced is in the pinned ClairCore dependency:
`rhel/rhel.go`, `rhel/repositoryscanner.go`, `rhel/packagescanner.go`,
`internal/dnf/dnf.go`, `internal/dnf/sql/`, and `rhel/coalescer.go`.
ACS applies its package filter and adds synthetic image packages in `indexer.go`.

## Browse DNF history

ClairCore prefers `usr/lib/sysimage/libdnf5/transaction_history.sqlite`, then
`var/lib/dnf/history.sqlite`, relative to the **index root**. It chooses the
query family from the database tables, not the filename: both `config` and
`trans_item_action` mean DNF5; otherwise it uses DNF4.

Use a captured database on your workstation. The helper uses only Python's
standard library and opens SQLite in read-only mode:

```sh
DB="$SCAN/index-root/var/lib/dnf/history.sqlite"
python3 compliance/node/index/inspect_dnf.py "$DB" \
  --table repo --table rpm --table trans_item > dnf-raw.json
```

For a DNF5 database, point `DB` to the captured libdnf5 path instead. The output
includes schemas, raw repository rows, and the exact repository selection query
and its results. For a single installed RPM, pass its **separate** NEVRA fields
(take the values from that node, rather than this illustrative example):

```sh
python3 compliance/node/index/inspect_dnf.py "$DB" \
  --package glibc 0 2.34 100.el9 x86_64 > glibc-dnf.json
```

This prints the parameterized query, matching repository IDs, and every
transaction row for the matching RPM item. The query mirrors ClairCore
`5fced1da0e77`: DNF4 selects the maximum transaction-item ID per item after
excluding `action = 8`; DNF5 excludes action name `Remove`. It does not first
select the latest row and then discard removals. A per-package lookup matches
name, epoch, version, release, and architecture exactly. No matching row leaves
the package's hint unchanged. Source hints inherit the binary package lookup.

To browse interactively if you have the SQLite CLI:

```sh
sqlite3 -readonly "$DB"
.tables
.schema repo
.schema rpm
.schema trans_item
.headers on
.mode column
SELECT * FROM repo;
```

If the DB exists only under `identity-root`, it was visible through the host
root but not necessarily available to RPM indexing. Do not treat its rows as
inputs to that scan. Compare both roots before testing a different mount layout.

The snapshots are sequential copies, not transactional backups. ClairCore itself
copies the main DNF database file before querying it and does not copy its WAL.
The instrumentation also captures visible `-wal` and `-shm` files. For a stable
repro, collect when no package transaction is running. To emulate ClairCore's
main-file-only view, query a separate copy of just the `.sqlite` file; retain the
full capture for comparison with a WAL-aware SQLite reader.

## Compare with the host

OpenShift's supported node debug workflow mounts the host at `/host`:

```sh
oc debug "node/$NODE"
chroot /host
cat /etc/os-release
cat /usr/share/buildinfo/labels.json
ls -l /var/lib/dnf/history.sqlite /usr/lib/sysimage/libdnf5/transaction_history.sqlite
rpm -qa --qf '%{NAME}\t%{EPOCHNUM}\t%{VERSION}\t%{RELEASE}\t%{ARCH}\n'
```

Some files can be absent. Use this to establish host facts, then compare with
what the Compliance container can read. The helper and captures avoid needing
SQLite or DNF executables installed on the immutable node. See the
[OpenShift 4.19 node troubleshooting documentation](https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html-multi/support/troubleshooting).

For incident tracking, retain the exact OCP patch release, node name/UID,
architecture, booted OS image identity, ACS revision and image digests, selected
scan directory, mapping bytes, advisory update identity, and a specific expected
RPM/image CVE outcome. OCP 4.19 alone is not a complete reproduction description.
