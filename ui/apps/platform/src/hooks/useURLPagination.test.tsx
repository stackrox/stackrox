import React from 'react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { renderHook, act } from '@testing-library/react';

import { URLSearchParams } from 'url';
import useURLPagination from './useURLPagination';

beforeAll(() => {
    jest.useFakeTimers();
});

function actAndRunTicks(callback) {
    return act(() => {
        callback();
        jest.runAllTicks();
    });
}

test('should update pagination parameters in the URL', async () => {
    let params;
    let testLocation;

    const { result } = renderHook(
        () => {
            testLocation = useLocation();
            return useURLPagination(10);
        },
        {
            wrapper: ({ children }) => (
                <MemoryRouter initialEntries={['']}>{children}</MemoryRouter>
            ),
        }
    );

    // Check new and existing values before setter function is called
    params = new URLSearchParams(testLocation.search);
    expect(result.current.page).toBe(1);
    expect(result.current.perPage).toBe(10);
    // When default values equal the current values, the URL parameters are not set
    expect(params.get('page')).toBe(null);
    expect(params.get('perPage')).toBe(null);

    // Check new and existing values when URL parameter is set
    actAndRunTicks(() => {
        const { setPage } = result.current;
        setPage(2);
    });
    params = new URLSearchParams(testLocation.search);
    expect(result.current.page).toBe(2);
    expect(result.current.perPage).toBe(10);
    expect(params.get('page')).toBe('2');
    expect(params.get('perPage')).toBe(null);

    // Check that updating the perPage parameter also resets the page parameter to 1
    actAndRunTicks(() => {
        const { setPerPage } = result.current;
        setPerPage(20);
    });
    params = new URLSearchParams(testLocation.search);
    expect(result.current.page).toBe(1);
    expect(result.current.perPage).toBe(20);
    expect(params.get('page')).toBe('1');
    expect(params.get('perPage')).toBe('20');
});

test('should not add history states when setting values with a "replace" action', async () => {
    let params;
    let historyLength;
    let testLocation;

    const { result } = renderHook(
        () => {
            testLocation = useLocation();
            historyLength = window.history.length;
            return useURLPagination(10);
        },
        {
            wrapper: ({ children }) => (
                <MemoryRouter initialEntries={['']}>{children}</MemoryRouter>
            ),
        }
    );

    // Check the length of the initial history stack
    params = new URLSearchParams(testLocation.search);
    expect(historyLength).toBe(1);
    expect(params.get('page')).toBe(null);
    expect(params.get('perPage')).toBe(null);

    // Update the page parameter with a 'replace' action
    actAndRunTicks(() => {
        const { setPage } = result.current;
        setPage(2, 'replace');
    });

    // Check the length of the history stack after updating the page parameter
    params = new URLSearchParams(testLocation.search);
    expect(historyLength).toBe(1);
    expect(params.get('page')).toBe('2');
    expect(params.get('perPage')).toBe(null);

    // Update the perPage parameter with a 'replace' action
    actAndRunTicks(() => {
        const { setPerPage } = result.current;
        setPerPage(20, 'replace');
    });

    // Check the length of the history stack after updating the perPage parameter
    params = new URLSearchParams(testLocation.search);
    expect(historyLength).toBe(1);
    expect(params.get('page')).toBe('1');
    expect(params.get('perPage')).toBe('20');
});

test('should only add a single history state when setting perPage without an action parameter', async () => {
    let params;
    let historyLength;
    let testLocation;

    const { result } = renderHook(
        () => {
            testLocation = useLocation();
            historyLength = window.history.length;
            return useURLPagination(10);
        },
        {
            wrapper: ({ children }) => (
                <MemoryRouter initialEntries={['']}>{children}</MemoryRouter>
            ),
        }
    );

    // Check the length of the initial history stack
    params = new URLSearchParams(testLocation.search);
    expect(historyLength).toBe(1);
    expect(params.get('page')).toBe(null);
    expect(params.get('perPage')).toBe(null);

    // Update the page parameter
    actAndRunTicks(() => {
        const { setPage } = result.current;
        setPage(2);
    });

    // Check the length of the history stack after updating the page parameter
    params = new URLSearchParams(testLocation.search);
    expect(historyLength).toBe(2);
    expect(params.get('page')).toBe('2');
    expect(params.get('perPage')).toBe(null);

    // Update the perPage parameter and check the length of the history stack
    actAndRunTicks(() => {
        const { setPerPage } = result.current;
        setPerPage(20);
    });

    // Check the length of the history stack after updating the perPage parameter
    params = new URLSearchParams(testLocation.search);
    expect(historyLength).toBe(3);
    expect(params.get('page')).toBe('1');
    expect(params.get('perPage')).toBe('20');
});
