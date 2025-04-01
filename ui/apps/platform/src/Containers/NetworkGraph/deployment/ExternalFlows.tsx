import React, { useCallback } from 'react';

import { UseURLPaginationResult } from 'hooks/useURLPagination';
import useRestQuery from 'hooks/useRestQuery';
import { UseUrlSearchReturn } from 'hooks/useURLSearch';
import { getExternalIpsFlowsMetadata } from 'services/NetworkService';
import { ExternalNetworkFlowsMetadataResponse } from 'types/networkFlow.proto';
import { getTableUIState } from 'utils/getTableUIState';

import ExternalIpsTable from '../external/ExternalIpsTable';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

type ExternalFlowsProps = {
    deploymentName: string;
    scopeHierarchy: NetworkScopeHierarchy;
    urlSearchFiltering: UseUrlSearchReturn;
    urlPagination: UseURLPaginationResult;
    onExternalIPSelect: (externalIP: string) => void;
};

function ExternalFlows({
    deploymentName,
    scopeHierarchy,
    urlSearchFiltering,
    urlPagination,
    onExternalIPSelect,
}: ExternalFlowsProps) {
    const clusterId = scopeHierarchy.cluster.id;
    const { namespaces } = scopeHierarchy;
    const { searchFilter } = urlSearchFiltering;
    const { page, perPage } = urlPagination;
    const fetchExternalIpsFlowsMetadata =
        useCallback((): Promise<ExternalNetworkFlowsMetadataResponse> => {
            return getExternalIpsFlowsMetadata(clusterId, namespaces, [deploymentName], {
                sortOption: {},
                page,
                perPage,
                advancedFilters: searchFilter,
            });
        }, [page, perPage, clusterId, deploymentName, namespaces, searchFilter]);

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

export default ExternalFlows;
