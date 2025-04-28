import React, { useCallback } from 'react';

import { TimeWindow } from 'constants/timeWindows';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import useRestQuery from 'hooks/useRestQuery';
import { UseUrlSearchReturn } from 'hooks/useURLSearch';
import { getExternalIpsFlowsMetadata } from 'services/NetworkService';
import { ExternalNetworkFlowsMetadataResponse } from 'types/networkFlow.proto';
import { getTableUIState } from 'utils/getTableUIState';
import timeWindowToDate from 'utils/timeWindows';

import ExternalIpsTable from './ExternalIpsTable';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

type ExternalIpsContainerProps = {
    scopeHierarchy: NetworkScopeHierarchy;
    onExternalIPSelect: (externalIP: string) => void;
    timeWindow: TimeWindow;
    urlSearchFiltering: UseUrlSearchReturn;
    urlPagination: UseURLPaginationResult;
};

function ExternalIpsContainer({
    scopeHierarchy,
    onExternalIPSelect,
    timeWindow,
    urlSearchFiltering,
    urlPagination,
}: ExternalIpsContainerProps) {
    const clusterId = scopeHierarchy.cluster.id;
    const { namespaces, deployments } = scopeHierarchy;
    const { searchFilter } = urlSearchFiltering;
    const { page, perPage } = urlPagination;
    const fetchExternalIpsFlowsMetadata =
        useCallback((): Promise<ExternalNetworkFlowsMetadataResponse> => {
            const fromTimestamp = timeWindowToDate(timeWindow);
            return getExternalIpsFlowsMetadata(clusterId, namespaces, deployments, fromTimestamp, {
                sortOption: {},
                page,
                perPage,
                advancedFilters: searchFilter,
            });
        }, [clusterId, deployments, namespaces, page, perPage, searchFilter, timeWindow]);

    const {
        data: externalIpsFlowsMetadata,
        isLoading,
        error,
    } = useRestQuery(fetchExternalIpsFlowsMetadata);

    const tableState = getTableUIState({
        isLoading,
        data: externalIpsFlowsMetadata?.entities,
        error,
        searchFilter,
    });

    return (
        <ExternalIpsTable
            onExternalIPSelect={onExternalIPSelect}
            tableState={tableState}
            totalEntities={externalIpsFlowsMetadata?.totalEntities ?? 0}
            urlPagination={urlPagination}
            urlSearchFiltering={urlSearchFiltering}
        />
    );
}

export default ExternalIpsContainer;
