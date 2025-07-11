# StackRox UI - Quick Reference Guide

## 🚀 Most Common Patterns

### Component Template

```typescript
import React from 'react';
import { ComponentProps } from './types';

interface Props extends ComponentProps {
    // Component-specific props
}

function ComponentName({ prop1, prop2, ...props }: Props) {
    // 1. Hooks
    // 2. Event handlers
    // 3. Computed values
    // 4. Render logic

    return <div {...props}>{/* JSX */}</div>;
}

export default ComponentName;
```

### Service Function Template

```typescript
export async function fetchData(): Promise<DataType[]> {
    try {
        const response = await axios.get<DataResponse>('/api/data');
        return response.data.items || [];
    } catch (error) {
        throw new ServiceError('Failed to fetch data', error.status, error);
    }
}
```

### Custom Hook Template

```typescript
export function useDataFetcher<T>(queryFn: () => Promise<T>) {
    const [data, setData] = useState<T | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const fetchData = useCallback(async () => {
        setIsLoading(true);
        setError(null);
        try {
            const result = await queryFn();
            setData(result);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Error occurred');
        } finally {
            setIsLoading(false);
        }
    }, [queryFn]);

    useEffect(() => {
        fetchData();
    }, [fetchData]);

    return { data, isLoading, error, refetch: fetchData };
}
```

## 🎨 PatternFly Quick Patterns

### Form Components

```typescript
import { Form, FormGroup, TextInput, Button } from '@patternfly/react-core';

<Form onSubmit={handleSubmit}>
    <FormGroup label="Name" isRequired fieldId="name">
        <TextInput
            id="name"
            value={name}
            onChange={(value) => setName(value)}
            isRequired
        />
    </FormGroup>
    <Button type="submit">Save</Button>
</Form>
```

### Table Components

```typescript
import { Table, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';

<Table>
    <Thead>
        <Tr>
            <Th>Name</Th>
            <Th>Status</Th>
        </Tr>
    </Thead>
    <Tbody>
        {items.map(item => (
            <Tr key={item.id}>
                <Td>{item.name}</Td>
                <Td>{item.status}</Td>
            </Tr>
        ))}
    </Tbody>
</Table>
```

### Card Layout

```typescript
import { Card, CardBody, CardHeader, Title } from '@patternfly/react-core';

<Card>
    <CardHeader>
        <Title headingLevel="h3">Card Title</Title>
    </CardHeader>
    <CardBody>
        Card content
    </CardBody>
</Card>
```

## 🔍 Testing Quick Patterns

### Component Test

```typescript
import { render, screen, fireEvent } from '@testing-library/react';
import { vi } from 'vitest';

describe('Component', () => {
    it('should handle user interaction', async () => {
        const mockHandler = vi.fn();
        render(<Component onAction={mockHandler} />);

        fireEvent.click(screen.getByRole('button'));

        expect(mockHandler).toHaveBeenCalled();
    });
});
```

### Service Test

```typescript
import { vi } from 'vitest';
import { fetchData } from './service';

vi.mock('services/instance', () => ({
    axios: { get: vi.fn() },
}));

describe('Service', () => {
    it('should fetch data successfully', async () => {
        vi.mocked(axios.get).mockResolvedValue({ data: { items: [] } });

        const result = await fetchData();

        expect(result).toEqual([]);
    });
});
```

## ♿ Accessibility Quick Patterns

### Form Accessibility

```typescript
<label htmlFor="email">Email (required)</label>
<input
    id="email"
    type="email"
    required
    aria-describedby="email-error"
    aria-invalid={hasError ? 'true' : 'false'}
/>
{hasError && (
    <div id="email-error" role="alert">
        Please enter a valid email
    </div>
)}
```

### Button Accessibility

```typescript
<button
    type="button"
    onClick={handleAction}
    aria-label={`Delete ${itemName}`}
    aria-describedby="delete-description"
>
    Delete
</button>
<div id="delete-description" className="sr-only">
    This action cannot be undone
</div>
```

## 📝 Common TypeScript Patterns

### Interface Design

```typescript
interface BaseProps {
    id: string;
    name: string;
}

interface ComponentProps extends BaseProps {
    onAction: (id: string) => void;
    isLoading?: boolean;
}

// Union types for controlled values
type Status = 'active' | 'inactive' | 'pending';

// Generic types
interface ApiResponse<T> {
    data: T;
    success: boolean;
    error?: string;
}
```

### Type Guards

```typescript
function isValidUser(obj: unknown): obj is User {
    return typeof obj === 'object' && obj !== null && 'id' in obj && 'name' in obj;
}
```

## 🚨 Error Handling Quick Patterns

### Service Error Handling

```typescript
export class ServiceError extends Error {
    constructor(
        message: string,
        public code: number,
        public cause: Error
    ) {
        super(message);
        this.name = 'ServiceError';
    }
}

// Usage
try {
    const data = await fetchData();
} catch (error) {
    if (error instanceof ServiceError) {
        // Handle service error
    } else {
        // Handle other errors
    }
}
```

### Component Error Handling

```typescript
function DataComponent() {
    const [error, setError] = useState<string | null>(null);

    const handleAction = async () => {
        try {
            await performAction();
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Action failed');
        }
    };

    if (error) {
        return <Alert variant="danger" title={error} />;
    }

    return <div>Component content</div>;
}
```

## 📚 Common Import Patterns

### Standard Imports

```typescript
// External libraries
import React, { useState, useEffect } from 'react';
import { Button, Card } from '@patternfly/react-core';

// Internal modules
import { fetchData } from 'services/DataService';
import { useAuth } from 'hooks/useAuth';

// Types
import type { User } from 'types/user';

// Relative imports
import './Component.css';
```

## 🏗️ File Organization

```
src/
├── Components/
│   ├── PatternFly/          # PatternFly wrappers
│   └── ComponentName/       # Component + types + tests
├── Containers/              # Page-level components
├── hooks/                   # Custom hooks
├── services/               # API services
├── types/                  # TypeScript types
├── constants/              # Application constants
└── utils/                  # Utility functions
```

## 🔧 Development Commands

```bash
# Development
npm run start              # Start dev server
npm run build             # Build for production
npm run lint              # Run ESLint
npm run lint:fix          # Fix ESLint issues
npm run tsc               # TypeScript check

# Testing
npm run test              # Run unit tests
npm run test:coverage     # Run with coverage
npm run test:e2e          # Run Cypress tests
npm run test:component    # Run component tests
```

## 📋 Pre-Commit Checklist

- [ ] ESLint passes (`npm run lint`)
- [ ] TypeScript compiles (`npm run tsc`)
- [ ] Tests pass (`npm run test`)
- [ ] Component has proper types
- [ ] Error handling is implemented
- [ ] Accessibility attributes added
- [ ] No console.log statements
- [ ] Imports are organized correctly

## 🎯 Quick Fixes for Common Issues

### "Property does not exist on type"

```typescript
// Instead of 'any'
interface ApiResponse {
    data: YourDataType[];
    success: boolean;
}
```

### "Missing dependencies in useEffect"

```typescript
// Add all used variables to dependency array
useEffect(() => {
    fetchData(userId);
}, [userId]); // Include userId
```

### "Element implicitly has 'any' type"

```typescript
// Add proper event types
const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    setValue(event.target.value);
};
```

### "Missing accessibility attributes"

```typescript
// Add proper ARIA labels
<button
    type="button"
    aria-label="Close dialog"
    onClick={onClose}
>
    ×
</button>
```

---

💡 **Remember**: These patterns are enforced by Cursor rules. Follow them for consistency across the team!
