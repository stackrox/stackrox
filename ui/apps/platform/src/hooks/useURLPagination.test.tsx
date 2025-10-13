import React from 'react';
import type { ReactNode } from 'react';
// import { createMemoryHistory } from 'history';
import { Route, MemoryRouter as Router } from 'react-router-dom';
import type { RouteComponentProps } from 'react-router-dom';
// import { HistoryRouter as Router } from 'redux-first-history/rr6';
import { CompatRouter, MemoryRouter, useLocation } from 'react-router-dom-v5-compat';
import { renderHook } from '@testing-library/react';

import actAndFlushTaskQueue from 'test-utils/flushTaskQueue';
import { URLSearchParams } from 'url';
import useURLPagination from './useURLPagination';

type WrapperProps = {
    children: ReactNode;
    onRouteRender: (renderResult: RouteComponentProps) => void;
    initialEntries: string[];
};

function Wrapper({ children, onRouteRender, initialEntries = [] }: WrapperProps) {
    return (
        <Router
            initialEntries={initialEntries}
            initialIndex={Math.max(0, initialEntries.length - 1)}
        >
            <CompatRouter>
                <Route path="*" render={onRouteRender} />
                {children}
            </CompatRouter>
        </Router>
    );
}

const createWrapper = (props) => {
    return function CreatedWrapper({ children }) {
        return <Wrapper {...props}>{children}</Wrapper>;
    };
};

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
    await actAndFlushTaskQueue(() => {
        const { setPage } = result.current;
        setPage(2);
    });
    params = new URLSearchParams(testLocation.search);
    expect(result.current.page).toBe(2);
    expect(result.current.perPage).toBe(10);
    expect(params.get('page')).toBe('2');
    expect(params.get('perPage')).toBe(null);

    // Check that updating the perPage parameter also resets the page parameter
    await actAndFlushTaskQueue(() => {
        const { setPerPage } = result.current;
        setPerPage(20);
    });
    params = new URLSearchParams(testLocation.search);
    expect(result.current.page).toBe(1);
    expect(result.current.perPage).toBe(20);
    expect(params.get('page')).toBe(null);
    expect(params.get('perPage')).toBe('20');
});

test('should not add history states when setting values with a "replace" action', async () => {
    let params;
    let historyLength;
    let testLocation;

    const { result } = renderHook(() => useURLPagination(10), {
        wrapper: createWrapper({
            children: [],
            onRouteRender: ({ location, history }) => {
                testLocation = location;
                historyLength = history.length;
            },
            initialEntries: [''],
        }),
    });
    // Check the length of the initial history stack
    params = new URLSearchParams(testLocation.search);
    expect(historyLength).toBe(1);
    expect(params.get('page')).toBe(null);
    expect(params.get('perPage')).toBe(null);

    // Update the page parameter with a 'replace' action
    await actAndFlushTaskQueue(() => {
        const { setPage } = result.current;
        setPage(2, 'replace');
    });

    // Check the length of the history stack after updating the page parameter
    params = new URLSearchParams(testLocation.search);
    expect(historyLength).toBe(1);
    expect(params.get('page')).toBe('2');
    expect(params.get('perPage')).toBe(null);

    // Update the perPage parameter with a 'replace' action
    await actAndFlushTaskQueue(() => {
        const { setPerPage } = result.current;
        setPerPage(20, 'replace');
    });

    // Check the length of the history stack after updating the perPage parameter
    params = new URLSearchParams(testLocation.search);
    expect(historyLength).toBe(1);
    expect(params.get('page')).toBe(null);
    expect(params.get('perPage')).toBe('20');
});

test('should only add a single history state when setting perPage without an action parameter', async () => {
    let params;
    let historyLength;
    let testLocation;

    const { result } = renderHook(() => useURLPagination(10), {
        wrapper: createWrapper({
            children: [],
            onRouteRender: ({ location, history }) => {
                testLocation = location;
                historyLength = history.length;
            },
            initialEntries: [''],
        }),
    });

    // Check the length of the initial history stack
    params = new URLSearchParams(testLocation.search);
    expect(historyLength).toBe(1);
    expect(params.get('page')).toBe(null);
    expect(params.get('perPage')).toBe(null);

    // Update the page parameter
    await actAndFlushTaskQueue(() => {
        const { setPage } = result.current;
        setPage(2);
    });

    // Check the length of the history stack after updating the page parameter
    params = new URLSearchParams(testLocation.search);
    expect(historyLength).toBe(2);
    expect(params.get('page')).toBe('2');
    expect(params.get('perPage')).toBe(null);

    // Update the perPage parameter and check the length of the history stack
    await actAndFlushTaskQueue(() => {
        const { setPerPage } = result.current;
        setPerPage(20);
    });

    // Check the length of the history stack after updating the perPage parameter
    params = new URLSearchParams(testLocation.search);
    expect(historyLength).toBe(3);
    expect(params.get('page')).toBe(null);
    expect(params.get('perPage')).toBe('20');
});
