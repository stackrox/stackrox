import { createContext, useCallback } from 'react';
import type { ReactNode } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import { listComplianceScanConfigOverviews } from 'services/ComplianceScanConfigurationService';
import type {
    ComplianceScanConfigOverview,
    ListComplianceScanConfigOverviewsResponse,
} from 'services/ComplianceScanConfigurationService';
import useURLParameter from 'hooks/useURLParameter';
import type { HistoryAction } from 'hooks/useURLParameter';

type ScanConfigurationsContextValue = {
    scanConfigOverviewsQuery: {
        response: ListComplianceScanConfigOverviewsResponse;
        isLoading: boolean;
        error: Error | undefined;
    };
    selectedScanConfigName: string | undefined;
    setSelectedScanConfigName: (
        scanConfigName: string | undefined,
        historyAction?: HistoryAction
    ) => void;
};

const defaultResponse: ListComplianceScanConfigOverviewsResponse = {
    configs: [],
    totalCount: 0,
};

const defaultContextValue: ScanConfigurationsContextValue = {
    scanConfigOverviewsQuery: {
        response: defaultResponse,
        isLoading: true,
        error: undefined,
    },
    selectedScanConfigName: undefined,
    setSelectedScanConfigName: () => {},
};

export const ScanConfigurationsContext =
    createContext<ScanConfigurationsContextValue>(defaultContextValue);

export type { ComplianceScanConfigOverview };

function ScanConfigurationsProvider({ children }: { children: ReactNode }) {
    const [selectedScanConfigName, setSelectedScanConfigName] = useURLParameter(
        'scanSchedule',
        undefined
    );

    const fetchOverviews = useCallback(() => listComplianceScanConfigOverviews(), []);
    const { data: overviewsResponse, isLoading, error } = useRestQuery(fetchOverviews);

    const selectedScanConfigNameString =
        typeof selectedScanConfigName === 'string' ? selectedScanConfigName : undefined;

    const wrappedSetSelectedScanConfig = (
        scanConfigName: string | undefined,
        historyAction?: HistoryAction
    ) => {
        setSelectedScanConfigName(scanConfigName, historyAction);
    };

    const effectiveResponse = overviewsResponse ?? defaultResponse;

    const sortedConfigs = [...effectiveResponse.configs].sort((a, b) =>
        a.scanConfigName.localeCompare(b.scanConfigName)
    );

    const contextValue: ScanConfigurationsContextValue = {
        scanConfigOverviewsQuery: {
            response: {
                configs: sortedConfigs,
                totalCount: effectiveResponse.totalCount,
            },
            isLoading,
            error,
        },
        selectedScanConfigName: selectedScanConfigNameString,
        setSelectedScanConfigName: wrappedSetSelectedScanConfig,
    };

    return (
        <ScanConfigurationsContext.Provider value={contextValue}>
            {children}
        </ScanConfigurationsContext.Provider>
    );
}

export default ScanConfigurationsProvider;
