import React, { useMemo } from 'react';
import type { ReactNode } from 'react';

import { PublicConfigContext } from 'hooks/usePublicConfig';
import useRestQuery from 'hooks/useRestQuery';
import { fetchPublicConfig } from 'services/SystemConfigService';
import type { PublicConfig } from 'types/config.proto';

export type PublicConfigContextType = {
    isLoadingPublicConfig: boolean;
    error: Error | undefined;
    publicConfig: PublicConfig | undefined;
    refetchPublicConfig: () => void;
};

export function PublicConfigProvider({ children }: { children: ReactNode }) {
    const {
        data,
        isLoading: isLoadingPublicConfig,
        error,
        refetch,
    } = useRestQuery(fetchPublicConfig);

    const value: PublicConfigContextType = useMemo(
        () => ({
            isLoadingPublicConfig,
            error,
            publicConfig: data,
            refetchPublicConfig: refetch,
        }),
        [data, isLoadingPublicConfig, error, refetch]
    );

    return <PublicConfigContext.Provider value={value}>{children}</PublicConfigContext.Provider>;
}
