# Violations

Lists policy violations in workflow views (user workloads, platform, nodes, all), filters them with the compound search filter, and opens a violation detail page with tabs for the violation, enforcement, deployment or node, policy, and network policies.

## Sub-features

- `violations-nav` opens the list from the left nav.
- `violations-views` switches between the User Workloads, Platform, Nodes, and All Violations views.
- `violations-filter` narrows the table by Severity, Lifecycle stage, Category, or Policy.
- `violations-detail` opens a violation and shows its tabs.
- `violations-row-actions` offers exclude, resolve, and resolve-plus-baseline from the row kebab.

## How to get to it (user POV)

- Left nav > `Violations` (`/main/violations`, default view User Workloads).
- Horizontal subnav: `User Workloads`, `Platform`, `Nodes`, `All Violations` (`?filteredWorkflowView=`).
- Dashboard `Violation` tile.
- Deep link `/main/violations/<alertId>`.

## Driving it with verify-ui

Preconditions:

- `doctor.sh` healthy.
- Import from `cypress/integration/violations/Violations.helpers.js`: `visitViolationsFromLeftNav`, `selectFilteredWorkflowView`, `interactAndWaitForViolationsResponses`, `routeMatcherMapForViolations`.

- **Open from nav.** `visitViolationsFromLeftNav()`. Expect pathname `/main/violations` and `h1:contains("User workload violations")`. `captureState('01-user-workloads')`.
- **All violations.** `selectFilteredWorkflowView('All Violations')`, then `cy.get('h1:contains("All violations")')`, `cy.get('table tbody[aria-busy="false"]')`, and `cy.get('tbody tr').should('have.length.greaterThan', 0)`. `captureState('02-all-violations')`.
- **Filter by severity.** `interactAndWaitForViolationsResponses(() => { cy.get('button[aria-label="Filter by Severity"]').click(); cy.get('div[aria-label="Filter by Severity select menu"] li:contains("Critical") input[type="checkbox"]').check(); })`. Each `td[data-label="Severity"]` reads `Critical`, or the empty state shows. `captureState('03-critical-only')`.
- **Open detail.** Read the first `td[data-label="Policy"] a` text as `policyName`, then click it inside `interactAndWaitForResponses(cb, { alert: { method: 'GET', url: '/v1/alerts/*' } })`. Expect `h1` to contain `policyName`, the pathname to match `/main/violations/[\w-]+`, and `[data-ouia-component-type="PF6/TabButton"]:contains("Violation")`. `captureState('04-violation-detail')`.
- **Policy tab.** Click the `Policy` tab button. Expect `h2:contains("Policy overview")`. `captureState('05-policy-tab')`.

## Gotchas

- The default view, User Workloads, can be empty on a demo cluster. Use `All Violations` for non-empty data.
- Detail tabs are conditional: Enforcement only when enforced, Deployment vs Node by entity type, and Network policies needs deployment plus read access. Do not assert a fixed tab count.
- `violations.test.js` and `violationsSearchFilter.test.ts` register `cy.intercept` after the action, which races. Always intercept first (`interactAndWaitFor...`).
- Autocomplete inputs debounce 500ms. Wait for the menu, not a sleep.
- Row actions (`Mark as resolved`, `Exclude deployment from policy`) mutate Central. Only use them on a throwaway cluster, and prove the result by reloading the list.
- Fixture helpers to avoid: `visitViolationsWithFixture`, `visitViolationFromTableWithFixture`, `visitViolationWithFixture`, `clickDeploymentTabWithFixture`.
