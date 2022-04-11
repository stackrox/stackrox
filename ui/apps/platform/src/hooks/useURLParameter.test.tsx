import React, { ReactNode } from 'react';
import { MemoryRouter, Route } from 'react-router-dom';
import { renderHook, act } from '@testing-library/react-hooks';

import { URLSearchParams } from 'url';
import useURLParameter from './useURLParameter';

type WrapperProps = {
    children: ReactNode;
    onRouteRender: ({ location: any }) => void;
    initialEntries: string[];
};

// This Wrapper component allows the `useURLParameter` hook to simulate the browser's
// URL bar in JSDom via the MemoryRouter
function Wrapper({ children, onRouteRender, initialEntries = [] }: WrapperProps) {
    return (
        <MemoryRouter initialEntries={initialEntries}>
            <Route path="*" render={onRouteRender} />
            {children}
        </MemoryRouter>
    );
}

test('should read/write scoped string value in URL parameter without changing existing URL parameters', async () => {
    let params;
    let testLocation;

    const { result } = renderHook(() => useURLParameter<string | undefined>('testKey', undefined), {
        initialProps: {
            children: [],
            onRouteRender: ({ location }) => {
                testLocation = location;
            },
            initialEntries: ['?oldKey=test'],
        },
        wrapper: Wrapper,
    });

    // Check new and existing values before setter function is called
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0]).toBeUndefined();
    expect(params.get('testKey')).toBeNull();
    expect(params.get('oldKey')).toBe('test');
    expect(params.get('bogusKey')).toBeNull();

    // Check new and existing values when URL parameter is set
    act(() => {
        const [, setParam] = result.current;
        setParam('testValue');
    });
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0]).toBe('testValue');
    expect(params.get('testKey')).toBe('testValue');
    expect(params.get('oldKey')).toBe('test');
    expect(params.get('bogusKey')).toBeNull();

    // Check new and existing values when URL parameter is cleared
    act(() => {
        const [, setParam] = result.current;
        setParam(undefined);
    });
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0]).toBeUndefined();
    expect(params.get('testKey')).toBeNull();
    expect(params.get('oldKey')).toBe('test');
    expect(params.get('bogusKey')).toBeNull();
});

test('should read/write scoped complex object in URL parameter without changing existing URL parameters', async () => {
    let params: URLSearchParams;
    let testLocation;

    type StateObject = {
        clusters: {
            id: string;
            name: string;
            namespaces: {
                id: string;
                name: string;
            }[];
        }[];
    };

    const emptyState: StateObject = { clusters: [] };
    const { result } = renderHook(() => useURLParameter<StateObject>('testKey', emptyState), {
        initialProps: {
            children: [],
            onRouteRender: ({ location }) => {
                testLocation = location;
            },
            initialEntries: ['?oldKey=test'],
        },
        wrapper: Wrapper,
    });

    // Check new and existing values before setter function is called
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0].clusters).toHaveLength(0);
    expect(params.get('testKey')).toBeNull();
    expect(params.get('oldKey')).toBe('test');
    expect(Array.from(params.entries())).toHaveLength(1);

    act(() => {
        const [, setParam] = result.current;
        setParam({
            clusters: [
                {
                    id: 'c-1',
                    name: 'production',
                    namespaces: [
                        { id: 'ns-1', name: 'stackrox' },
                        { id: 'ns-2', name: 'payments' },
                    ],
                },
            ],
        });
    });

    // Check new and existing values before setter function is called
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0].clusters).toHaveLength(1);
    expect(result.current[0].clusters[0].id).toBe('c-1');
    expect(result.current[0].clusters[0].name).toBe('production');
    expect(result.current[0].clusters[0].namespaces).toHaveLength(2);
    expect(params.get('testKey')).toBeNull();
    expect(params.get('oldKey')).toBe('test');
    expect(params.get('testKey[clusters][0][id]')).toBe('c-1');
    expect(params.get('testKey[clusters][0][name]')).toBe('production');
    expect(params.get('testKey[clusters][0][namespaces][0][id]')).toBe('ns-1');
    expect(params.get('testKey[clusters][0][namespaces][0][name]')).toBe('stackrox');
    expect(params.get('testKey[clusters][0][namespaces][1][id]')).toBe('ns-2');
    expect(params.get('testKey[clusters][0][namespaces][1][name]')).toBe('payments');

    // Clear value and ensure URL search is removed
    act(() => {
        const [, setParam] = result.current;
        setParam(emptyState);
    });
    params = new URLSearchParams(testLocation.search);
    expect(params.get('testKey')).toBeNull();
    expect(params.get('oldKey')).toBe('test');
    expect(Array.from(params.entries())).toHaveLength(1);
});
