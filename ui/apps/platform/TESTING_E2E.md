# E2E Testing — Cypress

See [TESTING.md](./TESTING.md) for shared principles.

## Setup

- **Framework**: Cypress 15 with `cypress-vite` preprocessor
- **Base URL**: `https://localhost:3000`
- **Viewport**: 1440 x 850
- **Timeouts**: 8s commands, 20s requests
- **Retries**: 1 in headless, 0 in interactive
- **Config**: `cypress.config.js`

## Authentication

Every test suite that requires authentication calls `withAuth()`:

```javascript
import withAuth from '../helpers/basicAuth';

describe('My Feature', () => {
    withAuth();

    it('does something', () => { ... });
});
```

This sets `localStorage.access_token` from the `ROX_AUTH_TOKEN` env var in `beforeEach`.

See `cypress/helpers/basicAuth.js`.

## Visiting Pages

Use helpers from `cypress/helpers/visit.js` — they automatically intercept and wait on prerequisite auth/config requests:

```javascript
import { visit } from '../helpers/visit';

it('loads the page', () => {
    visit('/main/dashboard');
});
```

For testing with specific permissions or auth states:

```javascript
import {
    visitWithStaticResponseForPermissions,
    visitWithStaticResponseForAuthStatus,
} from '../helpers/visit';

// Mock specific permissions
visitWithStaticResponseForPermissions('/main/violations', {
    body: { resourceToAccess: { Alert: 'READ_WRITE_ACCESS' } },
});
```

## API Interception

The core pattern uses **route matcher maps** — objects mapping alias names to `{ method, url }` matchers.

### Basic intercept → interact → wait

```javascript
// From cypress/integration/logout.test.js
import { interactAndWaitForResponses } from '../helpers/request';

const routeMatcherMapForLogout = {
    logout: {
        method: 'POST',
        url: '/sso/session/logout',
    },
};

const staticResponseMapForLogout = {
    logout: { body: {} },
};

it('goes to login page after logout', () => {
    visitMainDashboard();

    interactAndWaitForResponses(
        () => {
            cy.get(navSelectors.menuButton).click();
            cy.get(navSelectors.menuList.logoutButton).click();
        },
        routeMatcherMapForLogout,
        staticResponseMapForLogout
    );

    cy.location('pathname').should('eq', loginUrl);
});
```

### Step by step

For more control, use `interceptRequests` and `waitForResponses` separately:

```javascript
import { interceptRequests, waitForResponses } from '../helpers/request';

interceptRequests(routeMatcherMap, staticResponseMap);
// ... trigger interaction ...
waitForResponses(routeMatcherMap);
```

### Watching multiple requests

For tests that need to assert on request payloads:

```javascript
import { interceptAndWatchRequests } from '../helpers/request';

interceptAndWatchRequests(routeMatcherMap).then(({ waitAndYieldRequestBodyVariables }) => {
    // trigger sort click
    cy.get('th').click();
    waitAndYieldRequestBodyVariables().then(expectRequestedSort({ field: 'name', reversed: false }));
});
```

### GraphQL requests

```javascript
import { getRouteMatcherMapForGraphQL } from '../helpers/request';

const routeMatcherMap = getRouteMatcherMapForGraphQL(['getDeployments', 'searchOptions']);
```

### Overriding permissions and feature flags

```javascript
import {
    interceptAndOverridePermissions,
    interceptAndOverrideFeatureFlags,
} from '../helpers/request';

interceptAndOverridePermissions({ Alert: 'READ_ACCESS' });
interceptAndOverrideFeatureFlags({ ROX_WHATEVER: true });
```

## Selectors

### Selector priority

1. **OUIA attributes** — `[data-ouia-component-type="PF6/Button"]` (PatternFly standard, survives upgrades)
2. **`data-testid`** — for custom components without OUIA support
3. **Text content** — `:contains('Submit')` for visible text
4. **Accessible attributes** — `[aria-label="Close"]`, `[role="dialog"]`

Avoid raw PatternFly CSS classes (`.pf-v6-c-button`) — they change between PF versions.

### OUIA selectors

The `cypress/selectors/pf6.ts` file provides constants for common PF6 components:

```typescript
// From cypress/selectors/pf6.ts
const buttonSelectors = {
    button: '[data-ouia-component-type="PF6/Button"]',
} as const;

const selectSelectors = {
    select: 'div[data-ouia-component-type="PF6/Select"]',
    selectItem: 'div[data-ouia-component-type="PF6/Select"] *[role="listbox"] li',
} as const;
```

Import and use:

```javascript
import pf6 from '../selectors/pf6';

cy.get(pf6.button).contains('Save').click();
```

### Selector constants

Feature-specific selectors live in `cypress/constants/` and use `scopeSelectors()` for namespacing:

```javascript
// cypress/constants/AccessPage.js
import scopeSelectors from '../helpers/scopeSelectors';

export const selectors = {
    authProviders: scopeSelectors('[data-testid="auth-providers"]', {
        addButton: 'button:contains("Add")',
        table: { rows: '.rt-tr' },
    }),
};
```

## Helpers Reference

| Module | Purpose | Key exports |
|--------|---------|-------------|
| `helpers/request.js` | API interception and waiting | `interceptRequests`, `waitForResponses`, `interactAndWaitForResponses`, `interceptAndWatchRequests`, `interceptAndOverridePermissions`, `interceptAndOverrideFeatureFlags` |
| `helpers/visit.js` | Page navigation with auto auth handling | `visit`, `visitWithStaticResponseForPermissions`, `visitWithStaticResponseForAuthStatus`, `visitWithStaticResponseForCapabilities` |
| `helpers/basicAuth.js` | Test authentication setup | `withAuth` (default export) |
| `helpers/nav.ts` | Left navigation interactions | `visitFromLeftNav`, `visitFromLeftNavExpandable` |
| `helpers/formHelpers.js` | Form element interactions | `getInputByLabel`, `getSelectButtonByLabel`, `getSelectOption`, `generateNameWithDate` |
| `helpers/tableHelpers.ts` | Table row/column interactions | `getTableRowLinkByName`, `openTableRowActionMenu`, `sortByTableHeader`, `assertOnEachRowForColumn` |

## Custom Commands

Defined in `cypress/support/commands.js`:

### `cy.checkAccessibility()`

Runs axe-core accessibility checks. Call after page load:

```javascript
it('has no accessibility violations', () => {
    visit('/main/dashboard');
    cy.checkAccessibility();
});
```

### `cy.spyTelemetry()` / `cy.getTelemetryEvents()`

Mock and assert on analytics events:

```javascript
cy.spyTelemetry();
// ... interact with the page ...
cy.getTelemetryEvents().then((events) => {
    expect(events).to.deep.include({ event: 'Page Viewed', properties: { page: 'Dashboard' } });
});
```

## Test Organization

```sh
cypress/
├── integration/          # E2E tests grouped by feature
│   ├── access.test.js
│   ├── clusters/
│   │   ├── clusters.test.js
│   │   └── discoveredClusters.test.ts
│   ├── vulnerabilities/
│   │   ├── workloadCves/
│   │   └── exceptionManagement/
│   └── ...
├── helpers/              # Reusable test utilities
├── constants/            # Selector constants by feature
├── selectors/            # Framework-level selectors (PF6, nav)
├── fixtures/             # Mock response data (JSON)
├── mocks/                # Mock implementations
└── support/              # Cypress config, commands, plugins
```

### Fixtures

Store mock API responses in `cypress/fixtures/` organized by domain:

```sh
fixtures/
├── auth/
│   ├── authProviders.json
│   └── roles.json
├── clusters/
│   └── clusters.json
└── ...
```

Reference in static response maps:

```javascript
const staticResponseMap = {
    getAuthProviders: { fixture: 'auth/authProviders.json' },
};
```

## Avoid

- **`cy.wait(N)` with milliseconds** — always wait on aliases: `cy.wait('@aliasName')`
- **Raw PF CSS selectors** — use OUIA attributes or `data-testid`
- **Monolithic test files** — split into feature-focused files, use nested `describe` blocks
- **Skipping auth setup** — always call `withAuth()` unless testing unauthenticated flows
- **Direct `cy.request()` for page data** — use `cy.intercept()` to control API responses

## Canonical Examples

| Pattern | File |
|---------|------|
| Clean intercept + visit | `cypress/integration/logout.test.js` |
| Fixture-based mocking | `cypress/integration/userinfo.test.js` |
| Complex UI flows | `cypress/integration/access.test.js` |
| TypeScript e2e | `cypress/integration/telemetry.test.ts` |
| OUIA selectors | `cypress/selectors/pf6.ts` |
| Helper-driven workflows | `cypress/integration/vulnerabilities/exceptionManagement/approveRequestFlow.test.ts` |
