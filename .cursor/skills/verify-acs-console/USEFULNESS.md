# Why a live ACS console skill exists

Agents working in `ui/apps/platform` can already run Vitest and Cypress **component** tests. Those produce fast screenshots of a mounted PatternFly island with `cy.intercept()` fixtures, no Central, no left nav, no login, no Vite proxy. That is useful for component regressions. It is **not** proof that a user can open ACS and complete a workflow.

ACS is a console in front of Central. Dashboard tiles, Workload CVE tables, search autocomplete, and Network Graph all depend on live `/v1` and GraphQL, feature flags, and RBAC. The same JSX can look correct in a component test and still fail when Central is empty, Scanner is not ready, or the Vite proxy is pointed at the wrong place.

## Benefits

- One place that names the real Launch path (`npm run deploy-local`, `UI_START_TARGET`, `npm run start` from `ui/`), ports (3000 / 8000), and admin password file.
- Doctor that fails closed when Central is down, so the next agent reports **inconclusive** instead of shipping a component screenshot.
- Drive recipe that prefers the existing Cypress **e2e** harness (`cypress/integration/`) and real PatternFly / ACS routes, then browser/CDP — not invented coordinates.
- Feature map so “verify the console” is not one Dashboard visit while Workload CVEs and Search go untested.
- Isolation rules: do not double-drive a shared `stackrox` cluster; do not teardown what this run did not start.

## Limits

- Full Central + k8s usually will not start on a Cursor cloud VM (no cluster, no container engine). Live prove-once is often blocked; that is a real limit, not a pass.
- One cluster / one namespace / one 8000 forward. This skill will not spin a second isolated ACS next to a developer’s session.
- Many Cypress e2e specs stub GraphQL. A green fixture spec is not live-data proof; the skill says so.
- Component Cypress remains the wrong artifact for “the user can use the console.”
- OCP plugin flows are out of scope (different auth, different base URL).
