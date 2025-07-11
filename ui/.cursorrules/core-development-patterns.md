---
description: Core development patterns and consistency rules for StackRox UI team
globs: ['**/*.{js,jsx,ts,tsx}']
alwaysApply: true
---

# Core Development Patterns

## File Organization & Naming

### File Structure

-   **Components**: PascalCase directories with index files
-   **Services**: camelCase files with descriptive names
-   **Utils**: camelCase files with specific purposes
-   **Types**: PascalCase files with `.proto.ts` suffix for API types
-   **Hooks**: camelCase files with `use` prefix

```
✅ GOOD
src/
├── Components/
│   ├── UserProfile/
│   │   ├── UserProfile.tsx
│   │   ├── UserProfile.test.tsx
│   │   └── index.ts
│   └── ComplianceReport/
│       ├── ComplianceReport.tsx
│       └── ComplianceReport.test.tsx
├── services/
│   ├── userService.ts
│   └── complianceService.ts
├── hooks/
│   ├── useUserProfile.ts
│   └── useComplianceData.ts
└── types/
    ├── user.proto.ts
    └── compliance.proto.ts

❌ AVOID
src/
├── components/userprofile.jsx
├── Services/USER_SERVICE.js
├── hooks/UseUserProfile.js
└── types/user-types.ts
```

### File Extensions

-   **TypeScript**: Use `.ts` for utilities, services, types
-   **React TypeScript**: Use `.tsx` for React components
-   **Legacy**: `.js/.jsx` only for existing files being migrated

```typescript
// ✅ GOOD - New files
UserProfile.tsx;
userService.ts;
useUserProfile.ts;

// ❌ AVOID - New files
UserProfile.jsx;
userService.js;
useUserProfile.js;
```

## Import Organization

### Import Order (Consistent across ALL files)

```typescript
// ✅ GOOD - Consistent import order
import React from 'react';
import { useState, useEffect } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { Button, Card, CardBody } from '@patternfly/react-core';

import { UserProfile } from 'types/user.proto';
import { useUserProfile } from 'hooks/useUserProfile';
import { userService } from 'services/userService';
import { formatDate } from 'utils/dateUtils';
import { validateEmail } from 'utils/validationUtils';

import ComponentSpecificComponent from './ComponentSpecificComponent';
import './UserProfile.css';

// ❌ AVOID - Random import order
import './UserProfile.css';
import { userService } from 'services/userService';
import React from 'react';
import { Button } from '@patternfly/react-core';
import { UserProfile } from 'types/user.proto';
```

### Import Grouping Rules

1. **React & React ecosystem** (react, react-dom, react-router, etc.)
2. **Third-party libraries** (lodash, axios, @patternfly, etc.)
3. **Type imports** (separated or with `type` keyword)
4. **Internal hooks** (custom hooks)
5. **Internal services** (API services)
6. **Internal utilities** (utils functions)
7. **Relative imports** (components, local files)
8. **Styles** (CSS/SCSS files)

## Error Handling Patterns

### Service Layer Error Handling

```typescript
// ✅ GOOD - Consistent error handling
export async function fetchUserProfile(userId: string): Promise<UserProfile> {
    try {
        const response = await axios.get<UserProfile>(`/api/users/${userId}`);
        return response.data;
    } catch (error) {
        console.error('Failed to fetch user profile:', error);
        throw new Error(`Unable to load user profile: ${error.message}`);
    }
}

// ❌ AVOID - Inconsistent error handling
export async function fetchUserProfile(userId: string) {
    const response = await axios.get(`/api/users/${userId}`);
    return response.data; // No error handling
}
```

### Component Error Handling

```typescript
// ✅ GOOD - Consistent error display
const UserProfile: React.FC<UserProfileProps> = ({ userId }) => {
    const [user, setUser] = useState<UserProfile | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        async function loadUser() {
            try {
                setError(null);
                setIsLoading(true);
                const userData = await fetchUserProfile(userId);
                setUser(userData);
            } catch (err) {
                setError(err instanceof Error ? err.message : 'An error occurred');
            } finally {
                setIsLoading(false);
            }
        }
        loadUser();
    }, [userId]);

    if (isLoading) {
        return <Loader message="Loading user profile..." />;
    }

    if (error) {
        return (
            <Alert variant="danger" title="Error">
                {error}
            </Alert>
        );
    }

    return <div>{/* Component content */}</div>;
};
```

## Logging & Debugging

### Console Usage

```typescript
// ✅ GOOD - Use structured logging
console.error('User authentication failed:', { userId, error: error.message });
console.warn('Deprecated API usage detected:', { endpoint: '/api/v1/users' });

// ❌ AVOID - console.log in production code
console.log('User loaded'); // Remove before committing
```

### Debug Information

```typescript
// ✅ GOOD - Environment-specific debugging
if (process.env.NODE_ENV === 'development') {
    console.debug('UserProfile component rendered:', { userId, user });
}
```

## Code Organization

### Function Organization

```typescript
// ✅ GOOD - Consistent function organization
export class UserProfileComponent extends React.Component {
    // 1. Static properties
    static defaultProps = {
        showAvatar: true,
        theme: 'light',
    };

    // 2. Constructor
    constructor(props) {
        super(props);
        this.state = { isEditing: false };
    }

    // 3. Lifecycle methods
    componentDidMount() {
        this.loadUserData();
    }

    // 4. Event handlers (grouped together)
    handleEdit = () => {
        this.setState({ isEditing: true });
    };

    handleSave = async () => {
        await this.saveUserData();
        this.setState({ isEditing: false });
    };

    // 5. Helper methods
    loadUserData = async () => {
        // Implementation
    };

    saveUserData = async () => {
        // Implementation
    };

    // 6. Render method
    render() {
        return <div>{/* Component JSX */}</div>;
    }
}
```

### Variable Declaration

```typescript
// ✅ GOOD - Consistent variable patterns
const userProfile = useUserProfile(userId);
const { data: complianceData, error, isLoading } = useComplianceData();
const [selectedUsers, setSelectedUsers] = useState<string[]>([]);

// ❌ AVOID - Inconsistent patterns
var userProfile = useUserProfile(userId);
let { data: complianceData, error, isLoading } = useComplianceData();
const [selectedUsers, setSelectedUsers] = useState([]);
```

## Documentation Standards

### Function Documentation

```typescript
// ✅ GOOD - Consistent JSDoc format
/**
 * Fetches user profile data from the API
 * @param userId - The unique identifier for the user
 * @returns Promise resolving to user profile data
 * @throws Error when user is not found or API is unavailable
 */
export async function fetchUserProfile(userId: string): Promise<UserProfile> {
    // Implementation
}

/**
 * UserProfile component for displaying user information
 * @param userId - The ID of the user to display
 * @param showAvatar - Whether to show the user's avatar
 * @param onEdit - Callback fired when user clicks edit button
 */
interface UserProfileProps {
    userId: string;
    showAvatar?: boolean;
    onEdit?: () => void;
}
```

### Component Documentation

```typescript
// ✅ GOOD - Component usage examples
/**
 * UserProfile displays user information in a card format
 *
 * @example
 * <UserProfile
 *   userId="user-123"
 *   showAvatar={true}
 *   onEdit={() => console.log('Edit clicked')}
 * />
 */
const UserProfile: React.FC<UserProfileProps> = ({ userId, showAvatar = true, onEdit }) => {
    // Implementation
};
```

## Performance Patterns

### Memoization

```typescript
// ✅ GOOD - Consistent memoization patterns
const UserProfileCard = React.memo<UserProfileProps>(({ user, onEdit }) => {
    const formattedDate = useMemo(() => formatDate(user.lastLogin), [user.lastLogin]);

    const handleEditClick = useCallback(() => {
        onEdit?.(user.id);
    }, [onEdit, user.id]);

    return (
        <Card>
            <CardBody>
                <p>Last login: {formattedDate}</p>
                <Button onClick={handleEditClick}>Edit</Button>
            </CardBody>
        </Card>
    );
});
```

### Lazy Loading

```typescript
// ✅ GOOD - Consistent lazy loading
const UserProfile = React.lazy(() => import('./UserProfile'));
const ComplianceReport = React.lazy(() => import('./ComplianceReport'));

// Usage with consistent loading fallback
<Suspense fallback={<Loader message="Loading user profile..." />}>
    <UserProfile userId={userId} />
</Suspense>;
```

## Configuration & Constants

### Constants Organization

```typescript
// ✅ GOOD - Consistent constants structure
export const USER_ROLES = {
    ADMIN: 'admin',
    ANALYST: 'analyst',
    VIEWER: 'viewer',
} as const;

export const API_ENDPOINTS = {
    USERS: '/api/v1/users',
    COMPLIANCE: '/api/v1/compliance',
    CLUSTERS: '/api/v1/clusters',
} as const;

export const UI_CONSTANTS = {
    PAGE_SIZE: 25,
    DEBOUNCE_DELAY: 300,
    TOAST_DURATION: 5000,
} as const;
```

### Environment Configuration

```typescript
// ✅ GOOD - Consistent environment handling
export const config = {
    apiBaseUrl: process.env.REACT_APP_API_URL || 'http://localhost:3001',
    isProduction: process.env.NODE_ENV === 'production',
    isDevelopment: process.env.NODE_ENV === 'development',
    logLevel: process.env.REACT_APP_LOG_LEVEL || 'warn',
} as const;
```

## Team Workflow Integration

### Code Review Checklist

Before submitting PRs, ensure:

-   [ ] All imports follow the established order
-   [ ] Error handling is consistent and informative
-   [ ] TypeScript types are properly defined
-   [ ] Function and component documentation is complete
-   [ ] Performance optimizations are applied where needed
-   [ ] Constants are properly organized and typed

### Development Workflow

1. **File Creation**: Use established naming conventions
2. **Import Organization**: Follow the consistent import order
3. **Error Handling**: Implement consistent error patterns
4. **Documentation**: Add JSDoc comments for public APIs
5. **Testing**: Include appropriate test coverage
6. **Type Safety**: Ensure proper TypeScript usage
