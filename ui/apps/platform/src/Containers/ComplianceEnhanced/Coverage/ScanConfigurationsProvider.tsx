import React, { createContext, useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import {
    listComplianceScanConfigurations,
    ListComplianceScanConfigurationsResponse,
} from 'services/ComplianceScanConfigurationService';
import useURLParameter, { HistoryAction, QueryValue } from 'hooks/useURLParameter';

type ScanConfigurationsContextValue = {
    scanConfigurationsQuery: {
        response: ListComplianceScanConfigurationsResponse;
        isLoading: boolean;
        error: Error | undefined;
    };
    selectedScanConfig: QueryValue;
    setSelectedScanConfig: (
        scanConfigName: QueryValue,
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
    selectedScanConfig: undefined,
    setSelectedScanConfig: () => {},
};

export const ScanConfigurationsContext =
    createContext<ScanConfigurationsContextValue>(defaultContextValue);

function ScanConfigurationsProvider({ children }: { children: React.ReactNode }) {
    const [selectedScanConfig, setSelectedScanConfig] = useURLParameter('scanSchedule', undefined);

    const fetchScanConfigurations = useCallback(() => listComplianceScanConfigurations(), []);
    const {
        data: scanConfigurationsResponse,
        loading: isLoading,
        error,
    } = useRestQuery(fetchScanConfigurations);

    const contextValue: ScanConfigurationsContextValue = {
        scanConfigurationsQuery: {
            response: scanConfigurationsResponse ?? defaultResponse,
            isLoading,
            error,
        },
        selectedScanConfig,
        setSelectedScanConfig,
    };

    return (
        <ScanConfigurationsContext.Provider value={contextValue}>
            {children}
        </ScanConfigurationsContext.Provider>
    );
}

export default ScanConfigurationsProvider;
