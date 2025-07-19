---
description: TypeScript patterns and type safety rules for StackRox UI team
globs: ['**/*.{ts,tsx}', '**/*types*.{js,jsx}']
alwaysApply: false
---

# TypeScript Patterns

## Type Definition Standards

### Interface Definitions

```typescript
// ✅ GOOD - Consistent interface patterns
interface UserProfile {
    /** Unique identifier for the user */
    id: string;
    /** User's display name */
    name: string;
    /** User's email address */
    email: string;
    /** User's avatar URL */
    avatar?: string;
    /** User's role in the system */
    role: UserRole;
    /** Whether the user is currently active */
    isActive: boolean;
    /** Timestamp of user's last login */
    lastLogin: string; // ISO 8601 date string
    /** User's permissions */
    permissions: Permission[];
    /** User's preferences */
    preferences: UserPreferences;
}

// ❌ AVOID - Inconsistent interface patterns
interface UserProfile {
    id: string;
    name: string;
    email: string;
    avatar: string | null; // Use optional ? instead
    role: any; // Should be specific type
    isActive: boolean;
    lastLogin: Date; // Should be string for API consistency
    permissions: string[]; // Should be specific type
    preferences: any; // Should be specific type
}
```

### Union Types and Enums

```typescript
// ✅ GOOD - Consistent union types and enums
export type UserRole = 'admin' | 'analyst' | 'viewer';

export const USER_ROLES = {
    ADMIN: 'admin',
    ANALYST: 'analyst',
    VIEWER: 'viewer',
} as const;

export type UserRoleKey = keyof typeof USER_ROLES;

// For more complex enums
export enum ComplianceStatus {
    COMPLIANT = 'compliant',
    NON_COMPLIANT = 'non_compliant',
    UNKNOWN = 'unknown',
}

// API response types
export type ApiResponse<T> = {
    data: T;
    message: string;
    success: boolean;
};

export type ApiError = {
    error: string;
    code: number;
    details?: Record<string, unknown>;
};
```

### Generic Types

```typescript
// ✅ GOOD - Consistent generic type patterns
export interface PaginatedResponse<T> {
    data: T[];
    pagination: {
        page: number;
        pageSize: number;
        total: number;
        totalPages: number;
    };
}

export interface CacheEntry<T> {
    data: T;
    timestamp: number;
    expiry: number;
}

export type ServiceResponse<T> =
    | {
          success: true;
          data: T;
      }
    | {
          success: false;
          error: string;
      };

// Generic utility types
export type Optional<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;
export type RequiredFields<T, K extends keyof T> = T & Required<Pick<T, K>>;
```

## API Type Patterns

### Proto Types (.proto.ts files)

```typescript
// ✅ GOOD - Consistent proto type patterns
// user.proto.ts
export interface UserProfile {
    id: string;
    name: string;
    email: string;
    avatar?: string;
    role: UserRole;
    isActive: boolean;
    lastLogin: string; // ISO 8601 date string
    createdAt: string; // ISO 8601 date string
    updatedAt: string; // ISO 8601 date string
    permissions: Permission[];
    preferences: UserPreferences;
}

export interface Permission {
    id: string;
    name: string;
    resource: string;
    action: string;
    effect: 'allow' | 'deny';
}

export interface UserPreferences {
    theme: 'light' | 'dark';
    language: string;
    timezone: string;
    notifications: NotificationPreferences;
}

export interface NotificationPreferences {
    email: boolean;
    push: boolean;
    inApp: boolean;
}

// compliance.proto.ts
export interface ComplianceReport {
    id: string;
    name: string;
    status: ComplianceStatus;
    clusterId: string;
    clusterName: string;
    standards: ComplianceStandard[];
    controls: ComplianceControl[];
    createdAt: string;
    updatedAt: string;
}

export interface ComplianceStandard {
    id: string;
    name: string;
    description: string;
    controls: ComplianceControl[];
}

export interface ComplianceControl {
    id: string;
    name: string;
    description: string;
    status: ComplianceStatus;
    severity: 'low' | 'medium' | 'high' | 'critical';
    remediation?: string;
}
```

### Service Types

```typescript
// ✅ GOOD - Consistent service type patterns
// userService.ts
import { UserProfile, Permission } from 'types/user.proto';
import { ApiResponse, PaginatedResponse } from 'types/common';

export interface UserSearchParams {
    query?: string;
    role?: UserRole;
    isActive?: boolean;
    page?: number;
    pageSize?: number;
}

export interface CreateUserRequest {
    name: string;
    email: string;
    role: UserRole;
    permissions: string[];
}

export interface UpdateUserRequest {
    name?: string;
    email?: string;
    role?: UserRole;
    permissions?: string[];
    isActive?: boolean;
}

export type UserServiceResponse = ApiResponse<UserProfile>;
export type UsersListResponse = PaginatedResponse<UserProfile>;
export type PermissionsResponse = ApiResponse<Permission[]>;
```

## Hook Type Patterns

### Custom Hook Types

```typescript
// ✅ GOOD - Consistent hook type patterns
// useUserProfile.ts
import { UserProfile } from 'types/user.proto';

export interface UseUserProfileOptions {
    autoRefresh?: boolean;
    refreshInterval?: number;
    onError?: (error: Error) => void;
    onSuccess?: (user: UserProfile) => void;
}

export interface UseUserProfileReturn {
    data: UserProfile | null;
    error: string | null;
    isLoading: boolean;
    refetch: () => Promise<void>;
}

export function useUserProfile(
    userId: string,
    options?: UseUserProfileOptions
): UseUserProfileReturn {
    // Hook implementation
}

// useComplianceData.ts
export interface UseComplianceDataOptions {
    clusterId?: string;
    standardId?: string;
    autoRefresh?: boolean;
}

export interface UseComplianceDataReturn {
    data: ComplianceReport | null;
    error: string | null;
    isLoading: boolean;
    refetch: () => Promise<void>;
}

export function useComplianceData(options?: UseComplianceDataOptions): UseComplianceDataReturn {
    // Hook implementation
}
```

### State Management Types

```typescript
// ✅ GOOD - Consistent state management types
// Redux state types
export interface UserState {
    byId: Record<string, UserProfile>;
    allIds: string[];
    loading: boolean;
    error: string | null;
    currentUserId: string | null;
}

export interface ComplianceState {
    reports: Record<string, ComplianceReport>;
    loading: boolean;
    error: string | null;
    selectedReportId: string | null;
}

export interface RootState {
    user: UserState;
    compliance: ComplianceState;
    // Other state slices
}

// Action types
export interface FetchUserAction {
    type: 'FETCH_USER';
    payload: { userId: string };
}

export interface FetchUserSuccessAction {
    type: 'FETCH_USER_SUCCESS';
    payload: { user: UserProfile };
}

export interface FetchUserFailureAction {
    type: 'FETCH_USER_FAILURE';
    payload: { error: string };
}

export type UserAction = FetchUserAction | FetchUserSuccessAction | FetchUserFailureAction;
```

## Component Type Patterns

### Props Types

```typescript
// ✅ GOOD - Consistent component prop types
import React from 'react';
import { UserProfile } from 'types/user.proto';

// Base props interface
interface BaseComponentProps {
    className?: string;
    testId?: string;
    children?: React.ReactNode;
}

// Specific component props
export interface UserProfileProps extends BaseComponentProps {
    userId: string;
    showAvatar?: boolean;
    onEdit?: (userId: string) => void;
    onUserLoaded?: (user: UserProfile) => void;
}

export interface ComplianceReportProps extends BaseComponentProps {
    reportId: string;
    clusterId?: string;
    onReportGenerated?: (report: ComplianceReport) => void;
}

// Form component props
export interface UserFormProps extends BaseComponentProps {
    initialValues?: Partial<UserProfile>;
    onSubmit: (values: CreateUserRequest) => Promise<void>;
    onCancel: () => void;
    isEditing?: boolean;
}

// List component props
export interface UserListProps extends BaseComponentProps {
    users: UserProfile[];
    selectedUsers: string[];
    onUserSelect: (userId: string) => void;
    onUserDeselect: (userId: string) => void;
    onUserEdit: (userId: string) => void;
    onUserDelete: (userId: string) => void;
}
```

### Event Handler Types

```typescript
// ✅ GOOD - Consistent event handler types
export type UserEventHandler = (userId: string) => void;
export type UserFormEventHandler = (values: CreateUserRequest) => Promise<void>;
export type UserSelectionEventHandler = (selectedUsers: string[]) => void;

export interface UserTableEventHandlers {
    onUserSelect: UserEventHandler;
    onUserEdit: UserEventHandler;
    onUserDelete: UserEventHandler;
    onUserView: UserEventHandler;
}

// Form event handlers
export interface FormEventHandlers<T> {
    onSubmit: (values: T) => Promise<void>;
    onCancel: () => void;
    onReset: () => void;
    onFieldChange: (field: keyof T, value: unknown) => void;
}
```

## Utility Type Patterns

### Type Guards

```typescript
// ✅ GOOD - Consistent type guard patterns
export function isUserProfile(value: unknown): value is UserProfile {
    return (
        typeof value === 'object' &&
        value !== null &&
        typeof (value as UserProfile).id === 'string' &&
        typeof (value as UserProfile).name === 'string' &&
        typeof (value as UserProfile).email === 'string'
    );
}

export function isComplianceReport(value: unknown): value is ComplianceReport {
    return (
        typeof value === 'object' &&
        value !== null &&
        typeof (value as ComplianceReport).id === 'string' &&
        typeof (value as ComplianceReport).name === 'string' &&
        typeof (value as ComplianceReport).status === 'string'
    );
}

export function isApiError(value: unknown): value is ApiError {
    return (
        typeof value === 'object' &&
        value !== null &&
        typeof (value as ApiError).error === 'string' &&
        typeof (value as ApiError).code === 'number'
    );
}
```

### Validation Types

```typescript
// ✅ GOOD - Consistent validation patterns
export interface ValidationRule<T> {
    validate: (value: T) => boolean;
    message: string;
}

export interface ValidationSchema<T> {
    [K in keyof T]?: ValidationRule<T[K]>[];
}

export type ValidationErrors<T> = {
    [K in keyof T]?: string[];
};

export type ValidationResult<T> = {
    isValid: boolean;
    errors: ValidationErrors<T>;
};

// Example validation schema
export const userValidationSchema: ValidationSchema<CreateUserRequest> = {
    name: [
        { validate: (value) => value.length > 0, message: 'Name is required' },
        { validate: (value) => value.length <= 50, message: 'Name must be 50 characters or less' }
    ],
    email: [
        { validate: (value) => value.includes('@'), message: 'Email must be valid' },
        { validate: (value) => value.length > 0, message: 'Email is required' }
    ],
    role: [
        { validate: (value) => USER_ROLES[value as UserRoleKey] !== undefined, message: 'Role must be valid' }
    ]
};
```

## Error Handling Types

### Error Types

```typescript
// ✅ GOOD - Consistent error type patterns
export interface BaseError {
    message: string;
    code: string;
    timestamp: string;
}

export interface ValidationError extends BaseError {
    field: string;
    value: unknown;
}

export interface ApiError extends BaseError {
    statusCode: number;
    details?: Record<string, unknown>;
}

export interface NetworkError extends BaseError {
    isNetworkError: true;
    originalError: Error;
}

export type AppError = ValidationError | ApiError | NetworkError;

// Error handling utilities
export function isValidationError(error: AppError): error is ValidationError {
    return 'field' in error;
}

export function isApiError(error: AppError): error is ApiError {
    return 'statusCode' in error;
}

export function isNetworkError(error: AppError): error is NetworkError {
    return 'isNetworkError' in error;
}
```

## Configuration Types

### Environment and Config Types

```typescript
// ✅ GOOD - Consistent configuration types
export interface AppConfig {
    apiBaseUrl: string;
    environment: 'development' | 'staging' | 'production';
    version: string;
    buildTime: string;
    features: FeatureFlags;
    logging: LoggingConfig;
}

export interface FeatureFlags {
    enableNewUserInterface: boolean;
    enableComplianceReports: boolean;
    enableAdvancedFiltering: boolean;
}

export interface LoggingConfig {
    level: 'debug' | 'info' | 'warn' | 'error';
    enableConsoleLogging: boolean;
    enableRemoteLogging: boolean;
}

// Runtime configuration
export interface RuntimeConfig {
    user: UserProfile;
    permissions: Permission[];
    preferences: UserPreferences;
    clusters: ClusterInfo[];
}

export interface ClusterInfo {
    id: string;
    name: string;
    status: 'healthy' | 'unhealthy' | 'unknown';
    version: string;
}
```

## Testing Types

### Test Utility Types

```typescript
// ✅ GOOD - Consistent testing types
export interface MockUserProfile extends Partial<UserProfile> {
    id: string;
    name: string;
    email: string;
}

export interface TestHookResult<T> {
    result: { current: T };
    rerender: (newProps?: any) => void;
    unmount: () => void;
}

export interface ComponentTestProps<T = Record<string, unknown>> {
    props?: T;
    initialState?: Partial<RootState>;
    mocks?: Record<string, jest.Mock>;
}

// Test data factory types
export type UserProfileFactory = (overrides?: Partial<UserProfile>) => UserProfile;
export type ComplianceReportFactory = (overrides?: Partial<ComplianceReport>) => ComplianceReport;
```

## Type Organization

### Index File Patterns

```typescript
// ✅ GOOD - Consistent type exports
// types/index.ts
export type { UserProfile, Permission, UserPreferences } from './user.proto';
export type { ComplianceReport, ComplianceStandard, ComplianceControl } from './compliance.proto';
export type { ClusterInfo } from './cluster.proto';

// Re-export common types
export type { ApiResponse, ApiError, PaginatedResponse, ServiceResponse } from './common';

// Hook types
export type { UseUserProfileReturn, UseComplianceDataReturn } from './hooks';

// Component types
export type { UserProfileProps, ComplianceReportProps } from './components';
```

### Namespace Organization

```typescript
// ✅ GOOD - Consistent namespace usage
declare namespace StackRox {
    namespace API {
        interface User extends UserProfile {}
        interface Compliance extends ComplianceReport {}
    }

    namespace Components {
        interface UserProfileProps {
            userId: string;
            showAvatar?: boolean;
        }
    }

    namespace Hooks {
        interface UseUserProfileReturn {
            data: UserProfile | null;
            error: string | null;
            isLoading: boolean;
        }
    }
}
```

## Best Practices Summary

### Type Safety Rules

1. **Always use specific types** instead of `any`
2. **Use optional properties** (`?`) appropriately
3. **Prefer interfaces over types** for object definitions
4. **Use union types** for constrained values
5. **Implement type guards** for runtime validation
6. **Use generic types** for reusable components
7. **Keep API types in `.proto.ts` files**
8. **Export types alongside implementations**

### Migration Guidelines

```typescript
// ✅ GOOD - Gradual TypeScript migration
// Step 1: Add types to new files
// Step 2: Convert existing .js files to .ts
// Step 3: Add proper interfaces
// Step 4: Remove any types
// Step 5: Add proper error handling types
// Step 6: Update tests with proper types

// Example migration
// Before (JavaScript)
function fetchUser(id) {
    return axios.get(`/api/users/${id}`);
}

// After (TypeScript)
export async function fetchUser(id: string): Promise<UserProfile> {
    try {
        const response = await axios.get<UserProfile>(`/api/users/${id}`);
        return response.data;
    } catch (error) {
        throw new Error(`Failed to fetch user: ${error.message}`);
    }
}
```
