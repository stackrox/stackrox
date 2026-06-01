# Testing Guidelines

Shared principles for all UI tests. For framework-specific patterns see:

- [TESTING_UNIT.md](./TESTING_UNIT.md) — Vitest unit and component tests
- [TESTING_E2E.md](./TESTING_E2E.md) — Cypress end-to-end tests

## Test Types

| Type | Framework | Location | Purpose |
|------|-----------|----------|---------|
| Unit | Vitest | `src/**/*.test.ts(x)` | Pure functions, hooks, components in isolation |
| E2E | Cypress | `cypress/integration/**/*.test.ts` | Full user flows against a running app |

**When to use which:**

- **Unit tests** — logic, data transformations, hooks, individual components. Fast, isolated, no network.
- **E2E tests** — multi-step user workflows, page navigation, API integration. Slow, requires a running stack.

If a behavior can be verified with a unit test, prefer that over e2e.

## File Placement and Naming

- **Unit tests** are co-located with source: `Component.tsx` → `Component.test.tsx`
- **E2E tests** live in `cypress/integration/` grouped by feature domain (e.g., `vulnerabilities/`, `policies/`)
- All test files use the `.test.ts` or `.test.tsx` extension

## What to Test

Test **behavior**, not implementation:

```typescript
// Good: tests what the user sees
expect(screen.getByText('Success')).toBeInTheDocument();

// Bad: tests internal state shape
expect(component.state.status).toBe('success');
```

Focus on:
- User-visible outcomes (text rendered, elements present/absent, navigation)
- Function inputs → outputs
- Side effects that matter (API calls made, events fired)

Don't test:
- Internal state or private methods
- Framework internals (React lifecycle, Redux dispatch mechanics)
- Whether a value is a function (`typeof x === 'function'`)

## Assertion Style

Use the most specific assertion available:

```typescript
// Preferred
expect(value).toBe(true);
expect(items).toHaveLength(3);
expect(element).toBeInTheDocument();
expect(fn).toHaveBeenCalledWith('arg');

// Avoid — too vague, poor error messages
expect(value).toBeTruthy();
expect(items.length > 0).toBe(true);
expect(element).not.toBeNull();
```

For DOM assertions, use `@testing-library/jest-dom` matchers:
`toBeInTheDocument()`, `toHaveTextContent()`, `toBeVisible()`, `toBeDisabled()`

## Test Naming

Use `describe` for the subject and `it`/`test` for behaviors:

```typescript
describe('useLocalStorage', () => {
    test('should safely read and write local storage', () => { ... });
    test('should reject invalid values from raw localStorage', () => { ... });
});
```

- `describe` — the component, hook, or function name
- `it`/`test` — a behavior statement: "should **do something** when **condition**"
- Nest `describe` blocks for logical grouping when testing multiple aspects

## Anti-Patterns

### Snapshot tests

Don't use `toMatchSnapshot()` or `toMatchInlineSnapshot()`. They are brittle — any markup change breaks them regardless of whether behavior changed.

```typescript
// Don't
expect(asFragment()).toMatchSnapshot();

// Do — assert specific behavior
expect(screen.getByRole('button', { name: 'Submit' })).toBeEnabled();
```

### Hardcoded waits

In e2e tests, never use `cy.wait(N)` with a millisecond value. Always wait on aliased intercepts.

```typescript
// Don't
cy.wait(2000);

// Do
cy.wait('@getDeployments');
```

### Over-mocking

Mock at boundaries (API calls, external services), not internal modules. If you need to mock five things to test one function, the test or the code needs restructuring.

### Testing implementation details

Don't assert on internal state, CSS classes, or DOM structure that users don't see.

## Accessibility

- **Unit tests**: prefer accessible queries (`getByRole`, `getByLabelText`, `getByPlaceholderText`) over `getByTestId`
- **E2E tests**: use `cy.checkAccessibility()` (wraps axe-core) after page loads to catch a11y violations
- Use `data-testid` only when no accessible query fits the element

## Running Tests

```bash
# Unit tests
npm run test                                  # All tests
npm run test -- --testNamePattern="Name"      # By test name
npm run test -- src/path/to/test.test.ts      # By file

# E2E tests
npm run cypress-open                          # Interactive runner
npm run test-e2e-local                        # Headless, all tests
npm run cypress-spec -- "test-name.js"        # Single spec

# Coverage
npm run test-coverage                         # Unit test coverage
```

All commands run from `ui/apps/platform/`.
