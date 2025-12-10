# AGENTS.md - StackRox UI Development

This guide provides instructions for AI agents working on the StackRox UI codebase.

## Communication Style

- Be extremely concise. Sacrifice grammar for the sake of concision.

---

## Commands

### Development

```bash
npm run start             # Start dev server (from apps/platform)
npm run build             # Build for production
npm run clean             # Clean artifacts and test results
npm run start:ocp-plugin  # OpenShift Console plugin dev server
npm run build:ocp-plugin  # Build OpenShift Console plugin
```

### Testing

```bash
npm run test                                  # Unit tests (Vitest)
npm run test-coverage                         # Unit tests with coverage
npm run test -- --testNamePattern="Name"      # Single unit test
npm run test -- src/path/to/test.test.ts      # Test by file path
npm run test-e2e                              # E2E tests (Cypress)
npm run cypress-spec -- "test-name.js"        # Single E2E test
npm run test-component                        # Component tests
npm run cypress-open                          # Interactive test runner
```

### Code Quality

```bash
npm run lint:fast-dev      # Quick ESLint check (skips slow rules)
npm run lint:fast-dev:fix  # Quick ESLint with auto-fix
npm run lint:fix           # Full ESLint with auto-fix
npm run tsc                # TypeScript type checking
```

**Important:** Use IDE diagnostics (real-time linting) instead of running full lint commands — they're much faster for development. Use `lint:fast-dev` only if you need to run lint from the command line.

---

## Code Guidelines

### Coding Patterns

#### Conditional Rendering

Process data before JSX to avoid unnecessary renders:

```tsx
// Process first, then render
const validItems = rawData ? normalizeArray(rawData).filter(isValid) : [];
return validItems.length > 0 ? (
  <Stack>
    {validItems.map((item) => (
      <Component key={item} />
    ))}
  </Stack>
) : (
  'Fallback'
);
```

### Data Fetching

Use custom hooks for data fetching with built-in state management:

```typescript
// Queries (GET)
const { data, isLoading, error, refetch } = useRestQuery(() =>
  fetchMyResources(filters, page)
);

// Mutations (POST/PUT/DELETE)
const { mutate, isLoading, isSuccess } = useRestMutation(createMyResource, {
  onSuccess: () => refetch(),
});
```

**Service pattern:** Create service functions in `src/services/` that extract and return data

```typescript
export function fetchMyResource(id: string): Promise<MyResource> {
  return axios
    .get<MyResource>(`/v1/resources/${id}`)
    .then((resource) => resource.data);
}
```

**Best practices:**

- Always check existing services before implementing new endpoints
- Use cancellation for heavy queries to prevent memory leaks

### State Management

**Hierarchy of preferred approaches:**

1. **Component state first:** Use `useState` for data that only one component needs
2. **Prop drilling:** Pass state down through props to child components if needed
3. **React Context:** Use if prop drilling becomes excessive (lift state to closest shared ancestor first)
4. **Redux/Redux-Saga:** Only modify existing Redux code — never add new Redux code for new features

**Decision tree:**
- Does only one component use this state? → `useState`
- Do a few related components need it? → Find closest ancestor, pass props
- Lots of prop drilling? → Ask developer preference; consider Context if approved
- State already in Redux? → Only modify it there, don't duplicate elsewhere

**Best practices:**
- **No state duplication:** Never sync the same data in multiple places — single source of truth
- **Complex state:** Use `useReducer` instead of multiple `useState` calls for related fields
- **Custom hooks:** Extract state logic into a hook when it's reused or complex (e.g., `useFormState`, `useFilters`)

### Performance

- **Avoid premature `useMemo`:** Profile first to confirm it's necessary
- **Use `useMemo` only for:** Genuinely expensive calculations or values passed as deps to memoized components
- **Avoid `useMemo` for:** Simple values, primitives, inline objects/arrays
- **Prefer `useCallback`** often better than memoizing results

### Styling

- **Use PatternFly components** for consistent design and accessibility
- **Avoid custom CSS** and PatternFly style overrides whenever possible
- **Use PatternFly layout components** (Flex, Split, Stack, Bullseye) instead of organizing components with utility classes or CSS
- **Avoid plain HTML elements** when a PatternFly alternative exists (Form, Text, Title, Table)
- **Use PatternFly CSS variables** for custom styling
- **No CSS-in-JS:** Avoid styled-components
- **No Tailwind:** Use PatternFly utilities

### Refactoring & Pattern Changes

- **Search first** with Grep/Glob for all occurrences before changing a pattern
- **Assess scope** before starting work
- **Test after** to verify linting/type-checking passes

---

## Testing Strategy

- **Unit tests** (Vitest): Components in isolation, mocked dependencies, user interactions
- **E2E tests** (Cypress): Critical user journeys, accessibility testing (cypress-axe)
- **Component tests** (Cypress): Real DOM rendering, PatternFly integrations

**Before committing:**

- If modifying existing code with tests, run those specific tests to ensure they still pass (don't run the full suite)
- For new features, ask the developer if they want tests written
- When writing tests, focus on happy path and critical user flows — avoid over-engineering

**For detailed testing guidance:** See [apps/platform/README.md#testing](./apps/platform/README.md#testing)

---

## For More Information

- **Setup & Architecture** → See [README.md](./README.md) and [apps/platform/README.md](./apps/platform/README.md)
- **Routes & API patterns** → See [apps/platform/README.md](./apps/platform/README.md#routes) and [API section](./apps/platform/README.md#api)
- **Feature flags** → See [apps/platform/README.md](./apps/platform/README.md#feature-flags)
- **Plugin development** → See [apps/platform/README.md](./apps/platform/README.md#running-as-an-openshift-console-plugin)
