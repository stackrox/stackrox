import React, { useMemo, useRef } from 'react';
import isEqual from 'lodash/isEqual';

import { MetadataContext } from 'hooks/useMetadata';
import useRestQuery from 'hooks/useRestQuery';
import useInterval from 'hooks/useInterval';
import { fetchMetadata } from 'services/MetadataService';
import type { Metadata } from 'types/metadataService.proto';

export type MetadataContextType = {
    isLoadingMetadata: boolean;
    error: Error | undefined;
    metadata: Metadata;
    isOutdatedVersion: boolean;
    refetchMetadata: () => void;
};

// Initial state arbitrarily assumes release build
const metadataInitialState: Metadata = {
    buildFlavor: 'release',
    licenseStatus: 'VALID',
    releaseBuild: true,
    version: '', // response for request before authentication does not reveal version
};

export function MetadataProvider({ children }: { children: React.ReactNode }) {
    const { data, isLoading, error, refetch } = useRestQuery(fetchMetadata);

    const stableMetadataRef = useRef<Metadata>(metadataInitialState);
    const stableErrorRef = useRef<Error | undefined>();

    const currentData = data ?? metadataInitialState;

    // Track version changes for outdated version detection
    let isOutdatedVersion = false;
    if (data?.version && !stableMetadataRef.current.version) {
        // The version is not outdated if it's the first time we see it
        stableMetadataRef.current.version = data.version;
    } else if (data?.version) {
        isOutdatedVersion = data.version !== stableMetadataRef.current.version;
    }

    // Only update the metadata reference if the data has actually changed
    if (!isEqual(stableMetadataRef.current, currentData)) {
        stableMetadataRef.current = currentData;
    }

    if (!isEqual(stableErrorRef.current, error)) {
        stableErrorRef.current = error;
    }

    const isLoadingMetadata = isLoading && !data;

    // Always return the stable reference to prevent unnecessary re-renders
    const stableMetadata = stableMetadataRef.current;
    const stableError = stableErrorRef.current;

    // Poll metadata every 10 seconds (or 1 second if we don't have metadata yet)
    useInterval(refetch, data?.version ? 10000 : 1000);

    const value: MetadataContextType = useMemo(
        () => ({
            isLoadingMetadata,
            error: stableError,
            metadata: stableMetadata,
            isOutdatedVersion,
            refetchMetadata: refetch,
        }),
        [stableMetadata, stableError, isLoadingMetadata, isOutdatedVersion, refetch]
    );

    return <MetadataContext.Provider value={value}>{children}</MetadataContext.Provider>;
}
