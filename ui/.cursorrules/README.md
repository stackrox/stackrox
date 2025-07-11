# StackRox UI Cursor Rules

A comprehensive set of Cursor rules designed to maintain consistency and enforce best practices across the StackRox UI team. These rules help ensure code quality, reduce review time, and guide developers toward established patterns.

## 📋 Table of Contents

-   [Overview](#overview)
-   [Rule Files](#rule-files)
-   [Getting Started](#getting-started)
-   [Team Workflow](#team-workflow)
-   [Rule System Architecture](#rule-system-architecture)
-   [Examples](#examples)
-   [Troubleshooting](#troubleshooting)
-   [Contributing](#contributing)
-   [Maintenance](#maintenance)

## 🎯 Overview

### Team Context

-   **Team Size**: 4 UI engineers with varying experience levels
-   **Primary Goal**: Enforce consistency and guide developers toward better patterns
-   **Technology Stack**: React 18, TypeScript, Redux, PatternFly, Tailwind CSS, Cypress/Vitest

### Key Benefits

-   **Reduced Code Review Time**: Focus on logic rather than style/consistency issues
-   **Faster Onboarding**: New team members learn patterns automatically
-   **Consistent Code Quality**: Predictable code structure across the project
-   **Fewer Bugs**: Consistent patterns reduce common mistakes
-   **Better Maintainability**: Unified approach to component architecture

## 📚 Rule Files

### Core Rules (Always Applied)

-   **[core-development-patterns.md](./core-development-patterns.md)** - Fundamental patterns for all code
    -   File organization and naming conventions
    -   Import organization and error handling
    -   Documentation standards and performance patterns

### Specialized Rules (Context-Specific)

-   **[react-component-patterns.md](./react-component-patterns.md)** - React component best practices

    -   Component definition and props patterns
    -   State management and event handling
    -   Performance optimization and testing integration

-   **[typescript-patterns.md](./typescript-patterns.md)** - TypeScript patterns and type safety

    -   Interface definitions and API types
    -   Generic types and utility types
    -   Error handling and validation types

-   **[testing-patterns.md](./testing-patterns.md)** - Testing strategies and patterns

    -   Unit testing with React Testing Library
    -   Cypress component and E2E testing
    -   Mock data management and test utilities

-   **[service-layer-patterns.md](./service-layer-patterns.md)** - API integration and service patterns

    -   Service organization and error handling
    -   GraphQL and REST API patterns
    -   Caching and file upload/download utilities

-   **[styling-ui-patterns.md](./styling-ui-patterns.md)** - UI styling and component patterns
    -   PatternFly and Tailwind CSS integration
    -   Layout patterns and responsive design
    -   Theme consistency and accessibility

## 🚀 Getting Started

### Installation

1. Ensure you have Cursor installed and configured
2. Clone the repository and navigate to the UI directory
3. The rules are automatically applied based on file patterns

### First Steps

1. **Read the Core Rules**: Start with `core-development-patterns.md`
2. **Review Component Patterns**: Understand React patterns in `react-component-patterns.md`
3. **Check TypeScript Guidelines**: Follow patterns in `typescript-patterns.md`
4. **Understand Testing Approach**: Review `testing-patterns.md`

### Validation

Create a simple component to test the rules:

```typescript
// src/Components/TestComponent/TestComponent.tsx
import React, { useState } from 'react';
import { Card, CardBody, Button } from '@patternfly/react-core';

import { UserProfile } from 'types/user.proto';

interface TestComponentProps {
    userId: string;
    onEdit?: (userId: string) => void;
}

const TestComponent: React.FC<TestComponentProps> = ({ userId, onEdit }) => {
    const [isLoading, setIsLoading] = useState(false);

    const handleEdit = () => {
        onEdit?.(userId);
    };

    return (
        <Card data-testid="test-component">
            <CardBody>
                <p>User ID: {userId}</p>
                <Button onClick={handleEdit} isLoading={isLoading}>
                    Edit User
                </Button>
            </CardBody>
        </Card>
    );
};

export default TestComponent;
```

If Cursor provides suggestions following our patterns, the rules are working correctly.

## 🔄 Team Workflow

### Development Process

1. **File Creation**: Use established naming conventions
2. **Import Organization**: Follow the consistent import order
3. **Component Structure**: Use established patterns for React components
4. **Error Handling**: Implement consistent error patterns
5. **Testing**: Add appropriate test coverage
6. **Documentation**: Include JSDoc comments for public APIs

### Code Review Checklist

Before submitting PRs, ensure:

-   [ ] All imports follow the established order
-   [ ] Error handling is consistent and informative
-   [ ] TypeScript types are properly defined
-   [ ] Components use established patterns
-   [ ] Test files follow testing patterns
-   [ ] Documentation is complete and accurate

### PR Review Focus

With consistent patterns enforced, code reviews can focus on:

-   **Business Logic**: Does the code solve the right problem?
-   **Architecture**: Is the approach scalable and maintainable?
-   **Performance**: Are there any performance concerns?
-   **Security**: Are there any security implications?
-   **User Experience**: Does the implementation meet UX requirements?

## 🏗️ Rule System Architecture

### Rule File Structure

```yaml
---
description: Clear description of what this rule covers
globs: ['specific/path/patterns/**/*.{ext}'] # Target specific files
alwaysApply: true/false # Use true for consistency rules
---
# Rule Content
- Clear categories and sections
- Practical code examples
- DO vs DON'T sections
- Integration examples
- Real-world scenarios
```

### Consistency Matrix

All rule files maintain consistency in:

-   **Examples**: Same component names (`UserProfile`, `ComplianceReport`)
-   **Patterns**: Identical import orders, error handling, etc.
-   **Terminology**: Same terms for same concepts
-   **Style**: Consistent formatting and explanation depth

### File Targeting

```typescript
// Core patterns - always applied
globs: ['**/*.{js,jsx,ts,tsx}'];
alwaysApply: true;

// React patterns - component files
globs: ['**/*.{jsx,tsx}', '**/*Component*.{js,ts}'];
alwaysApply: false;

// TypeScript patterns - TS files
globs: ['**/*.{ts,tsx}', '**/*types*.{js,jsx}'];
alwaysApply: false;

// Testing patterns - test files
globs: ['**/*.test.{js,jsx,ts,tsx}', '**/*.spec.{js,jsx,ts,tsx}'];
alwaysApply: false;

// Service patterns - service files
globs: ['**/services/**/*.{js,ts}', '**/*Service*.{js,ts}'];
alwaysApply: false;

// Styling patterns - all files for UI consistency
globs: ['**/*.{css,scss}', '**/*.{js,jsx,ts,tsx}'];
alwaysApply: false;
```

## 💡 Examples

### Consistent Component Example

```typescript
// ✅ GOOD - Follows all established patterns
import React, { useState, useEffect, useCallback } from 'react';
import { Card, CardBody, Button, Alert } from '@patternfly/react-core';

import { UserProfile } from 'types/user.proto';
import { useUserProfile } from 'hooks/useUserProfile';
import { formatDate } from 'utils/dateUtils';

interface UserProfileCardProps {
    userId: string;
    showAvatar?: boolean;
    onEdit?: (userId: string) => void;
}

const UserProfileCard: React.FC<UserProfileCardProps> = ({ userId, showAvatar = true, onEdit }) => {
    const { data: user, error, isLoading } = useUserProfile(userId);

    const handleEditClick = useCallback(() => {
        onEdit?.(userId);
    }, [onEdit, userId]);

    if (isLoading) {
        return (
            <Card>
                <CardBody>
                    <div>Loading user profile...</div>
                </CardBody>
            </Card>
        );
    }

    if (error) {
        return (
            <Alert variant="danger" title="Error">
                {error}
            </Alert>
        );
    }

    return (
        <Card data-testid="user-profile-card">
            <CardBody>
                {showAvatar && (
                    <img
                        src={user.avatar}
                        alt={`${user.name} avatar`}
                        className="h-12 w-12 rounded-full"
                    />
                )}
                <h2>{user.name}</h2>
                <p>Email: {user.email}</p>
                <p>Last login: {formatDate(user.lastLogin)}</p>
                <Button onClick={handleEditClick}>Edit Profile</Button>
            </CardBody>
        </Card>
    );
};

export default UserProfileCard;
```

### Consistent Service Example

```typescript
// ✅ GOOD - Follows service patterns
import axios from './instance';
import { UserProfile, CreateUserRequest } from 'types/user.proto';

const baseUrl = '/api/v1/users';

/**
 * Fetches user profile by ID
 * @param userId - The unique identifier for the user
 * @returns Promise resolving to user profile data
 * @throws Error when user is not found or API is unavailable
 */
export async function fetchUserProfile(userId: string): Promise<UserProfile> {
    if (!userId) {
        throw new Error('User ID is required');
    }

    try {
        const response = await axios.get<UserProfile>(`${baseUrl}/${userId}`);
        return response.data;
    } catch (error) {
        console.error('Failed to fetch user profile:', error);
        throw new Error(`Unable to load user profile: ${error.message}`);
    }
}
```

### Consistent Test Example

```typescript
// ✅ GOOD - Follows testing patterns
import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import UserProfileCard from './UserProfileCard';
import { mockUserProfile } from 'test-utils/mockData';

describe('UserProfileCard', () => {
    it('renders user profile correctly', () => {
        render(<UserProfileCard userId="user-123" />);

        expect(screen.getByText(mockUserProfile.name)).toBeInTheDocument();
        expect(screen.getByText(mockUserProfile.email)).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /edit profile/i })).toBeInTheDocument();
    });

    it('calls onEdit when edit button is clicked', async () => {
        const user = userEvent.setup();
        const mockOnEdit = jest.fn();

        render(<UserProfileCard userId="user-123" onEdit={mockOnEdit} />);

        await user.click(screen.getByRole('button', { name: /edit profile/i }));

        expect(mockOnEdit).toHaveBeenCalledWith('user-123');
    });
});
```

## 🔧 Troubleshooting

### Common Issues

#### Rules Not Applying

1. **Check file patterns**: Ensure your files match the glob patterns
2. **Verify Cursor configuration**: Make sure Cursor is reading the `.cursorrules` directory
3. **File location**: Ensure rule files are in the correct location

#### Conflicting Suggestions

1. **Check rule precedence**: Core rules (alwaysApply: true) take precedence
2. **Review glob patterns**: Ensure patterns don't overlap unexpectedly
3. **File specificity**: More specific patterns override general ones

#### Performance Issues

1. **Rule complexity**: Very complex rules may slow down suggestions
2. **File size**: Large rule files may impact performance
3. **Pattern matching**: Overly broad glob patterns may affect performance

### Debugging Steps

1. **Test with simple examples**: Create basic components to verify rules work
2. **Check Cursor logs**: Look for error messages in Cursor's developer tools
3. **Validate rule syntax**: Ensure YAML frontmatter is correctly formatted
4. **Test individual rules**: Temporarily disable other rules to isolate issues

## 🤝 Contributing

### Adding New Rules

1. **Identify the need**: What pattern or consistency issue needs addressing?
2. **Choose the right file**: Core rules vs. specialized rules
3. **Follow existing patterns**: Maintain consistency with established examples
4. **Add comprehensive examples**: Include both good and bad examples
5. **Update documentation**: Add to this README if needed

### Modifying Existing Rules

1. **Discuss with team**: Ensure changes align with team needs
2. **Update all examples**: Maintain consistency across rule files
3. **Test thoroughly**: Verify changes don't break existing patterns
4. **Update documentation**: Reflect changes in README and examples

### Rule Quality Guidelines

-   **Clear and specific**: Rules should be unambiguous
-   **Actionable**: Provide clear guidance on what to do
-   **Consistent**: Align with other rules and established patterns
-   **Well-documented**: Include rationale and examples
-   **Tested**: Verify rules work as expected

## 🔧 Maintenance

### Regular Reviews

-   **Monthly**: Review rule effectiveness and team feedback
-   **Quarterly**: Update rules based on new patterns or technologies
-   **As needed**: Address specific issues or inconsistencies

### Rule Updates

1. **Technology changes**: Update rules when libraries/frameworks change
2. **Team feedback**: Incorporate suggestions from team members
3. **New patterns**: Add rules for emerging patterns or best practices
4. **Performance optimization**: Improve rule efficiency and clarity

### Version Control

-   **Track changes**: Use meaningful commit messages for rule updates
-   **Document reasons**: Include rationale for rule changes
-   **Communicate updates**: Notify team of significant rule changes

### Success Metrics

-   **Code review time**: Measure time spent on style/consistency issues
-   **Onboarding speed**: Track how quickly new team members adapt
-   **Code consistency**: Monitor consistency across the codebase
-   **Bug reduction**: Track bugs related to inconsistent patterns

## 📞 Support

### Getting Help

-   **Team Discussion**: Bring up questions in team meetings
-   **Documentation**: Check this README and individual rule files
-   **Examples**: Look at existing code that follows the patterns
-   **Experimentation**: Try small examples to understand rule behavior

### Feedback and Suggestions

-   **Rule improvements**: Suggest enhancements to existing rules
-   **New patterns**: Propose new rules for emerging patterns
-   **Documentation**: Help improve clarity and examples
-   **Bug reports**: Report issues with rule behavior

---

## 📈 Success Metrics

### Expected Outcomes

-   **75% reduction** in code review time spent on style issues
-   **50% faster** onboarding for new team members
-   **Consistent code structure** across 95% of new components
-   **Reduced bugs** from inconsistent patterns
-   **Improved maintainability** and code reusability

### Measurement Approach

1. **Code Review Analytics**: Track time spent on different types of feedback
2. **Onboarding Surveys**: Measure new team member experience
3. **Code Quality Metrics**: Monitor consistency across the codebase
4. **Bug Tracking**: Track bugs related to inconsistent patterns
5. **Team Satisfaction**: Regular team feedback on rule effectiveness

---

_This rule system is designed to evolve with the team's needs. Regular feedback and updates ensure it remains effective and relevant._
