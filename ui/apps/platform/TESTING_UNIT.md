# Unit & Component Testing — Vitest

See [TESTING.md](./TESTING.md) for shared principles.

## Setup

- **Framework**: Vitest 4.x with `jsdom` environment
- **Globals**: `describe`, `it`, `test`, `expect`, `vi` are available without imports
- **Setup file**: `src/setupTests.js` runs before all tests — imports `@testing-library/jest-dom`, sets 15s timeout, and spies on `console.error` to catch unmocked API calls
- **Config**: `vite.config.js` → `test` section

## Rendering Components

### Simple components

Use `render` and `screen` from `@testing-library/react`:

```tsx
import { render, screen } from '@testing-library/react';
import MyComponent from './MyComponent';

test('renders the title', () => {
    render(<MyComponent title="Hello" />);
    expect(screen.getByText('Hello')).toBeInTheDocument();
});
```

### Components requiring providers

Use `ComponentTestProvider` when the component needs Redux, Router, or Apollo:

```tsx
import { render, screen } from '@testing-library/react';
import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import MyPage from './MyPage';

test('renders the page', () => {
    render(
        <ComponentTestProvider>
            <MyPage />
        </ComponentTestProvider>
    );
    expect(screen.getByText('Page Title')).toBeInTheDocument();
});
```

See `src/test-utils/ComponentTestProvider.tsx` for what it provides (Redux + BrowserRouter + CompatRouter + Apollo).

### Testing hooks

Use `renderHook` and `act` from `@testing-library/react`:

```typescript
// From src/hooks/useLocalStorage.test.ts
import { act, renderHook } from '@testing-library/react';
import useLocalStorage from './useLocalStorage';

test('should safely read and write local storage', () => {
    const { result } = renderHook(() =>
        useLocalStorage('test', 'initial', (v: unknown): v is string => typeof v === 'string')
    );
    expect(result.current[0]).toBe('initial');

    act(() => {
        result.current[1]('new value');
    });
    expect(result.current[0]).toBe('new value');
});
```

If a hook needs providers, pass a `wrapper` option to `renderHook`.

## User Interaction

Use `@testing-library/user-event` for realistic browser interactions:

```tsx
// From src/Components/BinderTabs/Binder.test.tsx
import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

test('selecting a new tab renders new content', async () => {
    const user = userEvent.setup();
    render(<BinderTabs>{/* ... */}</BinderTabs>);

    await act(() => user.click(screen.getByText('tab 2')));
    expect(screen.getByText('Tab 2 Content')).toBeInTheDocument();
});
```

Always call `userEvent.setup()` at the start — don't use the deprecated `userEvent.click()` directly.

## Mocking

### Module mocking with `vi.mock()`

Use `vi.hoisted()` to define mock variables that are accessible inside `vi.mock()`:

```typescript
// From src/ConsolePlugin/consoleFetchAxiosAdapter.test.ts
const mockConsoleFetch = vi.hoisted(() => vi.fn());
vi.mock('@openshift-console/dynamic-plugin-sdk', () => ({
    consoleFetch: mockConsoleFetch,
}));

beforeEach(() => {
    vi.clearAllMocks();
});
```

### Mocking custom hooks

```typescript
// From src/hooks/useMetadata.test.tsx
const mockUseRestQuery = vi.hoisted(() => vi.fn());
vi.mock('hooks/useRestQuery', () => ({ default: mockUseRestQuery }));

beforeEach(() => {
    vi.clearAllMocks();
    mockUseRestQuery.mockReturnValue({
        data: mockMetadata,
        isLoading: false,
        error: undefined,
        refetch: vi.fn(),
    });
});
```

### Function spies

```typescript
const hasReadAccess = vi.fn((resource) => resource === 'ImageAdministration');
// ... call code under test ...
expect(hasReadAccess).toHaveBeenCalledWith('ImageAdministration');
```

### Fake timers

```typescript
// From src/hooks/useRestMutation.test.ts
vi.useFakeTimers({ shouldAdvanceTime: true });
// ... trigger async operation ...
vi.runAllTimers();
await waitForNextUpdate(result);
```

Always pass `{ shouldAdvanceTime: true }` to avoid stalling promises.

### Mock at the right level

Mock **service modules** (the functions in `src/services/`), not the HTTP client:

```typescript
// Good — mock the service function
vi.mock('services/AlertService', () => ({
    fetchAlerts: vi.fn().mockResolvedValue([]),
}));

// Avoid — mocking axios directly couples tests to HTTP implementation
vi.mock('axios');
```

## Parameterized Tests

Use `it.each()` or `describe.each()` for table-driven tests:

```typescript
// From src/Containers/Policies/Wizard/Step3/policyCriteriaValidators.test.ts
const processCriteria = ['Process Name', 'Process Ancestor', 'Process Arguments'];

it.each(processCriteria)(
    'should fail when %s is present without File Path',
    (criterionName) => {
        const group = createPolicyGroup(criterionName);
        const result = validate(group);
        expect(result).toBe(false);
    }
);
```

## Async Patterns

### `waitFor` — wait for assertions to pass

```typescript
import { waitFor } from '@testing-library/react';

await waitFor(() => {
    expect(screen.getByText('Loaded')).toBeInTheDocument();
});
```

### `waitForNextUpdate` — wait for hook state change

```typescript
import waitForNextUpdate from 'test-utils/waitForNextUpdate';

const { result } = renderHook(() => useMyHook());
act(() => { result.current.trigger(); });
await waitForNextUpdate(result);
expect(result.current.data).toBeDefined();
```

See `src/test-utils/waitForNextUpdate.ts`.

### `actAndFlushTaskQueue` — flush microtasks

```typescript
import actAndFlushTaskQueue from 'test-utils/flushTaskQueue';

await actAndFlushTaskQueue(() => {
    result.current.update('value');
});
```

See `src/test-utils/flushTaskQueue.ts`.

## Test Utilities

| File | Purpose |
|------|---------|
| `src/test-utils/ComponentTestProvider.tsx` | Redux + Router + Apollo wrapper |
| `src/test-utils/renderWithRedux.jsx` | Redux-only wrapper (legacy, prefer ComponentTestProvider) |
| `src/test-utils/renderWithRouter.jsx` | Router-only wrapper |
| `src/test-utils/flushTaskQueue.ts` | Flush microtask queue in `act()` |
| `src/test-utils/waitForNextUpdate.ts` | Wait for hook state to change |
| `src/test-utils/apiEndpoints.ts` | GraphQL URL builder |

## Avoid

- **Snapshot tests** — use specific DOM assertions instead
- **`toBeTruthy()` for booleans** — use `toBe(true)` / `toBe(false)`
- **`typeof` checks** — test behavior, not types
- **Mocking internals** — mock at service boundaries
- **Direct `userEvent.click()`** — always call `userEvent.setup()` first
- **Missing `act()` wrappers** — wrap state updates in `act()` to avoid React warnings

## Canonical Examples

These files demonstrate preferred patterns:

| Pattern | File |
|---------|------|
| Hook testing | `src/hooks/useLocalStorage.test.ts` |
| Component + interaction | `src/Components/BinderTabs/Binder.test.tsx` |
| Parameterized tests | `src/Containers/Policies/Wizard/Step3/policyCriteriaValidators.test.ts` |
| Complex hook + localStorage | `src/hooks/useWidgetConfig.test.tsx` |
| Module mocking | `src/ConsolePlugin/consoleFetchAxiosAdapter.test.ts` |
| Utility function testing | `src/utils/searchUtils.test.ts` |
