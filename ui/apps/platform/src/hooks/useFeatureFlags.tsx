import { createContext, useContext } from 'react';

import type { FeatureFlagsContextType } from 'providers/FeatureFlagProvider';
import type { FeatureFlagEnvVar } from 'types/featureFlag';

export type IsFeatureFlagEnabled = (envVar: FeatureFlagEnvVar) => boolean;

export const FeatureFlagsContext = createContext<FeatureFlagsContextType | undefined>(undefined);

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
