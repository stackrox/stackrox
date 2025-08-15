import React, { useMemo } from 'react';
import type { ReactNode } from 'react';

import { FeatureFlagsContext } from 'hooks/useFeatureFlags';
import type { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import useRestQuery from 'hooks/useRestQuery';
import { fetchFeatureFlags } from 'services/FeatureFlagsService';
import type { FeatureFlagEnvVar } from 'types/featureFlag';

export type FeatureFlagsProviderProps = {
    children: ReactNode;
};

export type FeatureFlagsContextType = {
    isLoadingFeatureFlags: boolean;
    error: Error | undefined;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
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
