import React, { createContext, useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';

import {
    getComplianceProfilesStats,
    ListComplianceProfileScanStatsResponse,
} from 'services/ComplianceResultsService';

type ComplianceProfilesContextValue = {
    profileScanStats: ListComplianceProfileScanStatsResponse | undefined;
    isLoading: boolean;
    error: Error | undefined;
};

export const ComplianceProfilesContext = createContext<ComplianceProfilesContextValue | null>(null);

function ComplianceProfilesProvider({ children }: { children: React.ReactNode }) {
    const fetchProfilesStats = useCallback(() => getComplianceProfilesStats(), []);
    const { data: profileScanStats, loading: isLoading, error } = useRestQuery(fetchProfilesStats);

    const contextValue: ComplianceProfilesContextValue = {
        profileScanStats,
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
