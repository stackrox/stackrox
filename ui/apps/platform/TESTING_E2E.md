# E2E Testing — Cypress

See [TESTING.md](./TESTING.md) for shared principles.

E2E tests run against a fully deployed StackRox stack. This covers two contexts:
- **Standalone platform** — tests in `cypress/integration/` against the platform UI directly
- **OCP console plugin** — tests in `cypress/integration-ocp/` against the plugin running inside OpenShift Console

Both use the same Cypress framework, helpers, and assertion patterns. The main differences are authentication, navigation, and base URL. See [OCP Console Plugin Tests](#ocp-console-plugin-tests) below.

## Setup

- **Framework**: Cypress
- **Base URL**: defaults to `https://localhost:3000`
- **Config**: `cypress.config.js`

## Guiding Principles

### Behavior over implementation

Tests should break when behavior changes, never when implementation changes. A developer should be able to refactor component internals — restructure JSX, swap state management, rename internal variables — and have zero test failures. If a test breaks during a refactor that doesn't change what the user sees, that test was testing implementation and is negative-value.

This applies to every layer of a test: selectors should target what users see (text, roles, labels) rather than DOM structure. Assertions should verify visible outcomes rather than internal state. Waits should be driven by UI state rather than specific network calls when possible.

The route matcher map pattern is a pragmatic exception — it couples tests to API shape (endpoint URLs, operation names), which means backend renames or frontend API migrations can break tests even when user-facing behavior is identical. This tradeoff is acceptable for test reliability, but prefer asserting on visible UI outcomes over request/response details when both options are available.

### Purpose

Cypress e2e tests and integration tests should strive to demonstrate that a core piece of app functionality works.

In general, we should have happy-path cases for all major functionality in the app that cannot be covered by lower level tests. Major error handling should be considered, but we do
not need to test validation states for every form input (this should be covered by unit or component tests instead).

These tests can test workflows that are mostly server agnostic that would be too cumbersome or expansive to test with component tests, such as multi-page navigation.

### Approach

Aim to test as closely to the user experience as possible. Avoid implementation details that are invisible to the user. For example:

**Good**
Click a deployment link that reads "central", and then assert that the top level page heading contains the text "central". Continue testing the specifics of the deployment page.

**Bad**
Click the link in the first `table tr td`. Listen for a request to `getDeploymentDetails` and wait for the request to complete. Continue testing the specifics of the deployment page.

**Good**
Click a radio button with label "Option A" in the form. Assert that a section that is no longer relevant has `disabled` attributes on form elements.

**Bad**
Click on the third radio button in the form. Assert that a section that is no longer relevant has `.pf-m-disabled` class names.

### Data

Tests should not depend on data they didn't create. External data changes unpredictably — rely on data the test creates itself via e2e flows.

**Good**
- Asserting on the results of a policy we create and then view in the UI
- Asserting that a vulnerability exception we create has the correct expiration

**Less good**
- Asserting that `central-db` has a process executing `/usr/bin/postgres` (subject to external changes, updates)
- Asserting that **any** CVEs are listed in Vulnerability Management (temporary external server downtime, slow scanner initialization)

**Worst**
- Asserting that a particular CVE exists in a platform image (CVEs are fixed and change regularly)
- Asserting default API token events exist in Administration Event logs (subject to change as install and infrastructure evolve)
- Asserting presence of OpenShift deployments in inventory lists (OpenShift components subject to change, test runs against *KS clusters do not have these components)

Mocking and request interception should be used when we need server data, but the specifics of the data have low impact on the test:
- Creating a Vulnerability Exception needs _some_ CVE available in the UI, but the specific CVE is irrelevant. We can mock a request to provide data needed for 
the workflow and assert on constants in the mocked data.
- We want to create a test for a "Severity:Critical" vulnerability filter. We cannot guarantee that the system has zero Critical vulnerabilities, _only_ Critical vulnerabilities, or a mix
of vulnerability severities. Intercepting the request after a user action and asserting on the presence of the correct filter is an acceptable trade off between testing implementation
details and test reliability.

### Cleanup

Many e2e tests create data in the system and assert based on this mutation. Tests should **always** clean up test data that is created, both before and after the test.

**Before** - ensures orphaned test data is cleaned up before execution that would cause test failures
**After** - avoids orphaned test data and cruft accumulation in long running systems

This includes form submitted data, resource creation, and API tokens used for authentication during tests.

### Test isolation

Tests should never depend on pre-existing state in the system beyond the base deployment. If a test needs a policy, it creates the policy. If it needs a specific CVE, it mocks the data. If it needs a specific user role, it sets up the permissions.

Two tests should be able to run in any order, simultaneously, against the same deployment, and never interfere with each other. A test that passes in the suite but fails in isolation (or vice versa) is broken.

### Error states and reduced permissions

Every feature should have at least one error-state or edge-case test alongside its happy path — empty lists, failed requests, and missing data are the scenarios that actually break in production.

Test features under reduced permissions as well. Use `interceptAndOverridePermissions` to verify that mutation controls are hidden or disabled for read-only users, and that permission-gated pages show appropriate messaging.

### URL-driven state

Many features persist filters, sorting, pagination, and tab selection in URL search parameters. Test the round-trip: apply filters, verify the URL updates, then visit that URL directly and verify the same state is restored. Bookmarkable state is an implicit contract with users and breaks easily during refactors.

## Authentication

Every test suite that requires authentication calls `withAuth()`:

```javascript
import withAuth from '../helpers/basicAuth';

describe('My Feature', () => {
    withAuth();

    it('does something', () => { ... });
});
```

This sets `localStorage.access_token` from the `ROX_AUTH_TOKEN` env var in `beforeEach`. See `cypress/helpers/basicAuth.js`.

## Feature Flag Gating

Tests that depend on a feature flag should skip when the flag is not enabled. The test runner scripts (`scripts/cypress.sh`) fetch flags from the deployment API and export them as `CYPRESS_ROX_*` env vars.

For individual tests, skip inside the `it` block:

```javascript
import { hasFeatureFlag } from '../helpers/features';

it('should show the new widget', function () {
    if (!hasFeatureFlag('ROX_NEW_WIDGET')) {
        this.skip();
    }
    // ... test body
});
```

To skip an entire `describe` block, use a `before` hook — this is the more common pattern in the codebase:

```javascript
describe('New Widget', function () {
    before(function () {
        if (!hasFeatureFlag('ROX_NEW_WIDGET')) {
            this.skip();
        }
    });

    it('should render', () => { ... });
    it('should update on click', () => { ... });
});
```

**Important:** Use `function` keyword, not an arrow function — `this.skip()` requires the Mocha test context which arrow functions don't bind.

Similarly, `hasOrchestratorFlavor('openshift')` gates tests that only apply to OpenShift clusters.

**Coverage gaps:** Skipped tests are invisible — if a flag is off in CI for months, that feature has zero e2e coverage and nobody notices. When a feature flag graduates (becomes default-on or is removed), remove the flag gating from its tests as part of the same change. Treat leftover `hasFeatureFlag` guards in tests as cleanup debt after flag graduation.

## Visiting Pages

Use helpers from `cypress/helpers/visit.js` — they automatically intercept and wait on prerequisite auth/config requests:

```javascript
import { visit } from '../helpers/visit';

visit('/main/dashboard');
```

For testing with specific permissions: `visitWithStaticResponseForPermissions`, `visitWithStaticResponseForAuthStatus`, `visitWithStaticResponseForCapabilities`.

## API Interception

The core pattern uses **route matcher maps** — objects mapping alias names to `{ method, url }` matchers.

Intercepts must be registered **before** the action that triggers requests (`visit()`, navigation, button clicks). Intercepts registered after the triggering action will miss requests already in flight.

### `interactAndWaitForResponses` — intercept, interact, and wait in one call

```javascript
import { interactAndWaitForResponses } from '../helpers/request';

const routeMatcherMap = {
    logout: { method: 'POST', url: '/sso/session/logout' },
};

interactAndWaitForResponses(
    () => { cy.get(selectors.logoutButton).click(); },
    routeMatcherMap,
    { logout: { body: {} } }  // optional static responses
);
```

### `interceptRequests` / `waitForResponses` — for more control

```javascript
import { interceptRequests, waitForResponses } from '../helpers/request';

interceptRequests(routeMatcherMap, staticResponseMap);
// ... trigger interaction ...
waitForResponses(routeMatcherMap);
```

### `interceptAndWatchRequests` — assert on request payloads or perform multiple actions that each trigger an awaitable request

```javascript
import { interceptAndWatchRequests } from '../helpers/request';

interceptAndWatchRequests(routeMatcherMap).then(({ waitAndYieldRequestBodyVariables }) => {
    cy.get('th').click();
    waitAndYieldRequestBodyVariables().then(expectRequestedSort({ field: 'name', reversed: false }));
});
```

### GraphQL helpers

```javascript
import { getRouteMatcherMapForGraphQL } from '../helpers/request';

const routeMatcherMap = getRouteMatcherMapForGraphQL(['getDeployments', 'searchOptions']);
```

### Overriding permissions and feature flags

```javascript
import { interceptAndOverridePermissions, interceptAndOverrideFeatureFlags } from '../helpers/request';

interceptAndOverridePermissions({ Alert: 'READ_ACCESS' });
interceptAndOverrideFeatureFlags({ ROX_WHATEVER: true });
```

## Selectors

### Selector priority

1. Semantic global components selectors exported from `selectors/pf6`
2. Semantic component selectors exported from top level helper files
3. Semantic component selectors available to individual feature domains
4. User-visible document attributes - e.g. `:contains('Submit')` for visible text, `input[disabled]` for input elements
5. **Accessible attributes** — `[aria-label="Close"]`, `[role="dialog"]`
6. **OUIA attributes** — `[data-ouia-component-type="PF6/Button"]` (PatternFly standard, generally should be encapsulated in `selectors/pf6` instead of used directly)
7. Custom CSS classes - mostly when there is no other option, such as `svg` elements generated by a library that we have limited control over

In contrast, avoid the following selector patterns in tests:

- Raw PatternFly CSS classes (`.pf-v6-c-button`) — they change between PF versions
- Test-only attributes (`data-testid`)
- Deeply nested DOM selectors (`div div p > button`)

### OUIA selectors

`cypress/selectors/pf6.ts` provides constants for common PF6 components. Import and use:

```javascript
import pf6 from '../selectors/pf6';

cy.get(pf6.button).contains('Save').click();
```

**Note - the goal of the `pf6` selectors is to make a definitive semantic library matching common components exposed from PatternFly. As we encounter more needs for these, we should update the selector list.**

## Helpers Reference

Globally applicable helper functions (`cypress/helpers/*`). Use these frequently, and create additional helpers when true cross cutting concerns are needed.

| Module | Purpose | Key exports |
|--------|---------|-------------|
| `helpers/request.js` | API interception and waiting | `interceptRequests`, `waitForResponses`, `interactAndWaitForResponses`, `interceptAndWatchRequests`, `interceptAndOverridePermissions`, `interceptAndOverrideFeatureFlags` |
| `helpers/visit.js` | Page navigation with auto auth handling and request awaiting | `visit`, `visitWithStaticResponseForPermissions`, `visitWithStaticResponseForAuthStatus` |
| `helpers/basicAuth.js` | Test authentication setup | `withAuth` (default export) |
| `helpers/features.js` | Feature flag and orchestrator checks | `hasFeatureFlag`, `hasOrchestratorFlavor` |
| `helpers/nav.ts` | Left navigation interactions | `visitFromLeftNav`, `visitFromLeftNavExpandable` |
| `helpers/formHelpers.js` | Form element interactions | `getInputByLabel`, `getSelectButtonByLabel`, `getSelectOption`, `generateNameWithDate` |
| `helpers/tableHelpers.ts` | Table row/column interactions | `getTableRowLinkByName`, `openTableRowActionMenu`, `sortByTableHeader`, `assertOnEachRowForColumn` |

## Custom Commands

Defined in `cypress/support/commands.js`:

- **`cy.checkAccessibility()`** — runs axe-core accessibility checks after page load
- **`cy.spyTelemetry()` / `cy.getTelemetryEvents()`** — mock and assert on analytics events

## Test Organization

```txt
cypress/
├── integration/          # E2E tests grouped by feature
│   ├── access.test.js
│   ├── clusters/
│   ├── vulnerabilities/
│   └── ...
├── helpers/              # Reusable test utilities
├── constants/            # Selector constants by feature
├── selectors/            # Framework-level selectors (PF6, nav)
├── fixtures/             # Mock response data (JSON)
├── mocks/                # Mock implementations
└── support/              # Cypress config, commands, plugins
```

Fixtures live in `cypress/fixtures/` organized by domain and are referenced in static response maps via `{ fixture: 'auth/authProviders.json' }`.

## Canonical Examples

| Pattern | File |
|---------|------|
| Clean intercept + visit | `cypress/integration/logout.test.js` |
| Fixture-based mocking | `cypress/integration/userinfo.test.js` |
| Complex UI flows | `cypress/integration/access.test.js` |
| TypeScript e2e | `cypress/integration/telemetry.test.ts` |
| OUIA selectors | `cypress/selectors/pf6.ts` |
| Helper-driven workflows | `cypress/integration/vulnerabilities/exceptionManagement/approveRequestFlow.test.ts` |

---

## OCP Console Plugin Tests

Tests for the ACS OpenShift Console plugin live in `cypress/integration-ocp/` and run against the plugin inside a real OpenShift Console instance.

### Key Differences from Standalone

| Aspect | Standalone | OCP Plugin |
|--------|-----------|------------|
| Test location | `cypress/integration/` | `cypress/integration-ocp/` |
| Base URL | `https://localhost:3000` | `http://localhost:9000` (configurable) |
| Auth | Token-based (`ROX_AUTH_TOKEN`) | Cookie-based OCP session (`withOcpAuth()`) |
| Navigation | `visit('/main/...')` | `visitFromConsoleLeftNavExpandable('Security', '...')` |
| Run command | `npm run test-e2e-local` | `npm run test-e2e-local:ocp` |
| Runner script | `scripts/cypress.sh` | `scripts/cypress-ocp.sh` |
| Artifacts | `cypress/test-results/artifacts/` | `cypress/test-results/artifacts/ocp-console-plugin/` |
| Min version | — | OCP 4.19+ |

### Authentication

OCP tests use `withOcpAuth()` instead of `withAuth()`. It uses `cy.session()` for cookie-based persistence — navigates to the OCP login page, enters credentials, and waits for the console dashboard. Supports auth-disabled mode via `OCP_BRIDGE_AUTH_DISABLED` env var. See `cypress/helpers/ocpAuth.ts`.

### Navigation

Use `visitFromConsoleLeftNavExpandable()` from `cypress/helpers/nav.ts` to navigate via the OpenShift Console left nav.

### Namespace / Project Selection

`selectProject()` from `cypress/helpers/ocpConsole.ts` switches namespace scope in the OCP console header.

### Namespace-Scoped Request Headers

A key OCP-specific concern is verifying that the `acs-auth-namespace-scope` header is sent correctly on data requests. See `cypress/integration-ocp/routes.ts` for route matchers and the header constant.

### OCP Version Compatibility

`filterByField()` in `helpers/ocpConsole.ts` handles differences between OCP 4.19 (Dropdown) and 4.21+ (DataViewFilters MenuToggle). When writing OCP tests that interact with console-native UI elements, check for version-specific selectors.

### OCP Helpers Reference

| Module | Purpose | Key exports |
|--------|---------|-------------|
| `helpers/ocpAuth.ts` | OCP session authentication | `withOcpAuth` |
| `helpers/ocpConsole.ts` | OCP console interactions | `selectProject`, `filterByField` |
| `helpers/main.js` | Console dashboard navigation | `visitConsoleMainDashboard` |
| `helpers/nav.ts` | OCP console navigation | `visitFromConsoleLeftNavExpandable` |
| `integration-ocp/routes.ts` | Route matchers and auth headers | `acsAuthNamespaceHeader`, route matcher constants |

### Running OCP Tests

```bash
npm run cypress-open:ocp         # Interactive runner
npm run test-e2e-local:ocp       # Headless, all tests
npm run test-e2e:ocp             # CI mode with reporters
```

Two scenarios are supported:

**1. Local development console with bridge auth disabled**

```sh
# If necessary, export the target URL
export OPENSHIFT_CONSOLE_URL=<url-to-web-console-ui>
# Set ORCHESTRATOR_FLAVOR, which is typically only available in CI
export ORCHESTRATOR_FLAVOR='openshift'
# Runs Cypress OCP tests ignoring authentication
OCP_BRIDGE_AUTH_DISABLED=true npm run cypress-open:ocp
```

**2. Deployed console with username/password credentials**

```sh
# If necessary, export the target URL
export OPENSHIFT_CONSOLE_URL=<url-to-web-console-ui>
# Set ORCHESTRATOR_FLAVOR, which is typically only available in CI
export ORCHESTRATOR_FLAVOR='openshift'
# export credentials
export OPENSHIFT_CONSOLE_USERNAME='kubeadmin'
export OPENSHIFT_CONSOLE_PASSWORD=<password>

# Runs Cypress OCP tests with a session initialization step
npm run cypress-open:ocp
```

### OCP Test Organization

```txt
cypress/integration-ocp/
├── smoke.test.ts               # Basic plugin loading
├── routes.ts                   # Route matchers + auth header constants
├── security/
│   ├── vulnerabilities.test.ts
│   ├── cveDetail.test.ts
│   └── imageDetail.test.ts
├── workloads/
│   └── securityTab.test.ts
└── projects/
    └── projectSecurityTab.test.ts
```

### OCP Canonical Examples

| Pattern | File |
|---------|------|
| Plugin smoke test | `cypress/integration-ocp/smoke.test.ts` |
| Namespace header verification | `cypress/integration-ocp/security/vulnerabilities.test.ts` |
| OCP console native UI interaction | `cypress/integration-ocp/workloads/securityTab.test.ts` |
