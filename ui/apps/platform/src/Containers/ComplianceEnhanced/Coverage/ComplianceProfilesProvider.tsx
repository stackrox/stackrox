import React, { createContext, useCallback } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';

import {
    getComplianceProfilesStats,
    ListComplianceProfileScanStatsResponse,
} from 'services/ComplianceResultsService';

const defaultProfileStats: ListComplianceProfileScanStatsResponse = {
    scanStats: [],
    totalCount: 0,
};

export const ComplianceProfilesContext =
    createContext<ListComplianceProfileScanStatsResponse>(defaultProfileStats);

function ComplianceProfilesProvider({ children }: { children: React.ReactNode }) {
    const fetchProfilesStats = useCallback(() => getComplianceProfilesStats(), []);
    const { data: profileScanStats, loading: isLoading, error } = useRestQuery(fetchProfilesStats);

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    }

    if (error) {
        return <div>Error: {error.message}</div>;
    }

    if (profileScanStats?.scanStats.length === 0) {
        return <div>No results to display</div>;
    }

    const contextValue = profileScanStats ?? defaultProfileStats;

    return (
        <ComplianceProfilesContext.Provider value={contextValue}>
            {children}
        </ComplianceProfilesContext.Provider>
    );
}

export default ComplianceProfilesProvider;
