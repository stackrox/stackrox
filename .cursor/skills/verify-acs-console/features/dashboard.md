# Dashboard

Route: `/main/dashboard` (`dashboardPath` in `routePaths.ts`). Sidebar link **Dashboard**.

## Sub-features

- Scope bar (cluster / namespace filters) — `ScopeBar.tsx`
- Policy violations by severity widget — `ViolationsByPolicySeverity.tsx`
- Other widgets: aging images, compliance levels, deployments at risk (`src/Containers/Dashboard/Widgets/`)

## How to get to it (user POV)

1. Sign in to ACS console (standalone: open `https://localhost:3000`, complete login against Central).
2. Click **Dashboard** in the left navigation (first item).

## Driving it with Cypress component tests

```bash
cd ui/apps/platform
npm run test-component -- --spec src/Containers/Dashboard/Widgets/ViolationsByPolicySeverity.cy.jsx
```

Or:

```bash
.cursor/skills/verify-acs-console/helpers/drive-component.sh dashboard
```

Key interactions from `ViolationsByPolicySeverity.cy.jsx`:

- Assert title: `cy.findByText(\`${alertCount} policy violations by severity\`)`
- Severity tiles: `cy.get('a').contains(/220\s*Low/)` (counts from mocked GraphQL)
- Navigation: `cy.findByText('View all').click()` → pathname `violationsBasePath` (`/main/violations`)

Scope bar spec: `src/Containers/Dashboard/ScopeBar.cy.jsx` — `cy.findByLabelText('Select clusters')`.

## Gotchas

- Intercept GraphQL **before** `cy.mount()` (`graphqlUrl('alertCountsBySeverity')`, `graphqlUrl('mostRecentAlerts')`).
- URL assertions: clear Cypress `specPath` query param before mount when testing `cy.location()` (see `ScopeBar.cy.jsx`).
- Full dashboard E2E needs Central + auth; component tests do not.
