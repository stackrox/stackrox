// Package operatorbundle computes, for a set of vulnerable images, which operator
// bundle update a customer should install to fix CVEs and exactly which CVEs that
// update fixes, leaves active, or newly introduces.
//
// The flow, orchestrated by Advisor, is:
//
//  1. Resolve the operator bundle that ships a vulnerable image (the "installed bundle")
//     from an operator catalog, keyed by image digest.
//  2. Find newer bundle versions of the same package/channel and select the latest patch
//     release within the installed bundle's major.minor (the "update candidate").
//  3. Obtain CVEs for every image of the update candidate (by scanning) and for every
//     image of the installed bundle (from already-scanned data).
//  4. Diff the CVEs per image, pairing images by repository name, classifying each CVE as
//     fixed, still active, or newly introduced.
//
// All external data access is expressed through the CatalogClient, ImageScanner, and
// InstalledImageSource interfaces so the package carries no transport dependencies; the
// tools/operatorbundle CLI provides live (StackRox Central + Red Hat catalog)
// implementations and tests provide in-memory fakes.
package operatorbundle
