---
description: React component patterns and best practices for StackRox UI team
globs: ['**/*.{jsx,tsx}', '**/*Component*.{js,ts}']
alwaysApply: false
---

# React Component Patterns

## Component Definition Standards

### Functional Components (Preferred)

```typescript
// ✅ GOOD - Consistent functional component pattern
import React, { useState, useEffect, useCallback } from 'react';
import { Card, CardBody, Button } from '@patternfly/react-core';

import { UserProfile } from 'types/user.proto';
import { useUserProfile } from 'hooks/useUserProfile';
import { formatDate } from 'utils/dateUtils';

interface UserProfileProps {
    userId: string;
    showAvatar?: boolean;
    onEdit?: (userId: string) => void;
}

const UserProfile: React.FC<UserProfileProps> = ({ userId, showAvatar = true, onEdit }) => {
    const { data: user, error, isLoading } = useUserProfile(userId);
    const [isEditing, setIsEditing] = useState(false);

    const handleEditClick = useCallback(() => {
        setIsEditing(true);
        onEdit?.(userId);
    }, [userId, onEdit]);

    if (isLoading) {
        return <div>Loading user profile...</div>;
    }

    if (error) {
        return <div>Error: {error}</div>;
    }

    return (
        <Card data-testid="user-profile-card">
            <CardBody>
                {showAvatar && (
                    <img src={user.avatar} alt={`${user.name} avatar`} className="user-avatar" />
                )}
                <h2>{user.name}</h2>
                <p>Last login: {formatDate(user.lastLogin)}</p>
                <Button onClick={handleEditClick}>Edit Profile</Button>
            </CardBody>
        </Card>
    );
};

export default UserProfile;
```

### Class Components (Legacy Support)

```typescript
// ✅ GOOD - Consistent class component pattern (when needed)
import React, { Component } from 'react';
import { Card, CardBody, Button } from '@patternfly/react-core';

import { UserProfile } from 'types/user.proto';
import { fetchUserProfile } from 'services/userService';
import { formatDate } from 'utils/dateUtils';

interface UserProfileProps {
    userId: string;
    showAvatar?: boolean;
    onEdit?: (userId: string) => void;
}

interface UserProfileState {
    user: UserProfile | null;
    error: string | null;
    isLoading: boolean;
    isEditing: boolean;
}

class UserProfileComponent extends Component<UserProfileProps, UserProfileState> {
    static defaultProps = {
        showAvatar: true,
    };

    constructor(props: UserProfileProps) {
        super(props);
        this.state = {
            user: null,
            error: null,
            isLoading: true,
            isEditing: false,
        };
    }

    async componentDidMount() {
        await this.loadUserData();
    }

    async componentDidUpdate(prevProps: UserProfileProps) {
        if (prevProps.userId !== this.props.userId) {
            await this.loadUserData();
        }
    }

    loadUserData = async () => {
        try {
            this.setState({ isLoading: true, error: null });
            const user = await fetchUserProfile(this.props.userId);
            this.setState({ user, isLoading: false });
        } catch (error) {
            this.setState({
                error: error instanceof Error ? error.message : 'An error occurred',
                isLoading: false,
            });
        }
    };

    handleEditClick = () => {
        this.setState({ isEditing: true });
        this.props.onEdit?.(this.props.userId);
    };

    render() {
        const { showAvatar } = this.props;
        const { user, error, isLoading } = this.state;

        if (isLoading) {
            return <div>Loading user profile...</div>;
        }

        if (error) {
            return <div>Error: {error}</div>;
        }

        return (
            <Card data-testid="user-profile-card">
                <CardBody>
                    {showAvatar && (
                        <img
                            src={user?.avatar}
                            alt={`${user?.name} avatar`}
                            className="user-avatar"
                        />
                    )}
                    <h2>{user?.name}</h2>
                    <p>Last login: {formatDate(user?.lastLogin)}</p>
                    <Button onClick={this.handleEditClick}>Edit Profile</Button>
                </CardBody>
            </Card>
        );
    }
}

export default UserProfileComponent;
```

## Props & Interface Patterns

### Props Definition

```typescript
// ✅ GOOD - Consistent props interface
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
    /** Callback fired when user profile is loaded */
    onUserLoaded?: (user: UserProfile) => void;
}

// ❌ AVOID - Inconsistent props interface
interface UserProfileProps {
    userId: string;
    showAvatar: boolean; // Should be optional
    className: string; // Should be optional
    editCallback: Function; // Should be specific type
    userData: any; // Should be specific type
}
```

### Default Props (Functional Components)

```typescript
// ✅ GOOD - Default props in functional components
const UserProfile: React.FC<UserProfileProps> = ({
    userId,
    showAvatar = true,
    className = '',
    testId = 'user-profile',
    onEdit,
    onUserLoaded,
}) => {
    // Component implementation
};

// Alternative with destructuring
const UserProfile: React.FC<UserProfileProps> = (props) => {
    const {
        userId,
        showAvatar = true,
        className = '',
        testId = 'user-profile',
        onEdit,
        onUserLoaded,
    } = props;

    // Component implementation
};
```

## State Management Patterns

### Local State with Hooks

```typescript
// ✅ GOOD - Consistent state management
const UserProfile: React.FC<UserProfileProps> = ({ userId }) => {
    const [user, setUser] = useState<UserProfile | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [isEditing, setIsEditing] = useState(false);

    // Custom hook for complex state logic
    const { selectedUsers, toggleUserSelection, clearSelection, isUserSelected } =
        useUserSelection();

    // Derived state
    const hasMultipleUsers = selectedUsers.length > 1;
    const canEdit = user && !isLoading && !error;

    // Implementation
};
```

### Redux Integration

```typescript
// ✅ GOOD - Consistent Redux patterns
import { useSelector, useDispatch } from 'react-redux';
import { actions } from 'reducers/users';

const UserProfile: React.FC<UserProfileProps> = ({ userId }) => {
    const dispatch = useDispatch();
    const user = useSelector((state) => state.users.byId[userId]);
    const isLoading = useSelector((state) => state.users.loading);
    const error = useSelector((state) => state.users.error);

    useEffect(() => {
        if (!user) {
            dispatch(actions.fetchUser(userId));
        }
    }, [dispatch, userId, user]);

    const handleEdit = useCallback(() => {
        dispatch(actions.setUserEditing(userId, true));
    }, [dispatch, userId]);

    // Component implementation
};
```

## Event Handling Patterns

### Event Handler Naming

```typescript
// ✅ GOOD - Consistent event handler naming
const UserProfile: React.FC<UserProfileProps> = ({ userId, onEdit }) => {
    // Internal handlers - use "handle" prefix
    const handleEditClick = useCallback(() => {
        onEdit?.(userId);
    }, [userId, onEdit]);

    const handleSaveClick = useCallback(async () => {
        try {
            await saveUserProfile(userId, updatedData);
        } catch (error) {
            console.error('Failed to save user profile:', error);
        }
    }, [userId, updatedData]);

    const handleDeleteConfirm = useCallback(() => {
        if (window.confirm('Are you sure you want to delete this user?')) {
            deleteUser(userId);
        }
    }, [userId]);

    // Form handlers
    const handleNameChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setName(event.target.value);
    }, []);

    const handleFormSubmit = useCallback(
        (event: React.FormEvent) => {
            event.preventDefault();
            handleSaveClick();
        },
        [handleSaveClick]
    );

    return (
        <form onSubmit={handleFormSubmit}>
            <input onChange={handleNameChange} />
            <Button onClick={handleEditClick}>Edit</Button>
            <Button onClick={handleSaveClick}>Save</Button>
            <Button onClick={handleDeleteConfirm}>Delete</Button>
        </form>
    );
};
```

## Conditional Rendering Patterns

### Loading and Error States

```typescript
// ✅ GOOD - Consistent conditional rendering
const UserProfile: React.FC<UserProfileProps> = ({ userId }) => {
    const { data: user, error, isLoading } = useUserProfile(userId);

    // Early returns for loading/error states
    if (isLoading) {
        return (
            <Card>
                <CardBody>
                    <Loader message="Loading user profile..." />
                </CardBody>
            </Card>
        );
    }

    if (error) {
        return (
            <Card>
                <CardBody>
                    <Alert variant="danger" title="Error">
                        Failed to load user profile: {error}
                    </Alert>
                </CardBody>
            </Card>
        );
    }

    if (!user) {
        return (
            <Card>
                <CardBody>
                    <EmptyState>
                        <p>User not found</p>
                    </EmptyState>
                </CardBody>
            </Card>
        );
    }

    // Main render
    return (
        <Card>
            <CardBody>
                <h2>{user.name}</h2>
                <p>Email: {user.email}</p>
                {user.isActive && <Badge>Active</Badge>}
            </CardBody>
        </Card>
    );
};
```

### Complex Conditional Logic

```typescript
// ✅ GOOD - Extract complex conditions
const UserProfile: React.FC<UserProfileProps> = ({ user, permissions }) => {
    const canEditUser = permissions.includes('EDIT_USER');
    const canDeleteUser = permissions.includes('DELETE_USER');
    const isCurrentUser = user.id === currentUser.id;
    const showAdminActions = canEditUser || canDeleteUser;

    return (
        <Card>
            <CardBody>
                <h2>{user.name}</h2>

                {showAdminActions && (
                    <div className="admin-actions">
                        {canEditUser && <Button onClick={handleEdit}>Edit</Button>}
                        {canDeleteUser && !isCurrentUser && (
                            <Button variant="danger" onClick={handleDelete}>
                                Delete
                            </Button>
                        )}
                    </div>
                )}
            </CardBody>
        </Card>
    );
};
```

## Component Composition Patterns

### Children and Render Props

```typescript
// ✅ GOOD - Consistent composition patterns
interface UserProfileLayoutProps {
    children: React.ReactNode;
    header?: React.ReactNode;
    actions?: React.ReactNode;
    className?: string;
}

const UserProfileLayout: React.FC<UserProfileLayoutProps> = ({
    children,
    header,
    actions,
    className = '',
}) => {
    return (
        <div className={`user-profile-layout ${className}`}>
            {header && <div className="user-profile-header">{header}</div>}
            <div className="user-profile-content">{children}</div>
            {actions && <div className="user-profile-actions">{actions}</div>}
        </div>
    );
};

// Usage
<UserProfileLayout header={<h1>User Profile</h1>} actions={<Button>Edit</Button>}>
    <UserProfile userId={userId} />
</UserProfileLayout>;
```

### Compound Components

```typescript
// ✅ GOOD - Compound component pattern
interface UserProfileCardProps {
    children: React.ReactNode;
    className?: string;
}

const UserProfileCard: React.FC<UserProfileCardProps> = ({ children, className = '' }) => {
    return <Card className={`user-profile-card ${className}`}>{children}</Card>;
};

const UserProfileHeader: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    return <div className="user-profile-header">{children}</div>;
};

const UserProfileBody: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    return <CardBody className="user-profile-body">{children}</CardBody>;
};

const UserProfileActions: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    return <div className="user-profile-actions">{children}</div>;
};

// Attach compound components
UserProfileCard.Header = UserProfileHeader;
UserProfileCard.Body = UserProfileBody;
UserProfileCard.Actions = UserProfileActions;

// Usage
<UserProfileCard>
    <UserProfileCard.Header>
        <h2>John Doe</h2>
    </UserProfileCard.Header>
    <UserProfileCard.Body>
        <p>Email: john.doe@example.com</p>
    </UserProfileCard.Body>
    <UserProfileCard.Actions>
        <Button>Edit</Button>
    </UserProfileCard.Actions>
</UserProfileCard>;
```

## Performance Optimization

### Memoization Patterns

```typescript
// ✅ GOOD - Consistent memoization
const UserProfile = React.memo<UserProfileProps>(({ user, showAvatar, onEdit }) => {
    // Memoize expensive calculations
    const userStats = useMemo(() => {
        return calculateUserStats(user);
    }, [user]);

    // Memoize callback functions
    const handleEditClick = useCallback(() => {
        onEdit?.(user.id);
    }, [onEdit, user.id]);

    // Memoize derived values
    const formattedDate = useMemo(() => formatDate(user.lastLogin), [user.lastLogin]);

    return (
        <Card>
            <CardBody>
                <h2>{user.name}</h2>
                <p>Last login: {formattedDate}</p>
                <p>Posts: {userStats.postCount}</p>
                <Button onClick={handleEditClick}>Edit</Button>
            </CardBody>
        </Card>
    );
});

// Custom comparison function for complex props
const UserProfile = React.memo<UserProfileProps>(
    ({ user, permissions, onEdit }) => {
        // Component implementation
    },
    (prevProps, nextProps) => {
        return (
            prevProps.user.id === nextProps.user.id &&
            prevProps.user.updatedAt === nextProps.user.updatedAt &&
            isEqual(prevProps.permissions, nextProps.permissions)
        );
    }
);
```

## Testing Integration

### Test-Friendly Component Structure

```typescript
// ✅ GOOD - Testing-friendly component
const UserProfile: React.FC<UserProfileProps> = ({ userId, testId = 'user-profile' }) => {
    const { data: user, error, isLoading } = useUserProfile(userId);

    return (
        <Card data-testid={testId}>
            <CardBody>
                {isLoading && <div data-testid="loading-spinner">Loading...</div>}

                {error && <div data-testid="error-message">Error: {error}</div>}

                {user && (
                    <>
                        <h2 data-testid="user-name">{user.name}</h2>
                        <p data-testid="user-email">{user.email}</p>
                        <Button data-testid="edit-button" onClick={handleEdit}>
                            Edit
                        </Button>
                    </>
                )}
            </CardBody>
        </Card>
    );
};
```

## PatternFly Integration

### Consistent PatternFly Usage

```typescript
// ✅ GOOD - Consistent PatternFly component usage
import {
    Card,
    CardTitle,
    CardBody,
    Button,
    Alert,
    Badge,
    Spinner,
    EmptyState,
    EmptyStateIcon,
    EmptyStateBody,
} from '@patternfly/react-core';
import { UserIcon } from '@patternfly/react-icons';

const UserProfile: React.FC<UserProfileProps> = ({ userId }) => {
    const { data: user, error, isLoading } = useUserProfile(userId);

    if (isLoading) {
        return (
            <Card>
                <CardBody>
                    <Spinner size="md" />
                    <span className="ml-2">Loading user profile...</span>
                </CardBody>
            </Card>
        );
    }

    if (error) {
        return (
            <Alert variant="danger" title="Error loading user profile">
                {error}
            </Alert>
        );
    }

    if (!user) {
        return (
            <EmptyState>
                <EmptyStateIcon icon={UserIcon} />
                <EmptyStateBody>User not found</EmptyStateBody>
            </EmptyState>
        );
    }

    return (
        <Card>
            <CardTitle>
                {user.name}
                {user.isActive && <Badge className="ml-2">Active</Badge>}
            </CardTitle>
            <CardBody>
                <p>Email: {user.email}</p>
                <p>Role: {user.role}</p>
                <Button variant="primary" size="sm">
                    Edit Profile
                </Button>
            </CardBody>
        </Card>
    );
};
```

## Component Export Patterns

### Consistent Export Structure

```typescript
// ✅ GOOD - Consistent component exports
// UserProfile.tsx
import React from 'react';
// ... other imports

interface UserProfileProps {
    userId: string;
    showAvatar?: boolean;
}

const UserProfile: React.FC<UserProfileProps> = ({ userId, showAvatar = true }) => {
    // Component implementation
};

export default UserProfile;
export type { UserProfileProps };

// index.ts
export { default } from './UserProfile';
export type { UserProfileProps } from './UserProfile';

// Alternative for multiple exports
export { default as UserProfile } from './UserProfile';
export { default as UserProfileCard } from './UserProfileCard';
export type { UserProfileProps } from './UserProfile';
```

## Migration Guidelines

### Converting Class to Functional Components

```typescript
// ✅ GOOD - Migration strategy
// Step 1: Convert class to functional component
// Step 2: Replace lifecycle methods with hooks
// Step 3: Convert state to useState hooks
// Step 4: Convert methods to useCallback hooks
// Step 5: Add proper TypeScript types
// Step 6: Add proper test IDs
// Step 7: Update tests

// Before (Class Component)
class UserProfile extends Component {
    state = { user: null, loading: true };

    componentDidMount() {
        this.loadUser();
    }

    loadUser = async () => {
        // Implementation
    };
}

// After (Functional Component)
const UserProfile: React.FC<UserProfileProps> = ({ userId }) => {
    const [user, setUser] = useState<UserProfile | null>(null);
    const [loading, setLoading] = useState(true);

    const loadUser = useCallback(async () => {
        // Implementation
    }, [userId]);

    useEffect(() => {
        loadUser();
    }, [loadUser]);

    // Rest of component
};
```
