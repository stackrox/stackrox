# Policy Management

Route: `/main/policy-management` (`policyManagementBasePath`). Under sidebar **Platform Configuration → Policy Management**.

## Sub-features

- Policy list and search
- Policy wizard (criteria, lifecycle, actions)
- Policy categories and severity filters

## How to get to it (user POV)

1. Open **Platform Configuration** in the sidebar.
2. Click **Policy Management**.

## Driving it with Cypress

**E2E:** `cypress/integration/policies/` — create/edit policies, wizard flows.

```bash
cd ui/apps/platform
npm run cypress-spec -- "policies/<spec>.test.js"
```

Helpers and selectors often live beside specs (e.g. `Policies.helpers.js`, wizard step files).

**Component:** policy wizard pieces may have co-located tests under `src/Containers/Policies/`; search for `*.cy.jsx`.

Requires Central; tests typically create policies and clean up in `after` hooks.

## Gotchas

- Permission-gated actions: use `interceptAndOverridePermissions` patterns from `TESTING_E2E.md` for read-only scenarios.
- Policy criteria descriptors are feature-flagged in `policyCriteriaDescriptors.tsx`.
- Long wizard flows are better suited to E2E than component tests.
