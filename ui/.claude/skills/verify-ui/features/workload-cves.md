# Workload CVEs

Vulnerability Management > Results shows image CVEs for user workloads, platform components, and other image views. Users switch between CVE, Image, and Deployment tables, filter by severity and fixability, and open a CVE, image, or deployment detail page.

## Sub-features

- `wcve-nav` opens Results from the left nav (User Workloads view).
- `wcve-views` switches between User Workloads, Platform, and the `More Views` options (All vulnerable images, Inactive images, Images without CVEs).
- `wcve-entity-tabs` toggles the CVE, Image, and Deployment tables.
- `wcve-cve-detail` opens a CVE page.
- `wcve-image-detail` opens an image page with Vulnerabilities and Resources tabs.
- `wcve-deployment-detail` opens a deployment page.

## How to get to it (user POV)

- Left nav > `Vulnerability Management` > `Results` (`/main/vulnerabilities/user-workloads`).
- Horizontal subnav: `User Workloads`, `Platform`, `Nodes`, `Virtual Machines`, plus the `More Views` dropdown.
- Deep links: `/main/vulnerabilities/<view>/cves/<cveId>`, `/images/<imageId>`, `/deployments/<deploymentId>`.

## Driving it with verify-ui

Preconditions:

- `doctor.sh` healthy and the demo's scanner has finished its first image scans (Platform view shows CVEs).
- Import `visitFromLeftNavExpandable` from `cypress/helpers/nav.ts`, `getRouteMatcherMapForGraphQL` and `interactAndWaitForResponses` from `cypress/helpers/request.js`, and `selectors` from `cypress/integration/vulnerabilities/vulnerabilities.selectors.ts`.
- Do not use `visitWorkloadCveOverview` or the `interactAndWaitFor*List` helpers in `workloadCves/WorkloadCves.helpers.js`. They stub with fixtures.

- **Open from nav.** `visitFromLeftNavExpandable('Vulnerability Management', 'Results', getRouteMatcherMapForGraphQL(['getImageCVEList']))`. Expect `h1:contains("User workload vulnerabilities")`. `captureState('01-user-workloads')`.
- **Platform view.** `interactAndWaitForResponses(() => cy.get('nav.pf-m-horizontal.pf-m-subnav a:contains("Platform")').click(), getRouteMatcherMapForGraphQL(['getImageCVEList']))`. Expect `h1:contains("Platform vulnerabilities")` and `tbody tr` length > 0. `captureState('02-platform-cves')`.
- **Image tab.** `interactAndWaitForResponses(() => cy.get(selectors.entityTypeToggleItem('Image')).click(), getRouteMatcherMapForGraphQL(['getImageList']))`. The `td[data-label="Image"] a` rows render. `captureState('03-images')`.
- **Image detail.** Read the first `td[data-label="Image"] a` text as `imageName`, then click it inside `interactAndWaitForResponses(cb, getRouteMatcherMapForGraphQL(['getImageDetails']))`. Expect `h1` to contain the image name's repository part, the pathname to contain `/images/`, and `button[role="tab"]:contains("Vulnerabilities")`. `captureState('04-image-detail')`.
- **CVE detail.** Go back to the CVE tab, read the first `td[data-label="CVE"] a` text as `cveId`, then click it inside `interactAndWaitForResponses(cb, getRouteMatcherMapForGraphQL(['getImageCveMetadata']))`. Expect `h1:contains(cveId)` and the pathname to contain `/cves/`. `captureState('05-cve-detail')`.

## Gotchas

- User Workloads can be empty on a fresh demo. Use the Platform view for guaranteed data.
- Entity toggle labels include counts (`12 CVEs`); `:contains("Image")` still matches.
- Persisted default filters (severity/fixability) change results between runs. Clear them with `selectors.clearFiltersButton` if counts look odd.
- `ROX_FLATTEN_IMAGE_DATA` changes the image GraphQL response shape (`imageV2`), and its opnames may differ. Check `CYPRESS_ROX_FLATTEN_IMAGE_DATA` in the run env.
- `Virtual Machines` and `Kubernetes components` entries may be flag-gated. Check the screenshot before asserting them.
- Namespace view needs Namespace read access; the Scanner V4 banner shows with Integration read access.
- Autocomplete filters debounce 500ms.
