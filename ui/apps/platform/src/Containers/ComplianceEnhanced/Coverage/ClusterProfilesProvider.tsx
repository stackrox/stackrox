import React, { createContext, useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';

import {
    getComplianceProfilesStats,
    ListComplianceProfileScanStatsResponse,
} from 'services/ComplianceResultsStatsService';

type ClusterProfileData = {
    clusterProfileData: ListComplianceProfileScanStatsResponse;
    isLoading: boolean;
    error: Error | undefined;
};

const defaultProfileStats: ListComplianceProfileScanStatsResponse = {
    scanStats: [],
    totalCount: 0,
};

const defaultContextValue: ClusterProfileData = {
    clusterProfileData: defaultProfileStats,
    isLoading: true,
    error: undefined,
};

export const ClusterProfilesContext = createContext<ClusterProfileData>(defaultContextValue);

export type ClusterProfilesProviderProps = {
    clusterId: string;
    children: React.ReactNode;
};

function ClusterProfilesProvider({ clusterId, children }: ClusterProfilesProviderProps) {
    const fetchProfilesStats = useCallback(
        () => getComplianceProfilesStats(clusterId),
        [clusterId]
    );
    const {
        data: clusterProfileData,
        loading: isLoading,
        error,
    } = useRestQuery(fetchProfilesStats);

    const contextValue: ClusterProfileData = {
        clusterProfileData: clusterProfileData ?? defaultProfileStats,
        isLoading,
        error,
    };

    return (
        <ClusterProfilesContext.Provider value={contextValue}>
            {children}
        </ClusterProfilesContext.Provider>
    );
}

export default ClusterProfilesProvider;
