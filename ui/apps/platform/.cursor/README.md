# StackRox UI Cursor Rules System

## Overview

This document describes the Cursor rules system for the StackRox UI team. These rules are designed to maintain consistency, enforce best practices, and help our 4-person UI engineering team write high-quality, maintainable code.

## Rule Files Structure

### 📁 `.cursor/rules/` Directory

Our Cursor rules are organized into specialized files for different concerns:

| File                        | Description                                   | Always Applied      |
| --------------------------- | --------------------------------------------- | ------------------- |
| `01-core-consistency.mdc`   | Core consistency rules for uniform code style | ✅ Yes              |
| `02-react-typescript.mdc`   | React and TypeScript best practices           | ✅ Yes              |
| `03-patternfly-styling.mdc` | PatternFly components and styling standards   | ✅ Yes              |
| `04-service-layer.mdc`      | Service layer and API integration patterns    | ✅ Yes              |
| `05-testing-patterns.mdc`   | Testing patterns and guidelines               | ❌ Context-specific |
| `06-accessibility.mdc`      | Accessibility standards and best practices    | ❌ Context-specific |

## Technology Stack

Our rules are tailored for the following technology stack:

- **React 18** with **TypeScript**
- **PatternFly v5** UI components
- **Tailwind CSS** for utility styling
- **Redux + Redux-Saga** for state management
- **Apollo Client** for GraphQL
- **Vite** for build tooling
- **Vitest** for unit testing
- **Cypress** for E2E testing

## Key Patterns Enforced

### 1. File Organization & Naming

- **NEW files**: Always use `.tsx` for React components, `.ts` for utilities
- **Component files**: PascalCase matching component name
- **Utility files**: camelCase describing function
- **Constants**: camelCase with descriptive names

### 2. Import/Export Standards

- **Import order**: External libraries → Internal modules → Relative imports
- **Components**: Default exports for main component, named exports for utilities
- **Services**: Named exports for all functions
- **Constants**: Named exports with `as const` assertion

### 3. TypeScript Patterns

- **Interfaces over types** for object shapes
- **Proper prop typing** - avoid `any`, use specific types
- **Type guards** for runtime type checking
- **Generics** for reusable components

### 4. Error Handling

- **Typed errors** using custom error classes
- **Consistent error responses** following `AuthHttpError` pattern
- **Proper error propagation** - don't swallow errors

### 5. Component Structure

```typescript
// Standard component template
function ComponentName({ prop1, prop2, ...props }: Props) {
    // 1. Hooks (useEffect, useState, custom hooks)
    // 2. Event handlers
    // 3. Computed values
    // 4. Render logic

    return (
        <div {...props}>
            {/* JSX */}
        </div>
    );
}
```

## Team Workflow

### Daily Development

1. **Before Starting Work**

    - Pull latest changes
    - Check ESLint passes: `npm run lint`
    - Run TypeScript check: `npm run tsc`

2. **While Developing**

    - Cursor will automatically apply rules based on file context
    - Pay attention to rule suggestions and error highlighting
    - Follow the component templates provided in rules

3. **Before Committing**
    - [ ] ESLint passes without warnings
    - [ ] TypeScript compilation succeeds
    - [ ] Component has proper prop types
    - [ ] Error states are handled
    - [ ] Loading states are implemented
    - [ ] Tests are updated (if applicable)

### Code Review Process

#### Focus Areas

1. **Consistency**: Are naming conventions followed?
2. **Types**: Are proper TypeScript types used?
3. **Error handling**: Are errors handled appropriately?
4. **Performance**: Are there any obvious performance issues?
5. **Accessibility**: Are accessibility requirements met?

#### Common Issues to Watch For

- Missing TypeScript types or using `any`
- Inconsistent naming conventions
- Missing error handling
- Not following PatternFly component patterns
- Missing accessibility attributes
- Inefficient re-renders

## Rule Categories Explained

### Core Consistency Rules (`01-core-consistency.mdc`)

**Always Applied**: Yes

These rules ensure uniform code style across the entire codebase:

- File naming and organization
- Import/export patterns
- Basic component structure
- Error handling standards
- Code comments and documentation

### React & TypeScript Rules (`02-react-typescript.mdc`)

**Always Applied**: Yes

React and TypeScript specific best practices:

- Functional components over classes
- Custom hooks standards
- TypeScript interface design
- State management patterns
- Performance optimization
- Common pitfalls to avoid

### PatternFly & Styling Rules (`03-patternfly-styling.mdc`)

**Always Applied**: Yes

UI component and styling standards:

- PatternFly component usage priority
- Tailwind CSS integration
- CSS variables and theming
- Responsive design patterns
- Accessibility considerations

### Service Layer Rules (`04-service-layer.mdc`)

**Always Applied**: Yes

API integration and service layer patterns:

- Service organization structure
- Error handling standards
- Request/response patterns
- Custom hooks for API integration
- Caching strategies
- Testing approaches

### Testing Patterns (`05-testing-patterns.mdc`)

**Always Applied**: Context-specific

Testing guidelines and patterns:

- Unit testing with Vitest
- Component testing with React Testing Library
- Custom hook testing
- E2E testing with Cypress
- Test organization and data management

### Accessibility Rules (`06-accessibility.mdc`)

**Always Applied**: Context-specific

Accessibility standards and best practices:

- WCAG 2.1 AA compliance
- Semantic HTML and ARIA
- Form accessibility
- Keyboard navigation
- Screen reader support
- Testing for accessibility

## Integration with Existing Tools

### ESLint Integration

Our Cursor rules complement existing ESLint configuration:

- Cursor rules focus on patterns and architecture
- ESLint handles syntax and code quality
- Both work together for comprehensive code quality

### TypeScript Integration

- Cursor rules provide guidance on TypeScript patterns
- TypeScript compiler enforces type safety
- Rules help with proper interface design and usage

### Testing Integration

- Rules provide testing patterns and examples
- Work with existing Vitest and Cypress setups
- Help maintain consistent testing approaches

## Common Scenarios

### Creating a New Component

1. Use `.tsx` extension
2. Follow component template from rules
3. Define proper TypeScript interfaces
4. Use PatternFly components when possible
5. Include accessibility attributes
6. Add tests if complex logic

### Adding a New Service

1. Create in `src/services/` directory
2. Use proper error handling patterns
3. Define TypeScript interfaces for responses
4. Add proper JSDoc comments
5. Include unit tests
6. Use custom hooks for React integration

### Updating Existing Code

1. When making significant changes, migrate `.js` → `.ts`
2. Follow existing patterns in the file
3. Improve TypeScript types if needed
4. Update tests if behavior changes
5. Ensure accessibility is maintained

## Customization and Updates

### Adding New Rules

1. Create new `.mdc` file in `.cursor/rules/`
2. Include proper YAML frontmatter
3. Document the rule purpose and usage
4. Add examples and anti-patterns
5. Update this README

### Modifying Existing Rules

1. Discuss changes with team
2. Update the rule file
3. Test with existing codebase
4. Update documentation
5. Communicate changes to team

### Rule Conflicts

1. Core consistency rules take precedence
2. Discuss with team if rules seem contradictory
3. Update rules to resolve conflicts
4. Document resolution in README

## Troubleshooting

### Common Issues

**Rule not being applied**

- Check file path matches glob pattern
- Verify YAML frontmatter is correct
- Restart Cursor if needed

**Conflicting suggestions**

- Check if multiple rules apply to same pattern
- Review rule priorities
- Discuss with team if clarification needed

**Performance issues**

- Rules are cached, restart Cursor if needed
- Large rule files may slow down editor
- Consider splitting complex rules

### Getting Help

1. **Check this README** for common patterns
2. **Review existing code** for examples
3. **Ask team members** for clarification
4. **Update documentation** if something is unclear

## Best Practices Summary

### DO:

- ✅ Follow the established patterns in rules
- ✅ Use TypeScript properly with specific types
- ✅ Handle errors consistently
- ✅ Use PatternFly components when available
- ✅ Include accessibility attributes
- ✅ Write tests for complex logic
- ✅ Document complex business logic

### DON'T:

- ❌ Use `any` type unless absolutely necessary
- ❌ Skip error handling
- ❌ Ignore accessibility requirements
- ❌ Create duplicate components
- ❌ Use hardcoded values instead of constants
- ❌ Skip testing for new features
- ❌ Override PatternFly styles directly

## Team Guidelines

### Onboarding New Team Members

1. Review this README and rule files
2. Set up development environment
3. Review existing code examples
4. Pair with experienced team member
5. Start with small, well-defined tasks

### Knowledge Sharing

- Weekly code review sessions
- Monthly pattern discussion meetings
- Document new patterns as they emerge
- Share interesting solutions with team

### Rule Evolution

- Rules should evolve with team needs
- Regular review of rule effectiveness
- Update based on lessons learned
- Keep rules practical and enforceable

## Resources

### External Documentation

- [PatternFly Design System](https://www.patternfly.org/)
- [React TypeScript Cheatsheet](https://react-typescript-cheatsheet.netlify.app/)
- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/Understanding/)
- [Testing Library Documentation](https://testing-library.com/)

### Internal Resources

- ESLint configuration: `eslint.config.js`
- TypeScript configuration: `tsconfig.json`
- Testing setup: `vitest.config.ts`
- Cypress configuration: `cypress.config.js`
