import { useCallback, useEffect } from 'react';
import isEmpty from 'lodash/isEmpty';

import useAnalytics, { EXTERNAL_IPS_SIDE_PANEL } from 'hooks/useAnalytics';
import useRestQuery from 'hooks/useRestQuery';
import { getExternalIpsFlowsMetadata } from 'services/NetworkService';
import type { ExternalNetworkFlowsMetadataResponse } from 'types/networkFlow.proto';
import { getTableUIState } from 'utils/getTableUIState';

import ExternalIpsTable from './ExternalIpsTable';
import type { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';
import { timeWindowToISO } from '../utils/timeWindow';

import {
    usePagination,
    useSearchFilterSidePanel,
    useTimeWindow,
} from '../NetworkGraphURLStateContext';

type ExternalIpsContainerProps = {
    scopeHierarchy: NetworkScopeHierarchy;
    onExternalIPSelect: (externalIP: string) => void;
};

function ExternalIpsContainer({ scopeHierarchy, onExternalIPSelect }: ExternalIpsContainerProps) {
    const clusterId = scopeHierarchy.cluster.id;
    const { namespaces, deployments } = scopeHierarchy;

    const { analyticsTrack } = useAnalytics();
    const { searchFilter } = useSearchFilterSidePanel();
    const { timeWindow } = useTimeWindow();
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

    // Can consider removing this track event when network graph gets it's own routing. However, we
    // would lose out on ability to infer if collector feature flag is turned on.
    useEffect(() => {
        if (!isLoading) {
            const isEmptyTable = !externalIpsFlowsMetadata?.totalEntities;
            const isFilteredTable = !isEmpty(searchFilter);

            analyticsTrack({
                event: EXTERNAL_IPS_SIDE_PANEL,
                properties: { isEmptyTable, isFilteredTable },
            });
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [analyticsTrack, externalIpsFlowsMetadata, isLoading]);

    return (
        <ExternalIpsTable
            onExternalIPSelect={onExternalIPSelect}
            tableState={tableState}
            totalEntities={externalIpsFlowsMetadata?.totalEntities ?? 0}
        />
    );
}

export default ExternalIpsContainer;
