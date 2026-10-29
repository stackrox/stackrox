# Dashboard

The landing page after sign-in. It shows summary count tiles (clusters, nodes, violations, deployments, images, secrets) with a "Last updated" time, plus widgets for violations by severity, recent alerts, images at most risk, and aging images.

## Sub-features

- `dashboard-load` renders the page with every widget request answered.
- `dashboard-counts` shows numeric summary tiles that link to their entity pages.
- `dashboard-nav` makes the left-nav `Dashboard` item current.

## How to get to it (user POV)

- Sign in; `/` redirects to `/main/dashboard`.
- Left nav > `Dashboard`.

## Driving it with verify-ui

Preconditions:

- `doctor.sh` healthy.
- Admin token (default), so every tile renders.

- **Load.** `visitMainDashboard()` from `cypress/helpers/main.js`. It visits `/main/dashboard`, waits on the GraphQL opnames `summary_counts`, `getAllNamespacesByCluster`, `alertCountsBySeverity`, `mostRecentAlerts`, `getImagesAtMostRisk`, and `agingImagesQuery` plus REST `deploymentswithprocessinfo` and `alerts/summary/counts`, then asserts `h1:contains("Dashboard")` and the current nav link. `captureState('01-dashboard')`.
- **Counts.** For each noun in `['Cluster', 'Node', 'Violation', 'Deployment', 'Image', 'Secret']`, `cy.get(\`main section:first-child a.pf-v6-c-button.pf-m-link:contains("${noun}")\`)` exists and its text matches `/^\d+ /`. `main section:first-child div:contains("Last updated")` exists.
- **Tile link.** Click the `Violation` tile. Expect the pathname to start with `/main/violations`. `captureState('02-violations-from-tile')`.

## Gotchas

- `visitMainDashboard` waits on every widget request. A slow or forbidden widget times out the whole visit; read `cypress.log` for the alias that never arrived.
- Tiles only render for resources the token can read. Admin sees all six.
- `getAggregatedResults` is also awaited when `ROX_DEPRECATED_COMPLIANCE_DASHBOARD` is on in Cypress env (drive.sh mirrors Central's flags).
- Counts move as the demo runs. Never assert exact values.
