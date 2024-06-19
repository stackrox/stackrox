import React, { createContext, useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';

import {
    listComplianceScanConfigProfiles,
    ListComplianceScanConfigsProfileResponse,
} from 'services/ComplianceScanConfigurationService';

type ComplianceProfilesContextValue = {
    scanConfigProfilesResponse: ListComplianceScanConfigsProfileResponse;
    isLoading: boolean;
    error: Error | undefined;
};

const defaultProfilesResponse: ListComplianceScanConfigsProfileResponse = {
    profiles: [],
    totalCount: 0,
};

const defaultContextValue: ComplianceProfilesContextValue = {
    scanConfigProfilesResponse: defaultProfilesResponse,
    isLoading: true,
    error: undefined,
};

export const ComplianceProfilesContext =
    createContext<ComplianceProfilesContextValue>(defaultContextValue);

function ComplianceProfilesProvider({ children }: { children: React.ReactNode }) {
    const fetchProfiles = useCallback(() => listComplianceScanConfigProfiles(), []);
    const {
        data: scanConfigProfilesResponse,
        loading: isLoading,
        error,
    } = useRestQuery(fetchProfiles);

    const contextValue: ComplianceProfilesContextValue = {
        scanConfigProfilesResponse: scanConfigProfilesResponse ?? defaultProfilesResponse,
        isLoading,
        error,
    };

    return (
        <ComplianceProfilesContext.Provider value={contextValue}>
            {children}
        </ComplianceProfilesContext.Provider>
    );
}

export default ComplianceProfilesProvider;
