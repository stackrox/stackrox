import { useCallback, useMemo } from 'react';
import type { ReactNode } from 'react';

import { LightspeedStatusContext } from 'hooks/useLightspeedStatus';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useRestQuery from 'hooks/useRestQuery';
import { fetchLightspeedStatus } from 'services/MetadataService';

export type LightspeedStatusProviderProps = {
    children: ReactNode;
};

export type LightspeedStatusContextType = {
    isAvailable: boolean;
    isLoading: boolean;
    error: Error | undefined;
};

export function LightspeedStatusProvider({ children }: LightspeedStatusProviderProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isRiskSummaryEnabled = isFeatureFlagEnabled('ROX_LIGHTSPEED_RISK_SUMMARY');

    const { data, isLoading, error } = useRestQuery(
        useCallback(
            () =>
                isRiskSummaryEnabled
                    ? fetchLightspeedStatus()
                    : Promise.resolve({ available: false, message: 'Feature disabled' }),
            [isRiskSummaryEnabled]
        )
    );

    const value: LightspeedStatusContextType = useMemo(
        () => ({
            isAvailable: data?.available ?? false,
            isLoading,
            error,
        }),
        [data?.available, isLoading, error]
    );

    return (
        <LightspeedStatusContext.Provider value={value}>
            {children}
        </LightspeedStatusContext.Provider>
    );
}
