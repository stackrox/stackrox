import React, { createContext, useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import {
    listComplianceScanConfigurations,
    ListComplianceScanConfigurationsResponse,
} from 'services/ComplianceScanConfigurationService';
import useURLParameter, { HistoryAction } from 'hooks/useURLParameter';

type ScanConfigurationsContextValue = {
    scanConfigurationsQuery: {
        response: ListComplianceScanConfigurationsResponse;
        isLoading: boolean;
        error: Error | undefined;
    };
    selectedScanConfigName: string | undefined;
    setSelectedScanConfigName: (
        scanConfigName: string | undefined,
        historyAction?: HistoryAction | undefined
    ) => void;
};

const defaultResponse: ListComplianceScanConfigurationsResponse = {
    configurations: [],
    totalCount: 0,
};

const defaultContextValue: ScanConfigurationsContextValue = {
    scanConfigurationsQuery: {
        response: defaultResponse,
        isLoading: true,
        error: undefined,
    },
    selectedScanConfigName: undefined,
    setSelectedScanConfigName: () => {},
};

export const ScanConfigurationsContext =
    createContext<ScanConfigurationsContextValue>(defaultContextValue);

function ScanConfigurationsProvider({ children }: { children: React.ReactNode }) {
    const [selectedScanConfigName, setSelectedScanConfigName] = useURLParameter(
        'scanSchedule',
        undefined
    );

    const fetchScanConfigurations = useCallback(() => listComplianceScanConfigurations(), []);
    const {
        data: scanConfigurationsResponse,
        loading: isLoading,
        error,
    } = useRestQuery(fetchScanConfigurations);

    const selectedScanConfigNameString =
        typeof selectedScanConfigName === 'string' ? selectedScanConfigName : undefined;

    const wrappedSetSelectedScanConfig = (
        scanConfigName: string | undefined,
        historyAction?: HistoryAction | undefined
    ) => {
        setSelectedScanConfigName(scanConfigName, historyAction);
    };

    const effectiveScanConfigurationsResponse = scanConfigurationsResponse ?? defaultResponse;

    const { configurations } = effectiveScanConfigurationsResponse;

    const sortedScanConfigurations = configurations.sort((a, b) =>
        a.scanName.localeCompare(b.scanName)
    );

    const contextValue: ScanConfigurationsContextValue = {
        scanConfigurationsQuery: {
            response: {
                configurations: sortedScanConfigurations,
                totalCount: effectiveScanConfigurationsResponse.totalCount,
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
