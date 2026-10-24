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
//  3. Obtain CVEs for the installed-bundle images that are used in the cluster (those the
//     InstalledImageSource/ACS already has scanned) and, restricted to that same
//     (repository, name) set, scan the matching update-candidate images. Unused bundle
//     images (e.g. the many Istio versions a multi-version bundle ships but the cluster does
//     not run) are neither scanned nor diffed. When a DeployedImageSource is configured
//     (WithRunningOnly), the set is further restricted to images referenced by a currently
//     running deployment.
//  4. Diff the CVEs per image, pairing images by their (repository, bundle name) key so that
//     multi-version bundles (many images sharing one repository) diff correctly, classifying
//     each CVE as fixed, still active, or newly introduced.
//
// All external data access is expressed through the CatalogClient, ImageScanner, and
// InstalledImageSource interfaces so the package carries no transport dependencies; the
// tools/operatorbundle CLI provides live (StackRox Central + Red Hat catalog)
// implementations and tests provide in-memory fakes.
package operatorbundle
