# 0012 - Re-indexing 

- **Author(s):** Ross Tannenbaum
- **Created:** [2024-09-03 Tue]

## Context

Scanner V4 clients (Central and Sensor) utilize the [client library provided in the pkg/ directory](https://github.com/stackrox/stackrox/blob/master/pkg/scannerv4/client/client.go) for all Scanner V4 communication.

Both (current) methods which potentially index images (`GetOrCreateImageIndex` and `IndexAndScanImage`) call Scanner V4's `GetOrCreateIndexReport` RPC which first checks if the image has already been indexed.
If it has already been indexed, then it simply returns the preexisting index report.

This is a problem, as there are several reasons why re-indexing is required:

* A ClairCore "versioned scanner" is updated.
  * A "versioned scanner" is either a package scanner (identifies things like RPMs, dpkgs, Go binaries, etc), repository scanner (identifies things like the CPEs for the related Red Hat repositories), or a distribution scanner (identifies the base operating system for the image). When one of these is updated (with perhaps a bug fix), it is best to re-index the image to ensure all updates are reflected in the Index Report.
* There is a bug fix in ClairCore outside the "versioned scanners"
  * For example, [ClairCore versions up to v1.5.27 were unable to follow hardlinks in image layers](https://github.com/quay/claircore/pull/1305). This was caught when indexing an image whose base operating system information was found in a hard-linked file. So, indexing prior to this resulted in an unknown operating system. Scanner V4 should re-index affected images.
* Required mapping data was not available during indexing
  * For example, it may be possible an image is indexed before it's related repository CPEs were made available [here](https://security.access.redhat.com/data/metrics/repository-to-cpe.json). Once the data is available, the image should be re-indexed to ensure it's complete.

Talking with the Clair team and inspecting the [Quay](https://github.com/quay/quay) and [Clair](https://github.com/quay/clair/blob/v4.7.4/httptransport/indexer_v1.go#L175) codebases, it is clear Clair has a built-in way to handle "versioned scanner" updates:
it is up to the client (typically Quay) to track the Indexer's state (which is a hash of all the "versioned scanners" and their respective versions) returned by Clair to the client via the `etag` HTTP header alongside the Index Report.
The client then sends that state back to Clair in an `If-None-Match` header, so it may determine if there have been any updates to the "versioned scanners" since indexing the image.
When done this way, the client does not need to generate a [`*claircore.Manifest`](https://github.com/quay/claircore/blob/v1.5.29/manifest.go#L5), which may be expensive.

It is also possible to forgo this state check and just request Clair to index the image again. ClairCore will check if an Index Report already exists, which "versioned scanners" were used, and compare them to the current versions.
ClairCore will then only re-index if there is a difference in "versioned scanner" versions and only with the scanners which updated.
However, this requires the client to generate a `*claircore.Manifest`, which, again, may be expensive. In StackRox's case, this may be expensive because
StackRox must reach out to the registry several times to determine the fully resolved layer URIs.
When a registry does not handle the `Range` header (Scanner V4 Indexer uses `Range: bytes=0-0`), then this is even more expensive, as the entire layer may potentially be sent over by the registry.

There is currently no built-in way for Clair to handle the other two mentioned reasons aside from deleting the Index Report and then requesting Clair to Index the image.

Related: Scanner V4 never deletes Index Reports, even when StackRox Central/Sensor no longer tracks the image, so the database is ever-growing.
This is a problem we would like to solve here, as well.

## Decision

We believe it is in our best interest to account for each of the re-indexing reasons listed above, which will be done as follows:

### Reuse ClairCore's 'checkManifest' Step

The first step of indexing is the [`checkManifest`](https://github.com/quay/claircore/blob/v1.5.29/indexer/controller/checkmanifest.go#L25) step.
This step checks if the image (identified by the passed in `*claircore.Manifest`) has already been indexed and with which "versioned scanners", so it can determine if there is a need to re-index.
This is all done requiring only the `*claircore.Manifest.Hash` field, which is rather easy to obtain. We copy the relevant parts of this (since it is private)
into our Indexer logic.

For example, it may be done via the following:

```go
package example

import (
    "context"
    "log"

    "github.com/quay/claircore"
    "github.com/quay/claircore/indexer"
    "github.com/quay/claircore/libindex"
)

func Example() {
    // Assuming there is already some instance of libindex.Libindex and a claircore.Digest
    var i *libindex.Libindex
    var digest claircore.Digest

    // Get all current versioned scanners. 
    pscnrs, dscnrs, rscnrs, fscnrs, err := indexer.EcosystemsToScanners(context.Background(), i.Ecosystems)
    if err != nil {
        log.Fatal(err)
    }
    vscnrs := indexer.MergeVS(pscnrs, dscnrs, rscnrs, fscnrs)

	// Check if the manifest was indexed with the latest versioned scanners.
    ok, err := i.Store.ManifestScanned(context.Background(), digest, vscnrs)
    if err != nil {
        log.Fatal(err)
    }
    if ok {
        // Image already indexed with latest versioned scanners.
    } else {
        // Image was either not previously indexed, or it was with older versioned scanners.
    }
}
```

This method does not require any kind of migration to account for previously indexed images, as the data/tables already have all required the information.

#### Alternative - Tracking Indexer State

Scanner V4 Indexer would maintain a table which tracks each successful Index Report (identified by the report's hash ID) 
alongside the Scanner V4 Indexer's state (equal to the `etag` sent by Clair).

This way, when clients reach out to Scanner V4 Indexer, it may first check not only if the image was already indexed,
but also the state of the Indexer at the time of indexing. If they do not match, then we will create the full `*claircore.Manifest`
and re-index the image. As mentioned in the [Context](#context) section, ClairCore already has some builtin performance optimizations for this.
Another benefit to this that `*claircore.Manifest` continues to only be generated, as needed.

However, there are cons which outweigh the pros:

* More data for Scanner V4 to track outside ClairCore
  * This is not a blocker, but it is worth mentioning
* Handling preexisting Index Reports
  * This is the challenging aspect, which would block this
  * There are a few options:
    * Write a migration
      * This involves doing essentially the same thing `ManifestScanned` does to determine the "versioned scanners"
      * This may not be able to be pure PostgeSQL, as the `etag` is generated in Go, so that is an extra challenge
    * Populate the table upon Index Report fetching
      * If the entry does not exist in the table (the case for all Index Reports created prior to this release), then create the Index Report and populate the table
      * This essentially forces a re-indexing of all previously indexed images, whether it is required or not, which may be costly

### Random Deletions

When it comes to variables outside "versioned scanners" there is no currently known clean way to handle the re-indexing.

Something like creating our own "versioned scanner" to track other types of changes is not a solution:

* Updating the version of the custom "versioned scanner" will only affect that scanner, and not the others which would all be skipped upon re-indexing
* It is not feasible to track every single change in ClairCore which may affect Index Reports

Instead, we will randomly delete manifests from the ClairCore database. Deleting a manifest from ClairCore
deletes all data related to it, including the Index Report.
Doing this solves both the remaining aspects of the re-indexing problem and the database ever-growing problem,
as unneeded Index Reports are permanently deleted and needed reports may just be regenerated.

**Note**: this approach also solves the "versioned scanner" update scenario; however, we opt to implement the [`checkManifest`](#reuse-claircores-checkmanifest-step)
step for faster turnaround upon "versioned scanner" updates.

For each manifest/image, once it is successfully indexed, a random expiry will be generated and associated with
the manifest. Of course, determining a timeframe for deletion is non-trivial. Deleting too quickly means Index Reports may be corrected
faster, but too many (probably) unnecessary requests to the image registry and (probably) unnecessary CPU/memory/disk usage.
Deleting too slowly means resource usage will be much less, but it may take (too) long to correct the Index Report.
Similarly, we would need to ensure not all Index Reports are all re-indexed at the same time (hence random deletions)
to ensure spikes are minimized.

The expirations will be chosen randomly from a range between **one week** and **one month**. We believe this is a sufficient timeframe
which balances speed (a manifest will always be re-indexed within a month of its previous indexing) and resource consumption
(choosing a random time within just a wide range will hopefully spread out the resource usage evenly and reduce massive spikes).

This will all be tracked in a new table (see [Manifest Metadata Table](#manifest-metadata-table) below).

The table will be queried periodically for passed timestamps via goroutine. These rows will be deleted and the related manifests/Index Reports will be deleted from ClairCore's tables.
In the chance the manifest is re-indexed but the row still exists in the Manifest Metadata table, the row will be updated with a new expiration timestamp.

#### Alternative - Client Delete Requests

Scanner V4 clients (Central and Sensor) may inform the Indexer of an image deletion and request the Index Report deleted.

However:

* It is non-trivial to pinpoint all places where images are deleted in Central and Sensor
* The proposed method is simpler, as it is all done from within the Indexer

### Manifest Metadata Table

The new Manifest Metadata table will look like the following:

| manifest_id   | expiration                    |
|---------------|-------------------------------|
| sha512:abc... | 2024-09-26 23:52:39.190285-07 |
| sha512:def... | 2024-09-12 05:48:46.361476-07 |

The `manifest_id` will be all that is needed to identify and delete a manifest and Index Report.
Note this ID is the same as the Index Report's ID which is the same as the related `*claircore.Manifest.Hash` (hence `manifest_id`).

### Handling preexisting Index Reports

A PostgreSQL migration is possible, but it may prove to be complicated:
we would need to ensure the ClairCore Indexer manifest table schema does not change.
This may be done by injecting our migration into ClairCore's migrations, but it would be best to not interfere with
ClairCore internals like this.

Instead, upon Indexer startup, and once the PostgreSQL migrations have been completed, the Indexer will populate the [`manifest_metadata`](#manifest-metadata-table)
with any manifests from ClairCore's `manifest` table missing from the `manifest_metadata` table. The `expiration` for
each manifest will be randomly generated.

### (Optional) Force Index Command

In case the seven to thirty days interval is too long, we may consider adding some way to force a re-indexing via `roxctl`.
The details for such is out of scope for this document, but it is worth mentioning, as it may prove to be useful
sometime in the future (ROX-26406).

## Consequences

* Yet another table to manage
* More storage space is required, though vulnerability data has significantly decreased with VEX, so this additional space may not be terrible
  * Plus the purpose of this space is for deletions, so this added space helps us sae space in the long-run.
  * Each row is on the order of tens of bytes in size, which is not a big deal.
* Scanner V4 will no longer track index reports for inactive images, which saves storage space, as the report will eventually be deleted.
* Re-indexing images indexed by older "versioned scanners" will not require re-downloading the entire image; however, re-indexing after random deletion will.
  * This puts more pressure on Scanner V4 Indexer's resources as well as the image registry.
  * This is meant to be alleviated by having a three-week timeframe to choose from when selecting a random expiration time.
* Scanner V4 clients do not need to change anything to support this new behavior, as all deletion and re-indexing decisions would be handled by Scanner V4 Indexer.
