# Component Testing — Cypress

See [TESTING.md](./TESTING.md) for shared principles.

Component tests render individual components in a real browser with Cypress, without needing a full app build or running backend. Use them for anything that requires DOM rendering, user interaction, or provider context. These tests should focus on complex, reusable components with real implemented logic. Simple components, one off components, and layout components do not each need individual component test files.

## Setup

- **Support file**: `cypress/support/component.js` — registers `cy.mount()`, imports global CSS, and loads Testing Library Cypress commands
- **Config**: `cypress.config.js` → `component` section

## What Belongs Here

- Components with user interactions (clicks, typing, selection)
- Components that depend on providers (Router, Redux, Apollo)
- Components with API-dependent behavior (mocked via `cy.intercept()`)
- Hook testing that requires React context
- PatternFly integration testing

## What Doesn't Belong Here

- Tests for Components that contain little to no logic
- Tests for Components that are primarily layout wrappers around PatternFly primitives
- Tests for one off Components specific to a single Page/Container

## Project Conventions

### Mounting with providers

Use `ComponentTestProvider` when the component needs Redux, Router, or Apollo, or other app-level providers. See `src/test-utils/ComponentTestProvider.tsx` for what it provides.

```jsx
import ComponentTestProvider from 'test-utils/ComponentTestProvider';

cy.mount(
    <ComponentTestProvider>
        <MyComponent />
    </ComponentTestProvider>
);
```

### Intercept before mount

`cy.intercept()` must be set up **before** `cy.mount()`. Requests fired during component initialization will be missed if intercepts are registered after mounting.

```jsx
// Correct — intercept is ready before the component renders
cy.intercept('POST', graphqlUrl('getData'), (req) => {
    req.reply({ data: mockData });
});
cy.mount(<ComponentTestProvider><MyComponent /></ComponentTestProvider>);

// Wrong — component fires requests during mount, intercept misses them
cy.mount(<ComponentTestProvider><MyComponent /></ComponentTestProvider>);
cy.intercept('POST', graphqlUrl('getData'), ...);
```

### Describe block naming

Use `Cypress.spec.relative` as the describe label so test output shows the file path:

```jsx
describe(Cypress.spec.relative, () => { ... });
```

### Element query priority

Prefer Testing Library Cypress commands for accessible queries:

`cy.findByRole` > `cy.findByLabelText` > `cy.findByText` > `cy.get('css selector')`

### Mocking API calls

Use `cy.intercept()` with the project's `graphqlUrl` helper for GraphQL:

```jsx
import { graphqlUrl } from 'test-utils/apiEndpoints';

cy.intercept('POST', graphqlUrl('getDeployments'), (req) => {
    req.reply({ data: { deployments: mockDeployments } });
}).as('getDeployments');
```

### OpenShift Console SDK mock

Components that import from `@openshift-console/dynamic-plugin-sdk` work automatically in component tests — the vite config aliases the SDK to `cypress/mocks/openshift-console-sdk.ts` when `CYPRESS_COMPONENT_TEST` is set. You don't need to mock the SDK yourself.

### URL state testing workaround

When testing components that assert on `cy.location()`, Cypress injects a `specPath` query parameter that interferes with URL assertions. Clear it before mounting:

```jsx
window.history.pushState({}, document.title, window.location.pathname);
cy.mount(<ComponentTestProvider><MyComponent /></ComponentTestProvider>);
cy.location('search').should('eq', '');
```

See [cypress-io/cypress#28021](https://github.com/cypress-io/cypress/issues/28021).

## File Placement

Component tests are **co-located** with their source files using `.cy.jsx` or `.cy.tsx` extension:

```txt
src/Components/
├── CodeViewer.tsx
├── CodeViewer.cy.jsx
```

## Running Tests

```bash
npm run cypress-component        # Interactive runner
npm run test-component           # Headless, all tests
```

## Canonical Examples

| Pattern | File |
|---------|------|
| Simple component | `src/Components/CodeViewer.cy.jsx` |
| Component with providers + GraphQL | `src/Containers/Dashboard/ScopeBar.cy.jsx` |
| Complex interactions + stubs | `src/Containers/Vulnerabilities/components/CompoundSearchFilter.cy.jsx` |
