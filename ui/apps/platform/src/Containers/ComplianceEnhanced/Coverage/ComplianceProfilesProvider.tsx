import React, { createContext, useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';

import {
    getComplianceProfilesStats,
    ListComplianceProfileScanStatsResponse,
} from 'services/ComplianceResultsStatsService';

type ComplianceProfilesContextValue = {
    profileScanStats: ListComplianceProfileScanStatsResponse;
    isLoading: boolean;
    error: Error | undefined;
};

const defaultProfileStats: ListComplianceProfileScanStatsResponse = {
    scanStats: [],
    totalCount: 0,
};

const defaultContextValue: ComplianceProfilesContextValue = {
    profileScanStats: defaultProfileStats,
    isLoading: true,
    error: undefined,
};

export const ComplianceProfilesContext =
    createContext<ComplianceProfilesContextValue>(defaultContextValue);

function ComplianceProfilesProvider({ children }: { children: React.ReactNode }) {
    const fetchProfilesStats = useCallback(() => getComplianceProfilesStats(), []);
    const { data: profileScanStats, loading: isLoading, error } = useRestQuery(fetchProfilesStats);

    const contextValue: ComplianceProfilesContextValue = {
        profileScanStats: profileScanStats ?? defaultProfileStats,
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
