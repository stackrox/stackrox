---
description: Styling and UI patterns for StackRox UI team using PatternFly and Tailwind CSS
globs: ['**/*.{css,scss}', '**/*.{js,jsx,ts,tsx}']
alwaysApply: false
---

# Styling and UI Patterns

## PatternFly Integration

### Component Usage

```typescript
// ✅ GOOD - Consistent PatternFly component usage
import React from 'react';
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
    EmptyStateActions,
} from '@patternfly/react-core';
import { UserIcon, EditIcon, DeleteIcon } from '@patternfly/react-icons';

const UserProfile: React.FC<UserProfileProps> = ({ user, onEdit, onDelete }) => {
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

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
            <Alert variant="danger" title="Error loading user profile" className="mb-4">
                {error}
            </Alert>
        );
    }

    if (!user) {
        return (
            <EmptyState>
                <EmptyStateIcon icon={UserIcon} />
                <EmptyStateBody>User not found</EmptyStateBody>
                <EmptyStateActions>
                    <Button variant="primary" onClick={() => window.history.back()}>
                        Go Back
                    </Button>
                </EmptyStateActions>
            </EmptyState>
        );
    }

    return (
        <Card>
            <CardTitle>
                <div className="flex items-center gap-2">
                    <UserIcon className="h-5 w-5" />
                    {user.name}
                    {user.isActive && (
                        <Badge className="ml-2" variant="success">
                            Active
                        </Badge>
                    )}
                </div>
            </CardTitle>
            <CardBody>
                <div className="space-y-2">
                    <p>
                        <strong>Email:</strong> {user.email}
                    </p>
                    <p>
                        <strong>Role:</strong> {user.role}
                    </p>
                    <p>
                        <strong>Last Login:</strong> {formatDate(user.lastLogin)}
                    </p>
                </div>
            </CardBody>
            <CardFooter>
                <div className="flex gap-2">
                    <Button
                        variant="primary"
                        size="sm"
                        icon={<EditIcon />}
                        onClick={() => onEdit(user.id)}
                    >
                        Edit
                    </Button>
                    <Button
                        variant="danger"
                        size="sm"
                        icon={<DeleteIcon />}
                        onClick={() => onDelete(user.id)}
                    >
                        Delete
                    </Button>
                </div>
            </CardFooter>
        </Card>
    );
};

// ❌ AVOID - Inconsistent PatternFly usage
const UserProfile = ({ user }) => {
    return (
        <div className="card">
            {' '}
            {/* Use PatternFly Card component */}
            <div className="card-header">
                <h3>{user.name}</h3>
                <span className="badge">Active</span> {/* Use PatternFly Badge */}
            </div>
            <div className="card-body">
                <p>Email: {user.email}</p>
                <button className="btn btn-primary">Edit</button> {/* Use PatternFly Button */}
            </div>
        </div>
    );
};
```

### Form Patterns

```typescript
// ✅ GOOD - Consistent form patterns with PatternFly
import {
    Form,
    FormGroup,
    FormSection,
    TextInput,
    TextArea,
    FormSelect,
    FormSelectOption,
    Checkbox,
    Radio,
    Button,
    ActionGroup,
    ValidatedOptions,
} from '@patternfly/react-core';

const UserForm: React.FC<UserFormProps> = ({
    initialValues,
    onSubmit,
    onCancel,
    isEditing = false,
}) => {
    const [formData, setFormData] = useState(initialValues);
    const [errors, setErrors] = useState<Record<string, string>>({});
    const [isSubmitting, setIsSubmitting] = useState(false);

    const validateForm = (): boolean => {
        const newErrors: Record<string, string> = {};

        if (!formData.name?.trim()) {
            newErrors.name = 'Name is required';
        }

        if (!formData.email?.trim()) {
            newErrors.email = 'Email is required';
        } else if (!/\S+@\S+\.\S+/.test(formData.email)) {
            newErrors.email = 'Email format is invalid';
        }

        if (!formData.role) {
            newErrors.role = 'Role is required';
        }

        setErrors(newErrors);
        return Object.keys(newErrors).length === 0;
    };

    const handleSubmit = async (event: React.FormEvent) => {
        event.preventDefault();

        if (!validateForm()) {
            return;
        }

        setIsSubmitting(true);
        try {
            await onSubmit(formData);
        } catch (error) {
            console.error('Form submission error:', error);
        } finally {
            setIsSubmitting(false);
        }
    };

    const handleFieldChange = (field: string, value: string) => {
        setFormData((prev) => ({ ...prev, [field]: value }));

        // Clear error when user starts typing
        if (errors[field]) {
            setErrors((prev) => ({ ...prev, [field]: '' }));
        }
    };

    return (
        <Form onSubmit={handleSubmit}>
            <FormSection title="User Information">
                <FormGroup
                    label="Name"
                    isRequired
                    fieldId="name"
                    validated={errors.name ? ValidatedOptions.error : ValidatedOptions.default}
                    helperTextInvalid={errors.name}
                >
                    <TextInput
                        id="name"
                        name="name"
                        value={formData.name || ''}
                        onChange={(_, value) => handleFieldChange('name', value)}
                        validated={errors.name ? ValidatedOptions.error : ValidatedOptions.default}
                        data-testid="name-input"
                    />
                </FormGroup>

                <FormGroup
                    label="Email"
                    isRequired
                    fieldId="email"
                    validated={errors.email ? ValidatedOptions.error : ValidatedOptions.default}
                    helperTextInvalid={errors.email}
                >
                    <TextInput
                        id="email"
                        name="email"
                        type="email"
                        value={formData.email || ''}
                        onChange={(_, value) => handleFieldChange('email', value)}
                        validated={errors.email ? ValidatedOptions.error : ValidatedOptions.default}
                        data-testid="email-input"
                    />
                </FormGroup>

                <FormGroup
                    label="Role"
                    isRequired
                    fieldId="role"
                    validated={errors.role ? ValidatedOptions.error : ValidatedOptions.default}
                    helperTextInvalid={errors.role}
                >
                    <FormSelect
                        id="role"
                        name="role"
                        value={formData.role || ''}
                        onChange={(_, value) => handleFieldChange('role', value)}
                        validated={errors.role ? ValidatedOptions.error : ValidatedOptions.default}
                        data-testid="role-select"
                    >
                        <FormSelectOption value="" label="Select a role" />
                        <FormSelectOption value="admin" label="Administrator" />
                        <FormSelectOption value="analyst" label="Security Analyst" />
                        <FormSelectOption value="viewer" label="Viewer" />
                    </FormSelect>
                </FormGroup>

                <FormGroup fieldId="permissions" label="Permissions">
                    <Checkbox
                        id="read-users"
                        name="permissions"
                        label="Read Users"
                        isChecked={formData.permissions?.includes('read_users')}
                        onChange={(_, checked) => {
                            const permissions = formData.permissions || [];
                            const newPermissions = checked
                                ? [...permissions, 'read_users']
                                : permissions.filter((p) => p !== 'read_users');
                            handleFieldChange('permissions', newPermissions);
                        }}
                    />
                    <Checkbox
                        id="write-users"
                        name="permissions"
                        label="Write Users"
                        isChecked={formData.permissions?.includes('write_users')}
                        onChange={(_, checked) => {
                            const permissions = formData.permissions || [];
                            const newPermissions = checked
                                ? [...permissions, 'write_users']
                                : permissions.filter((p) => p !== 'write_users');
                            handleFieldChange('permissions', newPermissions);
                        }}
                    />
                </FormGroup>
            </FormSection>

            <ActionGroup>
                <Button
                    variant="primary"
                    type="submit"
                    isDisabled={isSubmitting}
                    isLoading={isSubmitting}
                    data-testid="submit-button"
                >
                    {isEditing ? 'Update User' : 'Create User'}
                </Button>
                <Button variant="link" onClick={onCancel} data-testid="cancel-button">
                    Cancel
                </Button>
            </ActionGroup>
        </Form>
    );
};
```

## Tailwind CSS Integration

### Utility Classes

```typescript
// ✅ GOOD - Consistent Tailwind utility usage
const UserProfileCard: React.FC<UserProfileProps> = ({ user, showAvatar = true }) => {
    return (
        <div className="bg-white rounded-lg shadow-md p-6 border border-gray-200">
            {/* Header Section */}
            <div className="flex items-center space-x-4 mb-4">
                {showAvatar && (
                    <div className="flex-shrink-0">
                        <img
                            src={user.avatar}
                            alt={`${user.name} avatar`}
                            className="h-12 w-12 rounded-full object-cover border-2 border-gray-300"
                        />
                    </div>
                )}
                <div className="flex-1 min-w-0">
                    <h3 className="text-lg font-semibold text-gray-900 truncate">{user.name}</h3>
                    <p className="text-sm text-gray-500 truncate">{user.email}</p>
                </div>
                <div className="flex-shrink-0">
                    <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                            user.isActive
                                ? 'bg-green-100 text-green-800'
                                : 'bg-gray-100 text-gray-800'
                        }`}
                    >
                        {user.isActive ? 'Active' : 'Inactive'}
                    </span>
                </div>
            </div>

            {/* Content Section */}
            <div className="space-y-3">
                <div className="flex justify-between items-center">
                    <span className="text-sm font-medium text-gray-700">Role:</span>
                    <span className="text-sm text-gray-900 capitalize">{user.role}</span>
                </div>
                <div className="flex justify-between items-center">
                    <span className="text-sm font-medium text-gray-700">Last Login:</span>
                    <span className="text-sm text-gray-900">{formatDate(user.lastLogin)}</span>
                </div>
            </div>

            {/* Action Section */}
            <div className="mt-6 flex space-x-3">
                <button
                    type="button"
                    className="flex-1 inline-flex justify-center items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                    data-testid="edit-button"
                >
                    <EditIcon className="h-4 w-4 mr-2" />
                    Edit
                </button>
                <button
                    type="button"
                    className="flex-1 inline-flex justify-center items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                    data-testid="view-button"
                >
                    <UserIcon className="h-4 w-4 mr-2" />
                    View Details
                </button>
            </div>
        </div>
    );
};

// ❌ AVOID - Inconsistent or unclear utility usage
const UserProfileCard = ({ user }) => {
    return (
        <div className="bg-white shadow p-4">
            {' '}
            {/* Missing consistent spacing, border radius */}
            <div className="flex">
                <img src={user.avatar} className="w-10 h-10" />{' '}
                {/* Missing alt text, rounded corners */}
                <div className="ml-2">
                    <h3 className="text-lg">{user.name}</h3> {/* Missing font weight, color */}
                    <p className="text-gray-500">{user.email}</p>
                </div>
            </div>
            <button className="bg-blue-500 text-white px-4 py-2 mt-4">
                {' '}
                {/* Missing hover, focus, rounded */}
                Edit
            </button>
        </div>
    );
};
```

### Custom CSS Classes

```css
/* ✅ GOOD - Consistent custom CSS patterns */
/* styles/components/UserProfile.css */
.user-profile-card {
    @apply bg-white rounded-lg shadow-md border border-gray-200;
    transition: shadow 0.2s ease-in-out;
}

.user-profile-card:hover {
    @apply shadow-lg;
}

.user-profile-header {
    @apply flex items-center justify-between p-4 border-b border-gray-200;
}

.user-profile-avatar {
    @apply h-12 w-12 rounded-full object-cover border-2 border-gray-300;
}

.user-profile-info {
    @apply flex-1 min-w-0 ml-4;
}

.user-profile-name {
    @apply text-lg font-semibold text-gray-900 truncate;
}

.user-profile-email {
    @apply text-sm text-gray-500 truncate;
}

.user-profile-status {
    @apply inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium;
}

.user-profile-status--active {
    @apply bg-green-100 text-green-800;
}

.user-profile-status--inactive {
    @apply bg-gray-100 text-gray-800;
}

.user-profile-actions {
    @apply flex space-x-3 p-4;
}

.user-profile-button {
    @apply inline-flex justify-center items-center px-4 py-2 border text-sm font-medium rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50;
}

.user-profile-button--primary {
    @apply border-transparent text-white bg-blue-600 hover:bg-blue-700 focus:ring-blue-500;
}

.user-profile-button--secondary {
    @apply border-gray-300 text-gray-700 bg-white hover:bg-gray-50 focus:ring-blue-500;
}
```

## Layout Patterns

### Grid and Flexbox Layouts

```typescript
// ✅ GOOD - Consistent layout patterns
const UserDashboard: React.FC = () => {
    return (
        <div className="min-h-screen bg-gray-50">
            {/* Header */}
            <header className="bg-white shadow-sm border-b border-gray-200">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <div className="flex justify-between items-center h-16">
                        <div className="flex items-center">
                            <h1 className="text-2xl font-bold text-gray-900">User Dashboard</h1>
                        </div>
                        <div className="flex items-center space-x-4">
                            <Button variant="primary" size="sm">
                                Add User
                            </Button>
                            <Button variant="secondary" size="sm">
                                Export
                            </Button>
                        </div>
                    </div>
                </div>
            </header>

            {/* Main Content */}
            <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
                {/* Stats Grid */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                    <StatCard title="Total Users" value="1,234" change="+12%" trend="up" />
                    <StatCard title="Active Users" value="1,180" change="+8%" trend="up" />
                    <StatCard title="New This Month" value="54" change="+23%" trend="up" />
                    <StatCard title="Inactive Users" value="54" change="-4%" trend="down" />
                </div>

                {/* Content Grid */}
                <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                    {/* User List */}
                    <div className="lg:col-span-2">
                        <Card>
                            <CardTitle>Recent Users</CardTitle>
                            <CardBody>
                                <UserList users={users} />
                            </CardBody>
                        </Card>
                    </div>

                    {/* Sidebar */}
                    <div className="space-y-6">
                        <Card>
                            <CardTitle>User Statistics</CardTitle>
                            <CardBody>
                                <UserStatsChart />
                            </CardBody>
                        </Card>

                        <Card>
                            <CardTitle>Recent Activity</CardTitle>
                            <CardBody>
                                <ActivityFeed />
                            </CardBody>
                        </Card>
                    </div>
                </div>
            </main>
        </div>
    );
};

const StatCard: React.FC<StatCardProps> = ({ title, value, change, trend }) => {
    const trendColor = trend === 'up' ? 'text-green-600' : 'text-red-600';
    const trendIcon = trend === 'up' ? 'arrow-up' : 'arrow-down';

    return (
        <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between">
                <div>
                    <p className="text-sm font-medium text-gray-600">{title}</p>
                    <p className="text-3xl font-bold text-gray-900">{value}</p>
                </div>
                <div className={`flex items-center ${trendColor}`}>
                    <span className="text-sm font-medium">{change}</span>
                    <TrendIcon className="h-4 w-4 ml-1" />
                </div>
            </div>
        </div>
    );
};
```

### Responsive Design

```typescript
// ✅ GOOD - Responsive design patterns
const UserProfileGrid: React.FC<UserProfileGridProps> = ({ users }) => {
    return (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
            {users.map((user) => (
                <UserProfileCard key={user.id} user={user} className="w-full" />
            ))}
        </div>
    );
};

const ResponsiveTable: React.FC<ResponsiveTableProps> = ({ users }) => {
    return (
        <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                    <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                            User
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider hidden md:table-cell">
                            Role
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider hidden lg:table-cell">
                            Last Login
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                            Status
                        </th>
                        <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                            Actions
                        </th>
                    </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                    {users.map((user) => (
                        <tr key={user.id} className="hover:bg-gray-50">
                            <td className="px-6 py-4 whitespace-nowrap">
                                <div className="flex items-center">
                                    <img
                                        src={user.avatar}
                                        alt={`${user.name} avatar`}
                                        className="h-10 w-10 rounded-full"
                                    />
                                    <div className="ml-4">
                                        <div className="text-sm font-medium text-gray-900">
                                            {user.name}
                                        </div>
                                        <div className="text-sm text-gray-500 md:hidden">
                                            {user.role}
                                        </div>
                                    </div>
                                </div>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 hidden md:table-cell">
                                {user.role}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 hidden lg:table-cell">
                                {formatDate(user.lastLogin)}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap">
                                <span
                                    className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                                        user.isActive
                                            ? 'bg-green-100 text-green-800'
                                            : 'bg-gray-100 text-gray-800'
                                    }`}
                                >
                                    {user.isActive ? 'Active' : 'Inactive'}
                                </span>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                <Button variant="link" size="sm">
                                    Edit
                                </Button>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
};
```

## Component Styling

### Loading States

```typescript
// ✅ GOOD - Consistent loading state patterns
const LoadingCard: React.FC = () => (
    <div className="bg-white rounded-lg shadow p-6 animate-pulse">
        <div className="flex items-center space-x-4">
            <div className="h-12 w-12 bg-gray-300 rounded-full"></div>
            <div className="space-y-2 flex-1">
                <div className="h-4 bg-gray-300 rounded w-3/4"></div>
                <div className="h-3 bg-gray-300 rounded w-1/2"></div>
            </div>
        </div>
        <div className="mt-4 space-y-2">
            <div className="h-3 bg-gray-300 rounded"></div>
            <div className="h-3 bg-gray-300 rounded w-5/6"></div>
        </div>
    </div>
);

const LoadingSpinner: React.FC<{ size?: 'sm' | 'md' | 'lg' }> = ({ size = 'md' }) => {
    const sizeClasses = {
        sm: 'h-4 w-4',
        md: 'h-8 w-8',
        lg: 'h-12 w-12',
    };

    return (
        <div className="flex justify-center items-center">
            <div
                className={`${sizeClasses[size]} animate-spin rounded-full border-2 border-gray-300 border-t-blue-600`}
            ></div>
        </div>
    );
};
```

### Error States

```typescript
// ✅ GOOD - Consistent error state patterns
const ErrorCard: React.FC<{ error: string; onRetry?: () => void }> = ({ error, onRetry }) => (
    <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <div className="flex items-center">
            <ExclamationTriangleIcon className="h-5 w-5 text-red-600 mr-2" />
            <p className="text-sm text-red-800">{error}</p>
        </div>
        {onRetry && (
            <div className="mt-4">
                <Button variant="secondary" size="sm" onClick={onRetry}>
                    Try Again
                </Button>
            </div>
        )}
    </div>
);

const ErrorBoundaryFallback: React.FC<{ error: Error; resetError: () => void }> = ({
    error,
    resetError,
}) => (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-6">
            <div className="flex items-center mb-4">
                <ExclamationTriangleIcon className="h-8 w-8 text-red-600 mr-3" />
                <h2 className="text-lg font-semibold text-gray-900">Something went wrong</h2>
            </div>
            <p className="text-sm text-gray-600 mb-6">
                An unexpected error occurred. Please try refreshing the page.
            </p>
            <div className="flex space-x-3">
                <Button variant="primary" onClick={resetError}>
                    Try Again
                </Button>
                <Button variant="secondary" onClick={() => window.location.reload()}>
                    Refresh Page
                </Button>
            </div>
        </div>
    </div>
);
```

## Theme and Branding

### Color Scheme

```typescript
// ✅ GOOD - Consistent color usage
const theme = {
    colors: {
        primary: {
            50: '#eff6ff',
            100: '#dbeafe',
            500: '#3b82f6',
            600: '#2563eb',
            700: '#1d4ed8',
            900: '#1e3a8a',
        },
        success: {
            50: '#f0fdf4',
            100: '#dcfce7',
            500: '#22c55e',
            600: '#16a34a',
            700: '#15803d',
        },
        warning: {
            50: '#fffbeb',
            100: '#fef3c7',
            500: '#f59e0b',
            600: '#d97706',
            700: '#b45309',
        },
        danger: {
            50: '#fef2f2',
            100: '#fee2e2',
            500: '#ef4444',
            600: '#dc2626',
            700: '#b91c1c',
        },
    },
};

// Usage in components
const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
    const getStatusStyles = (status: string) => {
        switch (status) {
            case 'active':
                return 'bg-green-100 text-green-800';
            case 'inactive':
                return 'bg-gray-100 text-gray-800';
            case 'pending':
                return 'bg-yellow-100 text-yellow-800';
            case 'error':
                return 'bg-red-100 text-red-800';
            default:
                return 'bg-gray-100 text-gray-800';
        }
    };

    return (
        <span
            className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${getStatusStyles(
                status
            )}`}
        >
            {status}
        </span>
    );
};
```

### Typography

```typescript
// ✅ GOOD - Consistent typography patterns
const TypographyExample: React.FC = () => (
    <div className="space-y-4">
        {/* Headings */}
        <h1 className="text-3xl font-bold text-gray-900">Main Heading</h1>
        <h2 className="text-2xl font-semibold text-gray-900">Section Heading</h2>
        <h3 className="text-xl font-medium text-gray-900">Subsection Heading</h3>
        <h4 className="text-lg font-medium text-gray-900">Card Title</h4>

        {/* Body Text */}
        <p className="text-base text-gray-700">
            Regular body text with proper line height and spacing.
        </p>
        <p className="text-sm text-gray-600">
            Smaller secondary text for captions and descriptions.
        </p>

        {/* Links */}
        <a href="#" className="text-blue-600 hover:text-blue-800 underline">
            Standard link
        </a>
        <button className="text-blue-600 hover:text-blue-800 underline">
            Button styled as link
        </button>

        {/* Lists */}
        <ul className="space-y-2">
            <li className="flex items-center">
                <CheckIcon className="h-4 w-4 text-green-500 mr-2" />
                <span className="text-sm text-gray-700">List item with icon</span>
            </li>
            <li className="flex items-center">
                <CheckIcon className="h-4 w-4 text-green-500 mr-2" />
                <span className="text-sm text-gray-700">Another list item</span>
            </li>
        </ul>
    </div>
);
```

## Accessibility Patterns

### ARIA and Keyboard Navigation

```typescript
// ✅ GOOD - Accessible component patterns
const AccessibleButton: React.FC<AccessibleButtonProps> = ({
    children,
    onClick,
    disabled = false,
    loading = false,
    ariaLabel,
    ...props
}) => {
    return (
        <button
            type="button"
            onClick={onClick}
            disabled={disabled || loading}
            aria-label={ariaLabel}
            aria-busy={loading}
            className={`
                inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md
                focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500
                disabled:opacity-50 disabled:cursor-not-allowed
                ${disabled || loading ? 'opacity-50 cursor-not-allowed' : 'hover:bg-blue-700'}
            `}
            {...props}
        >
            {loading && <LoadingSpinner size="sm" className="mr-2" />}
            {children}
        </button>
    );
};

const AccessibleModal: React.FC<AccessibleModalProps> = ({ isOpen, onClose, title, children }) => {
    const modalRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (isOpen) {
            modalRef.current?.focus();
        }
    }, [isOpen]);

    const handleKeyDown = (event: KeyboardEvent) => {
        if (event.key === 'Escape') {
            onClose();
        }
    };

    if (!isOpen) return null;

    return (
        <div
            className="fixed inset-0 z-50 overflow-y-auto"
            aria-labelledby="modal-title"
            aria-describedby="modal-description"
            role="dialog"
            aria-modal="true"
        >
            <div className="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
                <div
                    className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"
                    onClick={onClose}
                    aria-hidden="true"
                />

                <div
                    ref={modalRef}
                    className="inline-block align-bottom bg-white rounded-lg px-4 pt-5 pb-4 text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full sm:p-6"
                    tabIndex={-1}
                    onKeyDown={handleKeyDown}
                >
                    <div>
                        <h3
                            id="modal-title"
                            className="text-lg leading-6 font-medium text-gray-900"
                        >
                            {title}
                        </h3>
                        <div id="modal-description" className="mt-2">
                            {children}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};
```

## Performance Optimization

### CSS Optimization

```css
/* ✅ GOOD - Performance-optimized CSS */
/* Use CSS custom properties for dynamic values */
:root {
    --primary-color: #3b82f6;
    --primary-hover: #2563eb;
    --border-radius: 0.375rem;
    --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
    --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}

/* Optimize animations */
.fade-in {
    animation: fadeIn 0.2s ease-in-out;
}

@keyframes fadeIn {
    from {
        opacity: 0;
        transform: translateY(-10px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

/* Use transform instead of changing layout properties */
.card-hover {
    transition: transform 0.2s ease-in-out;
}

.card-hover:hover {
    transform: translateY(-2px);
}

/* Optimize for will-change */
.animated-element {
    will-change: transform;
}
```

### Conditional Styling

```typescript
// ✅ GOOD - Efficient conditional styling
const ConditionalStyling: React.FC<ConditionalStylingProps> = ({
    variant,
    size,
    disabled,
    loading,
}) => {
    const baseClasses =
        'inline-flex items-center justify-center font-medium rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2';

    const variantClasses = {
        primary: 'bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500',
        secondary: 'bg-gray-200 text-gray-900 hover:bg-gray-300 focus:ring-gray-500',
        danger: 'bg-red-600 text-white hover:bg-red-700 focus:ring-red-500',
    };

    const sizeClasses = {
        sm: 'px-3 py-1.5 text-sm',
        md: 'px-4 py-2 text-sm',
        lg: 'px-6 py-3 text-base',
    };

    const stateClasses = {
        disabled: 'opacity-50 cursor-not-allowed',
        loading: 'opacity-75 cursor-wait',
    };

    const className = [
        baseClasses,
        variantClasses[variant],
        sizeClasses[size],
        disabled && stateClasses.disabled,
        loading && stateClasses.loading,
    ]
        .filter(Boolean)
        .join(' ');

    return (
        <button className={className} disabled={disabled || loading} aria-busy={loading}>
            {loading && <LoadingSpinner size="sm" className="mr-2" />}
            Button Text
        </button>
    );
};
```

## Style Guide Summary

### Best Practices

1. **Use PatternFly components** as the primary UI library
2. **Complement with Tailwind** for custom styling and layouts
3. **Maintain consistent spacing** using Tailwind's spacing scale
4. **Follow accessibility guidelines** with proper ARIA labels
5. **Optimize for performance** with efficient CSS and animations
6. **Use semantic HTML** elements where appropriate
7. **Implement responsive design** with mobile-first approach
8. **Consistent error and loading states** across components
9. **Follow color scheme** for consistent branding
10. **Document component styling** with clear examples
