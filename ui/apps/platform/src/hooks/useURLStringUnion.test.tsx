import React from 'react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { renderHook } from '@testing-library/react';
import actAndFlushTaskQueue from 'test-utils/flushTaskQueue';

import { URLSearchParams } from 'url';
import useURLStringUnion from './useURLStringUnion';

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

    await actAndFlushTaskQueue(() => {});

    // Check that default value is applied correctly
    params = new URLSearchParams(testLocation.search);
    expect(result.current[0]).toBe('Alpha');
    expect(params.get('urlKey')).toBe('Alpha');

    // Check that setting the value changes the parameter
    await actAndFlushTaskQueue(() => {
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

    for (let i = 0; i < invalidValues.length; i += 1) {
        // eslint-disable-next-line no-await-in-loop
        await actAndFlushTaskQueue(() => {
            const [, setParam] = result.current;
            setParam(invalidValues[i]);
        });
        params = new URLSearchParams(testLocation.search);
        expect(result.current[0]).toBe('Delta');
        expect(params.get('urlKey')).toBe('Delta');
    }

    // Check setting a valid value after invalid attempts correctly sets the new value
    await actAndFlushTaskQueue(() => {
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

    await actAndFlushTaskQueue(() => {});

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

    await actAndFlushTaskQueue(() => {});

    // Check that default value is applied correctly when the URL param is invalid
    const params = new URLSearchParams(testLocation.search);
    expect(initialInvalidResult.current[0]).toBe('Alpha');
    expect(params.get('urlKey')).toBe('Alpha');
});
