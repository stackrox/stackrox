# CLAUDE.md - StackRox UI Development Quick Reference

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
  "Fallback"
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
  return axios.get<MyResource>(`/v1/resources/${id}`).then((r) => r.data);
}
```

**Best practices:**

- Always check existing services before implementing new endpoints
- Use cancellation for heavy queries to prevent memory leaks
- Avoid creating wrapper hooks around `useRestQuery`—pass fetch functions directly

### State Management

- **Prefer React built-ins:** `useState`, `useReducer`, and React Context for new code
- **Avoid Redux/Redux-Saga:** Only modify existing code when migrating away
- **Component state:** Use `useState` for component-specific data
- **Shared state:** Use React Context for cross-component state

### Performance

- **Avoid premature `useMemo`:** Profile first to confirm it's necessary
- **Use `useMemo` only for:** Genuinely expensive calculations or values passed as deps to memoized components
- **Avoid `useMemo` for:** Simple values, primitives, inline objects/arrays
- **Prefer `useCallback`** often better than memoizing results

### Styling

- **Use PatternFly components** for consistent design and accessibility
- **Use PatternFly CSS variables** for custom styling
- **No CSS-in-JS:** Avoid styled-components
- **No Tailwind:** Use PatternFly utilities
- **Custom CSS:** CSS modules or plain CSS when PatternFly doesn't provide what you need

### Refactoring & Pattern Changes

- **Search first** with Grep/Glob for all occurrences before changing a pattern
- **Assess scope** before starting work
- **Batch fixes** all instances at once
- **Test after** to verify linting/type-checking passes

### RBAC (Role-Based Access Control)

Always check permissions when:

- Adding action components (buttons, modals, forms)
- Creating navigation links
- Implementing data fetching or mutations
- Rendering conditional UI

**Common pattern:**

```typescript
const { hasReadAccess, hasReadWriteAccess } = usePermissions();
const hasResourceAccess = hasReadAccess("ResourceName");
const shouldShowFeature = isFeatureEnabled && hasResourceAccess;
```

**Cross-link validation:** When Component A links to Page B, ensure users have access to both.

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

## Git Workflow

### Commit Message Format

- Use conventional commit format: `type: description` (e.g., `feat:`, `fix:`, `chore:`, `test:`)
- Default to `chore:` if unsure of the type
- Write messages explaining **why**, not just **what**
- Keep messages **concise and focused** — only include relevant information for a reviewer

**Example:**

```
feat: add dark mode toggle to settings

Users requested the ability to switch themes. This implementation
uses PatternFly's theme variables for consistency.
```

**Required:**

- Sign off commits with `-s` flag

**Prohibited:**

- Do NOT include "🤖 Generated with Claude Code" or similar markers
- Do NOT add any co-author lines
- Do NOT mention Claude as a contributor

### General Workflow

- Don't push changes or create PRs — let the developer handle that

---

## For More Information

- **Setup & Architecture** → See [README.md](./README.md) and [apps/platform/README.md](./apps/platform/README.md)
- **Routes & API patterns** → See [apps/platform/README.md](./apps/platform/README.md#routes) and [API section](./apps/platform/README.md#api)
- **Feature flags** → See [apps/platform/README.md](./apps/platform/README.md#feature-flags)
- **Plugin development** → See [apps/platform/README.md](./apps/platform/README.md#running-as-an-openshift-console-plugin)
