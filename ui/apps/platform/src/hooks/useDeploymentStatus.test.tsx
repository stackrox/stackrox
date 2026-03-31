import { MemoryRouter, useLocation } from 'react-router-dom-v5-compat';
import { renderHook } from '@testing-library/react';
import actAndFlushTaskQueue from 'test-utils/flushTaskQueue';

import useDeploymentStatus from './useDeploymentStatus';

test('defaults to DEPLOYED when no URL param is set', async () => {
    let testLocation;
    const { result } = renderHook(
        () => {
            testLocation = useLocation();
            return useDeploymentStatus();
        },
        {
            wrapper: ({ children }) => (
                <MemoryRouter initialEntries={['']}>{children}</MemoryRouter>
            ),
        }
    );

    await actAndFlushTaskQueue(() => {});

    expect(result.current).toBe('DEPLOYED');
    const params = new URLSearchParams(testLocation.search);
    expect(params.get('deploymentStatus')).toBe('DEPLOYED');
});

test('reflects DELETED when deploymentStatus=DELETED is in the URL', async () => {
    const { result } = renderHook(() => useDeploymentStatus(), {
        wrapper: ({ children }) => (
            <MemoryRouter initialEntries={['?deploymentStatus=DELETED']}>{children}</MemoryRouter>
        ),
    });

    await actAndFlushTaskQueue(() => {});

    expect(result.current).toBe('DELETED');
});

test('falls back to DEPLOYED for an invalid URL param value', async () => {
    const { result } = renderHook(() => useDeploymentStatus(), {
        wrapper: ({ children }) => (
            <MemoryRouter initialEntries={['?deploymentStatus=BOGUS']}>{children}</MemoryRouter>
        ),
    });

    await actAndFlushTaskQueue(() => {});

    expect(result.current).toBe('DEPLOYED');
});
