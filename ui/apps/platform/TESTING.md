# Testing Guidelines

Shared principles for all UI tests. For specific patterns for individual testing levels see:

- [TESTING_UNIT.md](./TESTING_UNIT.md) — Vitest unit tests (pure logic)
- [TESTING_COMPONENT.md](./TESTING_COMPONENT.md) — Cypress component tests
- [TESTING_E2E.md](./TESTING_E2E.md) — Cypress end-to-end tests

## Test Types

| Type | Framework | Location | Purpose |
|------|-----------|----------|---------|
| Unit | Vitest | `src/**/*.test.ts` | Pure functions, data transforms, validators — no DOM, no React |
| Component | Cypress | `src/**/*.cy.{jsx,tsx}` | Complex components with real DOM rendering |
| E2E | Cypress | `cypress/integration/**/*.test.{js,ts}` | Full user flows against a running app, UI focused integration flows |
| E2E (OCP) | Cypress | `cypress/integration-ocp/**/*.test.ts` | OCP console plugin flows against a running OpenShift cluster |

**When to use which:**

- **Unit tests** — pure logic: utility functions, data transformations, validators, formatters, reducers. No `render()`, no React, no DOM. Fast and deterministic.
- **Component tests** — interactive UI: components that need real DOM rendering, user interaction, provider context, or API mocking via `cy.intercept()`. No network calls, no full app build.
- **E2E tests** — full workflows: multi-page user journeys, API integration, cross-component flows against a running StackRox deployment. Covers both standalone platform and OCP console plugin.

If a behavior is pure logic, test it at the unit level. If it requires rendering a component but not the full application, use a component test. Reserve e2e for workflows that need the real stack or would be too cumbersome to test with component tests.

## Assessing Coverage

Before writing new tests, check what already exists:

- **Unit tests**: look for a co-located `.test.ts` file next to the source
- **Component tests**: look for a co-located `.cy.jsx` or `.cy.tsx` file next to the source
- **E2E tests**: check `cypress/integration/` for the feature domain directory (e.g., `vulnerabilities/`, `clusters/`)
- **OCP E2E tests**: check `cypress/integration-ocp/` for the feature domain

Not everything needs a dedicated test file. Use the "What Belongs Here" and "What Doesn't Belong Here" sections in each level's guide to decide whether the code warrants testing at that level. Simple utility wrappers, layout components, and thin pass-through components generally don't need their own tests — they get covered implicitly by the tests of the code that uses them.

## File Placement and Naming

- **Unit tests** are co-located with source: `utils.ts` → `utils.test.ts`
- **Component tests** are co-located with source: `Component.tsx` → `Component.cy.jsx`
- **E2E tests** live in `cypress/integration/` grouped by feature domain (e.g., `vulnerabilities/`, `policies/`)
- **OCP E2E tests** live in `cypress/integration-ocp/` grouped by feature domain

## Gotchas

### `console.error` fails unit tests

`setupTests.js` spies on `console.error` and fails any test that triggers one — even if the test's own assertions pass. The most common cause is an unmocked API call. If you hit this, check the console output for `ECONNREFUSED` errors indicating which service call needs mocking.

### testing-library in Vitest

Avoid `@testing-library/react` (`render`, `screen`, `userEvent`) in Vitest test files. Component rendering and interaction testing belongs in Cypress component tests. Vitest tests should only cover pure logic.

### Hardcoded waits

In Cypress tests (component and e2e), never use `cy.wait(N)` with a millisecond value. Wait on aliased intercepts or DOM state instead.

```typescript
// Don't
cy.wait(2000);

// Do
cy.wait('@getDeployments');
```

### Snapshot tests

Don't use `toMatchSnapshot()` or `toMatchInlineSnapshot()`. They are brittle — any markup change breaks them regardless of whether behavior changed.

## Accessibility

- **E2E tests**: use `cy.checkAccessibility()` after page loads
- Prefer Testing Library Cypress queries (`cy.findByRole`, `cy.findByLabelText`) over `cy.get` for accessibility semantics

## Running Tests

```bash
# Unit tests
npm run test                                  # All tests
npm run test -- --testNamePattern="Name"      # By test name
npm run test -- src/path/to/test.test.ts      # By file

# Component tests
npm run cypress-component                     # Interactive runner
npm run test-component                        # Headless, all tests

# E2E tests
npm run cypress-open                          # Interactive runner
npm run test-e2e-local                        # Headless, all tests
npm run cypress-spec -- "test-name.js"        # Single spec

# E2E tests (OCP plugin)
npm run cypress-open:ocp                      # Interactive runner
npm run test-e2e-local:ocp                    # Headless, all tests

# Coverage
npm run test-coverage                         # Unit test coverage
```

All commands run from `ui/apps/platform/`.
