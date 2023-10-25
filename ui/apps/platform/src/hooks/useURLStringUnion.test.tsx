import React, { ReactNode } from 'react';
import { MemoryRouter, Route, RouteComponentProps } from 'react-router-dom';
import { renderHook, act } from '@testing-library/react';

import { URLSearchParams } from 'url';
import useURLStringUnion from './useURLStringUnion';

type WrapperProps = {
    children: ReactNode;
    onRouteRender: (renderResult: RouteComponentProps) => void;
    initialEntries: string[];
};

// This Wrapper component allows the hook to simulate the browser's
// URL bar in JSDom via the MemoryRouter
function Wrapper({ children, onRouteRender, initialEntries = [] }: WrapperProps) {
    return (
        <MemoryRouter
            initialEntries={initialEntries}
            initialIndex={Math.max(0, initialEntries.length - 1)}
        >
            <Route path="*" render={onRouteRender} />
            {children}
        </MemoryRouter>
    );
}

test('should read/write only the specified set of strings to the URL parameter', async () => {
    let params;
    let testLocation;

    const possibleUrlValues = ['Alpha', 'Beta', 'Delta'] as const;

    const { result } = renderHook(() => useURLStringUnion('urlKey', possibleUrlValues), {
        initialProps: {
            children: [],
            onRouteRender: ({ location }) => {
                testLocation = location;
            },
            initialEntries: [''],
        },
        wrapper: Wrapper,
    });

    // Check that default value is applied correctly
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0]).toBe('Alpha');
    expect(params.get('urlKey')).toBe('Alpha');

    // Check that setting the value changes the parameter
    act(() => {
        const [, setParam] = result.current;
        setParam('Delta');
    });
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0]).toBe('Delta');
    expect(params.get('urlKey')).toBe('Delta');

    // Check that passing an invalid value does not update the parameter
    const invalidValues = [
        'Omega',
        '',
        'alpha',
        0,
        Infinity,
        { test: 'Object' },
        new Error('Test error'),
        null,
        undefined,
    ];

    invalidValues.forEach((invalid) => {
        act(() => {
            const [, setParam] = result.current;
            setParam(invalid);
        });
        params = new URLSearchParams(testLocation.search);
        expect(result.current[0]).toBe('Delta');
        expect(params.get('urlKey')).toBe('Delta');
    });

    // Check setting a valid value after invalid attempts correctly sets the new value
    act(() => {
        const [, setParam] = result.current;
        setParam('Beta');
    });
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0]).toBe('Beta');
    expect(params.get('urlKey')).toBe('Beta');
});

test('should default to the current URL parameter value on initialization, if it is valid', async () => {
    let testLocation;

    const possibleUrlValues = ['Alpha', 'Beta', 'Delta'] as const;

    const { result: initialValidResult } = renderHook(
        () => useURLStringUnion('urlKey', possibleUrlValues),
        {
            initialProps: {
                children: [],
                onRouteRender: ({ location }) => {
                    testLocation = location;
                },
                initialEntries: ['?urlKey=Beta'],
            },
            wrapper: Wrapper,
        }
    );

    // Check that default value is not applied if the URL param already contains a valid value
    const params = new URLSearchParams(testLocation.search);
    expect(initialValidResult.current[0]).toBe('Beta');
    expect(params.get('urlKey')).toBe('Beta');
});

test('should use the default value when an invalid value is entered directly into the URL', async () => {
    let testLocation;

    const possibleUrlValues = ['Alpha', 'Beta', 'Delta'] as const;
    const { result: initialInvalidResult } = renderHook(
        () => useURLStringUnion('urlKey', possibleUrlValues),
        {
            initialProps: {
                children: [],
                onRouteRender: ({ location }) => {
                    testLocation = location;
                },
                initialEntries: ['?urlKey=Bogus'],
            },
            wrapper: Wrapper,
        }
    );

    // Check that default value is applied correctly when the URL param is invalid
    const params = new URLSearchParams(testLocation.search);
    expect(initialInvalidResult.current[0]).toBe('Alpha');
    expect(params.get('urlKey')).toBe('Alpha');
});
