# StackRox UI Cursor Rules

This directory contains Cursor rules that help maintain consistency, best practices, and code quality across the StackRox UI codebase.

## 📁 Rule Files

### Core Rules (Always Applied)

-   **`ui-developer-rule.mdc`** - General UI development patterns and best practices
-   **`typescript-rules.mdc`** - TypeScript type safety and patterns
-   **`react-component-rules.mdc`** - React component structure and performance

### Context-Specific Rules (Applied Based on File Location)

-   **`testing-rules.mdc`** - Cypress and Jest testing patterns
-   **`context-specific-rules.mdc`** - Directory-specific patterns for services and hooks

## 🎯 Rule Application

### Glob Patterns

Rules use glob patterns to target specific files:

```yaml
# All TypeScript/JavaScript files
globs: ["apps/platform/src/**/*.{ts,tsx,js,jsx}"]

# Components only
globs: ["apps/platform/src/Components/**/*.{tsx,jsx}"]

# Tests only
globs: ["apps/platform/cypress/**/*.{js,ts}"]
```

### Rule Priority

-   `alwaysApply: true` - Applied regardless of context
-   `alwaysApply: false` - Applied only when working in matching files

## 🔧 Customization

### Adding New Rules

1. Create a new `.mdc` file in this directory
2. Add proper YAML frontmatter:
    ```yaml
    ---
    description: Your rule description
    globs: ['pattern/to/match/**/*.ts']
    alwaysApply: false
    ---
    ```
3. Write clear, actionable rules with examples

### Modifying Existing Rules

1. Update the relevant `.mdc` file
2. Test with sample code
3. Document changes in commit messages
4. Share updates with the team

## 📊 Rule Categories

### 1. **Code Structure**

-   Component organization
-   File naming conventions
-   Import/export patterns
-   Directory structure

### 2. **Type Safety**

-   TypeScript patterns
-   API response typing
-   Error handling
-   Generic types

### 3. **Performance**

-   React optimization
-   Bundle optimization
-   Memoization patterns
-   Lazy loading

### 4. **Testing**

-   Test structure
-   Mock patterns
-   Selector strategies
-   Accessibility testing

### 5. **Code Quality**

-   Error handling
-   Logging patterns
-   Documentation
-   Accessibility

## 🚀 Advanced Features

### Conditional Rules

Rules can be contextual based on file location:

```markdown
## Service Layer Rules (for /services/ directory)

When working in the `services/` directory:

-   Use consistent naming patterns
-   Include explicit return types
-   Handle errors consistently
```

### Code Examples

Rules include practical examples:

```typescript
// Good pattern
const Component = ({ title, isLoading = false }: Props) => {
    // Implementation
};

// Avoid pattern
const Component = (props: any) => {
    // Implementation
};
```

### Pattern Enforcement

Rules specify what to do AND what to avoid:

-   ✅ **DO**: Use explicit TypeScript types
-   ❌ **AVOID**: Using `any` type

## 📈 Measuring Success

### Key Metrics

-   **Code consistency** - Similar patterns across files
-   **Type safety** - Fewer `any` types, better error handling
-   **Performance** - Proper memoization and optimization
-   **Testing** - Consistent test patterns and coverage

### Review Process

1. **PR Reviews** - Check for rule adherence
2. **Code Quality** - Monitor TypeScript errors
3. **Performance** - Review bundle size and metrics
4. **Team Feedback** - Gather input on rule effectiveness

## 🔄 Rule Evolution

### Regular Updates

-   Review rules quarterly
-   Add new patterns as they emerge
-   Remove outdated patterns
-   Update based on team feedback

### Version Control

-   Track rule changes in git
-   Document breaking changes
-   Communicate updates to team
-   Maintain backward compatibility when possible

## 📚 Resources

### Documentation

-   [Cursor Rules Documentation](https://cursor.sh/docs/rules)
-   [TypeScript Best Practices](https://typescript-eslint.io/docs/)
-   [React Best Practices](https://reactjs.org/docs/thinking-in-react.html)
-   [Testing Best Practices](https://testing-library.com/docs/guiding-principles)

### Team Guidelines

-   Code review standards
-   TypeScript configuration
-   ESLint rules
-   Prettier configuration

---

_These rules are living documents that evolve with our codebase and team practices. Feedback and contributions are welcome!_
