# Clusters

Route: `/main/clusters` (`clustersBasePath`). Under sidebar **Platform Configuration → Clusters**.

## Sub-features

- Cluster list and health
- Init bundles, registration secrets, secure-a-cluster flows
- Delegated image scanning, discovered clusters (sub-routes under `clustersBasePath`)

## How to get to it (user POV)

1. Open **Platform Configuration** in the sidebar.
2. Click **Clusters**.

## Driving it with Cypress

**E2E:** `cypress/integration/clusters/` with helpers in `Clusters.helpers.js`.

```bash
cd ui/apps/platform
npm run cypress-spec -- "clusters/clusters.test.js"
```

**Component:** limited; cluster UI is mostly page-level. Prefer E2E or targeted component tests if added under `src/Containers/Clusters/`.

Requires Central with at least one secured cluster for happy-path tests; many specs create ephemeral cluster resources.

## Gotchas

- Cluster health and sensor compatibility tests depend on orchestrator flavor (`CYPRESS_ORCHESTRATOR_FLAVOR`).
- Deletion and certificate tests mutate cluster state — run cleanup hooks.
- Redirect tests link from Dashboard widgets; see `redirectFromDashboard.test.js`.
