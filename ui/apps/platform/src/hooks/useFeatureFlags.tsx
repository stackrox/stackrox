import React, { createContext, useContext, useMemo } from 'react';
import type { ReactNode } from 'react';

import type { FeatureFlagEnvVar } from 'types/featureFlag';
import { fetchFeatureFlags } from 'services/FeatureFlagsService';
import useRestQuery from './useRestQuery';

export type IsFeatureFlagEnabled = (envVar: FeatureFlagEnvVar) => boolean;

type FeatureFlagsContextType = {
    isLoadingFeatureFlags: boolean;
    error: Error | undefined;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

const FeatureFlagsContext = createContext<FeatureFlagsContextType | undefined>(undefined);

type FeatureFlagsProviderProps = {
    children: ReactNode;
};

export function FeatureFlagsProvider({ children }: FeatureFlagsProviderProps) {
    const { data, isLoading: isLoadingFeatureFlags, error } = useRestQuery(fetchFeatureFlags);

    const value: FeatureFlagsContextType = useMemo(
        () => ({
            isLoadingFeatureFlags,
            error,
            isFeatureFlagEnabled: (envVar: FeatureFlagEnvVar) => {
                const featureFlags = data?.featureFlags ?? [];
                const featureFlag = featureFlags.find((flag) => flag.envVar === envVar);
                if (!featureFlag) {
                    if (process.env.NODE_ENV === 'development') {
                        // eslint-disable-next-line no-console
                        console.warn(
                            `EnvVar ${envVar} not found in the backend list, possibly stale?`
                        );
                    }
                    return false;
                }
                return featureFlag.enabled;
            },
        }),
        [data?.featureFlags, isLoadingFeatureFlags, error]
    );

    return <FeatureFlagsContext.Provider value={value}>{children}</FeatureFlagsContext.Provider>;
}

type UseFeatureFlagsResult = {
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
    isLoadingFeatureFlags: boolean;
    error: Error | undefined;
};

function useFeatureFlags(): UseFeatureFlagsResult {
    const context = useContext(FeatureFlagsContext);
    if (context === undefined) {
        throw new Error('useFeatureFlags must be used within a FeatureFlagsProvider');
    }

    return {
        isFeatureFlagEnabled: context.isFeatureFlagEnabled,
        isLoadingFeatureFlags: context.isLoadingFeatureFlags,
        error: context.error,
    };
}

export default useFeatureFlags;
