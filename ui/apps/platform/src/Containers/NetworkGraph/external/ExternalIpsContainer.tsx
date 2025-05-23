import React, { useCallback } from 'react';

import { TimeWindow } from 'constants/timeWindows';
import useRestQuery from 'hooks/useRestQuery';
import { getExternalIpsFlowsMetadata } from 'services/NetworkService';
import { ExternalNetworkFlowsMetadataResponse } from 'types/networkFlow.proto';
import { getTableUIState } from 'utils/getTableUIState';

import ExternalIpsTable from './ExternalIpsTable';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';
import { timeWindowToISO } from '../utils/timeWindow';

import { usePagination, useSearchFilterSidePanel } from '../URLStateContext';

type ExternalIpsContainerProps = {
    scopeHierarchy: NetworkScopeHierarchy;
    onExternalIPSelect: (externalIP: string) => void;
    timeWindow: TimeWindow;
};

function ExternalIpsContainer({
    scopeHierarchy,
    onExternalIPSelect,
    timeWindow,
}: ExternalIpsContainerProps) {
    const clusterId = scopeHierarchy.cluster.id;
    const { namespaces, deployments } = scopeHierarchy;
    const { searchFilter } = useSearchFilterSidePanel();
    const { page, perPage } = usePagination();
    const fetchExternalIpsFlowsMetadata =
        useCallback((): Promise<ExternalNetworkFlowsMetadataResponse> => {
            const fromTimestamp = timeWindowToISO(timeWindow);
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
        />
    );
}

export default ExternalIpsContainer;
