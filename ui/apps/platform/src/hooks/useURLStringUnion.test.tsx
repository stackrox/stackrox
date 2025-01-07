import React, { ReactNode } from 'react';
import { Location, MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { renderHook, act } from '@testing-library/react';

import { URLSearchParams } from 'url';
import useURLStringUnion from './useURLStringUnion';

type WrapperProps = {
    children: ReactNode;
    onRouteRender: (location: Location) => void;
    initialEntries: string[];
};

// This Wrapper component allows the hook to simulate the browser's
// URL bar in JSDom via the MemoryRouter
function Wrapper({ children, onRouteRender, initialEntries = [] }: WrapperProps) {
    const RouteWatcher = () => {
        const location = useLocation();
        onRouteRender(location);
        return null;
    };

    return (
        <MemoryRouter
            initialEntries={initialEntries}
            initialIndex={Math.max(0, initialEntries.length - 1)}
        >
            <Routes>
                <Route path="*" element={<RouteWatcher />} />
            </Routes>
            {children}
        </MemoryRouter>
    );
}

const createWrapper = (props) => {
    return function CreatedWrapper({ children }) {
        return <Wrapper {...props}>{children}</Wrapper>;
    };
};

beforeAll(() => {
    jest.useFakeTimers();
});

function actAndRunTicks(callback) {
    return act(() => {
        callback();
        jest.runAllTicks();
    });
}

test('should read/write only the specified set of strings to the URL parameter', async () => {
    let params;
    let testLocation;

    const possibleUrlValues = ['Alpha', 'Beta', 'Delta'] as const;

    const { result } = renderHook(
        () => {
            testLocation = useLocation();
            return useURLStringUnion('urlKey', possibleUrlValues);
        },
        {
            wrapper: ({ children }) => (
                <MemoryRouter initialEntries={['']}>{children}</MemoryRouter>
            ),
        }
    );

    actAndRunTicks(() => {});

    // Check that default value is applied correctly
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0]).toBe('Alpha');
    expect(params.get('urlKey')).toBe('Alpha');

    // Check that setting the value changes the parameter
    actAndRunTicks(() => {
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
        actAndRunTicks(() => {
            const [, setParam] = result.current;
            setParam(invalid);
        });
        params = new URLSearchParams(testLocation.search);
        expect(result.current[0]).toBe('Delta');
        expect(params.get('urlKey')).toBe('Delta');
    });

    // Check setting a valid value after invalid attempts correctly sets the new value
    actAndRunTicks(() => {
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
        () => {
            testLocation = useLocation();
            return useURLStringUnion('urlKey', possibleUrlValues);
        },
        {
            wrapper: ({ children }) => (
                <MemoryRouter initialEntries={['?urlKey=Beta']}>{children}</MemoryRouter>
            ),
        }
    );

    actAndRunTicks(() => {});

    // Check that default value is not applied if the URL param already contains a valid value
    const params = new URLSearchParams(testLocation.search);
    expect(initialValidResult.current[0]).toBe('Beta');
    expect(params.get('urlKey')).toBe('Beta');
});

test('should use the default value when an invalid value is entered directly into the URL', async () => {
    let testLocation;

    const possibleUrlValues = ['Alpha', 'Beta', 'Delta'] as const;

    const { result: initialInvalidResult } = renderHook(
        () => {
            testLocation = useLocation();
            return useURLStringUnion('urlKey', possibleUrlValues);
        },
        {
            wrapper: ({ children }) => (
                <MemoryRouter initialEntries={['?urlKey=Bogus']}>{children}</MemoryRouter>
            ),
        }
    );

    actAndRunTicks(() => {});

    // Check that default value is applied correctly when the URL param is invalid
    const params = new URLSearchParams(testLocation.search);
    expect(initialInvalidResult.current[0]).toBe('Alpha');
    expect(params.get('urlKey')).toBe('Alpha');
});
