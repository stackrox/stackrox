import { createContext, useContext } from 'react';

import type { TelemetryConfigContextType } from 'providers/TelemetryConfigProvider';
import type { TelemetryConfig } from 'services/TelemetryConfigService';

export const TelemetryConfigContext = createContext<TelemetryConfigContextType | undefined>(
    undefined
);

type UseTelemetryConfigResult = {
    telemetryConfig: TelemetryConfig | undefined;
    isLoadingTelemetryConfig: boolean;
    errorTelemetryConfig: Error | undefined;
    isTelemetryConfigured: boolean;
};

function useTelemetryConfig(): UseTelemetryConfigResult {
    const context = useContext(TelemetryConfigContext);
    if (context === undefined) {
        throw new Error('useTelemetryConfig must be used within a TelemetryConfigProvider');
    }

    return {
        telemetryConfig: context.telemetryConfig,
        isLoadingTelemetryConfig: context.isLoadingTelemetryConfig,
        errorTelemetryConfig: context.errorTelemetryConfig,
        isTelemetryConfigured: context.isTelemetryConfigured,
    };
}

export default useTelemetryConfig;
