# Violations

Route: `/main/violations` (`violationsBasePath`). Sidebar link **Violations**.

## Sub-features

- Violation list with severity / policy filters
- Violation detail pages (`/main/violations/:id`)
- Links from Dashboard severity widget (see `ViolationsByPolicySeverity.cy.jsx`)

## How to get to it (user POV)

1. Click **Violations** in the left navigation, or
2. From Dashboard, click **View all** or a severity tile (Low / Medium / High / Critical).

## Driving it with Cypress

**Component (widget navigation only):**

```bash
.cursor/skills/verify-acs-console/helpers/drive-component.sh violations
```

Same spec as dashboard widget; asserts navigation to `/main/violations` with query params like `s[Severity]=CRITICAL_SEVERITY`.

**E2E (full list / detail):**

```bash
cd ui/apps/platform
npm run cypress-spec -- "violations/<spec>.test.js"
```

Browse `cypress/integration/violations/` for specs. Requires Central at `https://localhost:8000` and token from `scripts/cypress.sh`.

Selectors: prefer `cy.findByRole`, `cy.findByText`; see `TESTING_E2E.md` behavior-over-implementation guidance.

## Gotchas

- E2E tests must create and clean up their own data; do not assert on deployment-specific inventory.
- Feature flags from Central are exported as `CYPRESS_ROX_*` env vars in `scripts/cypress.sh`.
- Dashboard component test uses mocked alert IDs for detail links — not live violation data.
