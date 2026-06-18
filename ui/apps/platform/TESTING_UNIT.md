# Unit Testing — Vitest

See [TESTING.md](./TESTING.md) for shared principles.

Unit tests cover **pure logic only** — no DOM, no React, no rendering. If your code needs `render()`, `cy.mount()`, or any form of component rendering, write a [component test](./TESTING_COMPONENT.md) instead.

## Setup

- **Framework**: Vitest
- **Globals**: `describe`, `it`, `test`, `expect`, `vi` are available without imports
- **Config**: `vite.config.js` → `test` section

## What Belongs Here

- Utility functions (formatters, parsers, validators)
- Data transformations (sorting, filtering, mapping)
- Reducers and state logic
- Configuration builders
- Business logic helpers

## Project Conventions

### Mock at the service layer

Mock **service modules** (functions in `src/services/`), not the HTTP client:

```typescript
// Good — mock the service function
vi.mock('services/AlertService', () => ({
    fetchAlerts: vi.fn().mockResolvedValue([]),
}));

// Avoid — couples tests to HTTP implementation
vi.mock('axios');
```

_Note that mocking service functions is a warning that component or e2e tests may be more appropriate._

### Fake timers need `shouldAdvanceTime`

Always pass `{ shouldAdvanceTime: true }` to `vi.useFakeTimers()` — without it, promises stall indefinitely.

### `console.error` spy

`src/setupTests.js` spies on `console.error` and fails any test that triggers one. The most common cause is an unmocked API call — look for `ECONNREFUSED` errors in the output to identify which service function needs mocking.

## Canonical Examples

| Pattern | File |
|---------|------|
| Utility function testing | `src/utils/searchUtils.test.ts` |
| Parameterized tests | `src/Containers/Policies/Wizard/Step3/policyCriteriaValidators.test.ts` |
| Module mocking | `src/ConsolePlugin/consoleFetchAxiosAdapter.test.ts` |
