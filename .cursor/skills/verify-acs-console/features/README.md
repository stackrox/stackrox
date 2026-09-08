# ACS Console — feature map

Top-level navigation entries from `src/Containers/MainPage/Navigation/NavigationSidebar.tsx`. Each file lists sub-features, user navigation, Cypress harness entry points, and gotchas.

| Feature | File | Component spec (no Central) | E2E domain |
|---------|------|------------------------------|------------|
| Dashboard | [dashboard.md](./dashboard.md) | `ViolationsByPolicySeverity.cy.jsx`, `ScopeBar.cy.jsx` | `cypress/integration/dashboard/` |
| Violations | [violations.md](./violations.md) | Dashboard widget links into `/main/violations` | `cypress/integration/violations/` |
| Vulnerability Management | [vulnerability-management.md](./vulnerability-management.md) | (use E2E + fixtures) | `cypress/integration/vulnerabilities/` |
| Policy Management | [policy-management.md](./policy-management.md) | (use E2E) | `cypress/integration/policies/` |
| Clusters | [clusters.md](./clusters.md) | (use E2E) | `cypress/integration/clusters/` |

**Agent default:** drive component specs where they exist; otherwise document E2E blockers (Central, auth, feature flags).
