---
description: Service layer and API integration patterns for StackRox UI team
globs:
    ['**/services/**/*.{js,ts}', '**/api/**/*.{js,ts}', '**/*Service*.{js,ts}', '**/*API*.{js,ts}']
alwaysApply: false
---

# Service Layer Patterns

## Service File Organization

### Service Structure

```typescript
// ✅ GOOD - Consistent service organization
// userService.ts
import axios from './instance';
import { UserProfile, CreateUserRequest, UpdateUserRequest } from 'types/user.proto';
import { ApiResponse, PaginatedResponse } from 'types/common';

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

/**
 * Creates a new user
 * @param userData - The user data for creation
 * @returns Promise resolving to created user profile
 */
export async function createUser(userData: CreateUserRequest): Promise<UserProfile> {
    try {
        const response = await axios.post<UserProfile>(baseUrl, userData);
        return response.data;
    } catch (error) {
        console.error('Failed to create user:', error);
        throw new Error(`Unable to create user: ${error.message}`);
    }
}

/**
 * Updates an existing user
 * @param userId - The ID of the user to update
 * @param userData - The user data for update
 * @returns Promise resolving to updated user profile
 */
export async function updateUser(
    userId: string,
    userData: UpdateUserRequest
): Promise<UserProfile> {
    if (!userId) {
        throw new Error('User ID is required');
    }

    try {
        const response = await axios.put<UserProfile>(`${baseUrl}/${userId}`, userData);
        return response.data;
    } catch (error) {
        console.error('Failed to update user:', error);
        throw new Error(`Unable to update user: ${error.message}`);
    }
}

/**
 * Deletes a user
 * @param userId - The ID of the user to delete
 */
export async function deleteUser(userId: string): Promise<void> {
    if (!userId) {
        throw new Error('User ID is required');
    }

    try {
        await axios.delete(`${baseUrl}/${userId}`);
    } catch (error) {
        console.error('Failed to delete user:', error);
        throw new Error(`Unable to delete user: ${error.message}`);
    }
}

/**
 * Fetches paginated list of users
 * @param params - Search and pagination parameters
 * @returns Promise resolving to paginated user list
 */
export async function fetchUsers(
    params: {
        query?: string;
        role?: string;
        page?: number;
        pageSize?: number;
    } = {}
): Promise<PaginatedResponse<UserProfile>> {
    try {
        const queryString = new URLSearchParams(
            Object.entries(params)
                .filter(([_, value]) => value !== undefined)
                .map(([key, value]) => [key, String(value)])
        ).toString();

        const url = queryString ? `${baseUrl}?${queryString}` : baseUrl;
        const response = await axios.get<PaginatedResponse<UserProfile>>(url);
        return response.data;
    } catch (error) {
        console.error('Failed to fetch users:', error);
        throw new Error(`Unable to load users: ${error.message}`);
    }
}
```

### Service Instance Configuration

```typescript
// ✅ GOOD - Consistent axios instance setup
// services/instance.ts
import axios from 'axios';

const instance = axios.create({
    timeout: 10000,
    headers: {
        'Content-Type': 'application/json',
    },
});

// Request interceptor for authentication
instance.interceptors.request.use(
    (config) => {
        const token = localStorage.getItem('authToken');
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => {
        console.error('Request interceptor error:', error);
        return Promise.reject(error);
    }
);

// Response interceptor for error handling
instance.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.response?.status === 401) {
            // Handle unauthorized access
            localStorage.removeItem('authToken');
            window.location.href = '/login';
        }

        if (error.response?.status >= 500) {
            // Handle server errors
            console.error('Server error:', error.response);
        }

        return Promise.reject(error);
    }
);

export default instance;
```

## Error Handling Patterns

### Consistent Error Handling

```typescript
// ✅ GOOD - Consistent error handling across services
import { AxiosError } from 'axios';

export interface ServiceError {
    message: string;
    code: string;
    statusCode?: number;
    details?: Record<string, unknown>;
}

export function handleServiceError(error: unknown): ServiceError {
    if (axios.isAxiosError(error)) {
        const axiosError = error as AxiosError;

        return {
            message: axiosError.response?.data?.message || axiosError.message,
            code: axiosError.code || 'UNKNOWN_ERROR',
            statusCode: axiosError.response?.status,
            details: axiosError.response?.data,
        };
    }

    if (error instanceof Error) {
        return {
            message: error.message,
            code: 'GENERIC_ERROR',
        };
    }

    return {
        message: 'An unexpected error occurred',
        code: 'UNKNOWN_ERROR',
    };
}

// Usage in service functions
export async function fetchUserProfile(userId: string): Promise<UserProfile> {
    try {
        const response = await axios.get<UserProfile>(`/api/users/${userId}`);
        return response.data;
    } catch (error) {
        const serviceError = handleServiceError(error);
        console.error('Failed to fetch user profile:', serviceError);
        throw new Error(serviceError.message);
    }
}
```

### Service Response Types

```typescript
// ✅ GOOD - Consistent service response patterns
export type ServiceResult<T> =
    | {
          success: true;
          data: T;
      }
    | {
          success: false;
          error: ServiceError;
      };

export async function fetchUserProfileSafe(userId: string): Promise<ServiceResult<UserProfile>> {
    try {
        const user = await fetchUserProfile(userId);
        return { success: true, data: user };
    } catch (error) {
        return {
            success: false,
            error: handleServiceError(error),
        };
    }
}

// Usage in components
const loadUser = async () => {
    const result = await fetchUserProfileSafe(userId);

    if (result.success) {
        setUser(result.data);
        setError(null);
    } else {
        setError(result.error.message);
        setUser(null);
    }
};
```

## API Integration Patterns

### GraphQL Integration

```typescript
// ✅ GOOD - Consistent GraphQL patterns
import { gql } from '@apollo/client';
import { UserProfile } from 'types/user.proto';

// Fragment definitions
export const USER_PROFILE_FRAGMENT = gql`
    fragment UserProfileFragment on UserProfile {
        id
        name
        email
        avatar
        role
        isActive
        lastLogin
        createdAt
        updatedAt
        permissions {
            id
            name
            resource
            action
            effect
        }
        preferences {
            theme
            language
            timezone
            notifications {
                email
                push
                inApp
            }
        }
    }
`;

// Query definitions
export const GET_USER_PROFILE = gql`
    query GetUserProfile($userId: ID!) {
        userProfile(id: $userId) {
            ...UserProfileFragment
        }
    }
    ${USER_PROFILE_FRAGMENT}
`;

export const GET_USERS = gql`
    query GetUsers($filter: UserFilter, $pagination: PaginationInput) {
        users(filter: $filter, pagination: $pagination) {
            data {
                ...UserProfileFragment
            }
            pagination {
                page
                pageSize
                total
                totalPages
            }
        }
    }
    ${USER_PROFILE_FRAGMENT}
`;

// Mutation definitions
export const CREATE_USER = gql`
    mutation CreateUser($input: CreateUserInput!) {
        createUser(input: $input) {
            ...UserProfileFragment
        }
    }
    ${USER_PROFILE_FRAGMENT}
`;

export const UPDATE_USER = gql`
    mutation UpdateUser($userId: ID!, $input: UpdateUserInput!) {
        updateUser(id: $userId, input: $input) {
            ...UserProfileFragment
        }
    }
    ${USER_PROFILE_FRAGMENT}
`;

export const DELETE_USER = gql`
    mutation DeleteUser($userId: ID!) {
        deleteUser(id: $userId)
    }
`;
```

### REST API Patterns

```typescript
// ✅ GOOD - Consistent REST API patterns
// complianceService.ts
import axios from './instance';
import { ComplianceReport, ComplianceStandard } from 'types/compliance.proto';
import { PaginatedResponse } from 'types/common';

const baseUrl = '/api/v1/compliance';

export interface ComplianceSearchParams {
    clusterId?: string;
    standardId?: string;
    status?: 'compliant' | 'non_compliant' | 'unknown';
    severity?: 'low' | 'medium' | 'high' | 'critical';
    page?: number;
    pageSize?: number;
    sortBy?: string;
    sortOrder?: 'asc' | 'desc';
}

/**
 * Fetches compliance reports with filtering and pagination
 */
export async function fetchComplianceReports(
    params: ComplianceSearchParams = {}
): Promise<PaginatedResponse<ComplianceReport>> {
    try {
        const queryString = buildQueryString(params);
        const url = `${baseUrl}/reports${queryString}`;

        const response = await axios.get<PaginatedResponse<ComplianceReport>>(url);
        return response.data;
    } catch (error) {
        console.error('Failed to fetch compliance reports:', error);
        throw new Error(`Unable to load compliance reports: ${error.message}`);
    }
}

/**
 * Fetches a specific compliance report by ID
 */
export async function fetchComplianceReport(reportId: string): Promise<ComplianceReport> {
    if (!reportId) {
        throw new Error('Report ID is required');
    }

    try {
        const response = await axios.get<ComplianceReport>(`${baseUrl}/reports/${reportId}`);
        return response.data;
    } catch (error) {
        console.error('Failed to fetch compliance report:', error);
        throw new Error(`Unable to load compliance report: ${error.message}`);
    }
}

/**
 * Generates a new compliance report
 */
export async function generateComplianceReport(request: {
    clusterId: string;
    standardIds: string[];
    name: string;
}): Promise<ComplianceReport> {
    try {
        const response = await axios.post<ComplianceReport>(`${baseUrl}/reports`, request);
        return response.data;
    } catch (error) {
        console.error('Failed to generate compliance report:', error);
        throw new Error(`Unable to generate compliance report: ${error.message}`);
    }
}

/**
 * Downloads a compliance report as PDF
 */
export async function downloadComplianceReport(reportId: string): Promise<Blob> {
    try {
        const response = await axios.get(`${baseUrl}/reports/${reportId}/download`, {
            responseType: 'blob',
        });
        return response.data;
    } catch (error) {
        console.error('Failed to download compliance report:', error);
        throw new Error(`Unable to download compliance report: ${error.message}`);
    }
}

// Helper function for building query strings
function buildQueryString(params: Record<string, unknown>): string {
    const searchParams = new URLSearchParams();

    Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null && value !== '') {
            searchParams.append(key, String(value));
        }
    });

    const queryString = searchParams.toString();
    return queryString ? `?${queryString}` : '';
}
```

## Caching Patterns

### Service-Level Caching

```typescript
// ✅ GOOD - Consistent caching patterns
interface CacheEntry<T> {
    data: T;
    timestamp: number;
    expiry: number;
}

class ServiceCache {
    private cache = new Map<string, CacheEntry<unknown>>();
    private defaultTTL = 5 * 60 * 1000; // 5 minutes

    set<T>(key: string, data: T, ttl: number = this.defaultTTL): void {
        this.cache.set(key, {
            data,
            timestamp: Date.now(),
            expiry: Date.now() + ttl,
        });
    }

    get<T>(key: string): T | null {
        const entry = this.cache.get(key) as CacheEntry<T> | undefined;

        if (!entry) {
            return null;
        }

        if (Date.now() > entry.expiry) {
            this.cache.delete(key);
            return null;
        }

        return entry.data;
    }

    invalidate(pattern?: string): void {
        if (pattern) {
            const regex = new RegExp(pattern);
            for (const [key] of this.cache) {
                if (regex.test(key)) {
                    this.cache.delete(key);
                }
            }
        } else {
            this.cache.clear();
        }
    }
}

const cache = new ServiceCache();

// Usage in services
export async function fetchUserProfileCached(userId: string): Promise<UserProfile> {
    const cacheKey = `user:${userId}`;
    const cached = cache.get<UserProfile>(cacheKey);

    if (cached) {
        return cached;
    }

    const user = await fetchUserProfile(userId);
    cache.set(cacheKey, user);
    return user;
}

export async function updateUserCached(
    userId: string,
    userData: UpdateUserRequest
): Promise<UserProfile> {
    const updatedUser = await updateUser(userId, userData);

    // Update cache
    const cacheKey = `user:${userId}`;
    cache.set(cacheKey, updatedUser);

    // Invalidate related cache entries
    cache.invalidate(`users:.*`); // Invalidate user lists

    return updatedUser;
}
```

## File Upload/Download Patterns

### File Service Patterns

```typescript
// ✅ GOOD - Consistent file handling patterns
// fileService.ts
import axios from './instance';

export interface UploadProgressEvent {
    loaded: number;
    total: number;
    percentage: number;
}

export interface FileUploadOptions {
    onProgress?: (event: UploadProgressEvent) => void;
    timeout?: number;
}

/**
 * Uploads a file to the server
 */
export async function uploadFile(
    file: File,
    endpoint: string,
    options: FileUploadOptions = {}
): Promise<{ fileId: string; url: string }> {
    const formData = new FormData();
    formData.append('file', file);

    try {
        const response = await axios.post(endpoint, formData, {
            headers: {
                'Content-Type': 'multipart/form-data',
            },
            timeout: options.timeout || 30000,
            onUploadProgress: (progressEvent) => {
                if (options.onProgress && progressEvent.total) {
                    options.onProgress({
                        loaded: progressEvent.loaded,
                        total: progressEvent.total,
                        percentage: Math.round((progressEvent.loaded * 100) / progressEvent.total),
                    });
                }
            },
        });

        return response.data;
    } catch (error) {
        console.error('Failed to upload file:', error);
        throw new Error(`Unable to upload file: ${error.message}`);
    }
}

/**
 * Downloads a file from the server
 */
export async function downloadFile(url: string, filename: string): Promise<void> {
    try {
        const response = await axios.get(url, {
            responseType: 'blob',
        });

        const blob = new Blob([response.data]);
        const downloadUrl = window.URL.createObjectURL(blob);

        const link = document.createElement('a');
        link.href = downloadUrl;
        link.download = filename;
        document.body.appendChild(link);
        link.click();

        document.body.removeChild(link);
        window.URL.revokeObjectURL(downloadUrl);
    } catch (error) {
        console.error('Failed to download file:', error);
        throw new Error(`Unable to download file: ${error.message}`);
    }
}

/**
 * Uploads user avatar
 */
export async function uploadUserAvatar(
    userId: string,
    file: File,
    onProgress?: (percentage: number) => void
): Promise<string> {
    const endpoint = `/api/v1/users/${userId}/avatar`;

    const result = await uploadFile(file, endpoint, {
        onProgress: (event) => onProgress?.(event.percentage),
    });

    return result.url;
}
```

## Service Testing Patterns

### Service Unit Tests

```typescript
// ✅ GOOD - Consistent service testing
// userService.test.ts
import axios from 'axios';
import { fetchUserProfile, createUser, updateUser, deleteUser } from './userService';
import { mockUserProfile } from 'test-utils/mockData';

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
            expect(mockedAxios.get).toHaveBeenCalledWith('/api/v1/users/user-123');
        });

        it('throws error when API call fails', async () => {
            const errorMessage = 'Network error';
            mockedAxios.get.mockRejectedValue(new Error(errorMessage));

            await expect(fetchUserProfile('user-123')).rejects.toThrow(
                `Unable to load user profile: ${errorMessage}`
            );
        });

        it('validates input parameters', async () => {
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
            expect(mockedAxios.post).toHaveBeenCalledWith('/api/v1/users', newUserData);
        });

        it('handles validation errors', async () => {
            const validationError = {
                response: {
                    status: 400,
                    data: { message: 'Email is required' },
                },
            };
            mockedAxios.post.mockRejectedValue(validationError);

            await expect(createUser(newUserData)).rejects.toThrow(
                'Unable to create user: Email is required'
            );
        });
    });

    describe('error handling', () => {
        it('handles network errors consistently', async () => {
            const networkError = new Error('Network Error');
            mockedAxios.get.mockRejectedValue(networkError);

            await expect(fetchUserProfile('user-123')).rejects.toThrow(
                'Unable to load user profile: Network Error'
            );
        });

        it('handles HTTP error responses', async () => {
            const httpError = {
                response: {
                    status: 404,
                    data: { message: 'User not found' },
                },
            };
            mockedAxios.get.mockRejectedValue(httpError);

            await expect(fetchUserProfile('user-123')).rejects.toThrow(
                'Unable to load user profile: User not found'
            );
        });
    });
});
```

## API Documentation Patterns

### Service Documentation

````typescript
// ✅ GOOD - Consistent service documentation
/**
 * User Service
 *
 * Handles all user-related API operations including:
 * - User profile management
 * - User creation and updates
 * - User authentication
 * - User permissions
 *
 * @example
 * ```typescript
 * import { fetchUserProfile, createUser } from 'services/userService';
 *
 * // Fetch user profile
 * const user = await fetchUserProfile('user-123');
 *
 * // Create new user
 * const newUser = await createUser({
 *   name: 'John Doe',
 *   email: 'john.doe@example.com',
 *   role: 'analyst',
 *   permissions: ['read_users']
 * });
 * ```
 */

/**
 * Fetches user profile data from the API
 *
 * @param userId - The unique identifier for the user
 * @returns Promise resolving to user profile data
 * @throws {Error} When user ID is missing or invalid
 * @throws {Error} When user is not found (404)
 * @throws {Error} When API is unavailable (500+)
 *
 * @example
 * ```typescript
 * try {
 *   const user = await fetchUserProfile('user-123');
 *   console.log(user.name);
 * } catch (error) {
 *   console.error('Failed to load user:', error.message);
 * }
 * ```
 */
export async function fetchUserProfile(userId: string): Promise<UserProfile> {
    // Implementation
}
````

## Performance Optimization

### Request Optimization

```typescript
// ✅ GOOD - Request optimization patterns
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';

/**
 * Cancellable request utility
 */
export function makeCancellableRequest<T>(
    requestFn: (signal: AbortSignal) => Promise<T>
): CancellableRequest<T> {
    const controller = new AbortController();

    const request = requestFn(controller.signal).catch((error) => {
        if (error.name === 'AbortError') {
            throw new Error('Request was cancelled');
        }
        throw error;
    });

    return {
        request,
        cancel: () => controller.abort(),
    };
}

/**
 * Fetch users with cancellation support
 */
export function fetchUsersCancellable(
    params: ComplianceSearchParams = {}
): CancellableRequest<PaginatedResponse<UserProfile>> {
    return makeCancellableRequest(async (signal) => {
        const queryString = buildQueryString(params);
        const url = `/api/v1/users${queryString}`;

        const response = await axios.get<PaginatedResponse<UserProfile>>(url, { signal });
        return response.data;
    });
}

/**
 * Debounced search function
 */
export const debouncedSearchUsers = debounce(async (query: string) => {
    return fetchUsers({ query });
}, 300);
```

## Integration Guidelines

### Service Layer Best Practices

1. **Consistent error handling** across all services
2. **Input validation** for all service functions
3. **Proper TypeScript types** for requests and responses
4. **Comprehensive JSDoc documentation** for public APIs
5. **Cancellation support** for long-running requests
6. **Caching strategies** for frequently accessed data
7. **File upload/download** utilities for binary data
8. **Test coverage** for all service functions

### Migration Strategy

```typescript
// ✅ GOOD - Service migration approach
// Step 1: Create TypeScript interfaces for existing APIs
// Step 2: Add proper error handling to existing services
// Step 3: Implement consistent response patterns
// Step 4: Add input validation
// Step 5: Write comprehensive tests
// Step 6: Add JSDoc documentation
// Step 7: Implement caching where appropriate
// Step 8: Add cancellation support for new services
```
