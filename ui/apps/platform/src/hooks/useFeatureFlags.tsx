import React, { createContext, useContext, ReactNode } from 'react';

import type { FeatureFlagEnvVar } from 'types/featureFlag';
import type { FeatureFlag } from 'types/featureFlagService.proto';
import { fetchFeatureFlags } from 'services/FeatureFlagsService';
import useRestQuery from './useRestQuery';

export type IsFeatureFlagEnabled = (envVar: FeatureFlagEnvVar) => boolean;

type FeatureFlagsContextType = {
    featureFlags: FeatureFlag[];
    isLoadingFeatureFlags: boolean;
    error: Error | null;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

const FeatureFlagsContext = createContext<FeatureFlagsContextType | undefined>(undefined);

type FeatureFlagsProviderProps = {
    children: ReactNode;
};

export function FeatureFlagsProvider({ children }: FeatureFlagsProviderProps) {
    const { data, isLoading: isLoadingFeatureFlags, error } = useRestQuery(fetchFeatureFlags);

    const featureFlags = data?.response.featureFlags || [];

    function isFeatureFlagEnabled(envVar: FeatureFlagEnvVar): boolean {
        const featureFlag = featureFlags.find((flag) => flag.envVar === envVar);
        if (!featureFlag) {
            if (process.env.NODE_ENV === 'development') {
                // eslint-disable-next-line no-console
                console.warn(`EnvVar ${envVar} not found in the backend list, possibly stale?`);
            }
            return false;
        }
        return featureFlag.enabled;
    }

    const value: FeatureFlagsContextType = {
        featureFlags,
        isLoadingFeatureFlags,
        error: error || null,
        isFeatureFlagEnabled,
    };

    return <FeatureFlagsContext.Provider value={value}>{children}</FeatureFlagsContext.Provider>;
}

type UseFeatureFlagsResult = {
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
    isLoadingFeatureFlags: boolean;
    error: Error | null;
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
