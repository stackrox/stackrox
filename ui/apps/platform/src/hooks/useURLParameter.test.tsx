import React from 'react';
import { createMemoryHistory } from 'history';
import { MemoryRouter, useLocation, useNavigate } from 'react-router-dom';
import { HistoryRouter as Router } from 'redux-first-history/rr6';
import { renderHook, act } from '@testing-library/react';

import { URLSearchParams } from 'url';
import useURLParameter from './useURLParameter';

beforeAll(() => {
    jest.useFakeTimers();
});

function actAndRunTicks(callback) {
    return act(() => {
        callback();
        jest.runAllTicks();
    });
}

test('should read/write scoped string value in URL parameter without changing existing URL parameters', async () => {
    let params;
    let testLocation;

    const { result } = renderHook(
        () => {
            testLocation = useLocation();
            return useURLParameter('testKey', undefined);
        },
        {
            wrapper: ({ children }) => (
                <MemoryRouter initialEntries={['?oldKey=test']}>{children}</MemoryRouter>
            ),
        }
    );

    // Check new and existing values before setter function is called
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0]).toBeUndefined();
    expect(params.get('testKey')).toBeNull();
    expect(params.get('oldKey')).toBe('test');
    expect(params.get('bogusKey')).toBeNull();

    // Check new and existing values when URL parameter is set
    actAndRunTicks(() => {
        const [, setParam] = result.current;
        setParam('testValue');
    });
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0]).toBe('testValue');
    expect(params.get('testKey')).toBe('testValue');
    expect(params.get('oldKey')).toBe('test');
    expect(params.get('bogusKey')).toBeNull();

    // Check new and existing values when URL parameter is cleared
    actAndRunTicks(() => {
        const [, setParam] = result.current;
        setParam(undefined);
    });
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0]).toBeUndefined();
    expect(params.get('testKey')).toBeNull();
    expect(params.get('oldKey')).toBe('test');
    expect(params.get('bogusKey')).toBeNull();
});

test('should allow multiple sequential parameter updates without data loss', async () => {
    let params;
    let testLocation;

    const { result } = renderHook(
        () => {
            testLocation = useLocation();
            return [useURLParameter('key1', 'oldValue1'), useURLParameter('key2', undefined)];
        },
        {
            wrapper: ({ children }) => (
                <MemoryRouter initialEntries={['?key1=oldValue1']}>{children}</MemoryRouter>
            ),
        }
    );

    params = new URLSearchParams(testLocation.search);
    expect(params.get('key1')).toBe('oldValue1');
    expect(params.get('key2')).toBe(null);

    actAndRunTicks(() => {
        const [[, setParam1], [, setParam2]] = result.current;
        setParam1('newValue1');
        setParam2('newValue2');
    });
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0][0]).toBe('newValue1');
    expect(result.current[1][0]).toBe('newValue2');
    expect(params.get('key1')).toBe('newValue1');
    expect(params.get('key2')).toBe('newValue2');
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

    const { result } = renderHook(
        () => {
            testLocation = useLocation();
            return useURLParameter('testKey', emptyState);
        },
        {
            wrapper: ({ children }) => (
                <MemoryRouter initialEntries={['?oldKey=test']}>{children}</MemoryRouter>
            ),
        }
    );

    function isStateObject(obj: unknown): obj is StateObject {
        return typeof obj === 'object' && obj !== null && 'clusters' in obj;
    }

    // Check new and existing values before setter function is called
    params = new URLSearchParams(testLocation.search);
    if (!isStateObject(result.current[0])) {
        return;
    }
    expect(result.current[0].clusters).toHaveLength(0);
    expect(params.get('testKey')).toBeNull();
    expect(params.get('oldKey')).toBe('test');
    expect(Array.from(params.entries())).toHaveLength(1);

    actAndRunTicks(() => {
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
    actAndRunTicks(() => {
        const [, setParam] = result.current;
        setParam(emptyState);
    });
    params = new URLSearchParams(testLocation.search);
    expect(params.get('testKey')).toBeNull();
    expect(params.get('oldKey')).toBe('test');
    expect(Array.from(params.entries())).toHaveLength(1);
});

test('should implement push and replace state for navigate', async () => {
    let testNavigate;
    let testLocation;

    const { result } = renderHook(
        () => {
            testLocation = useLocation();
            testNavigate = useNavigate();
            return useURLParameter('testKey', undefined);
        },
        {
            wrapper: ({ children }) => (
                <MemoryRouter
                    initialEntries={['/main/dashboard', '/main/clusters?oldKey=test']}
                    initialIndex={1}
                >
                    {children}
                </MemoryRouter>
            ),
        }
    );

    // Test the the default behavior is to push URL parameter changes to the history stack
    actAndRunTicks(() => {
        const [, setParam] = result.current;
        setParam('testValue');
    });
    expect(testLocation.pathname).toBe('/main/clusters');
    expect(testLocation.search).toBe('?oldKey=test&testKey=testValue');
    actAndRunTicks(() => {
        testNavigate(-1);
    });
    expect(testLocation.pathname).toBe('/main/clusters');
    expect(testLocation.search).toBe('?oldKey=test');

    // Test that specifying a history action of 'replace' changes the history entry in-place
    actAndRunTicks(() => {
        const [, setParam] = result.current;
        setParam('newTestValue', 'replace');
    });
    expect(testLocation.pathname).toBe('/main/clusters');
    expect(testLocation.search).toBe('?oldKey=test&testKey=newTestValue');
    actAndRunTicks(() => {
        testNavigate(-1);
    });
    expect(testLocation.pathname).toBe('/main/dashboard');
    expect(testLocation.search).toBe('');
});

test('should batch URL parameter updates', async () => {
    let params;
    let testLocation;

    const history = createMemoryHistory({
        initialEntries: [''],
    });

    const { result } = renderHook(
        () => {
            testLocation = useLocation();
            return {
                hook1: useURLParameter('testKey1', undefined),
                hook2: useURLParameter('testKey2', undefined),
            };
        },
        {
            wrapper: ({ children }) => <Router history={history}>{children}</Router>,
        }
    );

    params = new URLSearchParams(testLocation.search);
    expect(history.index).toBe(0);
    expect(params.get('testKey1')).toBeNull();
    expect(params.get('testKey2')).toBeNull();
    expect(result.current.hook1[0]).toBeUndefined();
    expect(result.current.hook2[0]).toBeUndefined();

    // Default behavior is to push URL parameter changes to the history stack
    actAndRunTicks(() => {
        const [, setParam] = result.current.hook1;
        setParam('testValue');
    });

    params = new URLSearchParams(testLocation.search);
    expect(history.index).toBe(1);
    expect(params.get('testKey1')).toBe('testValue');
    expect(params.get('testKey2')).toBeNull();
    expect(result.current.hook1[0]).toBe('testValue');
    expect(result.current.hook2[0]).toBeUndefined();

    // Multiple URL updates should be batched into a single history entry
    actAndRunTicks(() => {
        const [, setParam1] = result.current.hook1;
        const [, setParam2] = result.current.hook2;
        setParam1('newValue1');
        setParam2('newValue2');
        setParam2('newValue3');
    });

    params = new URLSearchParams(testLocation.search);
    expect(history.index).toBe(2);
    expect(params.get('testKey1')).toBe('newValue1');
    expect(params.get('testKey2')).toBe('newValue3');
    expect(result.current.hook1[0]).toBe('newValue1');
    expect(result.current.hook2[0]).toBe('newValue3');

    // A single URL update with a 'replace' action should replace the current history entry
    actAndRunTicks(() => {
        const [, setParam] = result.current.hook1;
        setParam('newTestValue', 'replace');
    });

    params = new URLSearchParams(testLocation.search);
    expect(history.index).toBe(2);
    expect(params.get('testKey1')).toBe('newTestValue');
    expect(params.get('testKey2')).toBe('newValue3');
    expect(result.current.hook1[0]).toBe('newTestValue');
    expect(result.current.hook2[0]).toBe('newValue3');

    // A mix of 'push' and 'replace' actions should result in a single history entry
    actAndRunTicks(() => {
        const [, setParam1] = result.current.hook1;
        const [, setParam2] = result.current.hook2;
        setParam1('newValue4', 'replace');
        setParam2('newValue5');
    });

    params = new URLSearchParams(testLocation.search);
    expect(history.index).toBe(3);
    expect(params.get('testKey1')).toBe('newValue4');
    expect(params.get('testKey2')).toBe('newValue5');
    expect(result.current.hook1[0]).toBe('newValue4');
    expect(result.current.hook2[0]).toBe('newValue5');
});
