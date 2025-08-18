import React from 'react';
import type { ReactNode } from 'react';
import { renderHook } from '@testing-library/react';

import { MetadataProvider } from 'providers/MetadataProvider';
import type { Metadata } from 'types/metadataService.proto';
import useMetadata from './useMetadata';

// Mock the hooks that use network requests and interval polling
const mockUseRestQuery = vi.hoisted(() => vi.fn());
const mockUseInterval = vi.hoisted(() => vi.fn());
vi.mock('hooks/useRestQuery', () => ({ default: mockUseRestQuery }));
vi.mock('hooks/useInterval', () => ({ default: mockUseInterval }));

describe('useMetadata hook', () => {
    const mockMetadata: Metadata = {
        version: '1.0.0',
        buildFlavor: 'release',
        releaseBuild: true,
        licenseStatus: 'VALID',
    };

    const mockRefetch = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
        // Reset to default mock
        mockUseRestQuery.mockReturnValue({
            data: mockMetadata,
            isLoading: false,
            error: undefined,
            refetch: mockRefetch,
        });
    });

    it('should return correct context values when used within MetadataProvider', () => {
        // Pre-fetch: Validate initial loading state
        mockUseRestQuery.mockReturnValue({
            data: undefined,
            isLoading: true,
            error: undefined,
            refetch: mockRefetch,
        });

        const wrapper = ({ children }: { children: ReactNode }) => (
            <MetadataProvider key="context-test">{children}</MetadataProvider>
        );

        const { result, rerender } = renderHook(() => useMetadata(), { wrapper });

        // Validate pre-fetch state: initial values and loading
        expect(result.current.isLoadingMetadata).toBe(true);
        expect(result.current.version).toBe(''); // Initial empty version
        expect(result.current.buildFlavor).toBe('release'); // Initial default
        expect(result.current.releaseBuild).toBe(true); // Initial default
        expect(result.current.licenseStatus).toBe('VALID'); // Initial default

        // Now simulate data loading completion
        mockUseRestQuery.mockReturnValue({
            data: mockMetadata,
            isLoading: false,
            error: undefined,
            refetch: mockRefetch,
        });

        rerender();

        expect(result.current).toEqual({
            // Spread properties
            version: '1.0.0',
            buildFlavor: 'release',
            releaseBuild: true,
            licenseStatus: 'VALID',
            // Context properties
            isLoadingMetadata: false,
            error: undefined,
            isOutdatedVersion: false,
            refetchMetadata: mockRefetch,
        });
    });

    it('should detect outdated version correctly', () => {
        // First render with initial version
        const wrapper = ({ children }: { children: ReactNode }) => (
            <MetadataProvider key="version-test">{children}</MetadataProvider>
        );

        const { result, rerender } = renderHook(() => useMetadata(), { wrapper });

        // Initially not outdated
        expect(result.current.isOutdatedVersion).toBe(false);

        // Update with new version
        mockUseRestQuery.mockReturnValue({
            data: { ...mockMetadata, version: '2.0.0' },
            isLoading: false,
            error: undefined,
            refetch: mockRefetch,
        });

        rerender();

        expect(result.current.isOutdatedVersion).toBe(true);
        expect(result.current.version).toBe('2.0.0');
    });
});
