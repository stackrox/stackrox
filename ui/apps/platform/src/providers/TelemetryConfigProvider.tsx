import React, { useEffect, useMemo } from 'react';
import type { ReactNode } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import usePublicConfig from 'hooks/usePublicConfig';
import { TelemetryConfigContext } from 'hooks/useTelemetryConfig';
import { initializeAnalytics } from 'init/initializeAnalytics';
import { fetchTelemetryConfig } from 'services/TelemetryConfigService';
import type { TelemetryConfig } from 'services/TelemetryConfigService';

export type TelemetryConfigContextType = {
    telemetryConfig: TelemetryConfig | undefined;
    isLoadingTelemetryConfig: boolean;
    errorTelemetryConfig: Error | undefined;
    isTelemetryConfigured: boolean;
};

export function TelemetryConfigProvider({ children }: { children: ReactNode }) {
    const { publicConfig, isLoadingPublicConfig } = usePublicConfig();

    const telemetryConfigFetcher = useMemo(() => {
        if (publicConfig && !isLoadingPublicConfig) {
            return fetchTelemetryConfig;
        }
        return () =>
            new Promise<Awaited<ReturnType<typeof fetchTelemetryConfig>> | undefined>((resolve) =>
                resolve(undefined)
            );
    }, [publicConfig, isLoadingPublicConfig]);

    const { data, isLoading, error } = useRestQuery(telemetryConfigFetcher);

    const value: TelemetryConfigContextType = useMemo(
        () => ({
            telemetryConfig: data,
            isLoadingTelemetryConfig: isLoading,
            errorTelemetryConfig: error,
            isTelemetryConfigured: !isLoading && data !== undefined,
        }),
        [data, isLoading, error]
    );

    useEffect(() => {
        if (publicConfig?.telemetry?.enabled && data) {
            initializeAnalytics(data.storageKeyV1, data.endpoint, data.userId);
        }
    }, [data, publicConfig?.telemetry?.enabled]);

    return (
        <TelemetryConfigContext.Provider value={value}>{children}</TelemetryConfigContext.Provider>
    );
}
