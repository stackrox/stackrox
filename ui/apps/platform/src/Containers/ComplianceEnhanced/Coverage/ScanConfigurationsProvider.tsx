import React, { createContext, useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';

import {
    listComplianceScanConfigurations,
    ListComplianceScanConfigurationsResponse,
} from 'services/ComplianceScanConfigurationService';

type ScanConfigurationsContextValue = {
    scanConfigurationsResponse: ListComplianceScanConfigurationsResponse;
    isLoading: boolean;
    error: Error | undefined;
};

const defaultResponse: ListComplianceScanConfigurationsResponse = {
    configurations: [],
    totalCount: 0,
};

const defaultContextValue: ScanConfigurationsContextValue = {
    scanConfigurationsResponse: defaultResponse,
    isLoading: true,
    error: undefined,
};

export const ScanConfigurationsContext =
    createContext<ScanConfigurationsContextValue>(defaultContextValue);

function ScanConfigurationsProvider({ children }: { children: React.ReactNode }) {
    const fetchScanConfigurations = useCallback(() => listComplianceScanConfigurations(), []);
    const {
        data: scanConfigurationsResponse,
        loading: isLoading,
        error,
    } = useRestQuery(fetchScanConfigurations);

    const contextValue: ScanConfigurationsContextValue = {
        scanConfigurationsResponse: scanConfigurationsResponse ?? defaultResponse,
        isLoading,
        error,
    };

    return (
        <ScanConfigurationsContext.Provider value={contextValue}>
            {children}
        </ScanConfigurationsContext.Provider>
    );
}

export default ScanConfigurationsProvider;
