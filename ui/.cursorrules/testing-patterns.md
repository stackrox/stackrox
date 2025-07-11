---
description: Testing patterns and best practices for StackRox UI team
globs:
    [
        '**/*.test.{js,jsx,ts,tsx}',
        '**/*.spec.{js,jsx,ts,tsx}',
        '**/test-utils/**/*',
        '**/*test*.{js,jsx,ts,tsx}',
    ]
alwaysApply: false
---

# Testing Patterns

## Test File Organization

### File Structure

```
✅ GOOD - Consistent test organization
src/
├── Components/
│   ├── UserProfile/
│   │   ├── UserProfile.tsx
│   │   ├── UserProfile.test.tsx
│   │   └── UserProfile.cy.tsx (Cypress component test)
│   └── ComplianceReport/
│       ├── ComplianceReport.tsx
│       ├── ComplianceReport.test.tsx
│       └── __mocks__/
│           └── complianceData.ts
├── services/
│   ├── userService.ts
│   └── userService.test.ts
├── hooks/
│   ├── useUserProfile.ts
│   └── useUserProfile.test.ts
└── test-utils/
    ├── renderWithProviders.tsx
    ├── mockData.ts
    └── testHelpers.ts
```

### Test File Naming

```typescript
// ✅ GOOD - Consistent test file naming
UserProfile.test.tsx; // Unit tests
UserProfile.cy.tsx; // Cypress component tests
userService.test.ts; // Service tests
useUserProfile.test.ts; // Hook tests
testHelpers.ts; // Test utilities
mockData.ts; // Mock data

// ❌ AVOID - Inconsistent naming
UserProfile.spec.tsx;
UserProfile - test.tsx;
user - service.test.ts;
userProfileHook.test.ts;
```

## Unit Testing Patterns

### Component Testing

```typescript
// ✅ GOOD - Consistent component testing
import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Provider } from 'react-redux';
import { BrowserRouter } from 'react-router-dom';

import UserProfile from './UserProfile';
import { createMockStore } from 'test-utils/mockStore';
import { mockUserProfile } from 'test-utils/mockData';

// Test wrapper component
const renderUserProfile = (props = {}, initialState = {}) => {
    const store = createMockStore(initialState);
    const defaultProps = {
        userId: 'user-123',
        showAvatar: true,
        ...props,
    };

    return render(
        <Provider store={store}>
            <BrowserRouter>
                <UserProfile {...defaultProps} />
            </BrowserRouter>
        </Provider>
    );
};

describe('UserProfile', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    it('renders user profile with correct data', async () => {
        const initialState = {
            users: {
                byId: {
                    'user-123': mockUserProfile,
                },
                loading: false,
                error: null,
            },
        };

        renderUserProfile({}, initialState);

        expect(screen.getByText(mockUserProfile.name)).toBeInTheDocument();
        expect(screen.getByText(mockUserProfile.email)).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /edit profile/i })).toBeInTheDocument();
    });

    it('shows loading state initially', () => {
        const initialState = {
            users: {
                byId: {},
                loading: true,
                error: null,
            },
        };

        renderUserProfile({}, initialState);

        expect(screen.getByText(/loading user profile/i)).toBeInTheDocument();
    });

    it('displays error state when user fails to load', () => {
        const initialState = {
            users: {
                byId: {},
                loading: false,
                error: 'User not found',
            },
        };

        renderUserProfile({}, initialState);

        expect(screen.getByText(/error: user not found/i)).toBeInTheDocument();
    });

    it('calls onEdit when edit button is clicked', async () => {
        const user = userEvent.setup();
        const mockOnEdit = jest.fn();
        const initialState = {
            users: {
                byId: {
                    'user-123': mockUserProfile,
                },
                loading: false,
                error: null,
            },
        };

        renderUserProfile({ onEdit: mockOnEdit }, initialState);

        const editButton = screen.getByRole('button', { name: /edit profile/i });
        await user.click(editButton);

        expect(mockOnEdit).toHaveBeenCalledWith('user-123');
    });

    it('conditionally renders avatar when showAvatar is true', () => {
        const initialState = {
            users: {
                byId: {
                    'user-123': mockUserProfile,
                },
                loading: false,
                error: null,
            },
        };

        renderUserProfile({ showAvatar: true }, initialState);

        expect(screen.getByAltText(`${mockUserProfile.name} avatar`)).toBeInTheDocument();
    });

    it('does not render avatar when showAvatar is false', () => {
        const initialState = {
            users: {
                byId: {
                    'user-123': mockUserProfile,
                },
                loading: false,
                error: null,
            },
        };

        renderUserProfile({ showAvatar: false }, initialState);

        expect(screen.queryByAltText(`${mockUserProfile.name} avatar`)).not.toBeInTheDocument();
    });
});
```

### Hook Testing

```typescript
// ✅ GOOD - Consistent hook testing
import { renderHook, act } from '@testing-library/react';
import { Provider } from 'react-redux';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { useUserProfile } from './useUserProfile';
import { createMockStore } from 'test-utils/mockStore';
import { mockUserProfile } from 'test-utils/mockData';
import * as userService from 'services/userService';

// Mock the service
jest.mock('services/userService');
const mockUserService = userService as jest.Mocked<typeof userService>;

const createWrapper = (initialState = {}) => {
    const store = createMockStore(initialState);
    const queryClient = new QueryClient({
        defaultOptions: {
            queries: { retry: false },
            mutations: { retry: false },
        },
    });

    return ({ children }: { children: React.ReactNode }) => (
        <Provider store={store}>
            <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
        </Provider>
    );
};

describe('useUserProfile', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    it('returns user data when successful', async () => {
        mockUserService.fetchUserProfile.mockResolvedValue(mockUserProfile);

        const { result } = renderHook(() => useUserProfile('user-123'), {
            wrapper: createWrapper(),
        });

        expect(result.current.isLoading).toBe(true);
        expect(result.current.data).toBe(null);
        expect(result.current.error).toBe(null);

        await act(async () => {
            await new Promise((resolve) => setTimeout(resolve, 0));
        });

        expect(result.current.isLoading).toBe(false);
        expect(result.current.data).toEqual(mockUserProfile);
        expect(result.current.error).toBe(null);
    });

    it('returns error when fetch fails', async () => {
        const errorMessage = 'User not found';
        mockUserService.fetchUserProfile.mockRejectedValue(new Error(errorMessage));

        const { result } = renderHook(() => useUserProfile('user-123'), {
            wrapper: createWrapper(),
        });

        await act(async () => {
            await new Promise((resolve) => setTimeout(resolve, 0));
        });

        expect(result.current.isLoading).toBe(false);
        expect(result.current.data).toBe(null);
        expect(result.current.error).toBe(errorMessage);
    });

    it('refetches data when refetch is called', async () => {
        mockUserService.fetchUserProfile.mockResolvedValue(mockUserProfile);

        const { result } = renderHook(() => useUserProfile('user-123'), {
            wrapper: createWrapper(),
        });

        await act(async () => {
            await new Promise((resolve) => setTimeout(resolve, 0));
        });

        expect(mockUserService.fetchUserProfile).toHaveBeenCalledTimes(1);

        await act(async () => {
            await result.current.refetch();
        });

        expect(mockUserService.fetchUserProfile).toHaveBeenCalledTimes(2);
    });
});
```

### Service Testing

```typescript
// ✅ GOOD - Consistent service testing
import axios from 'axios';
import { fetchUserProfile, createUser, updateUser } from './userService';
import { mockUserProfile } from 'test-utils/mockData';

// Mock axios
jest.mock('axios');
const mockedAxios = axios as jest.Mocked<typeof axios>;

describe('userService', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    describe('fetchUserProfile', () => {
        it('returns user profile when successful', async () => {
            mockedAxios.get.mockResolvedValue({ data: mockUserProfile });

            const result = await fetchUserProfile('user-123');

            expect(result).toEqual(mockUserProfile);
            expect(mockedAxios.get).toHaveBeenCalledWith('/api/users/user-123');
        });

        it('throws error when API call fails', async () => {
            const errorMessage = 'Network error';
            mockedAxios.get.mockRejectedValue(new Error(errorMessage));

            await expect(fetchUserProfile('user-123')).rejects.toThrow(
                `Unable to load user profile: ${errorMessage}`
            );
        });

        it('throws error when user ID is invalid', async () => {
            await expect(fetchUserProfile('')).rejects.toThrow('User ID is required');
        });
    });

    describe('createUser', () => {
        const newUserData = {
            name: 'John Doe',
            email: 'john.doe@example.com',
            role: 'analyst' as const,
            permissions: ['read_users'],
        };

        it('creates user successfully', async () => {
            const createdUser = { ...mockUserProfile, ...newUserData };
            mockedAxios.post.mockResolvedValue({ data: createdUser });

            const result = await createUser(newUserData);

            expect(result).toEqual(createdUser);
            expect(mockedAxios.post).toHaveBeenCalledWith('/api/users', newUserData);
        });

        it('throws error when creation fails', async () => {
            const errorMessage = 'Validation error';
            mockedAxios.post.mockRejectedValue(new Error(errorMessage));

            await expect(createUser(newUserData)).rejects.toThrow(
                `Unable to create user: ${errorMessage}`
            );
        });
    });

    describe('updateUser', () => {
        const updateData = {
            name: 'Updated Name',
            email: 'updated@example.com',
        };

        it('updates user successfully', async () => {
            const updatedUser = { ...mockUserProfile, ...updateData };
            mockedAxios.put.mockResolvedValue({ data: updatedUser });

            const result = await updateUser('user-123', updateData);

            expect(result).toEqual(updatedUser);
            expect(mockedAxios.put).toHaveBeenCalledWith('/api/users/user-123', updateData);
        });

        it('throws error when update fails', async () => {
            const errorMessage = 'User not found';
            mockedAxios.put.mockRejectedValue(new Error(errorMessage));

            await expect(updateUser('user-123', updateData)).rejects.toThrow(
                `Unable to update user: ${errorMessage}`
            );
        });
    });
});
```

## Integration Testing Patterns

### Cypress Component Tests

```typescript
// ✅ GOOD - Consistent Cypress component testing
import React from 'react';
import { Provider } from 'react-redux';
import { BrowserRouter } from 'react-router-dom';

import UserProfile from './UserProfile';
import { createMockStore } from 'test-utils/mockStore';
import { mockUserProfile } from 'test-utils/mockData';

describe('UserProfile Component', () => {
    beforeEach(() => {
        cy.intercept('GET', '/api/users/*', { body: mockUserProfile }).as('getUserProfile');
    });

    it('renders user profile and handles edit interaction', () => {
        const store = createMockStore({
            users: {
                byId: { 'user-123': mockUserProfile },
                loading: false,
                error: null,
            },
        });

        cy.mount(
            <Provider store={store}>
                <BrowserRouter>
                    <UserProfile userId="user-123" />
                </BrowserRouter>
            </Provider>
        );

        // Verify initial render
        cy.get('[data-testid="user-profile-card"]').should('be.visible');
        cy.contains(mockUserProfile.name).should('be.visible');
        cy.contains(mockUserProfile.email).should('be.visible');

        // Test edit interaction
        cy.get('[data-testid="edit-button"]').click();
        cy.get('[data-testid="user-form"]').should('be.visible');
    });

    it('handles loading state correctly', () => {
        const store = createMockStore({
            users: {
                byId: {},
                loading: true,
                error: null,
            },
        });

        cy.mount(
            <Provider store={store}>
                <BrowserRouter>
                    <UserProfile userId="user-123" />
                </BrowserRouter>
            </Provider>
        );

        cy.get('[data-testid="loading-spinner"]').should('be.visible');
        cy.contains('Loading user profile').should('be.visible');
    });

    it('displays error state when user fails to load', () => {
        const store = createMockStore({
            users: {
                byId: {},
                loading: false,
                error: 'User not found',
            },
        });

        cy.mount(
            <Provider store={store}>
                <BrowserRouter>
                    <UserProfile userId="user-123" />
                </BrowserRouter>
            </Provider>
        );

        cy.get('[data-testid="error-message"]').should('be.visible');
        cy.contains('Error: User not found').should('be.visible');
    });
});
```

### E2E Testing Patterns

```typescript
// ✅ GOOD - Consistent E2E testing
describe('User Management Flow', () => {
    beforeEach(() => {
        cy.intercept('GET', '/api/users', { fixture: 'users.json' }).as('getUsers');
        cy.intercept('GET', '/api/users/*', { fixture: 'user.json' }).as('getUser');
        cy.intercept('POST', '/api/users', { fixture: 'user.json' }).as('createUser');
        cy.intercept('PUT', '/api/users/*', { fixture: 'user.json' }).as('updateUser');
        cy.intercept('DELETE', '/api/users/*', { statusCode: 204 }).as('deleteUser');

        cy.login('admin@example.com', 'password');
        cy.visit('/users');
    });

    it('displays user list and allows filtering', () => {
        cy.wait('@getUsers');

        // Verify user list is displayed
        cy.get('[data-testid="user-list"]').should('be.visible');
        cy.get('[data-testid="user-row"]').should('have.length.greaterThan', 0);

        // Test search functionality
        cy.get('[data-testid="search-input"]').type('john');
        cy.get('[data-testid="user-row"]').should('contain.text', 'john');

        // Test role filter
        cy.get('[data-testid="role-filter"]').select('admin');
        cy.get('[data-testid="user-row"]').each(($row) => {
            cy.wrap($row).should('contain.text', 'admin');
        });
    });

    it('creates a new user successfully', () => {
        cy.get('[data-testid="add-user-button"]').click();
        cy.get('[data-testid="user-form"]').should('be.visible');

        // Fill out form
        cy.get('[data-testid="name-input"]').type('Jane Doe');
        cy.get('[data-testid="email-input"]').type('jane.doe@example.com');
        cy.get('[data-testid="role-select"]').select('analyst');

        // Submit form
        cy.get('[data-testid="submit-button"]').click();

        // Verify user was created
        cy.wait('@createUser');
        cy.get('[data-testid="success-message"]').should(
            'contain.text',
            'User created successfully'
        );
        cy.get('[data-testid="user-list"]').should('contain.text', 'Jane Doe');
    });

    it('edits an existing user', () => {
        cy.get('[data-testid="user-row"]')
            .first()
            .within(() => {
                cy.get('[data-testid="edit-button"]').click();
            });

        cy.get('[data-testid="user-form"]').should('be.visible');

        // Update name
        cy.get('[data-testid="name-input"]').clear().type('John Smith');

        // Submit form
        cy.get('[data-testid="submit-button"]').click();

        // Verify user was updated
        cy.wait('@updateUser');
        cy.get('[data-testid="success-message"]').should(
            'contain.text',
            'User updated successfully'
        );
        cy.get('[data-testid="user-list"]').should('contain.text', 'John Smith');
    });

    it('deletes a user with confirmation', () => {
        cy.get('[data-testid="user-row"]')
            .first()
            .within(() => {
                cy.get('[data-testid="delete-button"]').click();
            });

        // Confirm deletion
        cy.get('[data-testid="confirm-dialog"]').should('be.visible');
        cy.get('[data-testid="confirm-button"]').click();

        // Verify user was deleted
        cy.wait('@deleteUser');
        cy.get('[data-testid="success-message"]').should(
            'contain.text',
            'User deleted successfully'
        );
    });
});
```

## Test Data Management

### Mock Data Organization

```typescript
// ✅ GOOD - Consistent mock data patterns
// test-utils/mockData.ts
import { UserProfile, ComplianceReport } from 'types';

export const mockUserProfile: UserProfile = {
    id: 'user-123',
    name: 'John Doe',
    email: 'john.doe@example.com',
    avatar: 'https://example.com/avatar.jpg',
    role: 'analyst',
    isActive: true,
    lastLogin: '2023-12-01T10:00:00Z',
    createdAt: '2023-01-01T00:00:00Z',
    updatedAt: '2023-12-01T10:00:00Z',
    permissions: [
        {
            id: 'perm-1',
            name: 'read_users',
            resource: 'users',
            action: 'read',
            effect: 'allow',
        },
    ],
    preferences: {
        theme: 'light',
        language: 'en',
        timezone: 'UTC',
        notifications: {
            email: true,
            push: false,
            inApp: true,
        },
    },
};

export const mockComplianceReport: ComplianceReport = {
    id: 'report-123',
    name: 'Q4 2023 Compliance Report',
    status: 'compliant',
    clusterId: 'cluster-123',
    clusterName: 'Production Cluster',
    standards: [
        {
            id: 'standard-1',
            name: 'PCI DSS 3.2.1',
            description: 'Payment Card Industry Data Security Standard',
            controls: [],
        },
    ],
    controls: [
        {
            id: 'control-1',
            name: 'Access Control',
            description: 'Implement proper access controls',
            status: 'compliant',
            severity: 'high',
            remediation: 'Review access permissions regularly',
        },
    ],
    createdAt: '2023-12-01T00:00:00Z',
    updatedAt: '2023-12-01T00:00:00Z',
};

// Factory functions for variations
export const createMockUser = (overrides: Partial<UserProfile> = {}): UserProfile => ({
    ...mockUserProfile,
    ...overrides,
});

export const createMockComplianceReport = (
    overrides: Partial<ComplianceReport> = {}
): ComplianceReport => ({
    ...mockComplianceReport,
    ...overrides,
});
```

### Test Utilities

```typescript
// ✅ GOOD - Consistent test utilities
// test-utils/renderWithProviders.tsx
import React from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { Provider } from 'react-redux';
import { BrowserRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { createMockStore } from './mockStore';
import { RootState } from 'types';

interface CustomRenderOptions extends RenderOptions {
    initialState?: Partial<RootState>;
    store?: ReturnType<typeof createMockStore>;
}

export function renderWithProviders(ui: React.ReactElement, options: CustomRenderOptions = {}) {
    const { initialState = {}, store = createMockStore(initialState), ...renderOptions } = options;

    const queryClient = new QueryClient({
        defaultOptions: {
            queries: { retry: false },
            mutations: { retry: false },
        },
    });

    function Wrapper({ children }: { children: React.ReactNode }) {
        return (
            <Provider store={store}>
                <QueryClientProvider client={queryClient}>
                    <BrowserRouter>{children}</BrowserRouter>
                </QueryClientProvider>
            </Provider>
        );
    }

    return {
        store,
        ...render(ui, { wrapper: Wrapper, ...renderOptions }),
    };
}

// Helper for testing hooks with providers
export function renderHookWithProviders<T>(hook: () => T, options: CustomRenderOptions = {}) {
    const { initialState = {}, store = createMockStore(initialState) } = options;

    const queryClient = new QueryClient({
        defaultOptions: {
            queries: { retry: false },
            mutations: { retry: false },
        },
    });

    function wrapper({ children }: { children: React.ReactNode }) {
        return (
            <Provider store={store}>
                <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
            </Provider>
        );
    }

    return { store, ...renderHook(hook, { wrapper }) };
}
```

## Test Quality Standards

### Assertions and Expectations

```typescript
// ✅ GOOD - Clear and specific assertions
describe('UserProfile', () => {
    it('displays user information correctly', () => {
        renderWithProviders(<UserProfile userId="user-123" />);

        // Specific assertions
        expect(screen.getByText('John Doe')).toBeInTheDocument();
        expect(screen.getByText('john.doe@example.com')).toBeInTheDocument();
        expect(screen.getByAltText('John Doe avatar')).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /edit profile/i })).toBeEnabled();
    });

    it('handles async operations correctly', async () => {
        const mockOnEdit = jest.fn();
        renderWithProviders(<UserProfile userId="user-123" onEdit={mockOnEdit} />);

        const editButton = screen.getByRole('button', { name: /edit profile/i });
        await userEvent.click(editButton);

        await waitFor(() => {
            expect(mockOnEdit).toHaveBeenCalledWith('user-123');
        });
    });
});

// ❌ AVOID - Vague or weak assertions
describe('UserProfile', () => {
    it('renders something', () => {
        renderWithProviders(<UserProfile userId="user-123" />);

        expect(screen.getByTestId('user-profile')).toBeTruthy(); // Too vague
        expect(document.querySelector('.user-profile')).toBeDefined(); // Fragile
    });
});
```

### Test Organization

```typescript
// ✅ GOOD - Well-organized test structure
describe('UserProfile', () => {
    // Setup
    const defaultProps = {
        userId: 'user-123',
        showAvatar: true,
    };

    const renderUserProfile = (props = {}) => {
        return renderWithProviders(<UserProfile {...defaultProps} {...props} />);
    };

    // Group related tests
    describe('rendering', () => {
        it('displays user information when loaded', () => {
            // Test implementation
        });

        it('shows loading state initially', () => {
            // Test implementation
        });

        it('displays error state when loading fails', () => {
            // Test implementation
        });
    });

    describe('interactions', () => {
        it('calls onEdit when edit button is clicked', () => {
            // Test implementation
        });

        it('handles form submission correctly', () => {
            // Test implementation
        });
    });

    describe('accessibility', () => {
        it('has proper ARIA labels', () => {
            // Test implementation
        });

        it('supports keyboard navigation', () => {
            // Test implementation
        });
    });
});
```

## Performance Testing

### Performance Assertions

```typescript
// ✅ GOOD - Performance-focused tests
describe('UserProfile Performance', () => {
    it('renders efficiently with large datasets', () => {
        const largeUserList = Array.from({ length: 1000 }, (_, i) =>
            createMockUser({ id: `user-${i}`, name: `User ${i}` })
        );

        const startTime = performance.now();
        renderWithProviders(<UserList users={largeUserList} />);
        const endTime = performance.now();

        expect(endTime - startTime).toBeLessThan(100); // Should render in under 100ms
    });

    it('does not cause unnecessary re-renders', () => {
        const mockRender = jest.fn();
        const UserProfileWithSpy = React.memo(() => {
            mockRender();
            return <UserProfile userId="user-123" />;
        });

        const { rerender } = renderWithProviders(<UserProfileWithSpy />);

        // Re-render with same props
        rerender(<UserProfileWithSpy />);

        expect(mockRender).toHaveBeenCalledTimes(1); // Should not re-render
    });
});
```

## Accessibility Testing

### A11y Testing Patterns

```typescript
// ✅ GOOD - Accessibility testing
import { axe, toHaveNoViolations } from 'jest-axe';

expect.extend(toHaveNoViolations);

describe('UserProfile Accessibility', () => {
    it('has no accessibility violations', async () => {
        const { container } = renderWithProviders(<UserProfile userId="user-123" />);

        const results = await axe(container);
        expect(results).toHaveNoViolations();
    });

    it('supports keyboard navigation', async () => {
        const user = userEvent.setup();
        renderWithProviders(<UserProfile userId="user-123" />);

        // Tab through interactive elements
        await user.tab();
        expect(screen.getByRole('button', { name: /edit profile/i })).toHaveFocus();

        // Activate with Enter key
        await user.keyboard('{Enter}');
        expect(screen.getByTestId('user-form')).toBeInTheDocument();
    });

    it('has proper ARIA attributes', () => {
        renderWithProviders(<UserProfile userId="user-123" />);

        expect(screen.getByRole('button', { name: /edit profile/i })).toHaveAttribute(
            'aria-label',
            'Edit profile for John Doe'
        );
    });
});
```

## Test Maintenance

### Test Cleanup

```typescript
// ✅ GOOD - Proper test cleanup
describe('UserProfile', () => {
    beforeEach(() => {
        jest.clearAllMocks();
        jest.clearAllTimers();
    });

    afterEach(() => {
        cleanup();
        jest.restoreAllMocks();
    });

    afterAll(() => {
        jest.resetAllMocks();
    });
});
```

### Test Documentation

```typescript
// ✅ GOOD - Well-documented tests
describe('UserProfile', () => {
    /**
     * Test suite for UserProfile component rendering behavior
     * Covers: loading states, error states, data display
     */
    describe('rendering', () => {
        /**
         * Verifies that user information is displayed correctly
         * when the component receives valid user data
         */
        it('displays user information when loaded', () => {
            // Test implementation
        });
    });
});
```
