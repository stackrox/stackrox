# Network Graph

Network Graph shows active deployment traffic for a selected cluster and one or more namespaces. Until scope is chosen, the page is an empty state. After scope, PatternFly topology chrome (graph, namespace groups, zoom toolbar, legend) appears.

## Sub-features

- `ng-nav` opens `/main/network-graph` from left nav **Network** → **Network Graph**.
- `ng-empty` shows the select-scope empty state.
- `ng-scope` selects a cluster and namespace (e2e uses `stackrox` when that namespace exists).
- `ng-chrome` shows topology toolbar **Zoom In**, **Zoom Out**, **Fit to Screen**, **Reset View**, **Legend**.

## How to get to it (user POV)

- Left nav: expand **Network** → **Network Graph**.
- Direct URL: `https://localhost:3000/main/network-graph`.
- Needs read on Deployment + NetworkGraph (`nonGlobalResourceNamesForNetworkGraph` in `routePaths.ts`).

## Driving it with Cypress e2e

Preconditions:

- Doctor READY.
- A secured cluster is in Central. Local `deploy-local` usually creates namespace `stackrox` with Central/Sensor workloads. If no cluster is registered, stop at the empty state and report **inconclusive** for `ng-scope` / `ng-chrome`.

- **Left nav.** `visitNetworkGraphFromLeftNav()` in `cypress/integration/networkGraph/networkGraph.helpers.js` (expandable **Network**, item **Network Graph**). Result: pathname `/main/network-graph`, title matches `Network Graph | (StackRox|Red Hat Advanced Cluster Security)`.
- **Empty state.** `checkNetworkGraphEmptyState()`: `.pf-v6-c-empty-state__content` contains `Select a cluster and at least one namespace to render active deployment traffic on the graph`.
- **Scope.** `selectCluster()` then `selectNamespace('stackrox')` — breadcrumb `nav[aria-label="Breadcrumb"]` controls `[aria-label="Select a cluster"]` and `[aria-label="Select namespaces"]`; menu items use `[data-testid="namespace-name"]` with an **exact** name match (avoid `stackrox-operator`). Result: group `[data-id="stackrox"]` under the topology groups layer; label via `filteredNamespaceGroupNode('stackrox')` in `networkGraph.selectors.js`.
- **Toolbar / legend.** `.pf-topology-control-bar` items contain Zoom In/Out, Fit to Screen, Reset View. **Legend** opens `.pf-v6-c-popover__content [data-testid="legend-title"]` with Node types / Namespace types. Close via `[aria-label="Close"]`.
- **Proof.** Screenshot of empty state **and** (if scoped) graph + toolbar. Graph node is `[data-id="stackrox-graph"]`. Do not click-drag by pixel coordinates; use toolbar buttons and labeled nodes (`[data-testid="deployment-name"]`, `.pf-topology__node__label`).

`npm run cypress-spec -- "networkGraph/networkGraph.test.js"` is a reasonable live spec **if** Doctor is READY and the cluster has `stackrox`. Still copy screenshots to the evidence dir.

## Gotchas

- Empty state without a cluster is not a product failure. Do not pass by stubbing `/v1/networkgraph/cluster/*` unless you label the result as mocked.
- `selectNamespace` must exact-match; `stackrox` ≠ `stackrox-operator`.
- CIDR **Manage CIDR blocks** writes Central config. Out of scope for a cheap proof; skip unless asked, then revert.
- Topology selectors changed between react-topology versions (see comments in `networkGraph.selectors.js`). Prefer `data-id` / toolbar text over SVG path indexes.
- Zoom/pan by dragging the canvas is not a stable driver. Use named toolbar items.
