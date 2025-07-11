# StackRox UI Quick Reference Guide

A fast reference for daily development tasks. For detailed information, see the full rule files.

## 📂 File Organization

### Naming Conventions

```
✅ GOOD
Components/UserProfile/UserProfile.tsx
services/userService.ts
hooks/useUserProfile.ts
types/user.proto.ts

❌ AVOID
components/userprofile.jsx
Services/USER_SERVICE.js
hooks/UseUserProfile.js
types/user-types.ts
```

### File Extensions

-   **TypeScript**: `.ts` for utilities, services, types
-   **React TypeScript**: `.tsx` for React components
-   **Legacy**: `.js/.jsx` only for existing files being migrated

## 📦 Import Order (ALWAYS use this order)

```typescript
// 1. React & React ecosystem
import React, { useState, useEffect } from 'react';
import { useSelector, useDispatch } from 'react-redux';

// 2. Third-party libraries
import { Button, Card } from '@patternfly/react-core';
import axios from 'axios';

// 3. Type imports
import { UserProfile } from 'types/user.proto';

// 4. Internal hooks
import { useUserProfile } from 'hooks/useUserProfile';

// 5. Internal services
import { userService } from 'services/userService';

// 6. Internal utilities
import { formatDate } from 'utils/dateUtils';

// 7. Relative imports
import ComponentSpecificComponent from './ComponentSpecificComponent';

// 8. Styles
import './UserProfile.css';
```

## ⚛️ React Component Template

```typescript
import React, { useState, useCallback } from 'react';
import { Card, CardBody, Button } from '@patternfly/react-core';

import { UserProfile } from 'types/user.proto';
import { useUserProfile } from 'hooks/useUserProfile';

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

    // Early returns for loading/error states
    if (isLoading) {
        return <div>Loading...</div>;
    }

    if (error) {
        return <div>Error: {error}</div>;
    }

    return (
        <Card data-testid="user-profile-card">
            <CardBody>
                <h2>{user.name}</h2>
                <Button onClick={handleEditClick}>Edit</Button>
            </CardBody>
        </Card>
    );
};

export default UserProfileCard;
```

## 🎯 TypeScript Patterns

### Interface Definition

```typescript
interface UserProfile {
    /** Unique identifier for the user */
    id: string;
    /** User's display name */
    name: string;
    /** User's email address */
    email: string;
    /** User's avatar URL */
    avatar?: string; // Use ? for optional properties
    /** Whether the user is currently active */
    isActive: boolean;
    /** Timestamp of user's last login */
    lastLogin: string; // ISO 8601 date string
}
```

### Props Interface

```typescript
interface UserProfileProps {
    /** The ID of the user to display */
    userId: string;
    /** Whether to show the user's avatar */
    showAvatar?: boolean;
    /** Custom CSS classes to apply */
    className?: string;
    /** Test ID for testing */
    testId?: string;
    /** Callback fired when user clicks edit button */
    onEdit?: (userId: string) => void;
}
```

## 🔧 Service Template

```typescript
import axios from './instance';
import { UserProfile } from 'types/user.proto';

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

## 🧪 Test Template

```typescript
import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import UserProfileCard from './UserProfileCard';
import { mockUserProfile } from 'test-utils/mockData';

describe('UserProfileCard', () => {
    it('renders user profile correctly', () => {
        render(<UserProfileCard userId="user-123" />);

        expect(screen.getByText(mockUserProfile.name)).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /edit/i })).toBeInTheDocument();
    });

    it('calls onEdit when edit button is clicked', async () => {
        const user = userEvent.setup();
        const mockOnEdit = jest.fn();

        render(<UserProfileCard userId="user-123" onEdit={mockOnEdit} />);

        await user.click(screen.getByRole('button', { name: /edit/i }));

        expect(mockOnEdit).toHaveBeenCalledWith('user-123');
    });
});
```

## 🎨 PatternFly Usage

### Common Components

```typescript
import {
    Card,
    CardTitle,
    CardBody,
    CardFooter,
    Button,
    Alert,
    Badge,
    Spinner,
    EmptyState,
    EmptyStateIcon,
    EmptyStateBody,
    Form,
    FormGroup,
    TextInput,
    FormSelect,
} from '@patternfly/react-core';
import { UserIcon, EditIcon } from '@patternfly/react-icons';

// Usage
<Card>
    <CardTitle>
        <UserIcon className="h-5 w-5 mr-2" />
        User Profile
    </CardTitle>
    <CardBody>
        <Alert variant="success" title="Success">
            User updated successfully
        </Alert>
        <Badge variant="success">Active</Badge>
    </CardBody>
    <CardFooter>
        <Button variant="primary" size="sm">
            Edit
        </Button>
    </CardFooter>
</Card>;
```

## 🎨 Tailwind Classes

### Common Patterns

```typescript
// Layout
className = 'flex items-center justify-between';
className = 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6';
className = 'space-y-4';

// Styling
className = 'bg-white rounded-lg shadow-md p-6 border border-gray-200';
className = 'text-lg font-semibold text-gray-900';
className = 'text-sm text-gray-500';

// Interactive
className = 'hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500';
className = 'disabled:opacity-50 disabled:cursor-not-allowed';

// Responsive
className = 'hidden md:block';
className = 'w-full sm:w-auto';
```

## 🔄 State Management

### Local State with Hooks

```typescript
const [user, setUser] = useState<UserProfile | null>(null);
const [error, setError] = useState<string | null>(null);
const [isLoading, setIsLoading] = useState(true);

// Custom hook usage
const { data: user, error, isLoading } = useUserProfile(userId);
```

### Redux Integration

```typescript
import { useSelector, useDispatch } from 'react-redux';
import { actions } from 'reducers/users';

const dispatch = useDispatch();
const user = useSelector((state) => state.users.byId[userId]);

useEffect(() => {
    if (!user) {
        dispatch(actions.fetchUser(userId));
    }
}, [dispatch, userId, user]);
```

## 🚨 Error Handling

### Component Error States

```typescript
// Loading
if (isLoading) {
    return <Spinner size="md" />;
}

// Error
if (error) {
    return (
        <Alert variant="danger" title="Error">
            {error}
        </Alert>
    );
}

// Empty state
if (!data) {
    return (
        <EmptyState>
            <EmptyStateIcon icon={UserIcon} />
            <EmptyStateBody>No data found</EmptyStateBody>
        </EmptyState>
    );
}
```

### Service Error Handling

```typescript
try {
    const response = await axios.get<UserProfile>(`/api/users/${userId}`);
    return response.data;
} catch (error) {
    console.error('Failed to fetch user profile:', error);
    throw new Error(`Unable to load user profile: ${error.message}`);
}
```

## 🎯 Event Handlers

### Naming Convention

```typescript
// Internal handlers - use "handle" prefix
const handleEditClick = useCallback(() => {
    onEdit?.(userId);
}, [onEdit, userId]);

const handleFormSubmit = useCallback((event: React.FormEvent) => {
    event.preventDefault();
    // Handle submission
}, []);

// Form field handlers
const handleNameChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    setName(event.target.value);
}, []);
```

## 📋 Common Validation

### Input Validation

```typescript
if (!userId) {
    throw new Error('User ID is required');
}

if (!email?.trim()) {
    errors.email = 'Email is required';
} else if (!/\S+@\S+\.\S+/.test(email)) {
    errors.email = 'Email format is invalid';
}
```

### Type Guards

```typescript
export function isUserProfile(value: unknown): value is UserProfile {
    return (
        typeof value === 'object' &&
        value !== null &&
        typeof (value as UserProfile).id === 'string' &&
        typeof (value as UserProfile).name === 'string'
    );
}
```

## 🔍 Testing Patterns

### Data Attributes

```typescript
// Add to components for testing
data-testid="user-profile-card"
data-testid="edit-button"
data-testid="loading-spinner"
data-testid="error-message"
```

### Mock Data

```typescript
export const mockUserProfile: UserProfile = {
    id: 'user-123',
    name: 'John Doe',
    email: 'john.doe@example.com',
    avatar: 'https://example.com/avatar.jpg',
    isActive: true,
    lastLogin: '2023-12-01T10:00:00Z',
};
```

## 📝 Documentation

### JSDoc Format

```typescript
/**
 * Fetches user profile data from the API
 * @param userId - The unique identifier for the user
 * @returns Promise resolving to user profile data
 * @throws Error when user is not found or API is unavailable
 */
```

## ⚡ Performance

### Memoization

```typescript
const UserProfile = React.memo<UserProfileProps>(({ user, onEdit }) => {
    const formattedDate = useMemo(() =>
        formatDate(user.lastLogin),
        [user.lastLogin]
    );

    const handleEdit = useCallback(() => {
        onEdit?.(user.id);
    }, [onEdit, user.id]);

    return (/* Component JSX */);
});
```

## 🚦 Common Mistakes to Avoid

### ❌ Don't Do This

```typescript
// Random import order
import './styles.css';
import { userService } from 'services/userService';
import React from 'react';

// Missing TypeScript types
const UserProfile = ({ userId, onEdit }) => { // No types

// Direct DOM manipulation
document.getElementById('user-name').innerHTML = user.name;

// console.log in production
console.log('User loaded:', user);

// Missing error handling
const user = await fetchUserProfile(userId); // No try/catch

// Inconsistent naming
const handle_edit_click = () => {}; // Use camelCase

// Missing test IDs
<button onClick={handleEdit}>Edit</button> // Add data-testid

// Inline styles instead of classes
<div style={{ padding: '16px' }}>Content</div> // Use className
```

## 📋 Daily Checklist

Before committing code:

-   [ ] Imports are in the correct order
-   [ ] Components have proper TypeScript interfaces
-   [ ] Error handling is implemented
-   [ ] Test IDs are added for interactive elements
-   [ ] JSDoc comments are added for public APIs
-   [ ] PatternFly components are used where appropriate
-   [ ] Loading and error states are handled
-   [ ] Event handlers use proper naming convention
-   [ ] No console.log statements in production code

## 🔗 Quick Links

-   [Full Documentation](./README.md)
-   [Core Patterns](./core-development-patterns.md)
-   [React Patterns](./react-component-patterns.md)
-   [TypeScript Patterns](./typescript-patterns.md)
-   [Testing Patterns](./testing-patterns.md)
-   [Service Patterns](./service-layer-patterns.md)
-   [Styling Patterns](./styling-ui-patterns.md)

---

_Keep this reference handy for quick pattern lookups during development!_
